package prompt

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
)

// FewShotExample represents a single few-shot example.
type FewShotExample struct {
	ID        string    `json:"id"`
	Category  string    `json:"category"`
	Input     string    `json:"input"`
	Analysis  string    `json:"analysis"`
	Output    string    `json:"output"`
	Embedding []float64 `json:"-"` // Vector representation for retrieval
}

// Embedder is an interface for generating embeddings (to avoid circular dependency with client).
type Embedder interface {
	Embed(text string) ([]float64, error)
}

// FewShotManager manages storage and retrieval of few-shot examples.
type FewShotManager struct {
	examples []*FewShotExample
	embedder Embedder
	mu       sync.RWMutex
}

// NewFewShotManager creates a new manager.
func NewFewShotManager(embedder Embedder) *FewShotManager {
	return &FewShotManager{
		examples: make([]*FewShotExample, 0),
		embedder: embedder,
	}
}

// AddExample adds a new example to the manager.
// If embedder is provided, it calculates the embedding for the input.
func (m *FewShotManager) AddExample(ex *FewShotExample) error {
	if m.embedder != nil && len(ex.Embedding) == 0 {
		emb, err := m.embedder.Embed(ex.Input)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for example %s: %w", ex.ID, err)
		}
		ex.Embedding = emb
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.examples = append(m.examples, ex)
	return nil
}

// RetrieveSimilar returns the top-k most relevant examples for a given query and category.
// If category is empty, it searches all examples.
// If embedder is nil, it falls back to exact category match or returns random examples (simplified).
// For this implementation, we will use a simple cosine similarity if embeddings exist,
// otherwise just filter by category.
func (m *FewShotManager) RetrieveSimilar(query string, category string, topK int) ([]*FewShotExample, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var candidates []*FewShotExample
	for _, ex := range m.examples {
		if category == "" || strings.EqualFold(ex.Category, category) {
			candidates = append(candidates, ex)
		}
	}

	if len(candidates) <= topK {
		return candidates, nil
	}

	// If we have an embedder and the query is not empty, perform similarity search
	if m.embedder != nil && query != "" {
		queryEmb, err := m.embedder.Embed(query)
		if err != nil {
			return nil, fmt.Errorf("failed to embed query: %w", err)
		}

		type scoredExample struct {
			ex    *FewShotExample
			score float64
		}
		scores := make([]scoredExample, len(candidates))

		for i, ex := range candidates {
			score := 0.0
			if len(ex.Embedding) > 0 {
				score = cosineSimilarity(queryEmb, ex.Embedding)
			}
			scores[i] = scoredExample{ex: ex, score: score}
		}

		sort.Slice(scores, func(i, j int) bool {
			return scores[i].score > scores[j].score
		})

		result := make([]*FewShotExample, topK)
		for i := 0; i < topK; i++ {
			result[i] = scores[i].ex
		}
		return result, nil
	}

	// Fallback: just return the first K
	return candidates[:topK], nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, magA, magB float64
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}
