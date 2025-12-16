package retriever

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/kubestack-ai/kubestack-ai/internal/rag/models"
)

// MockStore
type MockStore struct {
	mock.Mock
}

func (m *MockStore) Search(ctx context.Context, query string, topK int) ([]models.RetrievalResult, error) {
	args := m.Called(ctx, query, topK)
	return args.Get(0).([]models.RetrievalResult), args.Error(1)
}

func TestMultiStageRetriever_Retrieve(t *testing.T) {
	mockVec := new(MockStore)
	mockKw := new(MockStore)

	ctx := context.Background()
	query := "test query"

	vecRes := []models.RetrievalResult{
		{DocID: "1", Content: "A", Score: 0.8},
		{DocID: "2", Content: "B", Score: 0.6},
	}
	kwRes := []models.RetrievalResult{
		{DocID: "2", Content: "B", Score: 0.9},
		{DocID: "3", Content: "C", Score: 0.7},
	}

	mockVec.On("Search", ctx, query, 100).Return(vecRes, nil)
	mockKw.On("Search", ctx, query, 100).Return(kwRes, nil)

	// Fusion
	fusion := NewRRFFusion(60)

	// Reranker
	// Note: RRF scores are small (~0.01-0.03 for low ranks), so we need a low threshold
	reranker := NewThresholdReranker(0.01)

	config := &MultiStageConfig{
		RecallTopK: 100,
		CoarseTopK: 50,
		FineTopK: 20,
		FinalTopK: 10,
		MinScore: 0.01, // Low min score for RRF
		EnableKeyword: true,
	}

	mr := NewMultiStageRetriever(mockVec, mockKw, nil, []Reranker{reranker}, fusion, config)

	results, err := mr.Retrieve(ctx, query)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)

	// Check if results are merged
	// Doc 2 should be present (in both)
	found := false
	for _, r := range results {
		if r.DocID == "2" {
			found = true
			break
		}
	}
	assert.True(t, found, "Doc 2 should be in results")
}

func TestRRFFusion_Fuse(t *testing.T) {
	f := NewRRFFusion(60)

	res1 := []models.RetrievalResult{{DocID: "A", Score: 1.0}, {DocID: "B", Score: 0.8}}
	res2 := []models.RetrievalResult{{DocID: "B", Score: 1.0}, {DocID: "C", Score: 0.8}}

	fused, err := f.Fuse(context.Background(), [][]models.RetrievalResult{res1, res2})
	assert.NoError(t, err)

	// B is in both, should likely be first or second
	// A: 1/(60+1) = 1/61
	// B: 1/(60+2) + 1/(60+1) = 1/62 + 1/61
	// C: 1/(60+2) = 1/62
	// B score > A score > C score

	assert.Equal(t, "B", fused[0].DocID)
	assert.Equal(t, "A", fused[1].DocID)
	assert.Equal(t, "C", fused[2].DocID)
}

func TestThresholdReranker_Rerank(t *testing.T) {
	r := NewThresholdReranker(0.8)
	candidates := []models.RetrievalResult{
		{DocID: "A", Score: 0.9},
		{DocID: "B", Score: 0.7},
	}

	res, err := r.Rerank(context.Background(), "", candidates)
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "A", res[0].DocID)
}
