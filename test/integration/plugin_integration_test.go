package integration_test

import (
	"context"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockPlugin for integration
type MockPlugin struct {
	NameVal string
	TypeVal string
}

func (p *MockPlugin) Name() string { return p.NameVal }
func (p *MockPlugin) SupportedTypes() []string { return []string{p.TypeVal} }
func (p *MockPlugin) Version() string { return "1.0" }
func (p *MockPlugin) Init(config map[string]interface{}) error { return nil }
func (p *MockPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	return &models.DiagnosisResult{
		Issues: []*models.Issue{
			{Title: "Issue from " + p.NameVal},
		},
	}, nil
}
func (p *MockPlugin) Shutdown() error { return nil }

func TestMultiPluginDiagnosis(t *testing.T) {
	// Setup: 注册多个插件
	registry := plugin.NewRegistry()
	registry.Register(&MockPlugin{NameVal: "redis", TypeVal: "Redis"})

	// Setup Manager
	manager := diagnosis.NewManager(registry, []interfaces.DiagnosisAnalyzer{}, nil, "")

	// Action: 提交诊断请求
	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         "test-redis",
	}
	result, err := manager.RunDiagnosis(context.Background(), req, nil)
	require.NoError(t, err)

	// Assert: 验证结果
	assert.NotNil(t, result)
	assert.Len(t, result.Issues, 1)
	assert.Contains(t, result.Issues[0].Title, "redis")
}
