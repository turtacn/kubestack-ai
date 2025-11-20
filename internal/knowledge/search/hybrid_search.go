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

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"golang.org/x/sync/errgroup"
)

// HybridSearcher orchestrates a hybrid search by combining results from a
// semantic retriever and a keyword-based searcher (like BM25).
type HybridSearcher struct {
	vectorRetriever Retriever
	bm25Searcher    *BM25Searcher
	fusionStrategy  FusionStrategy
	reranker        Reranker
	cfg             config.RetrievalConfig
}

// NewHybridSearcher creates a new HybridSearcher.
func NewHybridSearcher(
	vectorRetriever Retriever,
	bm25Searcher *BM25Searcher,
	reranker Reranker,
	cfg config.RetrievalConfig,
) (Searcher, error) {
	var fusionStrategy FusionStrategy
	switch cfg.Fusion.Strategy {
	case "rrf":
		fusionStrategy = NewRRFFusion(cfg.Fusion.RRF.K)
	case "weighted":
		fusionStrategy = NewWeightedFusion([]float64{
			cfg.Fusion.Weighted.SemanticWeight,
			cfg.Fusion.Weighted.KeywordWeight,
		})
	default:
		return nil, fmt.Errorf("unknown fusion strategy: %s", cfg.Fusion.Strategy)
	}

	return &HybridSearcher{
		vectorRetriever: vectorRetriever,
		bm25Searcher:    bm25Searcher,
		fusionStrategy:  fusionStrategy,
		reranker:        reranker,
		cfg:             cfg,
	}, nil
}

// Search performs a hybrid search.
func (h *HybridSearcher) Search(ctx context.Context, query string) ([]Document, error) {
	g, gCtx := errgroup.WithContext(ctx)

	var semanticResults []*Document
	var bm25Results []*Document

	g.Go(func() error {
		var err error
		docs, err := h.vectorRetriever.Retrieve(gCtx, query, h.cfg.Semantic.TopK)
		for _, doc := range docs {
			semanticResults = append(semanticResults, &doc)
		}
		return err
	})

	g.Go(func() error {
		var err error
		bm25Results, err = h.bm25Searcher.Search(query, h.cfg.Keyword.TopK)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	fusedResults := h.fusionStrategy.Fuse(semanticResults, bm25Results)

	if h.cfg.Reranker.Enabled {
		rerankedDocs, err := h.reranker.Rerank(ctx, query, fusedResults, h.cfg.Reranker.TopK)
		if err != nil {
			return nil, err
		}
		fusedResults = rerankedDocs
	}

	finalDocs := make([]Document, 0, len(fusedResults))
	for _, doc := range fusedResults {
		finalDocs = append(finalDocs, *doc)
	}

	return finalDocs, nil
}
