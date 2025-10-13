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

// Package store provides interfaces and implementations for different knowledge storage backends.
package store

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// StoreDocument represents a single, embeddable unit of text (a "chunk") and its
// corresponding vector representation. This is the core data model for storage
// and retrieval in a vector database.
type StoreDocument struct {
	// ID is the unique identifier for this document chunk.
	ID string
	// Content is the text content of the chunk.
	Content string
	// Vector is the numerical representation of the content's semantics.
	Vector []float32
	// Metadata contains additional information about the chunk, such as its source document.
	Metadata map[string]interface{}
	// Score is populated during a search operation and indicates the relevance of the
	// document to the query. It is not a stored field.
	Score float32
}

// VectorStore defines the interface for a vector database. It abstracts the
// storage and retrieval of vector embeddings, allowing for different backends
// (e.g., in-memory, Chroma, Qdrant) to be used interchangeably.
type VectorStore interface {
	// AddDocuments adds a batch of documents (chunks and their vectors) to the store.
	AddDocuments(ctx context.Context, docs []StoreDocument) error
	// SimilaritySearch finds the top K documents in the store that are most
	// semantically similar to a given query vector.
	SimilaritySearch(ctx context.Context, queryVector []float32, topK int) ([]StoreDocument, error)
}

// --- In-Memory Vector Store Implementation ---

// inMemoryVectorStore is a simple, non-production implementation of VectorStore
// that holds all vectors in memory. It is useful for testing and small-scale use cases.
type inMemoryVectorStore struct {
	log       logger.Logger
	documents []StoreDocument
	mu        sync.RWMutex
}

// NewVectorStore is a factory function that creates and returns a VectorStore
// implementation based on the provided configuration. This allows the application
// to easily switch between different vector database backends.
func NewVectorStore(cfg *config.KnowledgeStoreConfig) (VectorStore, error) {
	switch cfg.VectorProvider {
	case "chroma":
		return NewChromaVectorStore(&cfg.Chroma)
	case "in-memory":
		return newInMemoryVectorStore()
	default:
		return nil, fmt.Errorf("unsupported vector store provider: %s", cfg.VectorProvider)
	}
}

// newInMemoryVectorStore creates a new, empty in-memory vector store.
// This implementation is simple and useful for testing or small-scale deployments,
// but it is not durable and performs a brute-force search, making it inefficient
// for large datasets.
//
// Returns:
//   VectorStore: A new instance of an in-memory vector store.
//   error: An error if initialization fails (nil in this implementation).
func newInMemoryVectorStore() (VectorStore, error) {
	return &inMemoryVectorStore{
		log:       logger.NewLogger("in-memory-vector-store"),
		documents: make([]StoreDocument, 0),
	}, nil
}

// AddDocuments appends a batch of documents to the in-memory store in a thread-safe manner.
func (s *inMemoryVectorStore) AddDocuments(_ context.Context, docs []StoreDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log.Infof("Adding %d documents to the in-memory store.", len(docs))
	s.documents = append(s.documents, docs...)
	return nil
}

// SimilaritySearch performs a brute-force k-Nearest-Neighbor (k-NN) search
// over all documents in the store. It calculates the cosine similarity between the
// query vector and every document vector, then returns the top K most similar documents.
// This operation is thread-safe.
func (s *inMemoryVectorStore) SimilaritySearch(_ context.Context, queryVector []float32, topK int) ([]StoreDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.documents) == 0 {
		return []StoreDocument{}, nil
	}
	s.log.Debugf("Performing similarity search for top %d results across %d documents.", topK, len(s.documents))

	// Create a copy of documents to avoid modifying scores on the stored objects directly
	searchableDocs := make([]StoreDocument, len(s.documents))
	copy(searchableDocs, s.documents)

	for i := range searchableDocs {
		score, err := cosineSimilarity(queryVector, searchableDocs[i].Vector)
		if err != nil {
			s.log.Warnf("Could not calculate similarity for doc ID %s: %v", searchableDocs[i].ID, err)
			searchableDocs[i].Score = -1.0 // Penalize on error
		} else {
			searchableDocs[i].Score = score
		}
	}

	// Sort documents by score in descending order.
	sort.Slice(searchableDocs, func(i, j int) bool {
		return searchableDocs[i].Score > searchableDocs[j].Score
	})

	if topK > len(searchableDocs) {
		topK = len(searchableDocs)
	}

	return searchableDocs[:topK], nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
// Result is between -1 and 1. A value of 1 means the vectors are identical.
func cosineSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector lengths do not match: %d vs %d", len(a), len(b))
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("cannot compute similarity with a zero-norm vector")
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB)))), nil
}

// TODO: Implement clients for real vector databases.
// Example for a hypothetical ChromaDB client:
//
// type ChromaVectorStore struct {
//   client *chroma.Client
//   collection *chroma.Collection
// }
//
// func (c *ChromaVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, topK int) ([]StoreDocument, error) {
//   // Logic to call the ChromaDB client's query method...
// }

//Personal.AI order the ending
