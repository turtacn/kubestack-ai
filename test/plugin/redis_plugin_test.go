package plugin_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/plugins/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisPluginDiagnose(t *testing.T) {
	// Setup: Mock Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	plugin := &redis.RedisPlugin{}
	config := map[string]interface{}{
		"addr": mr.Addr(),
	}
	err = plugin.Init(config)
	require.NoError(t, err)

	req := &models.DiagnosisRequest{TargetMiddleware: enum.Redis}

	// Action: 执行诊断
	result, err := plugin.Diagnose(context.Background(), req)
	assert.NoError(t, err)

	// Assert: 验证结果
	// Note: miniredis might not simulate high memory or slow logs unless we force it,
	// so we expect no issues but a valid result object.
	assert.NotNil(t, result)
	assert.Empty(t, result.Issues)

	// To test issues, we would need to mock the redis client inside the plugin
	// or use a mockable interface, but the plugin uses struct directly.
	// For this integration test, valid connection is the main check.
}
