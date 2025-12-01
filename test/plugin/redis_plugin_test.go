package plugin_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	redisplugin "github.com/kubestack-ai/kubestack-ai/plugins/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisPluginDiagnose(t *testing.T) {
	// Setup: Mock Redis
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	plugin := &redisplugin.RedisPlugin{}

	// Create a request with the mock server's address.
	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         s.Addr(),
	}

	// Action: Run diagnosis.
	result, err := plugin.Diagnose(context.Background(), req)
	assert.NoError(t, err)

	// Assert: A fresh miniredis instance should have no issues.
	assert.NotNil(t, result)
	assert.Empty(t, result.Issues)
}
