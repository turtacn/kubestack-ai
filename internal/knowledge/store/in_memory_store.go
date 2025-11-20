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

package store

import (
	"context"
	"sort"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/common/utils"
)

// InMemoryVectorStore is a simple in-memory vector store for testing and development.
type InMemoryVectorStore struct {
	mu   sync.RWMutex
	docs []StoreDocument
}

// NewInMemoryVectorStore creates a new in-memory vector store.
func NewInMemoryVectorStore() (VectorStore, error) {
	return &InMemoryVectorStore{
		docs: make([]StoreDocument, 0),
	}, nil
}

// AddDocuments adds documents to the in-memory store.
func (s *InMemoryVectorStore) AddDocuments(ctx context.Context, docs []StoreDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs = append(s.docs, docs...)
	return nil
}

// SimilaritySearch performs a similarity search in the in-memory store.
func (s *InMemoryVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, topK int) ([]StoreDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type docWithScore struct {
		doc   StoreDocument
		score float32
	}

	var scoredDocs []docWithScore
	for _, doc := range s.docs {
		score, err := utils.CosineSimilarity(queryVector, doc.Vector)
		if err != nil {
			return nil, err
		}
		scoredDocs = append(scoredDocs, docWithScore{doc: doc, score: score})
	}

	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	var results []StoreDocument
	for i := 0; i < topK && i < len(scoredDocs); i++ {
		results = append(results, scoredDocs[i].doc)
	}

	return results, nil
}
