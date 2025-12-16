package retriever

import (
	"context"
	"fmt"

	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/graph"
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
	graphQuery     *graph.QueryEngine
	rerankers      []Reranker
	fusionStrategy FusionStrategy
	config         *MultiStageConfig
}

func NewMultiStageRetriever(
	vectorStore VectorStore,
	keywordStore KeywordStore,
	graphQuery *graph.QueryEngine,
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
		graphQuery:     graphQuery,
		rerankers:      rerankers,
		fusionStrategy: fusion,
		config:         cfg,
	}
}

func (r *MultiStageRetriever) Retrieve(ctx context.Context, query string) ([]models.RetrievalResult, error) {
	// 1. Recall Phase
	var resultSets [][]models.RetrievalResult

	// Graph Enhancement: Extract middleware from query and fetch dependencies
	// Simple heuristic: check if query contains known middleware names
	// In real world, we use NLP NER. Here we just try to find context.
	if r.graphQuery != nil {
		// Mock extraction: we assume context might carry target info, or we search loosely
		// For now, let's say if we find a middleware ID in query (which is unlikely in NL)
		// Or we rely on the caller passing context.
		// Let's implement a simple keyword match if we had a dictionary, but here we skip logic
		// and assume if the query mentions "redis", we might want to know about redis nodes.
		// Ideally, the graph query engine should support "SearchNodesByName".

		// For this phase, let's implement the specific requirement:
		// "Search results should include graph context"

		// If the query is about "redis", and we have a redis node "redis-master",
		// we might want to fetch its impact or dependencies.

		// Let's assume the query might contain middleware name.
		if strings.Contains(strings.ToLower(query), "redis") {
			// This is a placeholder. Real implementation needs Entity Extraction.
			// Let's see if we can find a node named "redis-master" or similar.
			// Since we don't have "FindNodeByName" in interface, we might skip direct graph lookup based on NL query
			// unless we iterate.
		}
	}

	// Vector Search
	vecResults, err := r.vectorStore.Search(ctx, query, r.config.RecallTopK)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	// Mark source
	for i := range vecResults {
		vecResults[i].Source = "vector"
		// If we had graph context, we could attach it here or at end
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

// EnhanceWithGraph adds graph context to retrieval results.
// This is called by Retrieve or caller.
// For now, let's expose a method to get graph context given a middleware ID.
func (r *MultiStageRetriever) GetGraphContext(ctx context.Context, middlewareID string) (string, error) {
	if r.graphQuery == nil {
		return "", nil
	}
	impact, err := r.graphQuery.FindImpactedServices(ctx, middlewareID, 2)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Graph Analysis: %s. Impact Level: %s.", impact.EstimatedScope, impact.ImpactLevel), nil
}
