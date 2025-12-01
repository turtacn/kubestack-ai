package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/diagnosis/ai"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/parser"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockRetriever struct {
	mock.Mock
}

func (m *MockRetriever) Retrieve(ctx context.Context, query string, topK int) ([]search.Document, error) {
	args := m.Called(ctx, query, topK)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]search.Document), args.Error(1)
}

func (m *MockRetriever) HybridRetrieve(ctx context.Context, query string, opts *search.RetrieveOptions) ([]search.Document, error) {
	args := m.Called(ctx, query, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]search.Document), args.Error(1)
}

type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) SendMessage(ctx context.Context, req *interfaces.LLMRequest) (*interfaces.LLMResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*interfaces.LLMResponse), args.Error(1)
}

func (m *MockLLMClient) SendStreamingMessage(ctx context.Context, req *interfaces.LLMRequest) (<-chan interfaces.StreamingChunk, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(<-chan interfaces.StreamingChunk), args.Error(1)
}

func (m *MockLLMClient) GenerateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*interfaces.EmbeddingResponse), args.Error(1)
}

type MockPromptTemplate struct {
	mock.Mock
}

func (m *MockPromptTemplate) Render(data interface{}) (string, error) {
	// Simple render for test
	// We need to cast data to map to check fields
	dataMap, ok := data.(map[string]interface{})
	if ok {
		if kc, has := dataMap["KnowledgeContext"]; has && kc != "" {
			return fmt.Sprintf("Rendered prompt with knowledge: %v", kc), nil
		}
	}
	return "Rendered prompt without knowledge", nil
}

func (m *MockPromptTemplate) Validate() error {
	return nil
}

// --- Tests ---

func TestAnalyze_WithKnowledge(t *testing.T) {
	// Setup Mocks
	mockRetriever := new(MockRetriever)
	mockLLM := new(MockLLMClient)
	mockTemplate := new(MockPromptTemplate)

	// Prepare knowledge return
	docs := []search.Document{
		{
			Content: "OOM Killer kills Redis when memory is full.",
			Metadata: map[string]interface{}{"title": "Redis OOM"},
			Score: 0.9,
		},
	}
	mockRetriever.On("HybridRetrieve", mock.Anything, mock.Anything, mock.Anything).Return(docs, nil)

	// Prepare LLM return
	llmResponse := &interfaces.LLMResponse{
		Message: interfaces.Message{
			Content: `{"severity": "critical", "root_cause": "OOM", "confidence": 0.95, "affected_components": ["redis"]}`,
		},
	}

	mockLLM.On("SendMessage", mock.Anything, mock.MatchedBy(func(req *interfaces.LLMRequest) bool {
		return assert.Contains(t, req.Messages[0].Content, "OOM Killer kills Redis")
	})).Return(llmResponse, nil)

	// Helper dependencies
	realParser := parser.NewStructuredOutputParser()

	analyzer := ai.NewAIAnalyzer(
		mockLLM,
		mockRetriever,
		mockTemplate,
		realParser,
		nil, // KnowledgeInjector
		nil, // MultiTurnManager
		&prompt.FewShotManager{}, // Mock or Real
		&ai.AIAnalyzerConfig{
			Temperature: 0,
			MaxTokens: 1000,
			MinRelevanceScore: 0.5,
			MaxContextToken: 1000,
		},
	)

	req := &ai.DiagnosisRequest{
		PluginName: "Redis",
		Query:      "Redis is down",
		Logs:       "ERROR: Can't save in background",
		Metrics:    "used_memory: 1000000",
	}

	result, err := analyzer.Analyze(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "OOM", result.RootCause)
}

func TestAnalyze_Fallback(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockLLM := new(MockLLMClient)
	mockTemplate := new(MockPromptTemplate)

	// Mock retrieval failure
	mockRetriever.On("HybridRetrieve", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("timeout"))

	llmResponse := &interfaces.LLMResponse{
		Message: interfaces.Message{
			Content: `{"severity": "low", "root_cause": "Unknown", "confidence": 0.5, "affected_components": []}`,
		},
	}

	mockLLM.On("SendMessage", mock.Anything, mock.MatchedBy(func(req *interfaces.LLMRequest) bool {
		// Should proceed without knowledge
		return !strings.Contains(req.Messages[0].Content, "OOM Killer")
	})).Return(llmResponse, nil)

	analyzer := ai.NewAIAnalyzer(
		mockLLM,
		mockRetriever,
		mockTemplate,
		parser.NewStructuredOutputParser(),
		nil, nil, &prompt.FewShotManager{},
		&ai.AIAnalyzerConfig{MinRelevanceScore: 0.5},
	)

	req := &ai.DiagnosisRequest{Query: "Test"}
	_, err := analyzer.Analyze(context.Background(), req)

	// It should NOT return error from Analyze, but proceed
	assert.NoError(t, err)
}
