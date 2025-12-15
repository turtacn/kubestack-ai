package retriever

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/rag/models"
)

// Store interfaces for dependency injection
type VectorStore interface {
	Search(ctx context.Context, query string, topK int) ([]models.RetrievalResult, error)
}

type KeywordStore interface {
	Search(ctx context.Context, query string, topK int) ([]models.RetrievalResult, error)
}

type MultiStageConfig struct {
	RecallTopK    int
	CoarseTopK    int
	FineTopK      int
	FinalTopK     int
	MinScore      float64
	EnableKeyword bool
}

type MultiStageRetriever struct {
	vectorStore    VectorStore
	keywordStore   KeywordStore
	rerankers      []Reranker
	fusionStrategy FusionStrategy
	config         *MultiStageConfig
}

func NewMultiStageRetriever(
	vectorStore VectorStore,
	keywordStore KeywordStore,
	rerankers []Reranker,
	fusion FusionStrategy,
	cfg *MultiStageConfig,
) *MultiStageRetriever {
	if cfg == nil {
		cfg = &MultiStageConfig{
			RecallTopK: 100,
			CoarseTopK: 50,
			FineTopK:   20,
			FinalTopK:  10,
			MinScore:   0.5,
		}
	}
	return &MultiStageRetriever{
		vectorStore:    vectorStore,
		keywordStore:   keywordStore,
		rerankers:      rerankers,
		fusionStrategy: fusion,
		config:         cfg,
	}
}

func (r *MultiStageRetriever) Retrieve(ctx context.Context, query string) ([]models.RetrievalResult, error) {
	// 1. Recall Phase
	var resultSets [][]models.RetrievalResult

	// Vector Search
	vecResults, err := r.vectorStore.Search(ctx, query, r.config.RecallTopK)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	// Mark source
	for i := range vecResults {
		vecResults[i].Source = "vector"
	}
	resultSets = append(resultSets, vecResults)

	// Keyword Search (if enabled)
	if r.config.EnableKeyword && r.keywordStore != nil {
		kwResults, err := r.keywordStore.Search(ctx, query, r.config.RecallTopK)
		if err != nil {
			// Log error but continue? Or fail? Let's log and continue for robustness.
			// fmt.Printf("keyword search failed: %v\n", err)
		} else {
			for i := range kwResults {
				kwResults[i].Source = "keyword"
			}
			resultSets = append(resultSets, kwResults)
		}
	}

	// 2. Fusion / Coarse Ranking
	var merged []models.RetrievalResult
	if r.fusionStrategy != nil {
		merged, err = r.fusionStrategy.Fuse(ctx, resultSets)
		if err != nil {
			return nil, fmt.Errorf("fusion failed: %w", err)
		}
	} else {
		// Default merge if no strategy: just append and dedup?
		// For now assume fusion strategy is always provided or we fallback to just vector results
		if len(resultSets) > 0 {
			merged = resultSets[0] // Simple fallback
		}
	}

	// Apply Coarse TopK and MinScore (pre-rerank filtering)
	if len(merged) > r.config.CoarseTopK {
		merged = merged[:r.config.CoarseTopK]
	}

	// Filter by MinScore
	filtered := make([]models.RetrievalResult, 0, len(merged))
	for _, res := range merged {
		if res.Score >= r.config.MinScore {
			filtered = append(filtered, res)
		}
	}
	merged = filtered

	// 3. Fine Ranking (Reranking)
	for _, reranker := range r.rerankers {
		merged, err = reranker.Rerank(ctx, query, merged)
		if err != nil {
			return nil, fmt.Errorf("reranking failed with %s: %w", reranker.Name(), err)
		}

		// Apply FineTopK after each reranker? Or just at the end?
		// Usually good to prune if we have multiple rerankers
		if len(merged) > r.config.FineTopK {
			merged = merged[:r.config.FineTopK]
		}
	}

	// 4. Final Cut
	if len(merged) > r.config.FinalTopK {
		merged = merged[:r.config.FinalTopK]
	}

	return merged, nil
}
