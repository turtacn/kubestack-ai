package rca

import (
	"context"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/core/detection/models"
)

// SimilarCase represents a historical case found in the knowledge graph.
type SimilarCase struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Solution    string   `json:"solution"`
	Similarity  float64  `json:"similarity"`
}

// KnowledgeGraph represents the interface to the knowledge base.
type KnowledgeGraph struct {
	// In a real implementation, this would connect to a vector DB or graph DB.
	// For now, we use a simple in-memory mock.
	cases []SimilarCase
}

// NewKnowledgeGraph creates a new KnowledgeGraph instance.
func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{
		cases: []SimilarCase{
			{
				ID:          "CASE-001",
				Description: "Redis high memory due to large keys",
				Solution:    "Identified and deleted large keys using --bigkeys",
				Similarity:  0.0, // Calculated dynamically
			},
			{
				ID:          "CASE-002",
				Description: "MySQL high CPU due to missing index on users table",
				Solution:    "Added index on email column",
				Similarity:  0.0,
			},
		},
	}
}

// QuerySimilarCases searches for cases similar to the given anomaly.
func (kg *KnowledgeGraph) QuerySimilarCases(ctx context.Context, anomaly models.Anomaly) []SimilarCase {
	var results []SimilarCase

	// Simple keyword matching for "similarity" mock
	for _, c := range kg.cases {
		score := 0.0
		// Check for keyword matches based on anomaly type
		if anomaly.Type == models.AnomalyTypeHighMemory && strings.Contains(strings.ToLower(c.Description), "memory") {
			score = 0.85
		} else if anomaly.Type == models.AnomalyTypeHighCPU && strings.Contains(strings.ToLower(c.Description), "cpu") {
			score = 0.85
		}

		if score > 0 {
			match := c
			match.Similarity = score
			results = append(results, match)
		}
	}

	return results
}
