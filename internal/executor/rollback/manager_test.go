package rollback_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/executor/rollback"
)

type MockCollector struct {}

func (m *MockCollector) Collect(ctx context.Context, target *rollback.TargetInfo) (*rollback.StateSnapshot, error) {
	return &rollback.StateSnapshot{
		ID: "snap-123",
		TargetID: target.ID,
		CreatedAt: time.Now(),
		State: map[string]interface{}{"foo": "bar"},
	}, nil
}
func (m *MockCollector) SupportedTypes() []string { return []string{"redis"} }

func TestRollbackManager_CreateSnapshot(t *testing.T) {
	store := rollback.NewInMemorySnapshotStore()
	mgr := rollback.NewRollbackManager(store)
	mgr.RegisterCollector(&MockCollector{})

	ctx := context.Background()
	// Add steps to trigger inference of targets
	plan := &models.ExecutionPlan{
		ID: "plan-1",
		Steps: []*models.ExecutionStep{
			{Action: &models.FixAction{Command: "CONFIG SET maxmemory"}}, // should trigger redis target
		},
	}

	id, err := mgr.CreateCheckpoint(ctx, plan)
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestRollbackManager_Rollback_Success(t *testing.T) {
	store := rollback.NewInMemorySnapshotStore()
	mgr := rollback.NewRollbackManager(store)
	ctx := context.Background()

	// Seed snapshot
	planID := "plan-roll"
	snap := &rollback.StateSnapshot{
		ID: "snap-test",
		PlanID: planID,
		CreatedAt: time.Now(),
		TargetType: "redis",
		TargetID: "redis-1",
	}
	store.Save(ctx, snap)

	plan := &models.ExecutionPlan{ID: planID}
	result, err := mgr.Rollback(ctx, "any-checkpoint-id", plan)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Details)
}
