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

package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/contracts"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUnderlyingPlugin is a mock implementation of plugin.MiddlewarePlugin
type MockUnderlyingPlugin struct {
	mock.Mock
}

func (m *MockUnderlyingPlugin) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockUnderlyingPlugin) Type() plugin.MiddlewareType {
	args := m.Called()
	return args.Get(0).(plugin.MiddlewareType)
}

func (m *MockUnderlyingPlugin) Version() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockUnderlyingPlugin) Connect(ctx context.Context, config *plugin.ConnectionConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockUnderlyingPlugin) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockUnderlyingPlugin) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockUnderlyingPlugin) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockUnderlyingPlugin) CollectMetrics(ctx context.Context) (*plugin.MetricsSnapshot, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugin.MetricsSnapshot), args.Error(1)
}

func (m *MockUnderlyingPlugin) CollectSpecificMetric(ctx context.Context, metricName string) (interface{}, error) {
	args := m.Called(ctx, metricName)
	return args.Get(0), args.Error(1)
}

func (m *MockUnderlyingPlugin) Execute(ctx context.Context, cmd *plugin.Command) (*plugin.CommandResult, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugin.CommandResult), args.Error(1)
}

func (m *MockUnderlyingPlugin) SupportedCommands() []plugin.CommandSpec {
	args := m.Called()
	return args.Get(0).([]plugin.CommandSpec)
}

func (m *MockUnderlyingPlugin) GetDiagnosticData(ctx context.Context) (*plugin.DiagnosticData, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugin.DiagnosticData), args.Error(1)
}

func (m *MockUnderlyingPlugin) GetBuiltinRules() []plugin.DiagnosisRule {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]plugin.DiagnosisRule)
}

// TestAdapter_Diagnose_MapsAndReturns tests that the adapter correctly maps
// the Diagnose call to underlying plugin methods and returns consistent results.
func TestAdapter_Diagnose_MapsAndReturns(t *testing.T) {
	// Arrange
	mockPlugin := new(MockUnderlyingPlugin)
	adapter := NewPluginAdapter(mockPlugin)

	ctx := context.Background()
	config := &contracts.DiagnosisConfig{
		Target: &contracts.TargetConfig{
			Host: "localhost",
			Port: 6379,
		},
		Timeout: 30 * time.Second,
	}

	// Setup mock expectations
	mockPlugin.On("IsConnected").Return(false)
	mockPlugin.On("Connect", ctx, mock.AnythingOfType("*plugin.ConnectionConfig")).Return(nil)
	
	diagData := &plugin.DiagnosticData{
		Metrics: &plugin.MetricsSnapshot{
			Timestamp: time.Now(),
			Metrics: map[string]plugin.MetricValue{
				"memory_used": {
					Name:  "memory_used",
					Value: 1024.0,
					Unit:  "MB",
				},
			},
		},
		Config: map[string]interface{}{
			"maxmemory": "2gb",
		},
	}
	mockPlugin.On("GetDiagnosticData", ctx).Return(diagData, nil)
	mockPlugin.On("GetBuiltinRules").Return([]plugin.DiagnosisRule{})

	// Act
	result, err := adapter.Diagnose(ctx, config)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Metrics)
	assert.Equal(t, 1, len(result.Metrics.Metrics))
	assert.Contains(t, result.Metrics.Metrics, "memory_used")
	mockPlugin.AssertExpectations(t)
}

// TestAdapter_CollectMetrics_Success tests successful metrics collection
func TestAdapter_CollectMetrics_Success(t *testing.T) {
	// Arrange
	mockPlugin := new(MockUnderlyingPlugin)
	adapter := NewPluginAdapter(mockPlugin)

	ctx := context.Background()
	target := &contracts.TargetConfig{
		Host: "localhost",
		Port: 6379,
	}

	metricsSnapshot := &plugin.MetricsSnapshot{
		Timestamp: time.Now(),
		Metrics: map[string]plugin.MetricValue{
			"cpu_usage": {
				Name:  "cpu_usage",
				Value: 45.5,
				Unit:  "percent",
			},
		},
	}

	mockPlugin.On("IsConnected").Return(true)
	mockPlugin.On("CollectMetrics", ctx).Return(metricsSnapshot, nil)

	// Act
	result, err := adapter.CollectMetrics(ctx, target)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Metrics))
	assert.Contains(t, result.Metrics, "cpu_usage")
	assert.Equal(t, 45.5, result.Metrics["cpu_usage"].Value)
	mockPlugin.AssertExpectations(t)
}

