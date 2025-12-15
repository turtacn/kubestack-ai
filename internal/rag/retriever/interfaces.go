package retriever

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/rag/models"
)

// Reranker is the interface for re-ranking retrieval results.
type Reranker interface {
	// Rerank re-ranks the candidate results.
	// It returns a new slice of results, sorted by the new score.
	Rerank(ctx context.Context, query string, candidates []models.RetrievalResult) ([]models.RetrievalResult, error)

	// Name returns the name of the reranker.
	Name() string
}

// FusionStrategy is the interface for merging results from multiple retrieval sources.
type FusionStrategy interface {
	// Fuse merges multiple sets of retrieval results into a single set.
	Fuse(ctx context.Context, resultSets [][]models.RetrievalResult) ([]models.RetrievalResult, error)

	// Name returns the name of the fusion strategy.
	Name() string
}
