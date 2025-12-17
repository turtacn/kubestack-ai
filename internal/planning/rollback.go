package planning

import (
	"context"
	"fmt"
	"time"
)

// RollbackManager manages rollback operations
type RollbackManager struct {
	executor StepExecutor
}

// NewRollbackManager creates a new RollbackManager
func NewRollbackManager(executor StepExecutor) *RollbackManager {
	return &RollbackManager{
		executor: executor,
	}
}

// Rollback performs rollback for a failed plan execution
func (r *RollbackManager) Rollback(ctx context.Context, plan *Plan, state *ExecutionState) error {
	// Collect completed steps with rollback actions
	var stepsToRollback []*Step
	for _, stepID := range state.GetCompletedSteps() {
		step, exists := plan.GetStep(stepID)
		if !exists {
			continue
		}
		if step.Rollback != nil {
			stepsToRollback = append(stepsToRollback, step)
		}
	}

	if len(stepsToRollback) == 0 {
		return nil // Nothing to rollback
	}

	// Reverse order for rollback
	for i := len(stepsToRollback) - 1; i >= 0; i-- {
		step := stepsToRollback[i]
		if err := r.RollbackStep(ctx, step); err != nil {
			// Log error but continue with other rollbacks
			fmt.Printf("Warning: failed to rollback step %s: %v\n", step.ID, err)
			// Store the error in state
			if stepState, exists := state.StepStates[step.ID]; exists {
				stepState.Error = fmt.Sprintf("rollback failed: %v", err)
			}
		} else {
			state.MarkStepRolledBack(step.ID)
		}
	}

	return nil
}

// RollbackStep performs rollback for a single step
func (r *RollbackManager) RollbackStep(ctx context.Context, step *Step) error {
	if step.Rollback == nil {
		return nil
	}

	// Create a rollback step
	rollbackStep := &Step{
		ID:      step.ID + "_rollback",
		Name:    "Rollback: " + step.Name,
		Type:    step.Type,
		Action:  *step.Rollback,
		Timeout: step.Timeout,
	}

	// Execute rollback with retry if configured
	var lastErr error
	maxAttempts := 1
	if step.RetryPolicy != nil {
		maxAttempts = step.RetryPolicy.MaxRetries + 1
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			if step.RetryPolicy != nil && step.RetryPolicy.BackoffMs > 0 {
				select {
				case <-time.After(time.Duration(step.RetryPolicy.BackoffMs) * time.Millisecond):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}

		_, err := r.executor.Execute(ctx, rollbackStep, nil)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("rollback failed after %d attempts: %w", maxAttempts, lastErr)
}

// CanRollback checks if a plan can be rolled back
func (r *RollbackManager) CanRollback(plan *Plan, state *ExecutionState) bool {
	for _, stepID := range state.GetCompletedSteps() {
		step, exists := plan.GetStep(stepID)
		if exists && step.Rollback != nil {
			return true
		}
	}
	return false
}

// GetRollbackableSteps returns a list of steps that can be rolled back
func (r *RollbackManager) GetRollbackableSteps(plan *Plan, state *ExecutionState) []*Step {
	var rollbackable []*Step
	for _, stepID := range state.GetCompletedSteps() {
		step, exists := plan.GetStep(stepID)
		if exists && step.Rollback != nil {
			rollbackable = append(rollbackable, step)
		}
	}
	return rollbackable
}
