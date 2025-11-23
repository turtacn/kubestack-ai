package plugin_test

import (
	"os"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginLoader(t *testing.T) {
	// Setup: Create test config file
	configContent := `
enabled_plugins:
  - name: redis
    enabled: true
    config:
      addr: "localhost:6379"
  - name: kafka
    enabled: false
    config:
      brokers: ["localhost:9092"]
`
	configFile := "test_plugins.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)
	defer os.Remove(configFile)

	loader := plugin.NewLoader(configFile)

	// Action: 加载插件
	// Since redis is a built-in plugin and we registered the factory in its init(),
	// it should be loadable IF the package was imported.
	// But `internal/plugin` does NOT import `plugins/redis`.
	// The loader uses `pluginFactories` map.
	// We need to register a dummy factory for testing "redis" here because the actual `plugins/redis`
	// package might not be linked into this test binary unless imported.
	// However, this is `package plugin_test`, so we can register mock factories.

	plugin.RegisterPluginFactory("redis", func() plugin.DiagnosticPlugin {
		return &MockPlugin{NameVal: "redis", TypeVal: "redis"}
	})

	err = loader.LoadAll()
	assert.NoError(t, err)

	// Assert: Verify loaded
	registry := loader.GetRegistry()
	assert.NotNil(t, registry.Get("redis"))
	assert.Nil(t, registry.Get("kafka")) // Disabled
}
