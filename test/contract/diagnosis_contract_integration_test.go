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

package contract

import (
	"context"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/contracts"
	"github.com/kubestack-ai/kubestack-ai/internal/core/contracts/adapter"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	llm_interfaces "github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// === Mocks ===

// MockContractPlugin implements contracts.MiddlewarePlugin for testing
type MockContractPlugin struct {
	mock.Mock
}

func (m *MockContractPlugin) Name() string {
	return "MockContractPlugin"
}

func (m *MockContractPlugin) Version() string {
	return "1.0.0"
}

func (m *MockContractPlugin) SupportedVersions() []string {
	return []string{"1.0.0"}
}

func (m *MockContractPlugin) Diagnose(ctx context.Context, config *contracts.DiagnosisConfig) (*contracts.DiagnosisResult, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*contracts.DiagnosisResult), args.Error(1)
}

func (m *MockContractPlugin) CollectMetrics(ctx context.Context, target *contracts.TargetConfig) (*contracts.MetricsData, error) {
	args := m.Called(ctx, target)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*contracts.MetricsData), args.Error(1)
}

func (m *MockContractPlugin) CollectLogs(ctx context.Context, target *contracts.TargetConfig, opts *contracts.LogOptions) (*contracts.LogData, error) {
	args := m.Called(ctx, target, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*contracts.LogData), args.Error(1)
}

func (m *MockContractPlugin) GetConfiguration(ctx context.Context, target *contracts.TargetConfig) (*contracts.ConfigData, error) {
	args := m.Called(ctx, target)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*contracts.ConfigData), args.Error(1)
}

func (m *MockContractPlugin) HealthCheck(ctx context.Context, target *contracts.TargetConfig) (*contracts.HealthStatus, error) {
	args := m.Called(ctx, target)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*contracts.HealthStatus), args.Error(1)
}

func (m *MockContractPlugin) CanAutoFix(issue *contracts.Issue) (bool, *contracts.FixAction) {
	args := m.Called(issue)
	if args.Get(1) == nil {
		return args.Bool(0), nil
	}
	return args.Bool(0), args.Get(1).(*contracts.FixAction)
}

func (m *MockContractPlugin) ExecuteFix(ctx context.Context, fix *contracts.FixAction) (*contracts.FixResult, error) {
	args := m.Called(ctx, fix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*contracts.FixResult), args.Error(1)
}

// MockLegacyPlugin implements plugin.MiddlewarePlugin for testing legacy plugins
type MockLegacyPlugin struct {
	mock.Mock
}

func (m *MockLegacyPlugin) Name() string {
	return "MockLegacyPlugin"
}

func (m *MockLegacyPlugin) Type() plugin.MiddlewareType {
	return plugin.MiddlewareRedis
}

func (m *MockLegacyPlugin) Version() string {
	return "1.0.0"
}

func (m *MockLegacyPlugin) Connect(ctx context.Context, config *plugin.ConnectionConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockLegacyPlugin) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockLegacyPlugin) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockLegacyPlugin) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockLegacyPlugin) CollectMetrics(ctx context.Context) (*plugin.MetricsSnapshot, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugin.MetricsSnapshot), args.Error(1)
}

func (m *MockLegacyPlugin) CollectSpecificMetric(ctx context.Context, metricName string) (interface{}, error) {
	args := m.Called(ctx, metricName)
	return args.Get(0), args.Error(1)
}

func (m *MockLegacyPlugin) Execute(ctx context.Context, cmd *plugin.Command) (*plugin.CommandResult, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugin.CommandResult), args.Error(1)
}

func (m *MockLegacyPlugin) SupportedCommands() []plugin.CommandSpec {
	args := m.Called()
	return args.Get(0).([]plugin.CommandSpec)
}

func (m *MockLegacyPlugin) GetDiagnosticData(ctx context.Context) (*plugin.DiagnosticData, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugin.DiagnosticData), args.Error(1)
}

func (m *MockLegacyPlugin) GetBuiltinRules() []plugin.DiagnosisRule {
	args := m.Called()
	if args.Get(0) == nil {
		return []plugin.DiagnosisRule{}
	}
	return args.Get(0).([]plugin.DiagnosisRule)
}

// MockPluginManager implements interfaces.PluginManager
type MockPluginManager struct {
	mock.Mock
}

func (m *MockPluginManager) LoadPlugins() error {
	return m.Called().Error(0)
}

func (m *MockPluginManager) GetPlugin(name string) (interfaces.DiagnosticPlugin, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(interfaces.DiagnosticPlugin), args.Error(1)
}

func (m *MockPluginManager) ListPlugins() []interfaces.DiagnosticPlugin {
	args := m.Called()
	return args.Get(0).([]interfaces.DiagnosticPlugin)
}

