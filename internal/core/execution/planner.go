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
// high-level recommendations. It filters for auto-fixable recommendations,
// converts them into execution steps, and then performs a risk analysis on the
// resulting plan.
//
// Parameters:
//   ctx (context.Context): The context for the operation.
//   recommendations ([]*models.Recommendation): A slice of recommendations from a diagnosis.
//
// Returns:
//   *models.ExecutionPlan: A structured plan containing executable steps and a risk assessment.
//   error: An error if risk analysis fails.
func (p *planner) GeneratePlan(ctx context.Context, recommendations []*models.Recommendation) (*models.ExecutionPlan, error) {
	p.log.Infof("Generating execution plan from %d recommendations.", len(recommendations))

	steps := make([]*models.ExecutionStep, 0)
	actions := make([]*models.FixAction, 0)

	for _, rec := range recommendations {
		if !rec.CanAutoFix || rec.Command == "" {
			p.log.Debugf("Skipping non-autofixable recommendation: %s", rec.Description)
			continue
		}
		action := &models.FixAction{
			ID:               rec.ID,
			Description:      rec.Description,
			Command:          rec.Command,
			Category:         rec.Category,
			RollbackCommand:  rec.RollbackCommand, // Make sure the rollback command is included
			ValidationCommand: rec.ValidationCommand,
		}
		actions = append(actions, action)
	}

	// This is a placeholder for getting the current environment.
	// In a real application, this would come from configuration or context.
	environment := "Production"

	// Perform risk analysis on the collected actions.
	riskAssessment, err := AnalyzeRisk(ctx, actions, environment)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze plan risk: %w", err)
	}

	// Convert actions to steps after risk analysis.
	for _, action := range actions {
		step := &models.ExecutionStep{
			ID:          uuid.New().String(),
			Name:        fmt.Sprintf("Fix for '%s'", action.Description),
			Description: action.Description,
			Action:      action,
			Status:      models.StepStatusPending,
		}
		steps = append(steps, step)
	}

	// Build a dependency graph and sort the steps topologically.
	sortedSteps, err := p.buildDependencyGraphAndSort(steps)
	if err != nil {
		return nil, fmt.Errorf("failed to sort execution steps by dependency: %w", err)
	}

	plan := &models.ExecutionPlan{
		ID:       uuid.New().String(),
		Strategy: models.SerialExecution, // Serial execution is safest for dependent steps.
		Steps:    sortedSteps,
		Risk:     riskAssessment,
	}

	p.log.Infof("Successfully generated execution plan with risk level: %s", plan.Risk.MaxSeverity)
	return plan, nil
}

// AnalyzeRisk assesses the potential risks of an execution plan. It is now a wrapper
// that extracts actions from the plan and uses the centralized risk_rules logic.
func (p *planner) AnalyzeRisk(ctx context.Context, plan *models.ExecutionPlan) (*models.RiskAssessment, error) {
	p.log.Info("Analyzing risk for generated execution plan.")

	actions := make([]*models.FixAction, len(plan.Steps))
	for i, step := range plan.Steps {
		actions[i] = step.Action
	}

	// This is a placeholder for getting the current environment.
	// In a real application, this would come from configuration or context.
	environment := "Production"

	return AnalyzeRisk(ctx, actions, environment)
}

// OptimizeSequence is responsible for reordering the steps in a plan to ensure
// maximum safety and efficiency.
// NOTE: This is a placeholder implementation. A complete implementation would
// perform a topological sort on the dependency graph of the steps to guarantee
// correct execution order (e.g., apply a config change before restarting a service).
func (p *planner) OptimizeSequence(_ context.Context, plan *models.ExecutionPlan) (*models.ExecutionPlan, error) {
	p.log.Info("Optimizing execution sequence.")
	sortedSteps, err := p.buildDependencyGraphAndSort(plan.Steps)
	if err != nil {
		return nil, err
	}
	plan.Steps = sortedSteps
	return plan, nil
}

// buildDependencyGraphAndSort analyzes the steps, builds a dependency graph based on categories,
// and returns a new list of steps sorted topologically.
func (p *planner) buildDependencyGraphAndSort(steps []*models.ExecutionStep) ([]*models.ExecutionStep, error) {
	// Simple dependency rule: "Restart" actions depend on "ConfigChange" actions.
	// This can be expanded with more sophisticated rules.
	categoryMap := make(map[string][]*models.ExecutionStep)
	for _, step := range steps {
		category := step.Action.Category
		categoryMap[category] = append(categoryMap[category], step)
	}

	configChangeSteps, hasConfigChanges := categoryMap["ConfigChange"]
	restartSteps, hasRestarts := categoryMap["Restart"]

	if hasConfigChanges && hasRestarts {
		p.log.Debugf("Found %d config change steps and %d restart steps. Adding dependencies.", len(configChangeSteps), len(restartSteps))
		for _, restartStep := range restartSteps {
			for _, configStep := range configChangeSteps {
				restartStep.DependsOn = append(restartStep.DependsOn, configStep.ID)
			}
		}
	}

	// Now perform a topological sort (Kahn's algorithm)
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	for _, step := range steps {
		graph[step.ID] = []string{}
		inDegree[step.ID] = 0
	}

	for _, step := range steps {
		for _, depID := range step.DependsOn {
			graph[depID] = append(graph[depID], step.ID)
			inDegree[step.ID]++
		}
	}

	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var sortedOrder []*models.ExecutionStep
	stepMap := make(map[string]*models.ExecutionStep)
	for _, step := range steps {
		stepMap[step.ID] = step
	}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		sortedOrder = append(sortedOrder, stepMap[id])

		for _, neighborID := range graph[id] {
			inDegree[neighborID]--
			if inDegree[neighborID] == 0 {
				queue = append(queue, neighborID)
			}
		}
	}

	if len(sortedOrder) != len(steps) {
		return nil, fmt.Errorf("cycle detected in execution plan dependencies, cannot sort")
	}

	p.log.Info("Successfully sorted execution steps based on dependencies.")
	return sortedOrder, nil
}

//Personal.AI order the ending
