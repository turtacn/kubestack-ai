package diagnosis_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/kubestack-ai/kubestack-ai/internal/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// Mock Registry and Plugin (reuse mock from registry test or create local)
type MockPlugin struct {
	mock.Mock
}

func (m *MockPlugin) Name() string { return "Mock" }
func (m *MockPlugin) Type() plugin.MiddlewareType { return plugin.MiddlewareRedis }
func (m *MockPlugin) Version() string { return "1.0" }
func (m *MockPlugin) Connect(ctx context.Context, config *plugin.ConnectionConfig) error { return nil }
func (m *MockPlugin) Disconnect(ctx context.Context) error { return nil }
func (m *MockPlugin) Ping(ctx context.Context) error { return nil }
func (m *MockPlugin) IsConnected() bool { return true }
func (m *MockPlugin) CollectMetrics(ctx context.Context) (*plugin.MetricsSnapshot, error) {
	return nil, nil
}
func (m *MockPlugin) CollectSpecificMetric(ctx context.Context, name string) (interface{}, error) {
	return nil, nil
}
func (m *MockPlugin) Execute(ctx context.Context, cmd *plugin.Command) (*plugin.CommandResult, error) {
	return nil, nil
}
func (m *MockPlugin) SupportedCommands() []plugin.CommandSpec { return nil }
func (m *MockPlugin) GetDiagnosticData(ctx context.Context) (*plugin.DiagnosticData, error) {
	args := m.Called(ctx)
	return args.Get(0).(*plugin.DiagnosticData), args.Error(1)
}
func (m *MockPlugin) GetBuiltinRules() []plugin.DiagnosisRule {
	args := m.Called()
	return args.Get(0).([]plugin.DiagnosisRule)
}

func TestDiagnosisEngine_Diagnose(t *testing.T) {
	registry := plugin.NewPluginRegistry()
	mockP := new(MockPlugin)

	registry.RegisterFactory(plugin.MiddlewareRedis, func(cfg *plugin.PluginConfig) (plugin.MiddlewarePlugin, error) {
		return mockP, nil
	})
	registry.CreatePlugin(context.Background(), &plugin.PluginConfig{Type: plugin.MiddlewareRedis, Connection: &plugin.ConnectionConfig{}})

	// Setup data
	data := &plugin.DiagnosticData{
		Metrics: &plugin.MetricsSnapshot{
			Metrics: map[string]plugin.MetricValue{
				"val": {Value: 100},
			},
		},
	}
	mockP.On("GetDiagnosticData", mock.Anything).Return(data, nil)

	// Setup Rule
	rule := plugin.DiagnosisRule{
		ID: "test-rule",
		Condition: "metrics.val > 50",
		Severity: plugin.SeverityError,
		Name: "Test Rule",
	}
	mockP.On("GetBuiltinRules").Return([]plugin.DiagnosisRule{rule})

	engine := diagnosis.NewDiagnosisEngine(registry)

	res, err := engine.Diagnose(context.Background(), &diagnosis.DiagnosisRequest{
		MiddlewareType: plugin.MiddlewareRedis,
	})

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Len(t, res.Issues, 1)
	assert.Equal(t, "test-rule", res.Issues[0].RuleID)
	assert.Equal(t, plugin.SeverityError, res.Issues[0].Severity)
}
