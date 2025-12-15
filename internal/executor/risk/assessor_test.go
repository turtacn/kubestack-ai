package risk_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/executor/risk"
)

func TestRiskAssessor_HighRisk_DataDeletion(t *testing.T) {
	assessor := risk.NewRiskAssessor(risk.DefaultThresholds())
	ctx := context.Background()
	plan := &models.ExecutionPlan{
		Steps: []*models.ExecutionStep{
			{Action: &models.FixAction{Command: "FLUSHALL"}},
		},
	}
	result, err := assessor.Assess(ctx, plan)
	assert.NoError(t, err)
	assert.Equal(t, risk.RiskLevelHigh, result.Level)
	assert.True(t, result.RequiresConfirm)
}

func TestRiskAssessor_MediumRisk_ConfigChange(t *testing.T) {
	assessor := risk.NewRiskAssessor(risk.DefaultThresholds())
	ctx := context.Background()
	plan := &models.ExecutionPlan{
		Steps: []*models.ExecutionStep{
			{Action: &models.FixAction{Command: "CONFIG SET maxmemory 1gb"}},
		},
	}
	result, err := assessor.Assess(ctx, plan)
	assert.NoError(t, err)
	assert.Equal(t, risk.RiskLevelMedium, result.Level)
}

func TestRiskAssessor_LowRisk_ReadOnly(t *testing.T) {
	assessor := risk.NewRiskAssessor(risk.DefaultThresholds())
	ctx := context.Background()
	plan := &models.ExecutionPlan{
		Steps: []*models.ExecutionStep{
			{Action: &models.FixAction{Command: "INFO replication"}},
		},
	}
	result, err := assessor.Assess(ctx, plan)
	assert.NoError(t, err)
	assert.Equal(t, risk.RiskLevelLow, result.Level)
	assert.False(t, result.RequiresConfirm)
}

func TestRiskAssessor_CombinedRules(t *testing.T) {
	assessor := risk.NewRiskAssessor(risk.DefaultThresholds())
	ctx := context.Background()
	plan := &models.ExecutionPlan{
		Steps: []*models.ExecutionStep{
			{Action: &models.FixAction{Command: "CONFIG SET timeout 100"}},
			{Action: &models.FixAction{Command: "INFO"}},
		},
	}
	result, err := assessor.Assess(ctx, plan)
	assert.NoError(t, err)
	assert.Equal(t, risk.RiskLevelMedium, result.Level)
}
