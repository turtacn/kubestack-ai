package planning

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPlanEngine_ExecutePlan_Success(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.results["step1_tool"] = "result1"
	mockTools.results["step2_tool"] = "result2"
	mockTools.results["step3_tool"] = "result3"

	executor := NewDefaultStepExecutor(mockTools, nil)
	stateStore := NewMemoryStateStore()
	config := DefaultPlanEngineConfig()

	engine := NewPlanEngine(executor, stateStore, config)

	plan := NewPlan("plan1", "Test Plan", []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "step1_tool"}},
		{ID: "step2", Name: "Step 2", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "step2_tool"}, DependsOn: []string{"step1"}},
		{ID: "step3", Name: "Step 3", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "step3_tool"}, DependsOn: []string{"step2"}},
	})

	state, err := engine.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Status != PlanStatusCompleted {
		t.Errorf("expected status Completed, got %s", state.Status)
	}

	// Check all steps completed
	for _, stepID := range []string{"step1", "step2", "step3"} {
		stepState, exists := state.StepStates[stepID]
		if !exists {
			t.Errorf("step state for %s not found", stepID)
			continue
		}
		if stepState.Status != StepStatusCompleted {
			t.Errorf("expected step %s to be Completed, got %s", stepID, stepState.Status)
		}
	}

	// Verify execution order
	if len(mockTools.calls) != 3 {
		t.Errorf("expected 3 tool calls, got %d", len(mockTools.calls))
	}
}

func TestPlanEngine_ExecutePlan_PartialFailure(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.results["step1_tool"] = "result1"
	mockTools.errors["step2_tool"] = errors.New("step2 failed")

	executor := NewDefaultStepExecutor(mockTools, nil)
	stateStore := NewMemoryStateStore()
	config := DefaultPlanEngineConfig()
	config.EnableRollback = false // Disable for this test

	engine := NewPlanEngine(executor, stateStore, config)

	plan := NewPlan("plan2", "Failing Plan", []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "step1_tool"}},
		{ID: "step2", Name: "Step 2", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "step2_tool"}, DependsOn: []string{"step1"}},
		{ID: "step3", Name: "Step 3", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "step3_tool"}, DependsOn: []string{"step2"}},
	})

	state, err := engine.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Error("expected error for failing plan")
	}

	if state.Status != PlanStatusFailed {
		t.Errorf("expected status Failed, got %s", state.Status)
	}

	// Check step1 completed
	step1State := state.StepStates["step1"]
	if step1State.Status != StepStatusCompleted {
		t.Errorf("expected step1 to be Completed, got %s", step1State.Status)
	}

	// Check step2 failed
	step2State := state.StepStates["step2"]
	if step2State.Status != StepStatusFailed {
		t.Errorf("expected step2 to be Failed, got %s", step2State.Status)
	}

	// Check step3 not executed (or pending)
	step3State, exists := state.StepStates["step3"]
	if exists && step3State.Status == StepStatusCompleted {
		t.Error("step3 should not have completed after step2 failure")
	}
}

func TestPlanEngine_ExecutePlan_ParallelExecution(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.results["stepA_tool"] = "resultA"
	mockTools.results["stepB_tool"] = "resultB"
	mockTools.results["stepC_tool"] = "resultC"

	executor := NewDefaultStepExecutor(mockTools, nil)
	stateStore := NewMemoryStateStore()
	config := DefaultPlanEngineConfig()

	engine := NewPlanEngine(executor, stateStore, config)

	plan := NewPlan("plan3", "Parallel Plan", []Step{
		{ID: "stepA", Name: "Step A", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "stepA_tool"}},
		{ID: "stepB", Name: "Step B", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "stepB_tool"}},
		{ID: "stepC", Name: "Step C", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "stepC_tool"}, DependsOn: []string{"stepA", "stepB"}},
	})

	state, err := engine.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Status != PlanStatusCompleted {
		t.Errorf("expected status Completed, got %s", state.Status)
	}

	// All steps should be completed
	for _, stepID := range []string{"stepA", "stepB", "stepC"} {
		stepState := state.StepStates[stepID]
		if stepState.Status != StepStatusCompleted {
			t.Errorf("expected step %s to be Completed, got %s", stepID, stepState.Status)
		}
	}
}

func TestPlanEngine_ExecutePlan_WithRollback(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.results["step1_tool"] = "result1"
	mockTools.errors["step2_tool"] = errors.New("step2 failed")
	mockTools.results["step1_rollback"] = "rollback1"

	executor := NewDefaultStepExecutor(mockTools, nil)
	stateStore := NewMemoryStateStore()
	config := DefaultPlanEngineConfig()
	config.EnableRollback = true

	engine := NewPlanEngine(executor, stateStore, config)

	plan := NewPlan("plan4", "Rollback Plan", []Step{
		{
			ID:   "step1",
			Name: "Step 1",
			Type: StepTypeToolCall,
			Action: ActionSpec{
				ToolName: "step1_tool",
			},
			Rollback: &ActionSpec{
				ToolName: "step1_rollback",
			},
		},
		{
			ID:   "step2",
			Name: "Step 2",
			Type: StepTypeToolCall,
			Action: ActionSpec{
				ToolName: "step2_tool",
			},
			DependsOn: []string{"step1"},
		},
	})

	state, err := engine.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Error("expected error for failing plan")
	}

	if state.Status != PlanStatusRolledBack {
		t.Errorf("expected status RolledBack, got %s", state.Status)
	}

	// Check that step1 was rolled back
	step1State := state.StepStates["step1"]
	if step1State.Status != StepStatusRolledBack {
		t.Errorf("expected step1 to be RolledBack, got %s", step1State.Status)
	}
}

