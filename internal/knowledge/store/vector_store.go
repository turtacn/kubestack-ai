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

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// StoreDocument represents a single document (a text chunk and its vector)
// as it is stored in the vector database.
type StoreDocument struct {
	ID       string
	Content  string
	Vector   []float32
	Metadata map[string]interface{}
	Score    float32 // Used for returning search results, not for storage.
}

// VectorStore is the interface for any vector database implementation.
// It abstracts the storage and retrieval of vector embeddings, allowing for different
// backends like in-memory, Chroma, or Qdrant.
type VectorStore interface {
	AddDocuments(ctx context.Context, docs []StoreDocument) error
	SimilaritySearch(ctx context.Context, queryVector []float32, topK int) ([]StoreDocument, error)
	// TODO: Add other methods like DeleteDocuments, UpdateDocuments if needed.
}

// --- In-Memory Vector Store Implementation ---

// inMemoryVectorStore is a simple, non-production implementation of VectorStore
// that holds all vectors in memory. It is useful for testing and small-scale use cases.
type inMemoryVectorStore struct {
	log       logger.Logger
	documents []StoreDocument
	mu        sync.RWMutex
}

// NewInMemoryVectorStore creates a new, empty in-memory vector store.
func NewInMemoryVectorStore() (VectorStore, error) {
	return &inMemoryVectorStore{
		log:       logger.NewLogger("in-memory-vector-store"),
		documents: make([]StoreDocument, 0),
	}, nil
}

// AddDocuments adds a batch of documents to the in-memory store in a thread-safe manner.
func (s *inMemoryVectorStore) AddDocuments(_ context.Context, docs []StoreDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log.Infof("Adding %d documents to the in-memory store.", len(docs))
	s.documents = append(s.documents, docs...)
	return nil
}

// SimilaritySearch performs a brute-force k-Nearest-Neighbor search using cosine similarity.
// While inefficient for large datasets, it's perfect for a simple implementation.
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
