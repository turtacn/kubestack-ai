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

package execution

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// planner is the concrete implementation of interfaces.ExecutionPlanner.
// It is the "brains" of the execution engine, responsible for creating safe and optimized plans.
type planner struct {
	log logger.Logger
	// This could have dependencies like a risk database or a best-practices template engine.
}

// NewPlanner creates a new instance of the execution planner. The planner is
// responsible for the "brains" of the execution engine, converting high-level
// recommendations into a safe and optimized, step-by-step execution plan.
//
// Returns:
//   interfaces.ExecutionPlanner: A new instance of the execution planner.
func NewPlanner() interfaces.ExecutionPlanner {
	return &planner{
		log: logger.NewLogger("execution-planner"),
	}
}

// GeneratePlan creates a detailed, step-by-step execution plan from a list of
// high-level recommendations. It filters for autofixable recommendations,
// converts them into execution steps, and then performs a risk analysis on the
// resulting plan.
//
// Parameters:
//   _ (context.Context): The context for the operation (currently unused).
//   recommendations ([]*models.Recommendation): A slice of recommendations from a diagnosis.
//
// Returns:
//   *models.ExecutionPlan: A structured plan containing executable steps and a risk assessment.
//   error: An error if risk analysis fails.
func (p *planner) GeneratePlan(_ context.Context, recommendations []*models.Recommendation) (*models.ExecutionPlan, error) {
	p.log.Infof("Generating execution plan from %d recommendations.", len(recommendations))

	steps := make([]*models.ExecutionStep, 0, len(recommendations))
	for _, rec := range recommendations {
		if !rec.CanAutoFix || rec.Command == "" {
			p.log.Debugf("Skipping non-autofixable recommendation: %s", rec.Description)
			continue
		}
		step := &models.ExecutionStep{
			ID:          uuid.New().String(),
			Name:        fmt.Sprintf("Fix for '%s'", rec.Description),
			Description: rec.Description,
			Action: &models.FixAction{
				ID:          rec.ID,
				Description: rec.Description,
				Command:     rec.Command,
			},
			Status: "Pending",
		}
		steps = append(steps, step)
	}

	// Placeholder for dependency analysis. A real implementation would analyze the steps
	// to build a dependency graph and populate the `DependsOn` field of each step.
	// For example, a step that restarts a service should depend on a step that applies a config change.

	plan := &models.ExecutionPlan{
		ID:       uuid.New().String(),
		Strategy: models.SerialExecution, // Default to the safest strategy.
		Steps:    steps,
	}

	// After generating the basic plan, analyze its risk.
	risk, err := p.AnalyzeRisk(context.Background(), plan)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze plan risk: %w", err)
	}
	plan.Risk = risk

	return plan, nil
}

// AnalyzeRisk assesses the potential risks of an execution plan by inspecting the
// commands in each step. It uses a simple, keyword-based approach to classify
// commands as low, medium, or high risk. This is a critical safety feature to
// inform the user before they approve a plan.
//
// Parameters:
//   _ (context.Context): The context for the operation (currently unused).
//   plan (*models.ExecutionPlan): The plan to be analyzed.
//
// Returns:
//   *models.RiskAssessment: A struct containing the assessed risk level and a description.
//   error: An error if the analysis fails (nil in this implementation).
func (p *planner) AnalyzeRisk(_ context.Context, plan *models.ExecutionPlan) (*models.RiskAssessment, error) {
	p.log.Info("Analyzing risk for generated execution plan.")

	// This is a very basic risk analysis. A real-world system would have a more sophisticated engine,
	// possibly checking against a database of risky operations or using policies.
	var highRiskCommands = []string{"rm -rf", "kill -9", "reboot", "format", "mkfs"}
	var mediumRiskCommands = []string{"rm ", "kill ", "systemctl restart", "kubectl delete"}

	assessment := &models.RiskAssessment{
		Level:       "Low",
		Description: "No significant risks detected.",
	}

	for _, step := range plan.Steps {
		cmd := strings.ToLower(step.Action.Command)
		for _, riskyCmd := range highRiskCommands {
			if strings.Contains(cmd, riskyCmd) {
				assessment.Level = "High"
				assessment.Description = "Plan contains high-risk operations (e.g., data deletion, system reboot)."
				p.log.Warnf("High-risk command '%s' detected in step '%s'", riskyCmd, step.Name)
				return assessment, nil // Return on first high-risk finding
			}
		}
		for _, riskyCmd := range mediumRiskCommands {
			if strings.Contains(cmd, riskyCmd) && assessment.Level == "Low" {
				assessment.Level = "Medium"
				assessment.Description = "Plan contains medium-risk operations (e.g., service restarts, resource deletion)."
				p.log.Warnf("Medium-risk command '%s' detected in step '%s'", riskyCmd, step.Name)
			}
		}
	}

	return assessment, nil
}

// OptimizeSequence is responsible for reordering the steps in a plan to ensure
// maximum safety and efficiency.
// NOTE: This is a placeholder implementation. A complete implementation would
// perform a topological sort on the dependency graph of the steps to guarantee
// correct execution order (e.g., apply a config change before restarting a service).
//
// Parameters:
//   _ (context.Context): The context for the operation (currently unused).
//   plan (*models.ExecutionPlan): The plan to be optimized.
//
// Returns:
//   *models.ExecutionPlan: The optimized plan (or the original plan in this placeholder).
//   error: An error if optimization fails (nil in this implementation).
func (p *planner) OptimizeSequence(_ context.Context, plan *models.ExecutionPlan) (*models.ExecutionPlan, error) {
	p.log.Info("Optimizing execution sequence.")
	// This is a placeholder. A real implementation would perform a topological sort
	// on the dependency graph of the steps to ensure correct execution order.
	// It could also reorder independent steps to, for example, perform read-only
	// operations first, followed by config changes, and finally service restarts.
	p.log.Info("Sequence optimization is not yet implemented; returning original plan.")
	return plan, nil
}

//Personal.AI order the ending
