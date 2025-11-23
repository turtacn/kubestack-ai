package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockEmbedder struct{}

func (m *mockEmbedder) Embed(text string) ([]float64, error) {
	if text == "query" {
		return []float64{1.0, 0.0}, nil
	}
	return []float64{0.0, 1.0}, nil
}

func TestFewShotManager_Retrieve(t *testing.T) {
	mgr := NewFewShotManager(&mockEmbedder{})

	err := mgr.AddExample(&FewShotExample{
		ID:       "1",
		Category: "Redis",
		Input:    "redis issue",
		Embedding: []float64{1.0, 0.0}, // Matches query
	})
	assert.NoError(t, err)

	err = mgr.AddExample(&FewShotExample{
		ID:       "2",
		Category: "Kafka",
		Input:    "kafka issue",
		Embedding: []float64{0.0, 1.0}, // Orthogonal to query
	})
	assert.NoError(t, err)

	// Test category filtering
	results, err := mgr.RetrieveSimilar("query", "Redis", 5)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "1", results[0].ID)

	// Test similarity search (empty category)
	results, err = mgr.RetrieveSimilar("query", "", 5)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "1", results[0].ID) // High score first
	assert.Equal(t, "2", results[1].ID) // Low score second
}