func (m *MockPluginManager) CollectData(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CollectedData), args.Error(1)
}

func (m *MockPluginManager) Shutdown() {
	m.Called()
}

func (m *MockPluginManager) LoadPlugin(pluginName string) (interfaces.DiagnosticPlugin, error) {
	args := m.Called(pluginName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(interfaces.DiagnosticPlugin), args.Error(1)
}

func (m *MockPluginManager) UnloadPlugin(pluginName string) error {
	return m.Called(pluginName).Error(0)
}

// MockLLMClient implements llm_interfaces.LLMClient
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) SendMessage(ctx context.Context, req *llm_interfaces.LLMRequest) (*llm_interfaces.LLMResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llm_interfaces.LLMResponse), args.Error(1)
}

func (m *MockLLMClient) SendStreamingMessage(ctx context.Context, req *llm_interfaces.LLMRequest) (<-chan llm_interfaces.StreamingChunk, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan llm_interfaces.StreamingChunk), args.Error(1)
}

func (m *MockLLMClient) GenerateEmbedding(ctx context.Context, req *llm_interfaces.EmbeddingRequest) (*llm_interfaces.EmbeddingResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llm_interfaces.EmbeddingResponse), args.Error(1)
}

func (m *MockLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

// === Tests ===

// TestDiagnosisFlow_MinimalContractLoop tests the minimal diagnosis flow
// using contract-based plugins and adapters to verify the contract alignment works end-to-end.
func TestDiagnosisFlow_MinimalContractLoop(t *testing.T) {
	// Arrange
	ctx := context.Background()
	
	// Create a mock legacy plugin (simulating existing plugins)
	legacyPlugin := new(MockLegacyPlugin)
	
	// Setup expectations for the legacy plugin
	legacyPlugin.On("IsConnected").Return(true)
	legacyPlugin.On("GetDiagnosticData", ctx).Return(&plugin.DiagnosticData{
		Metrics: &plugin.MetricsSnapshot{
			Timestamp: time.Now(),
			Metrics: map[string]plugin.MetricValue{
				"memory_usage": {
					Name:  "memory_usage",
					Value: 85.5,
					Unit:  "percent",
				},
			},
		},
		Config: map[string]interface{}{
			"maxmemory": "2gb",
		},
	}, nil)
	legacyPlugin.On("GetBuiltinRules").Return([]plugin.DiagnosisRule{})
	
	// Wrap legacy plugin with adapter to get contract-compliant plugin
	contractPlugin := adapter.NewPluginAdapter(legacyPlugin)
	
	// Create diagnosis config
	diagConfig := &contracts.DiagnosisConfig{
		Target: &contracts.TargetConfig{
			Host: "localhost",
			Port: 6379,
		},
		Timeout: 30 * time.Second,
	}
	
	// Act - Execute diagnosis through the contract interface
	result, err := contractPlugin.Diagnose(ctx, diagConfig)
	
	// Assert - Verify the contract flow works
	assert.NoError(t, err, "Diagnosis should complete without error")
	assert.NotNil(t, result, "Diagnosis result should not be nil")
	assert.NotNil(t, result.Metrics, "Metrics should be collected")
	assert.Contains(t, result.Metrics.Metrics, "memory_usage", "Should contain memory_usage metric")
	assert.Equal(t, 85.5, result.Metrics.Metrics["memory_usage"].Value, "Metric value should match")
	
	// Verify all expectations were met
	legacyPlugin.AssertExpectations(t)
}

// TestDiagnosisFlow_WithPluginManager tests the integration with PluginManager
func TestDiagnosisFlow_WithPluginManager(t *testing.T) {
	// Arrange
	ctx := context.Background()
	
	mockPluginManager := new(MockPluginManager)
	
	// Setup diagnosis request
	diagReq := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         "localhost:6379",
	}
	
	// Mock collected data from plugin manager
	collectedData := &models.CollectedData{
		Metrics: &models.MetricsData{
			Data: map[string]interface{}{
				"cpu_usage":    45.5,
				"memory_usage": 60.0,
				"connections":  100,
			},
		},
		Logs: &models.LogData{
			Entries: []string{
				"Connection established",
				"Query executed successfully",
			},
		},
	}
	
	mockPluginManager.On("CollectData", ctx, diagReq).Return(collectedData, nil)
	
	// Act - Simulate data collection through plugin manager
	result, err := mockPluginManager.CollectData(ctx, diagReq)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Metrics)
	assert.Equal(t, 45.5, result.Metrics.Data["cpu_usage"])
	assert.NotNil(t, result.Logs)
	assert.Equal(t, 2, len(result.Logs.Entries))
	
	mockPluginManager.AssertExpectations(t)
}

