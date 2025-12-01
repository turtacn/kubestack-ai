package integration_test

import (
	"os"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"

	// Blank imports for plugins to trigger their registration within this test binary.
	_ "github.com/kubestack-ai/kubestack-ai/plugins/elasticsearch"
	_ "github.com/kubestack-ai/kubestack-ai/plugins/kafka"
	_ "github.com/kubestack-ai/kubestack-ai/plugins/mysql"
	_ "github.com/kubestack-ai/kubestack-ai/plugins/redis"
)

// mockPluginRegistry is a mock implementation of interfaces.PluginRegistry.
// It is defined here centrally to be used by all tests in the package.
type mockPluginRegistry struct{}

func (m *mockPluginRegistry) FindPlugin(name string, versionConstraint string) (*models.PluginManifest, error) {
	// For testing, we return a manifest that points to a statically linked plugin.
	// The plugin manager will use this entrypoint to find the factory registered
	// by the blank imports above.
	return &models.PluginManifest{
		Name:       name,
		Version:    "1.0.0",
		Entrypoint: "static:" + name,
	}, nil
}

func (m *mockPluginRegistry) ListAvailablePlugins() ([]*models.PluginManifest, error) {
	return []*models.PluginManifest{}, nil
}

// TestMain is the entry point for all tests in this package.
// Its primary role is to ensure that the blank imports above are processed,
// which in turn registers all the static plugins before any tests are run.
func TestMain(m *testing.M) {
	// Run all tests
	os.Exit(m.Run())
}
