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

package search

import (
	"context"
	"sort"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/rag"
)

// hybridSearcher implements the Searcher interface by combining results from
// a keyword-based search (lexical) and a vector-based search (semantic).
type hybridSearcher struct {
	log               logger.Logger
	keywordStore      store.DocumentStore
	semanticRetriever rag.Retriever
}

// NewHybridSearcher creates a new hybrid search component.
func NewHybridSearcher(keywordStore store.DocumentStore, semanticRetriever rag.Retriever) (Searcher, error) {
	return &hybridSearcher{
		log:               logger.NewLogger("hybrid-searcher"),
		keywordStore:      keywordStore,
		semanticRetriever: semanticRetriever,
	}, nil
}

// Search executes a keyword and a semantic search in parallel and fuses the results.
func (s *hybridSearcher) Search(ctx context.Context, query string) ([]rag.Document, error) {
	s.log.Infof("Performing hybrid search for query: %.50s...", query)

	var keywordResults []*store.RawDocument
	var semanticResults []rag.Document
	var wg sync.WaitGroup
	var errKeyword, errSemantic error

	wg.Add(2)

	// Run keyword search in a goroutine.
	go func() {
		defer wg.Done()
		keywordResults, errKeyword = s.keywordStore.Search(ctx, query, nil)
	}()

	// Run semantic search in a goroutine.
	go func() {
		defer wg.Done()
		const topK = 10 // Retrieve more results for better fusion potential.
		semanticResults, errSemantic = s.semanticRetriever.Retrieve(ctx, query, topK)
	}()

	wg.Wait()

	if errKeyword != nil {
		s.log.Warnf("Keyword search failed during hybrid search: %v", errKeyword)
	}
	if errSemantic != nil {
		s.log.Warnf("Semantic search failed during hybrid search: %v", errSemantic)
	}

	// Fuse the results from both searches.
	fusedDocs := s.reciprocalRankFusion(keywordResults, semanticResults)

	return fusedDocs, nil
}

// reciprocalRankFusion combines two lists of ranked results using the RRF algorithm.
// RRF is effective because it doesn't require tuning weights and focuses on rank order.
func (s *hybridSearcher) reciprocalRankFusion(keywordDocs []*store.RawDocument, semanticDocs []rag.Document) []rag.Document {
	// RRF Score = sum over result lists ( 1 / (k + rank) )
	// 'k' is a constant to mitigate the impact of high ranks; 60 is a common value.
	const rrfK = 60

	scores := make(map[string]float32)
	docsMap := make(map[string]rag.Document)

	// Process semantic results.
	for i, doc := range semanticDocs {
		sourceID, ok := doc.Metadata["SourceID"].(string)
		if !ok {
			continue // Skip chunks without a source document ID.
		}
		rank := i + 1
		score := 1.0 / float32(rrfK+rank)
		scores[sourceID] += score
		// Keep the first chunk we see from a source document.
		if _, exists := docsMap[sourceID]; !exists {
			docsMap[sourceID] = doc
		}
	}

	// Process keyword results.
	for i, doc := range keywordDocs {
		rank := i + 1
		score := 1.0 / float32(rrfK+rank)
		scores[doc.ID] += score
		// If we haven't seen this document from the semantic search, add it.
		// Note: The content here is the full document, not a chunk.
		if _, exists := docsMap[doc.ID]; !exists {
			docsMap[doc.ID] = rag.Document{
				Content: doc.Content,
				Metadata: map[string]interface{}{
					"SourceID":  doc.ID,
					"SourceURL": doc.Source,
				},
			}
		}
	}

	// Create a single list of documents with their new RRF scores.
	var fusedDocs []rag.Document
	for id, score := range scores {
		doc := docsMap[id]
		doc.Score = score
		fusedDocs = append(fusedDocs, doc)
	}

	// Sort the final list by the new RRF score in descending order.
	sort.Slice(fusedDocs, func(i, j int) bool {
		return fusedDocs[i].Score > fusedDocs[j].Score
	})

	return fusedDocs
}

//Personal.AI order the ending
