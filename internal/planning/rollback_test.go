package planning

import (
	"context"
	"errors"
	"testing"
)

func TestRollback_ReverseOrder(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.results["step1_rollback"] = "rollback1"
	mockTools.results["step2_rollback"] = "rollback2"

	executor := NewDefaultStepExecutor(mockTools, nil)
	rollbackManager := NewRollbackManager(executor)

	plan := NewPlan("rollback-plan", "Rollback Test", []Step{
		{
			ID:   "step1",
			Name: "Step 1",
			Type: StepTypeToolCall,
			Rollback: &ActionSpec{
				ToolName: "step1_rollback",
			},
		},
		{
			ID:   "step2",
			Name: "Step 2",
			Type: StepTypeToolCall,
			DependsOn: []string{"step1"},
			Rollback: &ActionSpec{
				ToolName: "step2_rollback",
			},
		},
		{
			ID:        "step3",
			Name:      "Step 3",
			Type:      StepTypeToolCall,
			DependsOn: []string{"step2"},
			// No rollback action
		},
	})

	// Create state with completed steps
	state := NewExecutionState(plan.ID)
	state.MarkStepCompleted("step1", "result1")
	state.MarkStepCompleted("step2", "result2")
	state.MarkStepFailed("step3", errors.New("step3 failed"))

	// Perform rollback
	err := rollbackManager.Rollback(context.Background(), plan, state)
	if err != nil {
		t.Errorf("unexpected error during rollback: %v", err)
	}

	// Check that rollback was performed in reverse order
	// step2 should be rolled back before step1
	if len(mockTools.calls) != 2 {
		t.Errorf("expected 2 rollback calls, got %d", len(mockTools.calls))
	}

	// Verify reverse order
	if mockTools.calls[0] != "step2_rollback" {
		t.Errorf("expected first rollback to be step2_rollback, got %s", mockTools.calls[0])
	}
	if mockTools.calls[1] != "step1_rollback" {
		t.Errorf("expected second rollback to be step1_rollback, got %s", mockTools.calls[1])
	}

	// Check that steps are marked as rolled back
	if state.StepStates["step1"].Status != StepStatusRolledBack {
		t.Errorf("expected step1 to be RolledBack, got %s", state.StepStates["step1"].Status)
	}
	if state.StepStates["step2"].Status != StepStatusRolledBack {
		t.Errorf("expected step2 to be RolledBack, got %s", state.StepStates["step2"].Status)
	}
}

func TestRollback_SkipNoRollback(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.results["step1_rollback"] = "rollback1"

	executor := NewDefaultStepExecutor(mockTools, nil)
	rollbackManager := NewRollbackManager(executor)

	plan := NewPlan("rollback-plan2", "Skip Test", []Step{
		{
			ID:   "step1",
			Name: "Step 1",
			Type: StepTypeToolCall,
			Rollback: &ActionSpec{
				ToolName: "step1_rollback",
			},
		},
		{
			ID:        "step2",
			Name:      "Step 2",
			Type:      StepTypeToolCall,
			DependsOn: []string{"step1"},
			// No rollback action
		},
	})

	state := NewExecutionState(plan.ID)
	state.MarkStepCompleted("step1", "result1")
	state.MarkStepCompleted("step2", "result2")

	err := rollbackManager.Rollback(context.Background(), plan, state)
	if err != nil {
		t.Errorf("unexpected error during rollback: %v", err)
	}

	// Only step1 should have been rolled back
	if len(mockTools.calls) != 1 {
		t.Errorf("expected 1 rollback call, got %d", len(mockTools.calls))
	}
	if mockTools.calls[0] != "step1_rollback" {
		t.Errorf("expected step1_rollback, got %s", mockTools.calls[0])
	}
}

func TestRollback_PartialRollback(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.results["step1_rollback"] = "rollback1"

	executor := NewDefaultStepExecutor(mockTools, nil)
	rollbackManager := NewRollbackManager(executor)

	plan := NewPlan("rollback-plan3", "Partial Rollback Test", []Step{
		{
			ID:   "step1",
			Name: "Step 1",
			Type: StepTypeToolCall,
			Rollback: &ActionSpec{
				ToolName: "step1_rollback",
			},
		},
		{
			ID:        "step2",
			Name:      "Step 2",
			Type:      StepTypeToolCall,
			DependsOn: []string{"step1"},
		},
	})

	state := NewExecutionState(plan.ID)
	state.MarkStepCompleted("step1", "result1")
	// step2 not completed

	err := rollbackManager.Rollback(context.Background(), plan, state)
	if err != nil {
		t.Errorf("unexpected error during rollback: %v", err)
	}

	// Only step1 should be rolled back (step2 was never completed)
	if len(mockTools.calls) != 1 {
		t.Errorf("expected 1 rollback call, got %d", len(mockTools.calls))
	}
}

