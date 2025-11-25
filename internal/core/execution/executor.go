// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package execution implements the core logic for the execution engine.
package execution

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// --- Action Executor (Worker) Implementation ---

// actionExecutor is the concrete implementation of interfaces.ActionExecutor.
// It is the "hands" of the engine, performing the low-level, actual work.
type actionExecutor struct {
	log logger.Logger
}

func newActionExecutor() interfaces.ActionExecutor {
	return &actionExecutor{
		log: logger.NewLogger("action-executor"),
	}
}

func (e *actionExecutor) ExecuteCommand(ctx context.Context, command string) (string, string, error) {
	e.log.Infof("Executing command: %s", command)
	// SECURITY NOTE: In a real-world scenario, never execute arbitrary commands.
	// Commands must be sanitized, validated, and originate from a trusted source (e.g., predefined in plugins).
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		e.log.Errorf("Command execution failed. Stderr: %s", stderr.String())
	}
	return stdout.String(), stderr.String(), err
}

func (e *actionExecutor) ApplyConfiguration(ctx context.Context, configChange *models.ConfigChange) error {
	e.log.Infof("Applying config change to file %s: set %s = %s", configChange.File, configChange.Key, configChange.Value)
	// This is a placeholder. A real implementation would need to safely parse and write to
	// different config formats (YAML, INI, etc.) and handle file permissions and backups.
	return fmt.Errorf("ApplyConfiguration is not implemented")
}

func (e *actionExecutor) RollbackChanges(ctx context.Context, steps []*models.ExecutionStep) error {
	e.log.Info("Starting rollback for failed execution.")
	var rollbackErrors []string
	// Iterate backwards through the already completed steps.
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Action.RollbackCommand == "" {
			e.log.Warnf("Step '%s' has no rollback command, skipping.", step.Name)
			continue
		}

		e.log.Infof("Rolling back step: %s (Command: `%s`)", step.Name, step.Action.RollbackCommand)
		_, stderr, err := e.ExecuteCommand(ctx, step.Action.RollbackCommand)
		if err != nil {
			e.log.Errorf("Rollback for step '%s' failed: %v. Stderr: %s", step.Name, err, stderr)
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("step '%s': %v", step.Name, err))
		} else {
			e.log.Infof("Rollback for step '%s' completed successfully.", step.Name)
		}
	}

	if len(rollbackErrors) > 0 {
		return fmt.Errorf("one or more rollback actions failed: %s", strings.Join(rollbackErrors, "; "))
	}

	e.log.Info("Rollback completed successfully.")
	return nil
}

// --- Execution Manager (Coordinator) Implementation ---

// manager is the concrete implementation of interfaces.ExecutionManager.
type manager struct {
	log      logger.Logger
	planner  interfaces.ExecutionPlanner
	executor interfaces.ActionExecutor
}

// NewManager creates a new instance of the execution manager. The manager is
// responsible for coordinating the entire execution process, from planning to
// applying and validating actions. It uses a planner to generate the execution
// steps and an internal executor to run them.
//
// Parameters:
//   planner (interfaces.ExecutionPlanner): The planner component used to generate execution plans.
//
// Returns:
//   interfaces.ExecutionManager: A new, configured execution manager.
func NewManager(planner interfaces.ExecutionPlanner) interfaces.ExecutionManager {
	return &manager{
		log:      logger.NewLogger("execution-manager"),
		planner:  planner,
		executor: newActionExecutor(),
	}
}

// PlanExecution delegates the task of generating an execution plan to the
// configured planner component.
//
// Parameters:
//   ctx (context.Context): The context for the planning operation.
//   recommendations ([]*models.Recommendation): A slice of recommendations from a diagnosis report.
//
// Returns:
//   *models.ExecutionPlan: The generated plan containing the steps to be executed.
//   error: An error if plan generation fails.
func (m *manager) PlanExecution(ctx context.Context, recommendations []*models.Recommendation) (*models.ExecutionPlan, error) {
	m.log.Info("Delegating execution planning to the planner component.")
	return m.planner.GeneratePlan(ctx, recommendations)
}

