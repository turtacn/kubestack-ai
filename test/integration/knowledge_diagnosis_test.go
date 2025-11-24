package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDiagnosticPlugin is a mock implementation of the DiagnosticPlugin interface.
type MockDiagnosticPlugin struct {
	mock.Mock
}

func (m *MockDiagnosticPlugin) Name() string {
	return "MockPlugin"
}
func (m *MockDiagnosticPlugin) Version() string {
	return "1.0.0"
}
func (m *MockDiagnosticPlugin) SupportedTypes() []string {
	return []string{"Redis"}
}
func (m *MockDiagnosticPlugin) Init(config map[string]interface{}) error {
	return nil
}
func (m *MockDiagnosticPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiagnosisResult), args.Error(1)
}
func (m *MockDiagnosticPlugin) Shutdown() error {
	return nil
}

func TestDiagnosisWithRuleEngine(t *testing.T) {
	// Setup: Initialize components
	kb := knowledge.NewKnowledgeBase()

	// Add a test rule
	err := kb.AddRule(&knowledge.Rule{
		ID:             "redis-mem-high",
		Name:           "High Memory Usage",
		MiddlewareType: "Redis",
		Condition:      "memory_usage > 80",
		Recommendation: "Scale up memory",
		Severity:       "HIGH",
		Priority:       100,
	})
	assert.NoError(t, err)

	// Setup Plugin Registry and Mock Plugin
	registry := plugin.NewRegistry()
	mockPlugin := new(MockDiagnosticPlugin)
	err = registry.Register(mockPlugin) // Corrected Register call
	assert.NoError(t, err)

	// Mock Plugin Response
	mockResult := &models.DiagnosisResult{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Status:    enum.StatusWarning,
		Summary:   "Mock diagnosis completed",
		Issues:    []*models.Issue{},
		Metrics: map[string]interface{}{
			"memory_usage": 85.0, // This should trigger the rule
		},
	}
	mockPlugin.On("Diagnose", mock.Anything, mock.Anything).Return(mockResult, nil)

	// Create Diagnosis Manager
	manager := diagnosis.NewManager(registry, nil, nil, "", kb)

	// Action: Run Diagnosis
	ctx := context.Background()
	req := &models.DiagnosisRequest{
		// ID field removed as per error message
		TargetMiddleware: enum.Redis, // Corrected enum usage
		Instance:         "redis-prod",
	}
	progressChan := make(chan interfaces.DiagnosisProgress, 100)

	result, err := manager.RunDiagnosis(ctx, req, progressChan)
	assert.NoError(t, err)

	// Assert: Verify results
	assert.NotNil(t, result)

	// Check if rule engine produced an issue
	foundRuleIssue := false
	for _, issue := range result.Issues {
		if issue.Source == "RuleEngine" {
			foundRuleIssue = true
			assert.NotEmpty(t, issue.Recommendations)
			assert.Contains(t, issue.Recommendations[0].Description, "Scale up memory")
		}
	}
	assert.True(t, foundRuleIssue, "Expected RuleEngine to generate an issue")
}
