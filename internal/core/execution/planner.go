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
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
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
	p.inferActionTypes(steps)

	plan := &models.ExecutionPlan{
		ID:       uuid.New().String(),
		Strategy: models.SerialExecution, // Default to the safest strategy.
		Steps:    steps,
	}

	// After generating the basic plan, analyze its risk and optimize the sequence.
	risk, err := p.AnalyzeRisk(context.Background(), plan)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze plan risk: %w", err)
	}
	plan.Risk = risk

	optimizedPlan, err := p.OptimizeSequence(context.Background(), plan)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize plan sequence: %w", err)
	}

	return optimizedPlan, nil
}

// inferActionTypes is a helper to temporarily classify actions based on their command string.
// In a real system, this information would likely come from the plugin or recommendation itself.
func (p *planner) inferActionTypes(steps []*models.ExecutionStep) {
	for _, step := range steps {
		cmd := strings.ToLower(step.Action.Command)
		if strings.Contains(cmd, "restart") || strings.Contains(cmd, "reload") {
			step.Action.ActionType = enum.FixActionRestart
		} else if strings.Contains(cmd, "set") || strings.Contains(cmd, "update") || strings.Contains(cmd, "config") {
			step.Action.ActionType = enum.FixActionConfigChange
		} else {
			step.Action.ActionType = enum.FixActionUnknown
		}
	}
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

	var criticalRiskCommands = []string{"rm -rf", "dd ", "format", "mkfs", "drop database", "delete from "}
	var highRiskCommands = []string{"reboot", "shutdown", "kill -9", "chmod -R 777"}
	var mediumRiskCommands = []string{"rm ", "kill ", "systemctl restart", "kubectl delete", "docker stop"}

	assessment := &models.RiskAssessment{
		Level:       "Low",
		Description: "No significant risks detected. Plan contains informational or configuration-read commands.",
	}

	for _, step := range plan.Steps {
		cmd := strings.ToLower(step.Action.Command)
		for _, riskyCmd := range criticalRiskCommands {
			if strings.Contains(cmd, riskyCmd) {
				assessment.Level = "Critical"
				assessment.Description = "Plan contains CRITICAL-risk operations (e.g., irreversible data deletion or disk formatting)."
				p.log.Warnf("Critical-risk command '%s' detected in step '%s'", riskyCmd, step.Name)
				return assessment, nil // Return on first critical finding
			}
		}
		for _, riskyCmd := range highRiskCommands {
			if strings.Contains(cmd, riskyCmd) {
				assessment.Level = "High"
				assessment.Description = "Plan contains high-risk operations (e.g., system reboots, permission changes)."
				p.log.Warnf("High-risk command '%s' detected in step '%s'", riskyCmd, step.Name)
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

// OptimizeSequence reorders the steps in a plan based on a predefined dependency graph.
// It uses a topological sort to ensure that steps are executed in a safe and logical order.
func (p *planner) OptimizeSequence(_ context.Context, plan *models.ExecutionPlan) (*models.ExecutionPlan, error) {
	p.log.Info("Optimizing execution sequence.")

	// 1. Define the dependency graph for action types.
	// A depends on B means B must be executed before A.
	dependencyGraph := map[enum.FixActionType][]enum.FixActionType{
		enum.FixActionRestart: {enum.FixActionConfigChange, enum.FixActionDataMigration},
		// Add other dependencies here, e.g., DATA_MIGRATION depends on CONFIG_CHANGE
	}

	// 2. Build the graph for the specific steps in the plan.
	// The graph is represented by an adjacency list and an in-degree map.
	adj := make(map[string][]string)
	inDegree := make(map[string]int)
	stepsMap := make(map[string]*models.ExecutionStep)

	for _, step := range plan.Steps {
		inDegree[step.ID] = 0
		stepsMap[step.ID] = step
	}

	for _, stepA := range plan.Steps {
		deps, ok := dependencyGraph[stepA.Action.ActionType]
		if !ok {
			continue
		}
		for _, stepB := range plan.Steps {
			if stepA.ID == stepB.ID {
				continue
			}
			for _, depType := range deps {
				if stepB.Action.ActionType == depType {
					adj[stepB.ID] = append(adj[stepB.ID], stepA.ID)
					inDegree[stepA.ID]++
				}
			}
		}
	}

	// 3. Perform Topological Sort (Kahn's algorithm).
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var sortedSteps []*models.ExecutionStep
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]
		sortedSteps = append(sortedSteps, stepsMap[currentID])

		for _, neighborID := range adj[currentID] {
			inDegree[neighborID]--
			if inDegree[neighborID] == 0 {
				queue = append(queue, neighborID)
			}
		}
	}

	// 4. Check for cycles.
	if len(sortedSteps) != len(plan.Steps) {
		return nil, fmt.Errorf("a cycle was detected in the execution plan dependencies, cannot optimize sequence")
	}

	plan.Steps = sortedSteps
	p.log.Info("Successfully optimized execution sequence.")
	return plan, nil
}

//Personal.AI order the ending
