package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/henomis/lingoose/embedder"
	"github.com/henomis/lingoose/index"
	"github.com/henomis/lingoose/index/vectordb/pinecone"
	"github.com/henomis/lingoose/textsplitter"
	"github.com/turtacn/kubestack-ai/internal/constants"
	appErr "github.com/turtacn/kubestack-ai/internal/errors"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/models"
)

// RAG 接口定义检索增强生成
type RAG interface {
	EmbedAndStore(ctx context.Context, docs map[string][]models.KnowledgeDocument) error
	Retrieve(ctx context.Context, query string, middleware string) (string, error)
	AddDocument(ctx context.Context, doc models.KnowledgeDocument) error
	GetDocumentCount(middleware string) int
}

type rag struct {
	embedder    embedder.Embedder
	index       index.Index
	splitter    textsplitter.TextSplitter
	documentMap map[string][]models.KnowledgeDocument
}

func NewRAG() RAG {
	embed := embedder.NewOpenAI()

	split := textsplitter.NewRecursiveCharacterTextSplitter(
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(200),
	)

	// 默认使用内存索引
	idx := index.NewInMemory()

	// 如需 Pinecone，取消注释：
	/*
		idx := pinecone.New(
			pinecone.WithAPIKey("YOUR_API_KEY"),
			pinecone.WithEnvironment("YOUR_ENVIRONMENT"),
			pinecone.WithIndexName("kubestack-ai-knowledge"),
		)
	*/

	return &rag{
		embedder:    embed,
		index:       idx,
		splitter:    split,
		documentMap: make(map[string][]models.KnowledgeDocument),
	}
}

func (r *rag) EmbedAndStore(ctx context.Context, docs map[string][]models.KnowledgeDocument) error {
	logging.Logger.Info("Embedding and storing knowledge documents")

	for middleware, documents := range docs {
		logging.Logger.Infof("Processing %d documents for %s", len(documents), middleware)

		for _, doc := range documents {
			doc.Middleware = middleware
			if doc.ID == "" {
				doc.ID = uuid.New().String()
			}
			if doc.CreatedAt.IsZero() {
				doc.CreatedAt = time.Now()
			}

			chunks, err := r.splitter.SplitText(doc.Content)
			if err != nil {
				logging.Logger.Errorf("Failed to split document %s: %v", doc.ID, err)
				continue
			}

			for i, chunk := range chunks {
				embedding, err := r.embedder.Embed(ctx, chunk)
				if err != nil {
					logging.Logger.Errorf("Failed to embed chunk %d of document %s: %v", i, doc.ID, err)
					continue
				}

				err = r.index.Upsert(ctx, []index.Record{
					{
						ID:     fmt.Sprintf("%s-chunk-%d", doc.ID, i),
						Vector: embedding,
						Metadata: map[string]interface{}{
							"document_id": doc.ID,
							"middleware":  middleware,
							"source":      doc.Source,
							"content":     chunk,
						},
					},
				})
				if err != nil {
					logging.Logger.Errorf("Failed to store chunk %d of document %s: %v", i, doc.ID, err)
				}
			}

			r.documentMap[middleware] = append(r.documentMap[middleware], doc)
			logging.Logger.Debugf("Processed document %s for %s", doc.ID, middleware)
		}
	}

	logging.Logger.Info("Completed embedding and storing knowledge documents")
	return nil
}

func (r *rag) Retrieve(ctx context.Context, query string, middleware string) (string, error) {
	logging.Logger.Debugf("Retrieving knowledge for query: %s (middleware: %s)", query, middleware)

	queryEmbedding, err := r.embedder.Embed(ctx, query)
	if err != nil {
		logging.Logger.Errorf("Failed to create query embedding: %v", err)
		return "", appErr.ErrRAGRetrievalFailed
	}

	params := index.SearchParams{
		TopK: 5,
		Filter: map[string]interface{}{
			"middleware": middleware,
		},
	}

	results, err := r.index.Search(ctx, queryEmbedding, params)
	if err != nil {
		logging.Logger.Errorf("Failed to search for similar embeddings: %v", err)
		return "", appErr.ErrRAGRetrievalFailed
	}

	if len(results) == 0 {
		logging.Logger.Debug("No relevant knowledge found")
		return "No relevant knowledge found", nil
	}

	var contextBuilder strings.Builder
	contextBuilder.WriteString("Relevant knowledge:\n\n")

	for i, result := range results {
		content, ok := result.Metadata["content"].(string)
		if !ok {
			continue
		}

		source, _ := result.Metadata["source"].(string)
		contextBuilder.WriteString(fmt.Sprintf(
			"Chunk %d (source: %s, similarity: %.2f):\n",
			i+1, source, result.Score))
		contextBuilder.WriteString(content)
		contextBuilder.WriteString("\n\n")
	}

	retrievedContext := contextBuilder.String()
	logging.Logger.Debugf("Retrieved %d knowledge chunks", len(results))
	return retrievedContext, nil
}

func (r *rag) AddDocument(ctx context.Context, doc models.KnowledgeDocument) error {
	if doc.Middleware == "" {
		return errors.New("document must specify middleware")
	}
	if doc.Content == "" {
		return errors.New("document content cannot be empty")
	}

	return r.EmbedAndStore(ctx, map[string][]models.KnowledgeDocument{
		doc.Middleware: {doc},
	})
}

func (r *rag) GetDocumentCount(middleware string) int {
	docs, ok := r.documentMap[middleware]
	if !ok {
		return 0
	}
	return len(docs)
}
