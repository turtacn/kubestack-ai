package rca_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/kubestack-ai/kubestack-ai/internal/core/detection/models"
	"github.com/kubestack-ai/kubestack-ai/internal/core/rca"
)

func TestRulesEngine(t *testing.T) {
	// Pass nil to use default rules
	engine := rca.NewRulesEngine(nil)
	anomalies := []models.Anomaly{
		{Type: models.AnomalyTypeHighCPU, Severity: models.SeverityHigh},
		{Type: models.AnomalyTypeHighConnections, Severity: models.SeverityHigh},
	}

	result, err := engine.Analyze(context.Background(), anomalies)

	assert.NoError(t, err)
	assert.Equal(t, "Connection Storm", result.RootCause) // Based on default rules
	assert.NotEmpty(t, result.Recommendations)
}

func TestKnowledgeGraphQuery(t *testing.T) {
	kg := rca.NewKnowledgeGraph()
	anomaly := models.Anomaly{Type: models.AnomalyTypeHighMemory}

	cases := kg.QuerySimilarCases(context.Background(), anomaly)

	assert.GreaterOrEqual(t, len(cases), 1)
	assert.NotEmpty(t, cases[0].Solution)
}
