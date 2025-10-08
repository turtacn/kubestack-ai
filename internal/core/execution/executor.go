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

func (e *actionExecutor) RollbackChanges(ctx context.Context, action *models.ExecutionStep) error {
	e.log.Infof("Rolling back action for step: %s", action.Name)
	// Placeholder. A real implementation would require a 'before' state or a pre-defined compensating action.
	return fmt.Errorf("RollbackChanges is not implemented")
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

// ExecuteActions carries out the steps defined in an execution plan. It follows
// the specified strategy (e.g., serial) and uses a confirmation function to
// prompt the user before executing each step, providing a critical safety check.
//
// Parameters:
//   ctx (context.Context): The context for the entire execution operation.
//   plan (*models.ExecutionPlan): The plan to be executed.
//   confirmFunc (interfaces.ConfirmationFunc): A function that prompts the user for confirmation.
//
// Returns:
//   *models.ExecutionResult: A struct containing the outcome of the execution, including logs and step results.
//   error: An error if a step fails and the execution is halted.
func (m *manager) ExecuteActions(ctx context.Context, plan *models.ExecutionPlan, confirmFunc interfaces.ConfirmationFunc) (*models.ExecutionResult, error) {
	m.log.Infof("Starting execution of plan ID: %s with strategy: %s", plan.ID, plan.Strategy)

	result := &models.ExecutionResult{
		PlanID:      plan.ID,
		Status:      "InProgress",
		StartTime:   time.Now().UTC(),
		StepResults: plan.Steps,
		Logs:        make([]*models.ExecutionLog, 0),
	}

	// For now, only Serial strategy is implemented. Parallel would require a more complex dependency graph traversal.
	if plan.Strategy != models.SerialExecution {
		return nil, fmt.Errorf("execution strategy '%s' is not implemented", plan.Strategy)
	}

	for _, step := range plan.Steps {
		// Permission Check (Placeholder)
		m.log.Debugf("Checking permissions for step: %s", step.Name)

		prompt := fmt.Sprintf("Action: %s\nDescription: %s\nCommand: `%s`", step.Name, step.Description, step.Action.Command)
		if !confirmFunc(prompt) {
			m.log.Warnf("Execution of step '%s' skipped by user.", step.Name)
			step.Status = "Skipped"
			result.Logs = append(result.Logs, &models.ExecutionLog{Timestamp: time.Now().UTC(), StepID: step.ID, Level: "Warn", Message: "Step skipped by user."})
			continue
		}

		step.Status = "Running"
		result.Logs = append(result.Logs, &models.ExecutionLog{Timestamp: time.Now().UTC(), StepID: step.ID, Level: "Info", Message: fmt.Sprintf("Executing step: %s", step.Name)})
		m.log.Infof("Executing step: %s", step.Name)

		stdout, stderr, err := m.executor.ExecuteCommand(ctx, step.Action.Command)
		if err != nil {
			step.Status = "Failed"
			step.Result = fmt.Sprintf("Error: %v\nStderr: %s", err, stderr)
			result.Logs = append(result.Logs, &models.ExecutionLog{Timestamp: time.Now().UTC(), StepID: step.ID, Level: "Error", Message: step.Result})
			m.log.Errorf("Step '%s' failed: %v", step.Name, err)

			// TODO: Implement rollback logic here, e.g., by executing the rollback plan.

			result.Status = "Failed"
			result.EndTime = time.Now().UTC()
			return result, fmt.Errorf("execution of step '%s' failed", step.Name)
		}

		step.Status = "Success"
		step.Result = stdout
		result.Logs = append(result.Logs, &models.ExecutionLog{Timestamp: time.Now().UTC(), StepID: step.ID, Level: "Info", Message: "Step completed successfully."})
		m.log.Infof("Step '%s' completed successfully.", step.Name)
	}

	result.Status = "Success"
	result.EndTime = time.Now().UTC()
	m.log.Infof("Execution of plan %s completed successfully.", plan.ID)
	return result, nil
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
