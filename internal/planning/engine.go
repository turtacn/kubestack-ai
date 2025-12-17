package planning

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PlanEngineConfig contains configuration for the plan engine
type PlanEngineConfig struct {
	MaxParallel      int  `yaml:"max_parallel" json:"max_parallel"`
	EnableReflection bool `yaml:"enable_reflection" json:"enable_reflection"`
	EnableRollback   bool `yaml:"enable_rollback" json:"enable_rollback"`
	DefaultTimeout   time.Duration `yaml:"default_timeout" json:"default_timeout"`
}

// DefaultPlanEngineConfig returns default configuration
func DefaultPlanEngineConfig() PlanEngineConfig {
	return PlanEngineConfig{
		MaxParallel:      5,
		EnableReflection: true,
		EnableRollback:   true,
		DefaultTimeout:   5 * time.Minute,
	}
}

// PlanEngine is the core execution engine for plans
type PlanEngine struct {
	executor         StepExecutor
	stateStore       StateStore
	reflection       *ReflectionLoop
	rollbackManager  *RollbackManager
	config           PlanEngineConfig
	parallelExecutor *ParallelExecutor
	mu               sync.RWMutex
	activePlans      map[string]context.CancelFunc
}

// NewPlanEngine creates a new PlanEngine
func NewPlanEngine(executor StepExecutor, stateStore StateStore, cfg PlanEngineConfig) *PlanEngine {
	engine := &PlanEngine{
		executor:         executor,
		stateStore:       stateStore,
		config:           cfg,
		parallelExecutor: NewParallelExecutor(cfg.MaxParallel),
		activePlans:      make(map[string]context.CancelFunc),
	}

	// Set plan engine reference in executor if it's DefaultStepExecutor
	if dse, ok := executor.(*DefaultStepExecutor); ok {
		dse.SetPlanEngine(engine)
	}

	engine.rollbackManager = NewRollbackManager(executor)

	return engine
}

// SetReflectionLoop sets the reflection loop for the engine
func (e *PlanEngine) SetReflectionLoop(reflection *ReflectionLoop) {
	e.reflection = reflection
}

// ExecutePlan executes a plan from start to finish
func (e *PlanEngine) ExecutePlan(ctx context.Context, plan *Plan) (*ExecutionState, error) {
	// Validate plan
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("plan validation failed: %w", err)
	}

	// Create execution state
	state := NewExecutionState(plan.ID)
	state.Status = PlanStatusRunning

	// Store cancel function for this plan
	planCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	e.mu.Lock()
	e.activePlans[plan.ID] = cancel
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.activePlans, plan.ID)
		e.mu.Unlock()
	}()

	// Save initial state
	if err := e.stateStore.Save(state); err != nil {
		return nil, fmt.Errorf("failed to save initial state: %w", err)
	}

	// Execute plan
	if err := e.executePlanSteps(planCtx, plan, state); err != nil {
		state.Status = PlanStatusFailed
		state.Error = err.Error()
		now := time.Now()
		state.CompletedAt = &now

		// Rollback if enabled and there was a failure
		if e.config.EnableRollback && state.HasFailedSteps() {
			if rollbackErr := e.rollbackManager.Rollback(planCtx, plan, state); rollbackErr != nil {
				state.Error = fmt.Sprintf("%s; rollback error: %v", state.Error, rollbackErr)
			} else {
				state.Status = PlanStatusRolledBack
			}
		}

		e.stateStore.Save(state)
		return state, err
	}

	// Mark as completed
	state.Status = PlanStatusCompleted
	now := time.Now()
	state.CompletedAt = &now

	// Save final state
	if err := e.stateStore.Save(state); err != nil {
		return state, fmt.Errorf("failed to save final state: %w", err)
	}

	// Perform reflection if enabled
	if e.config.EnableReflection && e.reflection != nil {
		if _, err := e.reflection.Evaluate(planCtx, plan, state); err != nil {
			// Log but don't fail the execution
			fmt.Printf("Reflection evaluation failed: %v\n", err)
		}
	}

	return state, nil
}

