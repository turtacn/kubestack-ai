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

// Package rag implements the Retrieval-Augmented Generation components.
package rag

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store" // Will be created later
)

// Document represents a single chunk of retrieved information, ready to be injected into an LLM prompt.
type Document struct {
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
	Score    float32                `json:"score"` // The relevance score from the search.
}

// Retriever is the interface for retrieving relevant documents from a knowledge base in response to a query.
type Retriever interface {
	Retrieve(ctx context.Context, query string, topK int) ([]Document, error)
}

// vectorRetriever implements the Retriever interface using an embedder and a vector store.
// This is the core of semantic search.
type vectorRetriever struct {
	log         logger.Logger
	embedder    Embedder      // From embedder.go
	vectorStore store.VectorStore // From knowledge/store/vector_store.go
}

// NewRetriever creates a new vector-based retriever. It requires an embedder to convert
// the query to a vector and a vector store to search for similar document vectors.
func NewRetriever(embedder Embedder, vectorStore store.VectorStore) (Retriever, error) {
	if embedder == nil {
		return nil, fmt.Errorf("embedder cannot be nil")
	}
	if vectorStore == nil {
		return nil, fmt.Errorf("vectorStore cannot be nil")
	}
	return &vectorRetriever{
		log:         logger.NewLogger("retriever"),
		embedder:    embedder,
		vectorStore: vectorStore,
	}, nil
}

// Retrieve finds the top K most relevant documents for a given query using semantic search.
func (r *vectorRetriever) Retrieve(ctx context.Context, query string, topK int) ([]Document, error) {
	r.log.Infof("Retrieving top %d documents for query: %s", topK, query)

	// 1. Convert the natural language query into a vector embedding.
	queryVector, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// 2. Use the vector store to find the most similar document vectors.
	// The vector store handles the actual similarity search (e.g., using ANN).
	similarDocs, err := r.vectorStore.SimilaritySearch(ctx, queryVector, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to perform similarity search in vector store: %w", err)
	}

	// 3. Convert the store's document format to our RAG document format.
	docs := make([]Document, len(similarDocs))
	for i, sDoc := range similarDocs {
		docs[i] = Document{
			Content:  sDoc.Content,
			Metadata: sDoc.Metadata,
			Score:    sDoc.Score,
		}
	}

	// TODO: Implement hybrid search by merging these results with a traditional keyword search (e.g., from Elasticsearch).
	// TODO: Implement re-ranking of results based on more complex relevance criteria (e.g., using a cross-encoder).

	r.log.Infof("Successfully retrieved %d documents.", len(docs))
	return docs, nil
}

//Personal.AI order the ending