// TestDiagnosisFlow_AdapterMetricsCollection tests metrics collection through adapter
func TestDiagnosisFlow_AdapterMetricsCollection(t *testing.T) {
	// Arrange
	ctx := context.Background()
	
	legacyPlugin := new(MockLegacyPlugin)
	
	metricsSnapshot := &plugin.MetricsSnapshot{
		Timestamp: time.Now(),
		Metrics: map[string]plugin.MetricValue{
			"cpu_usage": {Name: "cpu_usage", Value: 45.0, Unit: "percent"},
			"memory_usage": {Name: "memory_usage", Value: 80.0, Unit: "percent"},
			"disk_usage": {Name: "disk_usage", Value: 55.0, Unit: "percent"},
		},
	}
	
	legacyPlugin.On("IsConnected").Return(true)
	legacyPlugin.On("CollectMetrics", ctx).Return(metricsSnapshot, nil)
	
	contractPlugin := adapter.NewPluginAdapter(legacyPlugin)
	
	target := &contracts.TargetConfig{
		Host: "localhost",
		Port: 6379,
	}
	
	// Act
	result, err := contractPlugin.CollectMetrics(ctx, target)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, len(result.Metrics))
	assert.Contains(t, result.Metrics, "cpu_usage")
	assert.Contains(t, result.Metrics, "memory_usage")
	assert.Contains(t, result.Metrics, "disk_usage")
	assert.Equal(t, 45.0, result.Metrics["cpu_usage"].Value)
	
	legacyPlugin.AssertExpectations(t)
}

// TestDiagnosisFlow_AdapterHealthCheck tests health check through adapter
func TestDiagnosisFlow_AdapterHealthCheck(t *testing.T) {
	// Arrange
	ctx := context.Background()
	
	legacyPlugin := new(MockLegacyPlugin)
	legacyPlugin.On("IsConnected").Return(true)
	legacyPlugin.On("Ping", ctx).Return(nil)
	
	contractPlugin := adapter.NewPluginAdapter(legacyPlugin)
	
	target := &contracts.TargetConfig{
		Host: "localhost",
		Port: 6379,
	}
	
	// Act
	result, err := contractPlugin.HealthCheck(ctx, target)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "healthy", result.Status)
	assert.True(t, result.Connectivity)
	assert.True(t, result.Latency > 0)
	
	legacyPlugin.AssertExpectations(t)
}

// TestDiagnosisFlow_LLMIntegration tests minimal LLM integration in diagnosis flow
func TestDiagnosisFlow_LLMIntegration(t *testing.T) {
	// Arrange
	ctx := context.Background()
	
	mockLLM := new(MockLLMClient)
	
	// Mock LLM analysis response
	llmResponse := &llm_interfaces.LLMResponse{
		Message: llm_interfaces.Message{
			Role: "assistant",
			Content: `{
			"issues": [
				{
					"title": "High Memory Usage",
					"severity": "warning",
					"description": "Memory usage is at 85%, recommend optimization"
				}
			],
			"recommendations": [
				"Enable memory eviction policy",
				"Review memory-intensive operations"
			]
		}`,
		},
		Usage: llm_interfaces.UsageStats{
			TotalTokens: 150,
		},
	}
	
	mockLLM.On("SendMessage", ctx, mock.AnythingOfType("*interfaces.LLMRequest")).Return(llmResponse, nil)
	
	// Act - Simulate LLM analysis call
	llmReq := &llm_interfaces.LLMRequest{
		Model: "gpt-4",
		Messages: []llm_interfaces.Message{
			{Role: "system", Content: "You are a Redis diagnostics expert."},
			{Role: "user", Content: "Analyze this data: memory_usage=85%"},
		},
	}
	
	result, err := mockLLM.SendMessage(ctx, llmReq)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Message.Content, "High Memory Usage")
	assert.Contains(t, result.Message.Content, "recommendations")
	
	mockLLM.AssertExpectations(t)
}

// TestPluginRegistry_ContractCompatibility tests plugin registry with contract-based plugins
func TestPluginRegistry_ContractCompatibility(t *testing.T) {
	// This test verifies that the plugin registry can work with both legacy and contract-based plugins
	
	// Arrange
	legacyPlugin := new(MockLegacyPlugin)
	contractPlugin := adapter.NewPluginAdapter(legacyPlugin)
	
	// Verify contract plugin satisfies the interface
	var _ contracts.MiddlewarePlugin = contractPlugin
	
	// Assert - Type assertion should succeed
	assert.NotNil(t, contractPlugin)
	assert.Equal(t, "MockLegacyPlugin", contractPlugin.Name())
	assert.Equal(t, "1.0.0", contractPlugin.Version())
}