func TestRollback_NoCompletedSteps(t *testing.T) {
	mockTools := NewMockToolRegistry()
	executor := NewDefaultStepExecutor(mockTools, nil)
	rollbackManager := NewRollbackManager(executor)

	plan := NewPlan("rollback-plan4", "No Completed Steps", []Step{
		{
			ID:   "step1",
			Name: "Step 1",
			Type: StepTypeToolCall,
			Rollback: &ActionSpec{
				ToolName: "step1_rollback",
			},
		},
	})

	state := NewExecutionState(plan.ID)
	// No steps completed

	err := rollbackManager.Rollback(context.Background(), plan, state)
	if err != nil {
		t.Errorf("unexpected error during rollback: %v", err)
	}

	// No rollback calls should have been made
	if len(mockTools.calls) != 0 {
		t.Errorf("expected 0 rollback calls, got %d", len(mockTools.calls))
	}
}

func TestRollback_ContinueOnFailure(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.errors["step2_rollback"] = errors.New("rollback failed")
	mockTools.results["step1_rollback"] = "rollback1"

	executor := NewDefaultStepExecutor(mockTools, nil)
	rollbackManager := NewRollbackManager(executor)

	plan := NewPlan("rollback-plan5", "Continue On Failure", []Step{
		{
			ID:   "step1",
			Name: "Step 1",
			Type: StepTypeToolCall,
			Rollback: &ActionSpec{
				ToolName: "step1_rollback",
			},
		},
		{
			ID:   "step2",
			Name: "Step 2",
			Type: StepTypeToolCall,
			Rollback: &ActionSpec{
				ToolName: "step2_rollback",
			},
		},
	})

	state := NewExecutionState(plan.ID)
	state.MarkStepCompleted("step1", "result1")
	state.MarkStepCompleted("step2", "result2")

	err := rollbackManager.Rollback(context.Background(), plan, state)
	if err != nil {
		// Rollback manager continues on error but returns nil
		// Check that both rollbacks were attempted
	}

	// Both rollbacks should have been attempted
	if len(mockTools.calls) != 2 {
		t.Errorf("expected 2 rollback calls, got %d", len(mockTools.calls))
	}

	// step2 rollback should have failed but step1 should succeed
	if state.StepStates["step1"].Status != StepStatusRolledBack {
		t.Errorf("expected step1 to be RolledBack, got %s", state.StepStates["step1"].Status)
	}
}

func TestRollbackManager_CanRollback(t *testing.T) {
	mockTools := NewMockToolRegistry()
	executor := NewDefaultStepExecutor(mockTools, nil)
	rollbackManager := NewRollbackManager(executor)

	plan := NewPlan("rollback-plan6", "Can Rollback Test", []Step{
		{
			ID:   "step1",
			Name: "Step 1",
			Type: StepTypeToolCall,
			Rollback: &ActionSpec{
				ToolName: "step1_rollback",
			},
		},
		{
			ID:        "step2",
			Name:      "Step 2",
			Type:      StepTypeToolCall,
			DependsOn: []string{"step1"},
		},
	})

	state := NewExecutionState(plan.ID)
	state.MarkStepCompleted("step1", "result1")

	if !rollbackManager.CanRollback(plan, state) {
		t.Error("expected CanRollback to return true")
	}

	// Test with no rollbackable steps
	plan2 := NewPlan("rollback-plan7", "No Rollback", []Step{
		{
			ID:   "step1",
			Name: "Step 1",
			Type: StepTypeToolCall,
			// No rollback action
		},
	})

	state2 := NewExecutionState(plan2.ID)
	state2.MarkStepCompleted("step1", "result1")

	if rollbackManager.CanRollback(plan2, state2) {
		t.Error("expected CanRollback to return false")
	}
}

func TestRollbackManager_GetRollbackableSteps(t *testing.T) {
	mockTools := NewMockToolRegistry()
	executor := NewDefaultStepExecutor(mockTools, nil)
	rollbackManager := NewRollbackManager(executor)

	plan := NewPlan("rollback-plan8", "Get Rollbackable Steps", []Step{
		{
			ID:   "step1",
			Name: "Step 1",
			Type: StepTypeToolCall,
			Rollback: &ActionSpec{
				ToolName: "step1_rollback",
			},
		},
		{
			ID:        "step2",
			Name:      "Step 2",
			Type:      StepTypeToolCall,
			DependsOn: []string{"step1"},
		},
		{
			ID:   "step3",
			Name: "Step 3",
			Type: StepTypeToolCall,
			Rollback: &ActionSpec{
				ToolName: "step3_rollback",
			},
		},
	})

	state := NewExecutionState(plan.ID)
	state.MarkStepCompleted("step1", "result1")
	state.MarkStepCompleted("step3", "result3")

	rollbackableSteps := rollbackManager.GetRollbackableSteps(plan, state)

	if len(rollbackableSteps) != 2 {
		t.Errorf("expected 2 rollbackable steps, got %d", len(rollbackableSteps))
	}

	// Verify that step1 and step3 are in the list
	hasStep1 := false
	hasStep3 := false
	for _, step := range rollbackableSteps {
		if step.ID == "step1" {
			hasStep1 = true
		}
		if step.ID == "step3" {
			hasStep3 = true
		}
	}

	if !hasStep1 || !hasStep3 {
		t.Error("expected both step1 and step3 in rollbackable steps")
	}
}
