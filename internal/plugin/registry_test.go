package plugin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// MockMiddlewarePlugin
type MockMiddlewarePlugin struct {
	mock.Mock
}

func (m *MockMiddlewarePlugin) Name() string {
	args := m.Called()
	return args.String(0)
}
func (m *MockMiddlewarePlugin) Type() plugin.MiddlewareType {
	args := m.Called()
	return args.Get(0).(plugin.MiddlewareType)
}
func (m *MockMiddlewarePlugin) Version() string {
	args := m.Called()
	return args.String(0)
}
func (m *MockMiddlewarePlugin) Connect(ctx context.Context, config *plugin.ConnectionConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}
func (m *MockMiddlewarePlugin) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *MockMiddlewarePlugin) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *MockMiddlewarePlugin) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}
func (m *MockMiddlewarePlugin) CollectMetrics(ctx context.Context) (*plugin.MetricsSnapshot, error) {
	args := m.Called(ctx)
	return args.Get(0).(*plugin.MetricsSnapshot), args.Error(1)
}
func (m *MockMiddlewarePlugin) CollectSpecificMetric(ctx context.Context, metricName string) (interface{}, error) {
	args := m.Called(ctx, metricName)
	return args.Get(0), args.Error(1)
}
func (m *MockMiddlewarePlugin) Execute(ctx context.Context, cmd *plugin.Command) (*plugin.CommandResult, error) {
	args := m.Called(ctx, cmd)
	return args.Get(0).(*plugin.CommandResult), args.Error(1)
}
func (m *MockMiddlewarePlugin) SupportedCommands() []plugin.CommandSpec {
	args := m.Called()
	return args.Get(0).([]plugin.CommandSpec)
}
func (m *MockMiddlewarePlugin) GetDiagnosticData(ctx context.Context) (*plugin.DiagnosticData, error) {
	args := m.Called(ctx)
	return args.Get(0).(*plugin.DiagnosticData), args.Error(1)
}
func (m *MockMiddlewarePlugin) GetBuiltinRules() []plugin.DiagnosisRule {
	args := m.Called()
	return args.Get(0).([]plugin.DiagnosisRule)
}

func TestPluginRegistry_RegisterAndCreate(t *testing.T) {
	registry := plugin.NewPluginRegistry()
	mockPlugin := new(MockMiddlewarePlugin)

	mockPlugin.On("Connect", mock.Anything, mock.Anything).Return(nil)
	mockPlugin.On("Name").Return("Mock Plugin")

	factory := func(cfg *plugin.PluginConfig) (plugin.MiddlewarePlugin, error) {
		return mockPlugin, nil
	}

	err := registry.RegisterFactory(plugin.MiddlewareRedis, factory)
	assert.NoError(t, err)

	// Create
	cfg := &plugin.PluginConfig{
		Type:       plugin.MiddlewareRedis,
		Connection: &plugin.ConnectionConfig{Host: "localhost", Port: 6379},
	}
	p, err := registry.CreatePlugin(context.Background(), cfg)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, mockPlugin, p)

	// Satisfy Name() expectation
	_ = p.Name()

	mockPlugin.AssertExpectations(t)
}

func TestPluginRegistry_DuplicateRegister(t *testing.T) {
	registry := plugin.NewPluginRegistry()
	factory := func(cfg *plugin.PluginConfig) (plugin.MiddlewarePlugin, error) { return nil, nil }

	err := registry.RegisterFactory(plugin.MiddlewareRedis, factory)
	assert.NoError(t, err)

	err = registry.RegisterFactory(plugin.MiddlewareRedis, factory)
	assert.Error(t, err)
}

func TestPluginRegistry_GetPlugin(t *testing.T) {
	registry := plugin.NewPluginRegistry()
	// Manually inject plugin for testing Get without Create overhead if needed
	// But registry fields are private. So we must use Register + Create.
	mockPlugin := new(MockMiddlewarePlugin)
	mockPlugin.On("Connect", mock.Anything, mock.Anything).Return(nil)

	registry.RegisterFactory(plugin.MiddlewareRedis, func(cfg *plugin.PluginConfig) (plugin.MiddlewarePlugin, error) {
		return mockPlugin, nil
	})

	registry.CreatePlugin(context.Background(), &plugin.PluginConfig{Type: plugin.MiddlewareRedis, Connection: &plugin.ConnectionConfig{}})

	p, err := registry.GetPlugin(plugin.MiddlewareRedis)
	assert.NoError(t, err)
	assert.Equal(t, mockPlugin, p)
}
