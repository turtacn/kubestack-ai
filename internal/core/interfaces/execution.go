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

// ConfirmationFunc defines a function signature for requesting interactive user confirmation.
// This is a key part of the safety mechanism for the execution engine.
// The function should display the prompt and return true if the user confirms.
type ConfirmationFunc func(prompt string) bool

// ExecutionManager orchestrates the entire workflow for applying fixes, from planning to validation.
type ExecutionManager interface {
	// PlanExecution generates a detailed execution plan based on a set of recommendations from the diagnosis phase.
	PlanExecution(ctx context.Context, recommendations []*models.Recommendation) (*models.ExecutionPlan, error)

	// ExecuteActions executes all steps in a given execution plan.
	// It uses the provided confirmation function to get user approval for potentially risky steps.
	ExecuteActions(ctx context.Context, plan *models.ExecutionPlan, confirmFunc ConfirmationFunc) (*models.ExecutionResult, error)

	// ValidateExecution checks if the execution was successful and if the original issue has been resolved.
	ValidateExecution(ctx context.Context, result *models.ExecutionResult) error
}

// ExecutionPlanner is responsible for the "brains" of the execution process. It creates a safe,
// optimized, and understandable plan for the user to review.
type ExecutionPlanner interface {
	// GeneratePlan creates a detailed, step-by-step execution plan from a list of abstract recommendations.
	GeneratePlan(ctx context.Context, recommendations []*models.Recommendation) (*models.ExecutionPlan, error)

	// AnalyzeRisk assesses the potential risks associated with an execution plan and annotates the plan with its findings.
	AnalyzeRisk(ctx context.Context, plan *models.ExecutionPlan) (*models.RiskAssessment, error)

	// OptimizeSequence reorders the steps within a plan to ensure dependencies are met and risks are minimized.
	OptimizeSequence(ctx context.Context, plan *models.ExecutionPlan) (*models.ExecutionPlan, error)
}

// ActionExecutor is the "hands" of the execution engine. It's responsible for performing a single,
// atomic action on a target system, such as running a command or applying a configuration change.
type ActionExecutor interface {
	// ExecuteCommand runs a shell command on the target system and returns its output.
	ExecuteCommand(ctx context.Context, command string) (stdout string, stderr string, err error)

	// ApplyConfiguration applies a structured configuration change to a target system.
	ApplyConfiguration(ctx context.Context, configChange *models.ConfigChange) error

	// RollbackChanges reverts a specific action that was previously executed, which is crucial for transactional safety.
	RollbackChanges(ctx context.Context, action *models.ExecutionStep) error
}

//Personal.AI order the ending
