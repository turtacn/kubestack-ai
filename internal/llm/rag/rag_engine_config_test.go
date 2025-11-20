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
	"os"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type mockLLMClient struct{}

func (m *mockLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	return "mock answer", nil
}

type mockReranker struct{}

func (m *mockReranker) Rerank(ctx context.Context, query string, candidates []*search.Document, topK int) ([]*search.Document, error) {
	return candidates, nil
}

func TestRAGEngine_WithHybridConfig(t *testing.T) {
	// Create a test config file
	cfg := config.KnowledgeConfig{
		Retrieval: config.RetrievalConfig{
			Mode: "hybrid",
			Semantic: config.SemanticConfig{
				TopK: 10,
			},
			Keyword: config.KeywordConfig{
				TopK: 10,
			},
			Fusion: config.FusionConfig{
				Strategy: "rrf",
				RRF:      config.RRFConfig{K: 60},
			},
			Reranker: config.RerankerConfig{
				Enabled: false,
			},
		},
		RAG: config.RAGConfig{
			Engine: config.RAGEngineConfig{
				MaxContextTokens: 4096,
				MaxChunks:        10,
			},
		},
	}

	configFile, err := os.CreateTemp("", "knowledge.yaml")
	assert.NoError(t, err)
	defer os.Remove(configFile.Name())

	encoder := yaml.NewEncoder(configFile)
	err = encoder.Encode(map[string]config.KnowledgeConfig{"knowledge": cfg})
	assert.NoError(t, err)
	encoder.Close()

	// Create mock components
	indexPath := "./test_bm25.index"
	defer os.RemoveAll(indexPath)
	bm25Searcher, err := search.NewBM25Searcher(indexPath)
	assert.NoError(t, err)

	docs := []*store.StoreDocument{
		{ID: "1", Content: "keyword result 1"},
	}
	err = bm25Searcher.IndexDocuments(docs)
	assert.NoError(t, err)

	vectorRetriever := &mockRetriever{}
	reranker := &mockReranker{}
	llmClient := &mockLLMClient{}

	// Create the RAGEngine
	engine, err := NewRAGEngine(cfg, vectorRetriever, bm25Searcher, reranker, llmClient)
	assert.NoError(t, err)

	// Test the RAGEngine
	answer, err := engine.GenerateAnswer(context.Background(), "keyword")
	assert.NoError(t, err)
	assert.Equal(t, "mock answer", answer)
}

type mockRetriever struct{}

func (m *mockRetriever) Retrieve(ctx context.Context, query string, topK int) ([]search.Document, error) {
	return []search.Document{
		{Content: "semantic result 1"},
	}, nil
}

func (m *mockRetriever) HybridRetrieve(ctx context.Context, query string, opts *search.RetrieveOptions) ([]search.Document, error) {
	return m.Retrieve(ctx, query, opts.TopK)
}
