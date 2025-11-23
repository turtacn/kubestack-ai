package plugin_test

import (
	"context"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/stretchr/testify/assert"
)

// MockPlugin 用于测试的简单插件
type MockPlugin struct {
	NameVal string
	TypeVal string
}

func (p *MockPlugin) Name() string { return p.NameVal }
func (p *MockPlugin) SupportedTypes() []string { return []string{p.TypeVal} }
func (p *MockPlugin) Version() string { return "1.0" }
func (p *MockPlugin) Init(config map[string]interface{}) error { return nil }
func (p *MockPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	return nil, nil
}
func (p *MockPlugin) Shutdown() error { return nil }

func TestPluginRegistration(t *testing.T) {
	registry := plugin.NewRegistry()

	// Action: 注册插件
	redisPlugin := &MockPlugin{NameVal: "redis", TypeVal: "redis"}
	mysqlPlugin := &MockPlugin{NameVal: "mysql", TypeVal: "mysql"}
	registry.Register(redisPlugin)
	registry.Register(mysqlPlugin)

	// Assert: 验证注册
	plugins := registry.List()
	assert.Len(t, plugins, 2)
	assert.Contains(t, plugins, "redis")
	assert.Contains(t, plugins, "mysql")

	// Assert: Verify FindByType
	found := registry.FindByType("redis")
	assert.Len(t, found, 1)
	assert.Equal(t, "redis", found[0].Name())
}
