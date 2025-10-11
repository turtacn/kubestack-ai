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
	"fmt"
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

// NewHybridSearcher creates a new hybrid searcher that combines lexical search
// (from a document store) and semantic search (from a retriever). This approach
// leverages the strengths of both methods to provide more relevant results.
//
// Parameters:
//   keywordStore (store.DocumentStore): The document store to use for keyword-based search.
//   semanticRetriever (rag.Retriever): The retriever to use for vector-based semantic search.
//
// Returns:
//   Searcher: A new instance of a hybrid searcher.
//   error: An error if initialization fails (nil in this implementation).
func NewHybridSearcher(keywordStore store.DocumentStore, semanticRetriever rag.Retriever) (Searcher, error) {
	return &hybridSearcher{
		log:               logger.NewLogger("hybrid-searcher"),
		keywordStore:      keywordStore,
		semanticRetriever: semanticRetriever,
	}, nil
}

// Search implements the Searcher interface. It executes a keyword search and a
// semantic search in parallel for a given query. It then fuses the results from
// both searches using the Reciprocal Rank Fusion (RRF) algorithm to produce a
// single, re-ranked list of relevant documents.
//
// Parameters:
//   ctx (context.Context): The context for the search operations.
//   query (string): The user's search query.
//
// Returns:
//   []rag.Document: A slice of documents, sorted by their fused relevance score.
//   error: An error if both the keyword and semantic searches fail.
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

	// If both searches fail, return an error.
	if errKeyword != nil && errSemantic != nil {
		return nil, fmt.Errorf("both keyword and semantic searches failed (keyword: %w, semantic: %w)", errKeyword, errSemantic)
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
		// Use the document chunk's content as the key for fusion.
		key := doc.Content
		rank := i + 1
		score := 1.0 / float32(rrfK+rank)
		scores[key] += score
		docsMap[key] = doc
	}

	// Process keyword results by breaking them into chunks (simplified for now).
	// In a real system, you would have a more sophisticated chunking strategy.
	for i, doc := range keywordDocs {
		// Simplified chunking: treat the first 512 characters as the most relevant chunk.
		chunkContent := doc.Content
		if len(chunkContent) > 512 {
			chunkContent = chunkContent[:512]
		}

		key := chunkContent
		rank := i + 1
		score := 1.0 / float32(rrfK+rank)

		if existingScore, exists := scores[key]; exists {
			// If the chunk already exists from semantic search, just add to its score.
			scores[key] = existingScore + score
		} else {
			// If it's a new chunk, add it to the map.
			scores[key] = score
			docsMap[key] = rag.Document{
				Content: chunkContent,
				Metadata: map[string]interface{}{
					"SourceID":  doc.ID,
					"SourceURL": doc.Source,
				},
			}
		}
	}

	// Create a single list of documents with their new RRF scores.
	var fusedDocs []rag.Document
	for key, score := range scores {
		doc := docsMap[key]
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
