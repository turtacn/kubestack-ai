// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rag

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/llm"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
)

// RAGEngine orchestrates the Retrieval-Augmented Generation (RAG) pipeline.
type RAGEngine struct {
	searcher  search.Searcher
	reranker  search.Reranker
	llmClient llm.LLMClient
	cfg       config.RAGEngineConfig
	topK      int
}

// NewRAGEngine creates a new RAGEngine.
func NewRAGEngine(
	knowledgeCfg config.KnowledgeConfig,
	vectorRetriever search.Retriever,
	bm25Searcher *search.BM25Searcher,
	reranker search.Reranker,
	llmClient llm.LLMClient,
) (*RAGEngine, error) {
	var searcher search.Searcher
	var err error

	// Note: We might want to disable internal reranking in HybridSearcher if we do it here.
	// But HybridSearcher uses reranker only if cfg.Reranker.Enabled is true.
	// We assume knowledgeCfg passed here controls that.

	switch knowledgeCfg.Retrieval.Mode {
	case "hybrid":
		searcher, err = search.NewHybridSearcher(vectorRetriever, bm25Searcher, reranker, knowledgeCfg.Retrieval)
	case "semantic":
		searcher, err = search.NewSemanticSearcher(vectorRetriever)
	default:
		return nil, fmt.Errorf("unknown retrieval mode: %s", knowledgeCfg.Retrieval.Mode)
	}

	if err != nil {
		return nil, err
	}

	topK := knowledgeCfg.Retrieval.Reranker.TopK
	if topK <= 0 {
		topK = 5 // Default
	}

	return &RAGEngine{
		searcher:  searcher,
		reranker:  reranker,
		llmClient: llmClient,
		cfg:       knowledgeCfg.RAG.Engine,
		topK:      topK,
	}, nil
}

// GenerateAnswer performs the end-to-end RAG process.
func (e *RAGEngine) GenerateAnswer(ctx context.Context, question string) (string, error) {
	// 1. Retrieve relevant documents
	// The searcher might already do reranking if it is HybridSearcher and configured so.
	// However, the requirement is to explicitly integrate Reranker here.
	docs, err := e.searcher.Search(ctx, question)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve documents: %w", err)
	}

	// Explicitly rerank if reranker is available and documents are retrieved
	if e.reranker != nil && len(docs) > 0 {
		// Searcher returns []Document (values), but Rerank needs []*Document.
		docPtrs := make([]*search.Document, len(docs))
		for i := range docs {
			docPtrs[i] = &docs[i]
		}

		rerankedDocs, err := e.reranker.Rerank(ctx, question, docPtrs, e.topK)
		if err != nil {
			// Log warning but continue with original docs (or maybe sliced original docs)
			// For now, we return error as per usual go practice unless we want to fail open.
			// Let's just fallback to original docs sliced to topK if rerank fails?
			// The task implies adding reranking logic.
			return "", fmt.Errorf("failed to rerank documents: %w", err)
		}

		// Convert back to []Document
		docs = make([]search.Document, len(rerankedDocs))
		for i, doc := range rerankedDocs {
			docs[i] = *doc
		}
	} else {
		// If no reranker, just ensure we respect TopK
		if len(docs) > e.topK {
			docs = docs[:e.topK]
		}
	}

	// 2. Build the prompt with the retrieved context
	tmpl := prompt.Template{
		Name:    "rag",
		Content: "Answer the following question based on the provided context:\n\n{{range .context}}Context: {{.}}\n{{end}}\nQuestion: {{.question}}",
	}
	builder, err := prompt.NewBuilder(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to create prompt builder: %w", err)
	}

	docContents := make([]string, len(docs))
	for i, doc := range docs {
		docContents[i] = doc.Content
	}

	promptData, err := builder.
		WithData("context", docContents).
		WithData("question", question).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build prompt: %w", err)
	}

	// 3. Generate the answer using the language model
	answer, err := e.llmClient.Generate(ctx, promptData)
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}

	return answer, nil
}
