package executor_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	executor "github.com/kubestack-ai/kubestack-ai/internal/executor"
	"github.com/kubestack-ai/kubestack-ai/internal/executor/confirm"
	"github.com/kubestack-ai/kubestack-ai/internal/executor/plan"
	"github.com/kubestack-ai/kubestack-ai/internal/executor/risk"
	"github.com/kubestack-ai/kubestack-ai/internal/executor/rollback"
)

// --- Mocks ---

type MockPlanner struct{}

func (p *MockPlanner) GeneratePlan(ctx context.Context, issues []models.Issue) (*models.ExecutionPlan, error) {
	return nil, nil
}

func (p *MockPlanner) AnalyzeRisk(ctx context.Context, actions []*models.FixAction) (*models.RiskAssessment, error) {
	return nil, nil
}

func (p *MockPlanner) OptimizeSequence(ctx context.Context, plan *models.ExecutionPlan) (*models.ExecutionPlan, error) {
	return plan, nil
}

type MockConfirmationChannel struct {
	Response      *confirm.ConfirmResponse
	RequestCalled bool
	LastRequest   *confirm.ConfirmRequest
}

func (m *MockConfirmationChannel) Name() string { return "Mock" }
func (m *MockConfirmationChannel) RequestConfirmation(ctx context.Context, req *confirm.ConfirmRequest) (<-chan *confirm.ConfirmResponse, error) {
	m.RequestCalled = true
	m.LastRequest = req
	ch := make(chan *confirm.ConfirmResponse, 1)
	ch <- m.Response
	return ch, nil
}

type MockSnapshotCollector struct {
	Captured bool
}

func (m *MockSnapshotCollector) Collect(ctx context.Context, target *rollback.TargetInfo) (*rollback.StateSnapshot, error) {
	m.Captured = true
	return &rollback.StateSnapshot{
		ID:        "snap-1",
		CreatedAt: time.Now(),
		TargetID:  target.ID,
	}, nil
}
func (m *MockSnapshotCollector) SupportedTypes() []string { return []string{"redis"} }

// --- Tests ---

func TestExecutor_HighRiskOperation_RequiresConfirmation(t *testing.T) {
	ctx := context.Background()
	mockChannel := &MockConfirmationChannel{
		Response: &confirm.ConfirmResponse{Approved: false},
	}

	// Must pass list of channels
	confirmHandler := confirm.NewConfirmationHandler(1*time.Second, []confirm.ConfirmationChannel{mockChannel})

	opts := &executor.ManagerOptions{
		ConfirmationHdlr: confirmHandler,
		RiskThresholds:   risk.DefaultThresholds(),
	}

	mgr := executor.NewManager(&MockPlanner{}, opts)

	execPlan := &models.ExecutionPlan{
		ID:       "plan-high-risk",
		Strategy: models.SerialExecution,
		Steps: []*models.ExecutionStep{
			{
				ID: "step-1",
				Name: "Delete All Keys",
				Action: &models.FixAction{
					Command: "FLUSHALL",
				},
			},
		},
	}

	result, err := mgr.ExecutePlan(ctx, execPlan)

	// Expect error due to rejection
	assert.Error(t, err)
	assert.Nil(t, result) // Result is nil on confirmation rejection based on implementation
	assert.True(t, mockChannel.RequestCalled)
	assert.Equal(t, risk.RiskLevelHigh, mockChannel.LastRequest.RiskLevel)
}

func TestExecutor_PlanPersistence_Recovery(t *testing.T) {
	// Use temporary directory
	tmpDir := t.TempDir()
	store := plan.NewFilePlanStore(tmpDir)

	ctx := context.Background()
	p := &models.ExecutionPlan{
		ID: "plan-persist",
		Strategy: models.SerialExecution,
		Steps: []*models.ExecutionStep{},
	}

	err := store.Save(ctx, p)
	assert.NoError(t, err)

	loaded, err := store.Load(ctx, "plan-persist")
	assert.NoError(t, err)
	assert.Equal(t, p.ID, loaded.ID)
}

func TestRiskAssessor_CombinedRules(t *testing.T) {
	assessor := risk.NewRiskAssessor(risk.DefaultThresholds())
	ctx := context.Background()

	cases := []struct {
		name          string
		plan          *models.ExecutionPlan
		expectedLevel risk.RiskLevel
		expectConfirm bool
	}{
		{
			name: "readonly_low_risk",
			plan: &models.ExecutionPlan{
				Steps: []*models.ExecutionStep{
					{Action: &models.FixAction{Command: "INFO"}},
				},
			},
			expectedLevel: risk.RiskLevelLow,
			expectConfirm: false,
		},
		{
			name: "delete_high_risk",
			plan: &models.ExecutionPlan{
				Steps: []*models.ExecutionStep{
					{Action: &models.FixAction{Command: "DEL key1"}},
				},
			},
			expectedLevel: risk.RiskLevelHigh,
			expectConfirm: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, _ := assessor.Assess(ctx, tc.plan)
			assert.Equal(t, tc.expectedLevel, result.Level)
			assert.Equal(t, tc.expectConfirm, result.RequiresConfirm)
		})
	}
}
