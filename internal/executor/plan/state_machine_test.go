package plan_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/executor/plan"
)

func TestStateMachine_ValidTransitions(t *testing.T) {
	p := &models.ExecutionPlan{ID: "test-plan"}
	sm := plan.NewStateMachine(p)

	// Pending -> Approved
	err := sm.Transition("approve")
	assert.NoError(t, err)
	assert.Equal(t, plan.StateApproved, sm.CurrentState())

	// Approved -> Executing
	err = sm.Transition("start")
	assert.NoError(t, err)
	assert.Equal(t, plan.StateExecuting, sm.CurrentState())

	// Executing -> Completed
	err = sm.Transition("complete")
	assert.NoError(t, err)
	assert.Equal(t, plan.StateCompleted, sm.CurrentState())
}

func TestStateMachine_InvalidTransitions(t *testing.T) {
	p := &models.ExecutionPlan{ID: "test-plan"}
	sm := plan.NewStateMachine(p)

	// Pending -> Completed (Invalid)
	err := sm.Transition("complete")
	assert.Error(t, err)
	assert.Equal(t, plan.StatePending, sm.CurrentState())
}
