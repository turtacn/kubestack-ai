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
				Category:    rec.Category,
			},
			Status: "Pending",
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
	}

	// After generating the basic plan, analyze its risk.
	risk, err := p.AnalyzeRisk(context.Background(), plan)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze plan risk: %w", err)
	}
	plan.Risk = risk

	return plan, nil
}

// RiskRule defines a pattern and associated risk level for classifying commands.
type RiskRule struct {
	Pattern     string
	Level       string
	Description string
}

var riskRules = []RiskRule{
	{Pattern: "rm -rf", Level: "Critical", Description: "Potential for irreversible, widespread data loss."},
	{Pattern: "dd ", Level: "Critical", Description: "Potential for raw disk writes, which can cause catastrophic data loss."},
	{Pattern: "mkfs", Level: "Critical", Description: "Formats a filesystem, which will destroy all data on the target partition."},
	{Pattern: "format", Level: "Critical", Description: "Formats a disk, destroying all data."},
	{Pattern: "reboot", Level: "High", Description: "Will cause a service outage by rebooting the machine."},
	{Pattern: "shutdown", Level: "High", Description: "Will cause a service outage by shutting down the machine."},
	{Pattern: "kill -9", Level: "High", Description: "Forcibly terminates a process, which can lead to data corruption."},
	{Pattern: "drop database", Level: "High", Description: "Deletes an entire database, leading to major data loss."},
	{Pattern: "delete from", Level: "Medium", Description: "Deletes data from a table. If no WHERE clause is present, this could be high risk."},
	{Pattern: "systemctl restart", Level: "Medium", Description: "Restarts a service, which will cause a brief service interruption."},
	{Pattern: "kubectl delete", Level: "Medium", Description: "Deletes a Kubernetes resource, which could impact service availability."},
	{Pattern: "kill", Level: "Medium", Description: "Terminates a process, which could cause a service interruption."},
}

// AnalyzeRisk assesses the potential risks of an execution plan by inspecting the
// commands in each step against a predefined set of risk rules.
func (p *planner) AnalyzeRisk(_ context.Context, plan *models.ExecutionPlan) (*models.RiskAssessment, error) {
	p.log.Info("Analyzing risk for generated execution plan.")

	highestRisk := &models.RiskAssessment{
		Level:       "Low",
		Description: "No significant risks detected.",
	}

	riskLevels := map[string]int{"Low": 0, "Medium": 1, "High": 2, "Critical": 3}

	for _, step := range plan.Steps {
		cmd := strings.ToLower(step.Action.Command)
		for _, rule := range riskRules {
			if strings.Contains(cmd, rule.Pattern) {
				if riskLevels[rule.Level] > riskLevels[highestRisk.Level] {
					highestRisk.Level = rule.Level
					highestRisk.Description = rule.Description
					p.log.Warnf("Risk rule triggered. Level: %s, Pattern: '%s', Step: '%s'", rule.Level, rule.Pattern, step.Name)
				}
			}
		}
	}

	return highestRisk, nil
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