// TestAdapter_GetConfiguration_NotSupported tests the NotSupported error case
func TestAdapter_GetConfiguration_NotSupported(t *testing.T) {
	// Arrange
	mockPlugin := new(MockUnderlyingPlugin)
	adapter := NewPluginAdapter(mockPlugin)

	ctx := context.Background()
	target := &contracts.TargetConfig{
		Host: "localhost",
		Port: 6379,
	}

	// Mock returns diagnostic data without config
	diagData := &plugin.DiagnosticData{
		Config: nil, // No config available
	}

	mockPlugin.On("IsConnected").Return(true)
	mockPlugin.On("GetDiagnosticData", ctx).Return(diagData, nil)

	// Act
	result, err := adapter.GetConfiguration(ctx, target)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, contracts.ErrNotSupported, err)
	assert.Nil(t, result)
	mockPlugin.AssertExpectations(t)
}

// TestAdapter_GetConfiguration_Success tests successful configuration retrieval
func TestAdapter_GetConfiguration_Success(t *testing.T) {
	// Arrange
	mockPlugin := new(MockUnderlyingPlugin)
	adapter := NewPluginAdapter(mockPlugin)

	ctx := context.Background()
	target := &contracts.TargetConfig{
		Host: "localhost",
		Port: 6379,
	}

	diagData := &plugin.DiagnosticData{
		Config: map[string]interface{}{
			"maxmemory":        "2gb",
			"maxmemory-policy": "allkeys-lru",
		},
		Extra: map[string]interface{}{
			"version": "6.2.0",
		},
	}

	mockPlugin.On("IsConnected").Return(true)
	mockPlugin.On("GetDiagnosticData", ctx).Return(diagData, nil)

	// Act
	result, err := adapter.GetConfiguration(ctx, target)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Parameters))
	assert.Equal(t, "2gb", result.Parameters["maxmemory"])
	assert.NotNil(t, result.Runtime)
	mockPlugin.AssertExpectations(t)
}

// TestAdapter_HealthCheck_Healthy tests successful health check
func TestAdapter_HealthCheck_Healthy(t *testing.T) {
	// Arrange
	mockPlugin := new(MockUnderlyingPlugin)
	adapter := NewPluginAdapter(mockPlugin)

	ctx := context.Background()
	target := &contracts.TargetConfig{
		Host: "localhost",
		Port: 6379,
	}

	mockPlugin.On("IsConnected").Return(true)
	mockPlugin.On("Ping", ctx).Return(nil)

	// Act
	result, err := adapter.HealthCheck(ctx, target)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "healthy", result.Status)
	assert.True(t, result.Connectivity)
	mockPlugin.AssertExpectations(t)
}

// TestAdapter_HealthCheck_Unhealthy tests failed health check
func TestAdapter_HealthCheck_Unhealthy(t *testing.T) {
	// Arrange
	mockPlugin := new(MockUnderlyingPlugin)
	adapter := NewPluginAdapter(mockPlugin)

	ctx := context.Background()
	target := &contracts.TargetConfig{
		Host: "localhost",
		Port: 6379,
	}

	mockPlugin.On("IsConnected").Return(true)
	mockPlugin.On("Ping", ctx).Return(assert.AnError)

	// Act
	result, err := adapter.HealthCheck(ctx, target)

	// Assert
	assert.NoError(t, err) // HealthCheck returns status, not error
	assert.NotNil(t, result)
	assert.Equal(t, "unhealthy", result.Status)
	assert.False(t, result.Connectivity)
	mockPlugin.AssertExpectations(t)
}

