package retriever

import (
	"context"
	"sort"

	"github.com/kubestack-ai/kubestack-ai/internal/rag/models"
)

// RRFFusion implements Reciprocal Rank Fusion.
// Formula: RRF(d) = Σ 1/(k + rank_i(d))
type RRFFusion struct {
	k float64
}

// NewRRFFusion creates a new RRFFusion strategy.
// k is a constant, usually set to 60.
func NewRRFFusion(k float64) *RRFFusion {
	if k <= 0 {
		k = 60
	}
	return &RRFFusion{k: k}
}

func (f *RRFFusion) Name() string {
	return "RRF"
}

func (f *RRFFusion) Fuse(ctx context.Context, resultSets [][]models.RetrievalResult) ([]models.RetrievalResult, error) {
	// Map to store RRF scores for each document
	rrfScores := make(map[string]float64)
	// Map to store the document content/metadata to reconstruct the result list
	docMap := make(map[string]models.RetrievalResult)

	for _, resultSet := range resultSets {
		for rank, result := range resultSet {
			score := 1.0 / (f.k + float64(rank+1))
			rrfScores[result.DocID] += score

			if _, exists := docMap[result.DocID]; !exists {
				docMap[result.DocID] = result
			} else {
				// Optionally merge metadata or sources here
				// For simplicity, we keep the first one encountered or update source list
			}
		}
	}

	// Convert map to slice
	fusedResults := make([]models.RetrievalResult, 0, len(rrfScores))
	for docID, score := range rrfScores {
		res := docMap[docID]
		res.Score = score // Update score to RRF score
		fusedResults = append(fusedResults, res)
	}

	// Sort by score descending
	sort.Slice(fusedResults, func(i, j int) bool {
		return fusedResults[i].Score > fusedResults[j].Score
	})

	return fusedResults, nil
}

// WeightedFusion implements Weighted Fusion strategy.
type WeightedFusion struct {
	weights map[string]float64
}

func NewWeightedFusion(weights map[string]float64) *WeightedFusion {
	return &WeightedFusion{weights: weights}
}

func (f *WeightedFusion) Name() string {
	return "Weighted"
}

func (f *WeightedFusion) Fuse(ctx context.Context, resultSets [][]models.RetrievalResult) ([]models.RetrievalResult, error) {
	// Map to store aggregated scores
	scores := make(map[string]float64)
	docMap := make(map[string]models.RetrievalResult)

	for _, resultSet := range resultSets {
		for _, result := range resultSet {
			weight, ok := f.weights[result.Source]
			if !ok {
				weight = 1.0 // Default weight
			}

			// Assuming result.Score is normalized 0-1 or compatible scale
			scores[result.DocID] += result.Score * weight

			if _, exists := docMap[result.DocID]; !exists {
				docMap[result.DocID] = result
			}
		}
	}

	fusedResults := make([]models.RetrievalResult, 0, len(scores))
	for docID, score := range scores {
		res := docMap[docID]
		res.Score = score
		fusedResults = append(fusedResults, res)
	}

	sort.Slice(fusedResults, func(i, j int) bool {
		return fusedResults[i].Score > fusedResults[j].Score
	})

	return fusedResults, nil
}
