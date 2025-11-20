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
	"os"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store"
	"github.com/stretchr/testify/assert"
)

type mockRetriever struct{}

func (m *mockRetriever) Retrieve(ctx context.Context, query string, topK int) ([]Document, error) {
	return []Document{
		{Content: "semantic result 1"},
	}, nil
}

func (m *mockRetriever) HybridRetrieve(ctx context.Context, query string, opts *RetrieveOptions) ([]Document, error) {
	return m.Retrieve(ctx, query, opts.TopK)
}

type mockReranker struct{}

func (m *mockReranker) Rerank(ctx context.Context, query string, candidates []*Document, topK int) ([]*Document, error) {
	return candidates, nil
}

func TestHybridSearcher(t *testing.T) {
	indexPath := "./test_bm25.index"
	defer os.RemoveAll(indexPath)

	bm25Searcher, err := NewBM25Searcher(indexPath)
	assert.NoError(t, err)

	docs := []*store.StoreDocument{
		{ID: "1", Content: "keyword result 1"},
	}
	err = bm25Searcher.IndexDocuments(docs)
	assert.NoError(t, err)

	cfg := config.RetrievalConfig{
		Semantic: config.SemanticConfig{TopK: 10},
		Keyword:  config.KeywordConfig{TopK: 10},
		Fusion: config.FusionConfig{
			Strategy: "rrf",
			RRF:      config.RRFConfig{K: 60},
		},
		Reranker: config.RerankerConfig{Enabled: false},
	}

	hybridSearcher, err := NewHybridSearcher(&mockRetriever{}, bm25Searcher, &mockReranker{}, cfg)
	assert.NoError(t, err)

	results, err := hybridSearcher.Search(context.Background(), "keyword")
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	contents := make([]string, len(results))
	for i, doc := range results {
		contents[i] = doc.Content
	}
	assert.Contains(t, contents, "semantic result 1")
	assert.Contains(t, contents, "keyword result 1")
}
