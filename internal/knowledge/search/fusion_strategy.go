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
	"sort"
)

// FusionStrategy defines the interface for strategies that combine and re-rank
// search results from multiple sources (e.g., semantic and keyword search).
type FusionStrategy interface {
	// Fuse takes multiple lists of search results and merges them into a single,
	// re-ranked list based on the specific fusion algorithm.
	Fuse(results ...[]*Document) []*Document
}

// --- Reciprocal Rank Fusion (RRF) Implementation ---

// RRFFusion implements the FusionStrategy using Reciprocal Rank Fusion.
// RRF is a simple yet effective technique that scores documents based on their
// rank in the original result lists, without needing to normalize the scores.
type RRFFusion struct {
	K int // A constant used in the RRF formula, typically set to 60.
}

// NewRRFFusion creates a new RRFFusion strategy.
func NewRRFFusion(k int) FusionStrategy {
	return &RRFFusion{K: k}
}

// Fuse combines results using the RRF algorithm. The score for each document
// is calculated as the sum of `1 / (K + rank)` for each result list it appears in.
func (f *RRFFusion) Fuse(results ...[]*Document) []*Document {
	scores := make(map[string]float64)
	docs := make(map[string]*Document)

	for _, resultList := range results {
		for rank, doc := range resultList {
			docID := doc.Content // Using content as a unique identifier for simplicity
			scores[docID] += 1.0 / float64(f.K+rank+1)
			if _, exists := docs[docID]; !exists {
				docs[docID] = doc
			}
		}
	}

	var fusedResults []*Document
	for docID := range docs {
		fusedResults = append(fusedResults, docs[docID])
	}

	sort.Slice(fusedResults, func(i, j int) bool {
		docIDi := fusedResults[i].Content
		docIDj := fusedResults[j].Content
		return scores[docIDi] > scores[docIDj]
	})

	return fusedResults
}

// --- Weighted Sum Fusion Implementation ---

// WeightedFusion implements the FusionStrategy by calculating a weighted sum of
// the scores from different result lists. This requires the scores to be
// normalized beforehand.
type WeightedFusion struct {
	Weights []float64 // The weight for each result list.
}

// NewWeightedFusion creates a new WeightedFusion strategy.
func NewWeightedFusion(weights []float64) FusionStrategy {
	return &WeightedFusion{Weights: weights}
}

// Fuse combines results using a weighted sum of their normalized scores.
func (f *WeightedFusion) Fuse(results ...[]*Document) []*Document {
	scores := make(map[string]float64)
	docs := make(map[string]*Document)

	// Normalize scores for each result list
	normalizedResults := make([][]*Document, len(results))
	for i, resultList := range results {
		normalizedResults[i] = normalizeScores(resultList)
	}

	for i, resultList := range normalizedResults {
		weight := f.Weights[i]
		for _, doc := range resultList {
			docID := doc.Content // Using content as a unique identifier
			scores[docID] += weight * float64(doc.Score)
			if _, exists := docs[docID]; !exists {
				docs[docID] = doc
			}
		}
	}

	var fusedResults []*Document
	for docID := range docs {
		fusedResults = append(fusedResults, docs[docID])
	}

	sort.Slice(fusedResults, func(i, j int) bool {
		docIDi := fusedResults[i].Content
		docIDj := fusedResults[j].Content
		return scores[docIDi] > scores[docIDj]
	})

	return fusedResults
}

// normalizeScores scales the scores of a list of documents to the [0, 1] range.
func normalizeScores(docs []*Document) []*Document {
	if len(docs) == 0 {
		return docs
	}

	minScore, maxScore := docs[0].Score, docs[0].Score
	for _, doc := range docs {
		if doc.Score < minScore {
			minScore = doc.Score
		}
		if doc.Score > maxScore {
			maxScore = doc.Score
		}
	}

	// Avoid division by zero if all scores are the same
	if maxScore == minScore {
		for i := range docs {
			docs[i].Score = 1.0
		}
		return docs
	}

	for i := range docs {
		docs[i].Score = (docs[i].Score - minScore) / (maxScore - minScore)
	}

	return docs
}
