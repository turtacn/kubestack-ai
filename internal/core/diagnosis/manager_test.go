package diagnosis

import (
	"context"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/chain"
	llm_interfaces "github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/parser"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockPluginManager struct {
	mock.Mock
}

func (m *MockPluginManager) LoadPlugin(name string) (interfaces.MiddlewarePlugin, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(interfaces.MiddlewarePlugin), args.Error(1)
}

func (m *MockPluginManager) UnloadPlugin(name string) error {
	return m.Called(name).Error(0)
}

func (m *MockPluginManager) ListPlugins() []interfaces.MiddlewarePlugin {
	args := m.Called()
	return args.Get(0).([]interfaces.MiddlewarePlugin)
}

func (m *MockPluginManager) GetPlugin(name string) (interfaces.MiddlewarePlugin, bool) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(interfaces.MiddlewarePlugin), args.Bool(1)
}


type MockMiddlewarePlugin struct {
	mock.Mock
}

func (m *MockMiddlewarePlugin) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MetricsData), args.Error(1)
}

func (m *MockMiddlewarePlugin) CollectLogs(ctx context.Context, opts *models.LogOptions) (*models.LogData, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LogData), args.Error(1)
}

func (m *MockMiddlewarePlugin) GetConfiguration(ctx context.Context) (*models.ConfigData, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConfigData), args.Error(1)
}

func (m *MockMiddlewarePlugin) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(*models.HealthStatus), args.Error(1)
}

func (m *MockMiddlewarePlugin) Ping(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockMiddlewarePlugin) Name() string { return "mock" }
func (m *MockMiddlewarePlugin) Version() string { return "1.0" }
func (m *MockMiddlewarePlugin) Description() string { return "mock plugin" }
func (m *MockMiddlewarePlugin) SupportedVersions() []string { return []string{"1.0"} }
func (m *MockMiddlewarePlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	return nil, nil
}
func (m *MockMiddlewarePlugin) CanAutoFix(issue *models.Issue) bool { return false }
func (m *MockMiddlewarePlugin) ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error) { return nil, nil }
func (m *MockMiddlewarePlugin) ValidateFix(ctx context.Context, fix *models.FixAction) error { return nil }


type MockRetriever struct {
	mock.Mock
}

func (m *MockRetriever) Retrieve(ctx context.Context, query string, topK int) ([]search.Document, error) {
	args := m.Called(ctx, query, topK)
	return args.Get(0).([]search.Document), args.Error(1)
}

func (m *MockRetriever) HybridRetrieve(ctx context.Context, query string, opts *search.RetrieveOptions) ([]search.Document, error) {
	args := m.Called(ctx, query, opts)
	return args.Get(0).([]search.Document), args.Error(1)
}

type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) SendMessage(ctx context.Context, req *llm_interfaces.LLMRequest) (*llm_interfaces.LLMResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*llm_interfaces.LLMResponse), args.Error(1)
}

func (m *MockLLMClient) SendStreamingMessage(ctx context.Context, req *llm_interfaces.LLMRequest) (<-chan llm_interfaces.StreamingChunk, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(<-chan llm_interfaces.StreamingChunk), args.Error(1)
}

func (m *MockLLMClient) GenerateEmbedding(ctx context.Context, req *llm_interfaces.EmbeddingRequest) (*llm_interfaces.EmbeddingResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*llm_interfaces.EmbeddingResponse), args.Error(1)
}

// Test Cases

func TestRunDiagnosis_WithDiagnosisChain(t *testing.T) {
	// Setup Mocks
	mockPM := new(MockPluginManager)
	mockPlugin := new(MockMiddlewarePlugin)
	mockRetriever := new(MockRetriever)
	mockLLM := new(MockLLMClient)

	// Mock Plugin behavior
	mockPM.On("LoadPlugin", "Redis").Return(mockPlugin, nil)
	mockPlugin.On("CollectMetrics", mock.Anything).Return(&models.MetricsData{}, nil)
	mockPlugin.On("CollectLogs", mock.Anything, mock.Anything).Return(&models.LogData{Entries: []string{}}, nil)
	mockPlugin.On("GetConfiguration", mock.Anything).Return(&models.ConfigData{}, nil)

	// Mock Retriever behavior
	mockRetriever.On("Retrieve", mock.Anything, mock.Anything, 10).Return([]search.Document{
		{Content: "Redis memory OOM info", Score: 0.9},
	}, nil)

	// Mock LLM behavior
	llmResp := &llm_interfaces.LLMResponse{
		Message: llm_interfaces.Message{
			Content: `{
				"root_cause": "OOM due to maxmemory",
				"severity": "high",
				"confidence": 0.95,
				"contributing_factors": ["High key eviction", "No maxmemory-policy"],
				"affected_components": ["Redis"],
				"next_steps": ["Set maxmemory-policy to allkeys-lru"]
			}`,
		},
	}
	mockLLM.On("SendMessage", mock.Anything, mock.Anything).Return(llmResp, nil)

	// Setup Diagnosis Chain
	tmpl, _ := prompt.NewGoTemplate("test", "{{.Question}}")
	parserObj := parser.NewStructuredOutputParser()

	// Need to handle nil fewShotMgr in Chain
	diagChain := chain.NewDiagnosisChain(mockRetriever, mockLLM, tmpl, parserObj, nil)

	// Manager
	analyzers := []interfaces.DiagnosisAnalyzer{} // empty rule based analyzers
	manager := NewManager(mockPM, analyzers, diagChain, "reports_test", nil)

	// Action
	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         "redis-1",
	}
	// Use a channel to drain progress
	progressCh := make(chan interfaces.DiagnosisProgress, 10)
	go func() {
		for range progressCh {}
	}()

	result, err := manager.RunDiagnosis(context.Background(), req, progressCh)
	close(progressCh)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Issues)

	foundAI := false
	for _, issue := range result.Issues {
		if issue.Source == "AI" {
			foundAI = true
			assert.Equal(t, "OOM due to maxmemory", issue.Title)
			assert.Equal(t, enum.SeverityHigh, issue.Severity)
		}
	}
	assert.True(t, foundAI, "Should contain issue from AI source")

	mockRetriever.AssertExpectations(t)
	mockLLM.AssertExpectations(t)
}
