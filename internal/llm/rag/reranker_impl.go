package rag

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
)

// SimpleReranker implements a TF-IDF based reranker.
type SimpleReranker struct{}

// NewSimpleReranker creates a new SimpleReranker.
func NewSimpleReranker() *SimpleReranker {
	return &SimpleReranker{}
}

// Rerank reorders the candidate documents based on TF-IDF similarity with the query.
func (r *SimpleReranker) Rerank(ctx context.Context, query string, candidates []*search.Document, topK int) ([]*search.Document, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	// Step 1: Tokenize query
	queryTerms := tokenize(query)

	// Step 2: Calculate score for each document
	type scoredDoc struct {
		doc   *search.Document
		score float64
	}
	scored := make([]scoredDoc, len(candidates))

	for i, doc := range candidates {
		score := 0.0
		docTerms := tokenize(doc.Content)
		if len(docTerms) > 0 {
			for _, term := range queryTerms {
				// TF: Term Frequency in document
				tf := float64(countTerm(docTerms, term)) / float64(len(docTerms))

				// IDF: Inverse Document Frequency
				docCountWithTerm := countDocsWithTerm(candidates, term)
				var idf float64
				if docCountWithTerm > 0 {
					idf = math.Log(float64(len(candidates)) / float64(docCountWithTerm))
				} else {
					idf = 0 // Should not happen if term is in query and we are checking candidates? Actually it can happen if query term is not in any doc.
				}

				score += tf * idf
			}
		}
		scored[i] = scoredDoc{doc: doc, score: score}
	}

	// Step 3: Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Step 4: Return topK
	if topK > len(scored) {
		topK = len(scored)
	}

	reranked := make([]*search.Document, topK)
	for i := 0; i < topK; i++ {
		reranked[i] = scored[i].doc
		// Update the score with the new reranking score
		reranked[i].Score = float32(scored[i].score)
	}

	return reranked, nil
}

// tokenize splits text into tokens. Simple implementation using space.
func tokenize(text string) []string {
	return strings.Fields(strings.ToLower(text))
}

func countTerm(terms []string, target string) int {
	count := 0
	for _, t := range terms {
		if t == target {
			count++
		}
	}
	return count
}

func countDocsWithTerm(docs []*search.Document, term string) int {
	count := 0
	for _, doc := range docs {
		if strings.Contains(strings.ToLower(doc.Content), term) {
			count++
		}
	}
	return count
}