// executePlanSteps executes the steps of a plan according to DAG
func (e *PlanEngine) executePlanSteps(ctx context.Context, plan *Plan, state *ExecutionState) error {
	dag := NewDAG(plan.Steps)
	parallelGroups := dag.GetParallelGroups()

	for _, group := range parallelGroups {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Get steps for this group
		steps := plan.GetStepsByIDs(group)

		if len(steps) == 0 {
			continue
		}

		// Execute group
		if len(steps) == 1 {
			// Serial execution for single step
			if err := e.executeStep(ctx, steps[0], state, make(map[string]any)); err != nil {
				return err
			}
		} else {
			// Parallel execution for multiple steps
			if err := e.executeParallelGroup(ctx, steps, state); err != nil {
				return err
			}
		}

		// Save state after each group
		if err := e.stateStore.Save(state); err != nil {
			fmt.Printf("Warning: failed to save state: %v\n", err)
		}

		// Check for failures
		if state.HasFailedSteps() {
			return fmt.Errorf("step execution failed")
		}
	}

	return nil
}

// executeStep executes a single step
func (e *PlanEngine) executeStep(ctx context.Context, step *Step, state *ExecutionState, input map[string]any) error {
	// Mark step as started
	state.MarkStepStarted(step.ID)

	// Execute step
	output, err := e.executor.Execute(ctx, step, input)
	if err != nil {
		state.MarkStepFailed(step.ID, err)
		return fmt.Errorf("step %s failed: %w", step.ID, err)
	}

	// Mark step as completed
	state.MarkStepCompleted(step.ID, output)
	return nil
}

// executeParallelGroup executes a group of steps in parallel
func (e *PlanEngine) executeParallelGroup(ctx context.Context, steps []*Step, state *ExecutionState) error {
	// Mark all steps as started
	for _, step := range steps {
		state.MarkStepStarted(step.ID)
	}

	// Execute in parallel
	results := e.parallelExecutor.ExecuteParallel(ctx, steps, e.executor, make(map[string]any))

	// Process results
	var hasError bool
	for stepID, result := range results {
		if result.Error != nil {
			state.MarkStepFailed(stepID, result.Error)
			hasError = true
		} else {
			state.MarkStepCompleted(stepID, result.Output)
		}
	}

	if hasError {
		return fmt.Errorf("one or more parallel steps failed")
	}

	return nil
}

// ResumePlan resumes execution of a paused or failed plan
func (e *PlanEngine) ResumePlan(ctx context.Context, planID string) (*ExecutionState, error) {
	// Load existing state
	state, err := e.stateStore.Load(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan state: %w", err)
	}

	if state.Status == PlanStatusCompleted {
		return state, fmt.Errorf("plan already completed")
	}

	// This is a simplified resume - in a full implementation, we would:
	// 1. Load the original plan
	// 2. Identify incomplete steps
	// 3. Continue execution from where it stopped
	// For now, return the state as-is
	return state, fmt.Errorf("resume not fully implemented")
}

// GetState retrieves the execution state for a plan
func (e *PlanEngine) GetState(planID string) (*ExecutionState, error) {
	return e.stateStore.Load(planID)
}

// CancelPlan cancels an active plan execution
func (e *PlanEngine) CancelPlan(planID string) error {
	e.mu.Lock()
	cancel, exists := e.activePlans[planID]
	e.mu.Unlock()

	if !exists {
		return fmt.Errorf("plan %s is not currently executing", planID)
	}

	// Cancel the context
	cancel()

	// Update state
	state, err := e.stateStore.Load(planID)
	if err != nil {
		return fmt.Errorf("failed to load plan state: %w", err)
	}

	state.Status = PlanStatusCancelled
	now := time.Now()
	state.CompletedAt = &now
	state.Error = "execution cancelled"

	return e.stateStore.Save(state)
}

// PausePlan pauses an active plan execution
func (e *PlanEngine) PausePlan(planID string) error {
	// In a full implementation, this would pause execution
	// For now, we'll just update the state
	state, err := e.stateStore.Load(planID)
	if err != nil {
		return fmt.Errorf("failed to load plan state: %w", err)
	}

	if state.Status != PlanStatusRunning {
		return fmt.Errorf("can only pause running plans")
	}

	state.Status = PlanStatusPaused
	return e.stateStore.Save(state)
}

// ListExecutions returns all plan executions
func (e *PlanEngine) ListExecutions() ([]*ExecutionState, error) {
	return e.stateStore.List()
}

// DeleteExecution deletes a plan execution state
func (e *PlanEngine) DeleteExecution(planID string) error {
	// Check if plan is currently executing
	e.mu.RLock()
	_, isActive := e.activePlans[planID]
	e.mu.RUnlock()

	if isActive {
		return fmt.Errorf("cannot delete active plan execution")
	}

	return e.stateStore.Delete(planID)
}
