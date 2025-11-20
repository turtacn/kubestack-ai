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
	"os"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store"
)

// BM25Searcher provides keyword-based search functionality using the BM25 algorithm.
type BM25Searcher struct {
	index bleve.Index
	mu    sync.RWMutex
}

// NewBM25Searcher creates a new BM25Searcher.
func NewBM25Searcher(indexPath string) (*BM25Searcher, error) {
	var index bleve.Index
	var err error

	if _, statErr := os.Stat(indexPath); os.IsNotExist(statErr) {
		mapping := bleve.NewIndexMapping()
		mapping.DefaultAnalyzer = JiebaAnalyzerName
		index, err = bleve.New(indexPath, mapping)
	} else {
		index, err = bleve.Open(indexPath)
	}

	if err != nil {
		return nil, err
	}

	return &BM25Searcher{index: index}, nil
}

// IndexDocuments indexes a batch of documents.
func (s *BM25Searcher) IndexDocuments(docs []*store.StoreDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	batch := s.index.NewBatch()
	for _, doc := range docs {
		if err := batch.Index(doc.ID, doc); err != nil {
			return err
		}
	}
	s.index.Batch(batch)
	return nil
}

// Search performs a keyword search and returns the top K results.
func (s *BM25Searcher) Search(query string, topK int) ([]*Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	searchRequest := bleve.NewSearchRequest(bleve.NewMatchQuery(query))
	searchRequest.Size = topK
	searchRequest.Fields = []string{"*"}
	searchResult, err := s.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var results []*Document
	for _, hit := range searchResult.Hits {
		var content string
		if c, ok := hit.Fields["Content"].(string); ok {
			content = c
		}

		var metadata map[string]interface{}
		if m, ok := hit.Fields["Metadata"].(map[string]interface{}); ok {
			metadata = m
		}

		doc := &Document{
			Content:  content,
			Metadata: metadata,
			Score:    float32(hit.Score),
		}
		results = append(results, doc)
	}

	return results, nil
}