func TestPlanEngine_GetState(t *testing.T) {
	mockTools := NewMockToolRegistry()
	executor := NewDefaultStepExecutor(mockTools, nil)
	stateStore := NewMemoryStateStore()
	config := DefaultPlanEngineConfig()

	engine := NewPlanEngine(executor, stateStore, config)

	plan := NewPlan("plan5", "State Test Plan", []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "tool1"}},
	})

	_, err := engine.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Retrieve state
	state, err := engine.GetState("plan5")
	if err != nil {
		t.Fatalf("failed to get state: %v", err)
	}

	if state.PlanID != "plan5" {
		t.Errorf("expected plan ID 'plan5', got '%s'", state.PlanID)
	}
}

func TestPlanEngine_CancelPlan(t *testing.T) {
	mockTools := NewMockToolRegistry()
	// Set custom execute function to make it slow
	mockTools.executeFunc = func(ctx context.Context, toolName string, args map[string]any) (any, error) {
		mockTools.calls = append(mockTools.calls, toolName)
		select {
		case <-time.After(5 * time.Second):
			return "result", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	executor := NewDefaultStepExecutor(mockTools, nil)
	stateStore := NewMemoryStateStore()
	config := DefaultPlanEngineConfig()

	engine := NewPlanEngine(executor, stateStore, config)

	plan := NewPlan("plan6", "Cancel Test Plan", []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "slow_tool"}},
	})

	// Start execution in background
	go func() {
		engine.ExecutePlan(context.Background(), plan)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the plan
	err := engine.CancelPlan("plan6")
	if err != nil {
		t.Errorf("unexpected error cancelling plan: %v", err)
	}

	// Give it time to process cancellation
	time.Sleep(100 * time.Millisecond)

	// Check state
	state, err := engine.GetState("plan6")
	if err != nil {
		t.Fatalf("failed to get state: %v", err)
	}

	if state.Status != PlanStatusCancelled {
		t.Errorf("expected status Cancelled, got %s", state.Status)
	}
}

func TestPlanEngine_ListExecutions(t *testing.T) {
	mockTools := NewMockToolRegistry()
	executor := NewDefaultStepExecutor(mockTools, nil)
	stateStore := NewMemoryStateStore()
	config := DefaultPlanEngineConfig()

	engine := NewPlanEngine(executor, stateStore, config)

	// Execute multiple plans
	for i := 1; i <= 3; i++ {
		plan := NewPlan("plan"+string(rune('0'+i)), "Plan "+string(rune('0'+i)), []Step{
			{ID: "step1", Name: "Step 1", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "tool"}},
		})
		engine.ExecutePlan(context.Background(), plan)
	}

	executions, err := engine.ListExecutions()
	if err != nil {
		t.Fatalf("failed to list executions: %v", err)
	}

	if len(executions) != 3 {
		t.Errorf("expected 3 executions, got %d", len(executions))
	}
}

func TestPlanEngine_ValidatePlan(t *testing.T) {
	mockTools := NewMockToolRegistry()
	executor := NewDefaultStepExecutor(mockTools, nil)
	stateStore := NewMemoryStateStore()
	config := DefaultPlanEngineConfig()

	engine := NewPlanEngine(executor, stateStore, config)

	// Invalid plan (empty steps)
	plan := NewPlan("invalid", "Invalid Plan", []Step{})

	_, err := engine.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Error("expected validation error for empty plan")
	}
}

func TestExecutionState_Helpers(t *testing.T) {
	state := NewExecutionState("test-plan")

	// Test GetStepState
	stepState := state.GetStepState("step1")
	if stepState.StepID != "step1" {
		t.Errorf("expected step ID 'step1', got '%s'", stepState.StepID)
	}

	// Test MarkStepStarted
	state.MarkStepStarted("step1")
	if state.StepStates["step1"].Status != StepStatusRunning {
		t.Errorf("expected status Running, got %s", state.StepStates["step1"].Status)
	}

	// Test MarkStepCompleted
	state.MarkStepCompleted("step1", "result")
	if state.StepStates["step1"].Status != StepStatusCompleted {
		t.Errorf("expected status Completed, got %s", state.StepStates["step1"].Status)
	}
	if state.StepStates["step1"].Output != "result" {
		t.Errorf("expected output 'result', got %v", state.StepStates["step1"].Output)
	}

	// Test IsStepCompleted
	if !state.IsStepCompleted("step1") {
		t.Error("expected step1 to be completed")
	}

	// Test MarkStepFailed
	state.MarkStepFailed("step2", errors.New("test error"))
	if state.StepStates["step2"].Status != StepStatusFailed {
		t.Errorf("expected status Failed, got %s", state.StepStates["step2"].Status)
	}

	// Test HasFailedSteps
	if !state.HasFailedSteps() {
		t.Error("expected to have failed steps")
	}

	// Test GetCompletedSteps
	completed := state.GetCompletedSteps()
	if len(completed) != 1 || completed[0] != "step1" {
		t.Errorf("expected ['step1'], got %v", completed)
	}
}
