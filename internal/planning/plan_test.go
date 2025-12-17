package planning

import (
	"testing"
	"time"
)

func TestNewPlan(t *testing.T) {
	steps := []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall},
	}

	plan := NewPlan("plan1", "Test Plan", steps)

	if plan.ID != "plan1" {
		t.Errorf("expected ID 'plan1', got '%s'", plan.ID)
	}
	if plan.Name != "Test Plan" {
		t.Errorf("expected name 'Test Plan', got '%s'", plan.Name)
	}
	if len(plan.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(plan.Steps))
	}
	if plan.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestPlan_Validate_Success(t *testing.T) {
	plan := NewPlan("plan1", "Valid Plan", []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall, DependsOn: []string{}},
		{ID: "step2", Name: "Step 2", Type: StepTypeToolCall, DependsOn: []string{"step1"}},
	})

	err := plan.Validate()
	if err != nil {
		t.Errorf("expected validation to succeed, got error: %v", err)
	}
}

func TestPlan_Validate_EmptySteps(t *testing.T) {
	plan := NewPlan("plan1", "Empty Plan", []Step{})

	err := plan.Validate()
	if err == nil {
		t.Error("expected validation to fail for empty steps")
	}
}

func TestPlan_Validate_DuplicateID(t *testing.T) {
	plan := NewPlan("plan1", "Duplicate ID Plan", []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall},
		{ID: "step1", Name: "Step 2", Type: StepTypeToolCall},
	})

	err := plan.Validate()
	if err == nil {
		t.Error("expected validation to fail for duplicate IDs")
	}
}

func TestPlan_Validate_MissingDependency(t *testing.T) {
	plan := NewPlan("plan1", "Missing Dep Plan", []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall, DependsOn: []string{"nonexistent"}},
	})

	err := plan.Validate()
	if err == nil {
		t.Error("expected validation to fail for missing dependency")
	}
}

func TestPlan_Validate_CyclicDependency(t *testing.T) {
	plan := NewPlan("plan1", "Cyclic Plan", []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall, DependsOn: []string{"step2"}},
		{ID: "step2", Name: "Step 2", Type: StepTypeToolCall, DependsOn: []string{"step1"}},
	})

	err := plan.Validate()
	if err == nil {
		t.Error("expected validation to fail for cyclic dependency")
	}
}

func TestPlan_GetStep(t *testing.T) {
	plan := NewPlan("plan1", "Test Plan", []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall},
		{ID: "step2", Name: "Step 2", Type: StepTypeToolCall},
	})

	step, found := plan.GetStep("step1")
	if !found {
		t.Error("expected to find step1")
	}
	if step.ID != "step1" {
		t.Errorf("expected ID 'step1', got '%s'", step.ID)
	}

	_, found = plan.GetStep("nonexistent")
	if found {
		t.Error("expected not to find nonexistent step")
	}
}

func TestPlan_StepCount(t *testing.T) {
	plan := NewPlan("plan1", "Test Plan", []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall},
		{ID: "step2", Name: "Step 2", Type: StepTypeToolCall},
	})

	count := plan.StepCount()
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestPlan_AddStep(t *testing.T) {
	plan := NewPlan("plan1", "Test Plan", []Step{})

	step := Step{ID: "step1", Name: "Step 1", Type: StepTypeToolCall}
	err := plan.AddStep(step)
	if err != nil {
		t.Errorf("unexpected error adding step: %v", err)
	}

	if plan.StepCount() != 1 {
		t.Errorf("expected count 1, got %d", plan.StepCount())
	}

	// Try to add duplicate
	err = plan.AddStep(step)
	if err == nil {
		t.Error("expected error adding duplicate step")
	}
}

func TestPlan_RemoveStep(t *testing.T) {
	plan := NewPlan("plan1", "Test Plan", []Step{
		{ID: "step1", Name: "Step 1", Type: StepTypeToolCall},
		{ID: "step2", Name: "Step 2", Type: StepTypeToolCall},
	})

	err := plan.RemoveStep("step1")
	if err != nil {
		t.Errorf("unexpected error removing step: %v", err)
	}

	if plan.StepCount() != 1 {
		t.Errorf("expected count 1, got %d", plan.StepCount())
	}

	err = plan.RemoveStep("nonexistent")
	if err == nil {
		t.Error("expected error removing nonexistent step")
	}
}

func TestStep_Timeout(t *testing.T) {
	step := Step{
		ID:      "step1",
		Name:    "Timed Step",
		Type:    StepTypeToolCall,
		Timeout: 5 * time.Second,
	}

	if step.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", step.Timeout)
	}
}

func TestStep_RetryPolicy(t *testing.T) {
	retryPolicy := &RetryPolicy{
		MaxRetries: 3,
		BackoffMs:  100,
	}

	step := Step{
		ID:          "step1",
		Name:        "Retry Step",
		Type:        StepTypeToolCall,
		RetryPolicy: retryPolicy,
	}

	if step.RetryPolicy.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", step.RetryPolicy.MaxRetries)
	}
	if step.RetryPolicy.BackoffMs != 100 {
		t.Errorf("expected backoff 100ms, got %d", step.RetryPolicy.BackoffMs)
	}
}
