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

// GeneratePlan creates a detailed, step-by-step execution plan from a list of issues.
func (p *planner) GeneratePlan(ctx context.Context, issues []models.Issue) (*models.ExecutionPlan, error) {
	p.log.Infof("Generating execution plan from %d issues.", len(issues))

	var actions []*models.FixAction
	for _, issue := range issues {
		for _, rec := range issue.Recommendations {
			if rec.CanAutoFix {
				actions = append(actions, &rec.Fix)
			}
		}
	}

	if len(actions) == 0 {
		return nil, fmt.Errorf("no autofixable actions found in the provided issues")
	}

	riskAssessment, err := p.AnalyzeRisk(ctx, actions)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze risk: %w", err)
	}

	steps := make([]*models.ExecutionStep, len(actions))
	for i, action := range actions {
		steps[i] = &models.ExecutionStep{
			ID:          uuid.New().String(),
			Name:        fmt.Sprintf("Fix for '%s'", action.Description),
			Description: action.Description,
			Action:      action,
			Status:      models.StepStatusPending,
		}
	}

	sortedSteps, err := p.buildDependencyGraphAndSort(steps)
	if err != nil {
		return nil, fmt.Errorf("failed to sort execution steps: %w", err)
	}

	plan := &models.ExecutionPlan{
		ID:       uuid.New().String(),
		Strategy: models.SerialExecution,
		Steps:    sortedSteps,
		Risk:     riskAssessment,
	}

	p.log.Infof("Successfully generated execution plan with risk level: %s", plan.Risk.MaxSeverity)
	return plan, nil
}

// AnalyzeRisk assesses the potential risks of a set of actions based on predefined rules.
func (p *planner) AnalyzeRisk(ctx context.Context, actions []*models.FixAction) (*models.RiskAssessment, error) {
	p.log.Info("Analyzing risk for proposed actions.")

	assessment := &models.RiskAssessment{
		TotalScore:       0,
		MaxSeverity:      models.RiskLevelLow,
		RequiresApproval: false,
	}
	rules := DefaultRiskRules()

	for _, action := range actions {
		for _, rule := range rules {
			if rule.Condition(action) {
				assessment.TotalScore += rule.Score
				if rule.Severity > assessment.MaxSeverity {
					assessment.MaxSeverity = rule.Severity
				}
				p.log.Debugf("Action '%s' matched risk rule, adding score %d. New total: %d", action.Description, rule.Score, assessment.TotalScore)
			}
		}
	}

	// Define thresholds for requiring approval
	if assessment.TotalScore >= 80 || assessment.MaxSeverity >= models.RiskLevelHigh {
		assessment.RequiresApproval = true
		assessment.Description = "High-risk operations detected, manual approval is required."
	} else if assessment.TotalScore >= 40 {
		assessment.Description = "Moderate risk operations detected."
	} else {
		assessment.Description = "Low risk operations, no approval required."
	}

	return assessment, nil
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
