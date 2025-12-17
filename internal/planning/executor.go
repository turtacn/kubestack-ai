package planning

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

// StepExecutor is the interface for executing steps
type StepExecutor interface {
	Execute(ctx context.Context, step *Step, input map[string]any) (any, error)
}

// ToolRegistry is an interface for tool execution
type ToolRegistry interface {
	Execute(ctx context.Context, toolName string, args map[string]any) (any, error)
}

// LLMClient is an interface for LLM interactions
type LLMClient interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// ConditionEvaluator is an interface for evaluating conditions
type ConditionEvaluator interface {
	Evaluate(ctx context.Context, condition string, context map[string]any) (bool, error)
}

// DefaultStepExecutor is the default implementation of StepExecutor
type DefaultStepExecutor struct {
	toolRegistry       ToolRegistry
	llmClient          LLMClient
	conditionEvaluator ConditionEvaluator
	planEngine         *PlanEngine // For SubPlan execution
}

// NewDefaultStepExecutor creates a new DefaultStepExecutor
func NewDefaultStepExecutor(tools ToolRegistry, llm LLMClient) *DefaultStepExecutor {
	return &DefaultStepExecutor{
		toolRegistry:       tools,
		llmClient:          llm,
		conditionEvaluator: &SimpleConditionEvaluator{},
	}
}

// SetPlanEngine sets the plan engine for SubPlan execution
func (e *DefaultStepExecutor) SetPlanEngine(engine *PlanEngine) {
	e.planEngine = engine
}

// Execute executes a single step
func (e *DefaultStepExecutor) Execute(ctx context.Context, step *Step, input map[string]any) (any, error) {
	// Apply timeout
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	// Execute with retry
	var lastErr error
	maxAttempts := 1
	if step.RetryPolicy != nil {
		maxAttempts = step.RetryPolicy.MaxRetries + 1
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Backoff before retry
			if step.RetryPolicy != nil && step.RetryPolicy.BackoffMs > 0 {
				select {
				case <-time.After(time.Duration(step.RetryPolicy.BackoffMs) * time.Millisecond):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
		}

		result, err := e.executeOnce(ctx, step, input)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("step execution failed after %d attempts: %w", maxAttempts, lastErr)
}

// executeOnce executes a step once without retry
func (e *DefaultStepExecutor) executeOnce(ctx context.Context, step *Step, input map[string]any) (any, error) {
	switch step.Type {
	case StepTypeToolCall:
		if e.toolRegistry == nil {
			return nil, fmt.Errorf("tool registry not configured")
		}
		return e.toolRegistry.Execute(ctx, step.Action.ToolName, step.Action.ToolArgs)

	case StepTypeLLMQuery:
		if e.llmClient == nil {
			return nil, fmt.Errorf("LLM client not configured")
		}
		return e.llmClient.Complete(ctx, step.Action.Prompt)

	case StepTypeCondition:
		if e.conditionEvaluator == nil {
			return nil, fmt.Errorf("condition evaluator not configured")
		}
		return e.conditionEvaluator.Evaluate(ctx, step.Action.Condition, input)

	case StepTypeSubPlan:
		if e.planEngine == nil {
			return nil, fmt.Errorf("plan engine not configured for SubPlan execution")
		}
		// SubPlan execution would require a plan ID or embedded plan
		return nil, fmt.Errorf("SubPlan execution not yet implemented")

	default:
		return nil, fmt.Errorf("unknown step type: %s", step.Type)
	}
}

// ParallelExecutor executes multiple steps in parallel
type ParallelExecutor struct {
	maxParallel int
}

// NewParallelExecutor creates a new ParallelExecutor
func NewParallelExecutor(maxParallel int) *ParallelExecutor {
	if maxParallel <= 0 {
		maxParallel = 10 // default
	}
	return &ParallelExecutor{
		maxParallel: maxParallel,
	}
}

// StepResult represents the result of a step execution
type StepResult struct {
	StepID string
	Output any
	Error  error
}

// ExecuteParallel executes multiple steps in parallel
func (pe *ParallelExecutor) ExecuteParallel(ctx context.Context, steps []*Step, executor StepExecutor, input map[string]any) map[string]StepResult {
	results := make(map[string]StepResult)
	resultsChan := make(chan StepResult, len(steps))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(pe.maxParallel)

	for _, step := range steps {
		step := step // capture loop variable
		g.Go(func() error {
			output, err := executor.Execute(gctx, step, input)
			resultsChan <- StepResult{
				StepID: step.ID,
				Output: output,
				Error:  err,
			}
			return nil // Don't propagate errors through errgroup
		})
	}

	// Wait for all goroutines
	g.Wait()
	close(resultsChan)

	// Collect results
	for result := range resultsChan {
		results[result.StepID] = result
	}

	return results
}

// SimpleConditionEvaluator is a simple implementation of ConditionEvaluator
type SimpleConditionEvaluator struct{}

// Evaluate evaluates a simple condition
func (e *SimpleConditionEvaluator) Evaluate(ctx context.Context, condition string, context map[string]any) (bool, error) {
	// Simple implementation: just check if condition is "true"
	// In a real implementation, this would parse and evaluate expressions
	switch condition {
	case "true", "True", "TRUE":
		return true, nil
	case "false", "False", "FALSE":
		return false, nil
	default:
		// For now, return true for any other condition
		return true, nil
	}
}
