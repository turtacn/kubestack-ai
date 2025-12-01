package integration_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiPluginDiagnosis(t *testing.T) {
	// Start a mock Redis server
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer s.Close()

	// Use the globally registered mock plugins
	loader := manager.NewLoader()
	pm := manager.NewManager(&mockPluginRegistry{}, loader)

	// Setup Manager
	diagManager := diagnosis.NewManager(pm, []interfaces.DiagnosisAnalyzer{}, nil, "", nil)

	// Action: Submit diagnosis request
	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         s.Addr(), // Use the mock server's address
	}
	result, err := diagManager.RunDiagnosis(context.Background(), req, nil)
	require.NoError(t, err)

	// Assert: A fresh miniredis instance should have no issues.
	assert.NotNil(t, result)
	assert.Empty(t, result.Issues)
}
