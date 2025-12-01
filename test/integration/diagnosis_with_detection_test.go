package integration_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/manager"
)

func TestDiagnosisWithAnomalyDetection(t *testing.T) {
	// Start a mock Redis server
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer s.Close()

	// The mock plugins are registered in TestMain.
	loader := manager.NewLoader()
	pm := manager.NewManager(&mockPluginRegistry{}, loader)

	// Create Manager
	diagManager := diagnosis.NewManager(pm, nil, nil, "reports_test", nil)

	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         s.Addr(), // Use the mock server's address
	}

	// Run diagnosis
	result, err := diagManager.RunDiagnosis(context.Background(), req, nil)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// A fresh miniredis instance should have no issues.
	assert.Empty(t, result.Issues)
}