// ExecutePlan carries out the steps defined in an execution plan. It is a more
// modern entry point for execution that encapsulates the logic for handling
// different execution strategies, logging, and rollbacks.
func (m *manager) ExecutePlan(ctx context.Context, plan *models.ExecutionPlan) (*models.ExecutionResult, error) {
	m.log.Infof("Starting execution of plan ID: %s with strategy: %s", plan.ID, plan.Strategy)
	result := &models.ExecutionResult{
		PlanID:    plan.ID,
		Status:    models.ExecutionStatusInProgress,
		StartTime: time.Now().UTC(),
		Logs:      make([]*models.ExecutionLog, 0),
	}

	// A stack to keep track of successfully completed steps for potential rollback.
	completedSteps := make([]*models.ExecutionStep, 0)

	// For now, only Serial strategy is implemented.
	if plan.Strategy != models.SerialExecution {
		err := fmt.Errorf("execution strategy '%s' is not implemented", plan.Strategy)
		result.Status = models.ExecutionStatusFailed
		result.EndTime = time.Now().UTC()
		return result, err
	}

	for _, step := range plan.Steps {
		step.Status = models.StepStatusRunning
		m.log.Infof("Executing step: %s", step.Name)
		result.Logs = append(result.Logs, &models.ExecutionLog{Timestamp: time.Now().UTC(), StepID: step.ID, Level: "Info", Message: fmt.Sprintf("Executing step: %s", step.Name)})

		// The actual execution is delegated to the internal executor.
		stdout, stderr, err := m.executor.ExecuteCommand(ctx, step.Action.Command)
		if err != nil {
			step.Status = models.StepStatusFailed
			step.Result = fmt.Sprintf("Error: %v\nStderr: %s", err, stderr)
			m.log.Errorf("Step '%s' failed: %v", step.Name, err)
			result.Logs = append(result.Logs, &models.ExecutionLog{Timestamp: time.Now().UTC(), StepID: step.ID, Level: "Error", Message: step.Result})

			// Trigger rollback for all previously completed steps.
			rollbackErr := m.executor.RollbackChanges(ctx, completedSteps)
			if rollbackErr != nil {
				result.Status = models.ExecutionStatusFailedWithRollbackFailure
				m.log.Errorf("Rollback failed: %v", rollbackErr)
				result.Logs = append(result.Logs, &models.ExecutionLog{Timestamp: time.Now().UTC(), Level: "Critical", Message: fmt.Sprintf("Rollback failed: %v", rollbackErr)})
			} else {
				result.Status = models.ExecutionStatusFailedWithRollbackSuccess
				m.log.Info("Rollback completed successfully.")
				result.Logs = append(result.Logs, &models.ExecutionLog{Timestamp: time.Now().UTC(), Level: "Info", Message: "Rollback completed successfully."})
			}

			result.EndTime = time.Now().UTC()
			return result, fmt.Errorf("execution of step '%s' failed", step.Name)
		}

		step.Status = models.StepStatusSuccess
		step.Result = stdout
		completedSteps = append(completedSteps, step) // Push to stack
		m.log.Infof("Step '%s' completed successfully.", step.Name)
		result.Logs = append(result.Logs, &models.ExecutionLog{Timestamp: time.Now().UTC(), StepID: step.ID, Level: "Info", Message: "Step completed successfully."})
	}

	result.Status = models.ExecutionStatusSuccess
	result.EndTime = time.Now().UTC()
	m.log.Infof("Execution of plan %s completed successfully.", plan.ID)
	return result, nil
}

// ExecuteActions is the legacy entry point for execution, which includes a
// user confirmation step. It's kept for backward compatibility and for CLI
// scenarios where user interaction is required.
func (m *manager) ExecuteActions(ctx context.Context, plan *models.ExecutionPlan, confirmFunc interfaces.ConfirmationFunc) (*models.ExecutionResult, error) {
	m.log.Info("Executing plan with user confirmation step.")

	// A simple check for high-risk plans. A more robust implementation
	// would use the `RequiresApproval` flag from the risk assessment.
	if plan.Risk.MaxSeverity >= models.RiskLevelHigh {
		prompt := fmt.Sprintf("High-risk operation detected (Level: %s). Do you want to proceed with the execution?", plan.Risk.MaxSeverity)
		if !confirmFunc(prompt) {
			m.log.Warn("Execution aborted by user due to high risk.")
			return &models.ExecutionResult{
				PlanID: plan.ID,
				Status: models.ExecutionStatusAborted,
			}, nil
		}
	}

	// This method now wraps ExecutePlan, adding the confirmation layer.
	// The core execution logic is centralized in ExecutePlan.
	return m.ExecutePlan(ctx, plan)
}

// ValidateExecution is responsible for verifying that an executed plan has
// successfully resolved the underlying issue.
// NOTE: This is a placeholder implementation. A real implementation would
// re-run a relevant health check or diagnosis to confirm the fix.
//
// Parameters:
//   ctx (context.Context): The context for the validation operation.
//   result (*models.ExecutionResult): The result of the execution to be validated.
//
// Returns:
//   error: An error if validation fails or if the execution was not successful.
func (m *manager) ValidateExecution(ctx context.Context, result *models.ExecutionResult) error {
	m.log.Infof("Validating execution result for plan ID: %s", result.PlanID)
	// Placeholder. A real implementation would re-run a health check or a specific
	// diagnostic check (from the original plugin) to confirm the issue is resolved.
	if result.Status != "Success" {
		return fmt.Errorf("cannot validate a failed or incomplete execution")
	}
	m.log.Info("Validation successful (placeholder implementation).")
	return nil
}

//Personal.AI order the ending