// TestAdapter_ExecuteFix_Success tests successful fix execution
func TestAdapter_ExecuteFix_Success(t *testing.T) {
	// Arrange
	mockPlugin := new(MockUnderlyingPlugin)
	adapter := NewPluginAdapter(mockPlugin)

	ctx := context.Background()
	fix := &contracts.FixAction{
		ID:          "config-change",
		Type:        contracts.FixTypeConfiguration,
		Description: "Update maxmemory policy",
		Parameters: map[string]interface{}{
			"key":   "maxmemory-policy",
			"value": "allkeys-lru",
		},
	}

	commandResult := &plugin.CommandResult{
		Success: true,
		Output:  "OK",
	}

	mockPlugin.On("IsConnected").Return(true)
	mockPlugin.On("Execute", ctx, mock.AnythingOfType("*plugin.Command")).Return(commandResult, nil)

	// Act
	result, err := adapter.ExecuteFix(ctx, fix)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "OK", result.Message)
	mockPlugin.AssertExpectations(t)
}

// TestAdapter_ExecuteFix_NotConnected tests fix execution when not connected
func TestAdapter_ExecuteFix_NotConnected(t *testing.T) {
	// Arrange
	mockPlugin := new(MockUnderlyingPlugin)
	adapter := NewPluginAdapter(mockPlugin)

	ctx := context.Background()
	fix := &contracts.FixAction{
		ID:   "test-fix",
		Type: contracts.FixTypeCommand,
	}

	mockPlugin.On("IsConnected").Return(false)

	// Act
	result, err := adapter.ExecuteFix(ctx, fix)

	// Assert
	assert.NoError(t, err) // Returns result with error message, not Go error
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not connected")
	mockPlugin.AssertExpectations(t)
}

// TestAdapter_CollectLogs_WithFiltering tests log collection with filtering
func TestAdapter_CollectLogs_WithFiltering(t *testing.T) {
	// Arrange
	mockPlugin := new(MockUnderlyingPlugin)
	adapter := NewPluginAdapter(mockPlugin)

	ctx := context.Background()
	target := &contracts.TargetConfig{
		Host: "localhost",
		Port: 6379,
	}

	now := time.Now()
	opts := &contracts.LogOptions{
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now,
		Limit:     10,
	}

	diagData := &plugin.DiagnosticData{
		SlowLogs: []plugin.SlowLogEntry{
			{
				ID:       "1",
				Time:     now.Add(-30 * time.Minute),
				Duration: 500 * time.Millisecond,
				Query:    "GET key1",
				ClientIP: "127.0.0.1",
			},
			{
				ID:       "2",
				Time:     now.Add(-2 * time.Hour), // Outside time range
				Duration: 300 * time.Millisecond,
				Query:    "SET key2 value",
				ClientIP: "127.0.0.1",
			},
		},
	}

	mockPlugin.On("IsConnected").Return(true)
	mockPlugin.On("GetDiagnosticData", ctx).Return(diagData, nil)

	// Act
	result, err := adapter.CollectLogs(ctx, target, opts)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Entries)) // Only one entry in time range
	assert.Equal(t, "1", result.Entries[0].Fields["id"])
	mockPlugin.AssertExpectations(t)
}

// TestAdapter_Metadata tests basic metadata methods
func TestAdapter_Metadata(t *testing.T) {
	// Arrange
	mockPlugin := new(MockUnderlyingPlugin)
	adapter := NewPluginAdapter(mockPlugin)

	mockPlugin.On("Name").Return("Test Plugin")
	mockPlugin.On("Version").Return("1.0.0")

	// Act & Assert
	assert.Equal(t, "Test Plugin", adapter.Name())
	assert.Equal(t, "1.0.0", adapter.Version())
	
	versions := adapter.SupportedVersions()
	assert.NotNil(t, versions)
	assert.Contains(t, versions, "1.0.0")
	
	mockPlugin.AssertExpectations(t)
}

// TestAdapter_CanAutoFix_ReturnsFalse tests that CanAutoFix returns false
// since the underlying plugin interface doesn't support this capability
func TestAdapter_CanAutoFix_ReturnsFalse(t *testing.T) {
	// Arrange
	mockPlugin := new(MockUnderlyingPlugin)
	adapter := NewPluginAdapter(mockPlugin)

	issue := &contracts.Issue{
		ID:       "test-issue",
		Title:    "Test Issue",
		Severity: contracts.SeverityWarning,
	}

	// Act
	canFix, action := adapter.CanAutoFix(issue)

	// Assert
	assert.False(t, canFix)
	assert.Nil(t, action)
}
