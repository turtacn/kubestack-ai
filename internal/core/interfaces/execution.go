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

package interfaces

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// ConfirmationFunc defines the function signature for a callback that requests
// interactive user confirmation. This is a critical safety mechanism for the
// execution engine, allowing it to pause and seek approval before performing
// potentially risky actions. The function should display the provided prompt to the
// user and return true if the user confirms, or false if they deny.
type ConfirmationFunc func(prompt string) bool

// ExecutionManager defines the contract for the component that orchestrates the
// entire workflow for applying automated fixes, from planning through to validation.
type ExecutionManager interface {
	// PlanExecution generates a detailed, multi-step execution plan based on a set of
	// high-level recommendations from a diagnosis report.
	//
	// Parameters:
	//   ctx (context.Context): The context for the planning operation.
	//   recommendations ([]*models.Recommendation): A slice of recommendations to be turned into a plan.
	//
	// Returns:
	//   *models.ExecutionPlan: The generated plan, ready for user review and execution.
	//   error: An error if plan generation fails.
	PlanExecution(ctx context.Context, recommendations []*models.Recommendation) (*models.ExecutionPlan, error)

	// ExecuteActions executes all steps in a given execution plan, respecting its defined
	// strategy (e.g., serial). It uses the provided confirmation function to get user
	// approval before executing each step.
	//
	// Parameters:
	//   ctx (context.Context): The context for the entire execution operation.
	//   plan (*models.ExecutionPlan): The plan to be executed.
	//   confirmFunc (ConfirmationFunc): The callback function for user confirmation.
	//
	// Returns:
	//   *models.ExecutionResult: A struct containing the outcome of the execution.
	//   error: An error if a step fails and the execution is halted.
	ExecuteActions(ctx context.Context, plan *models.ExecutionPlan, confirmFunc ConfirmationFunc) (*models.ExecutionResult, error)

	// ExecutePlan carries out the steps defined in an execution plan. It is a more
	// modern entry point for execution that encapsulates the logic for handling
	// different execution strategies, logging, and rollbacks.
	ExecutePlan(ctx context.Context, plan *models.ExecutionPlan) (*models.ExecutionResult, error)

	// ValidateExecution checks if the execution was successful and if the original issue
	// has been resolved. This typically involves re-running a targeted health check.
	//
	// Parameters:
	//   ctx (context.Context): The context for the validation operation.
	//   result (*models.ExecutionResult): The result of the execution to be validated.
	//
	// Returns:
	//   error: An error if validation fails.
	ValidateExecution(ctx context.Context, result *models.ExecutionResult) error
}

// ExecutionPlanner defines the contract for the "brains" of the execution engine.
// It is responsible for creating a safe, optimized, and understandable plan from
// a set of high-level recommendations, which a user can then review and approve.
type ExecutionPlanner interface {
	// GeneratePlan creates a detailed, step-by-step execution plan from a list of
	// abstract recommendations. This involves translating recommendations into concrete
	// actions and determining their dependencies.
	//
	// Parameters:
	//   ctx (context.Context): The context for the planning operation.
	//   recommendations ([]*models.Recommendation): A slice of recommendations to be planned.
	//
	// Returns:
	//   *models.ExecutionPlan: The generated plan.
	//   error: An error if plan generation fails.
	GeneratePlan(ctx context.Context, recommendations []*models.Recommendation) (*models.ExecutionPlan, error)

	// AnalyzeRisk assesses the potential risks associated with an execution plan by
	// inspecting its actions. This is a crucial safety step.
	//
	// Parameters:
	//   ctx (context.Context): The context for the risk analysis.
	//   plan (*models.ExecutionPlan): The plan to be analyzed.
	//
	// Returns:
	//   *models.RiskAssessment: A struct containing the assessed risk level and description.
	//   error: An error if risk analysis fails.
	AnalyzeRisk(ctx context.Context, plan *models.ExecutionPlan) (*models.RiskAssessment, error)

	// OptimizeSequence reorders the steps within a plan to ensure dependencies are met
	// and risks are minimized. For example, it might use a topological sort on a
	// dependency graph of the plan's steps.
	//
	// Parameters:
	//   ctx (context.Context): The context for the optimization.
	//   plan (*models.ExecutionPlan): The plan to be optimized.
	//
	// Returns:
	//   *models.ExecutionPlan: The optimized plan.
	//   error: An error if sequence optimization fails.
	OptimizeSequence(ctx context.Context, plan *models.ExecutionPlan) (*models.ExecutionPlan, error)
}

// ActionExecutor defines the contract for the "hands" of the execution engine. It
// is responsible for performing a single, atomic action on a target system, such
// as running a command or applying a configuration change. This provides a clear
// separation between the logic of *what* to do (the plan) and the mechanics of
// *how* to do it (the executor).
type ActionExecutor interface {
	// ExecuteCommand runs a shell command on the target system.
	// SECURITY: This is a high-risk operation and should only execute commands from
	// a trusted source (e.g., predefined in a plugin or generated by a secure planner).
	//
	// Parameters:
	//   ctx (context.Context): The context for the command execution.
	//   command (string): The command to be executed.
	//
	// Returns:
	//   string: The standard output from the command.
	//   string: The standard error from the command.
	//   error: An error if the command fails to execute (e.g., non-zero exit code).
	ExecuteCommand(ctx context.Context, command string) (stdout string, stderr string, err error)

	// ApplyConfiguration applies a structured configuration change to a target system.
	// This is a safer alternative to executing raw shell commands for config management.
	//
	// Parameters:
	//   ctx (context.Context): The context for the operation.
	//   configChange (*models.ConfigChange): A struct describing the change to be made.
	//
	// Returns:
	//   error: An error if applying the configuration fails.
	ApplyConfiguration(ctx context.Context, configChange *models.ConfigChange) error

	// RollbackChanges reverts a series of previously executed actions. This is
	// crucial for transactional safety and is a key part of a robust execution engine.
	//
	// Parameters:
	//   ctx (context.Context): The context for the rollback operation.
	//   steps ([]*models.ExecutionStep): The steps containing the actions to be rolled back.
	//
	// Returns:
	//   error: An error if the rollback fails.
	RollbackChanges(ctx context.Context, steps []*models.ExecutionStep) error
}

//Personal.AI order the ending
