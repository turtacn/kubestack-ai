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
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store" // Will be created later
)

// vectorRetriever implements the Retriever interface using an embedder and a vector store.
// This is the core of semantic search.
type vectorRetriever struct {
	log         logger.Logger
	embedder    Embedder      // From embedder.go
	vectorStore store.VectorStore // From knowledge/store/vector_store.go
}

// NewRetriever creates a new vector-based retriever. It composes an embedder
// (to convert the query to a vector) and a vector store (to search for similar
// document vectors), which together form the core of a semantic search pipeline.
//
// Parameters:
//   embedder (Embedder): The component used to create query embeddings.
//   vectorStore (store.VectorStore): The vector database to search against.
//
// Returns:
//   search.Retriever: A new instance of a vector-based retriever.
//   error: An error if either the embedder or vector store is nil.
func NewRetriever(embedder Embedder, vectorStore store.VectorStore) (search.Retriever, error) {
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

// Retrieve implements the Retriever interface. It finds the top K most relevant
// documents for a given query by first converting the query into a vector
// embedding and then using that vector to perform a similarity search in the vector store.
//
// Parameters:
//   ctx (context.Context): The context for the retrieval operation.
//   query (string): The natural language user query.
//   topK (int): The number of top matching documents to retrieve.
//
// Returns:
//   []search.Document: A ranked list of the most relevant documents.
//   error: An error if embedding the query or searching the vector store fails.
func (r *vectorRetriever) Retrieve(ctx context.Context, query string, topK int) ([]search.Document, error) {
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
	docs := make([]search.Document, len(similarDocs))
	for i, sDoc := range similarDocs {
		docs[i] = search.Document{
			Content:  sDoc.Content,
			Metadata: sDoc.Metadata,
			Score:    sDoc.Score,
		}
	}

	r.log.Infof("Successfully retrieved %d documents.", len(docs))
	return docs, nil
}

// HybridRetrieve implements the Retriever interface for hybrid search.
// For the vectorRetriever, this will just call the regular Retrieve method.
func (r *vectorRetriever) HybridRetrieve(ctx context.Context, query string, opts *search.RetrieveOptions) ([]search.Document, error) {
	return r.Retrieve(ctx, query, opts.TopK)
}
