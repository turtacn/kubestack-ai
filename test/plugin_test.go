package test

import (
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// MockPlugin for testing
type MockPlugin struct {
	name   string
	state  string
}

func (p *MockPlugin) Name() string { return p.name }
func (p *MockPlugin) Version() string { return "1.0.0" }
func (p *MockPlugin) Description() string { return "Mock Plugin" }
func (p *MockPlugin) SupportedMiddlewareVersions() []string { return []string{"1.0"} }
func (p *MockPlugin) Initialize(config *plugin.PluginConfig) error { return nil }
func (p *MockPlugin) Shutdown() error { return nil }
func (p *MockPlugin) Collector() plugin.DataCollector { return nil }
func (p *MockPlugin) Parser() plugin.MetricParser { return nil }
func (p *MockPlugin) HealthChecker() plugin.HealthChecker { return nil }

type MockPluginFactory struct {
	name string
}

func (f *MockPluginFactory) Create() plugin.Plugin {
	return &MockPlugin{name: f.name}
}

func (f *MockPluginFactory) Metadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:       f.name,
		Version:    "1.0.0",
		APIVersion: "v1",
	}
}

func TestPluginManager_Lifecycle(t *testing.T) {
	logger := zap.NewNop()
	registry := plugin.NewRegistry()
	manager := plugin.NewPluginManager(registry, logger)

	pluginName := "mock_lifecycle"
	err := registry.Register(&MockPluginFactory{name: pluginName})
	assert.NoError(t, err)

	config := &plugin.PluginConfig{
		Name:    pluginName,
		Enabled: true,
	}

	// Load
	err = manager.LoadPlugin(pluginName, config)
	assert.NoError(t, err)

	// Enable
	err = manager.EnablePlugin(pluginName)
	assert.NoError(t, err)

	p, err := manager.GetPlugin(pluginName)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// Disable
	err = manager.DisablePlugin(pluginName)
	assert.NoError(t, err)

	_, err = manager.GetPlugin(pluginName)
	assert.Error(t, err)

	// Unload
	err = manager.UnloadPlugin(pluginName)
	assert.NoError(t, err)
}

func TestRegistry_Conflict(t *testing.T) {
	registry := plugin.NewRegistry()
	name := "mock_conflict"

	err := registry.Register(&MockPluginFactory{name: name})
	assert.NoError(t, err)

	err = registry.Register(&MockPluginFactory{name: name})
	assert.ErrorIs(t, err, plugin.ErrPluginNameConflict)
}

func TestConfigWatcher_Integration(t *testing.T) {
    // This would require file system operations, mock it or use a temp dir.
    // For unit test, we can check basic initialization.
    manager := plugin.NewPluginManager(plugin.NewRegistry(), zap.NewNop())
    watcher, err := plugin.NewConfigWatcher(manager, "/tmp", zap.NewNop())
    assert.NoError(t, err)
    watcher.Stop()
}
