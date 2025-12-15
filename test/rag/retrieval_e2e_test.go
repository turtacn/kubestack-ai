package rag_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kubestack-ai/kubestack-ai/internal/rag"
	"github.com/kubestack-ai/kubestack-ai/internal/rag/models"
	"github.com/kubestack-ai/kubestack-ai/internal/rag/query"
	"github.com/kubestack-ai/kubestack-ai/internal/rag/retriever"
)

// Mocks
type MockVectorStore struct {
	mock.Mock
}

func (m *MockVectorStore) Search(ctx context.Context, query string, topK int) ([]models.RetrievalResult, error) {
	args := m.Called(ctx, query, topK)
	return args.Get(0).([]models.RetrievalResult), args.Error(1)
}

type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

func TestRAGEngine_MultiStageRetrieval_E2E(t *testing.T) {
	// Setup
	mockVec := new(MockVectorStore)
	mockLLM := new(MockLLMClient)

	// Mock vector store results
	vecRes := []models.RetrievalResult{
		{DocID: "doc1", Content: "Redis uses a lot of memory", Score: 0.9},
		{DocID: "doc2", Content: "Kafka is a message queue", Score: 0.5},
	}
	// Allow any query for search
	mockVec.On("Search", mock.Anything, mock.Anything, mock.Anything).Return(vecRes, nil)

	// Mock LLM response
	mockLLM.On("Generate", mock.Anything, mock.Anything).Return("Based on the context, Redis is memory intensive.", nil)

	// Components
	fusion := retriever.NewRRFFusion(60)
	reranker := retriever.NewThresholdReranker(0.5)

	msRetriever := retriever.NewMultiStageRetriever(
		mockVec,
		nil, // no keyword store
		[]retriever.Reranker{reranker},
		fusion,
		nil, // default config
	)

	rewriter := query.NewQueryRewriter(nil, nil)
	expander := query.NewQueryExpander(mockLLM, false) // Disable expander for simple test

	engine := rag.NewRAGEngine(msRetriever, rewriter, expander, mockLLM)

	// Execute
	answer, err := engine.Query(context.Background(), "Why is Redis high memory?")

	// Assert
	require.NoError(t, err)
	assert.Contains(t, answer, "Redis")

	mockVec.AssertExpectations(t)
	mockLLM.AssertExpectations(t)
}
