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

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// RiskRule defines a condition and an associated risk score.
type RiskRule struct {
	// Condition is a function that evaluates if the rule applies to a given action and context.
	Condition func(action *models.FixAction, env string) bool
	// Score is the risk score to assign if the condition is met.
	Score int
	// Severity is the qualitative risk level (e.g., Low, Medium, High).
	Severity models.RiskLevel
	// Description provides context for why this rule is triggered.
	Description string
}

// DefaultRiskRules returns a predefined slice of risk rules for the planner to use.
// These rules help in assessing the potential impact of a fix action.
func DefaultRiskRules() []RiskRule {
	return []RiskRule{
		{
			// Rule: Restarting a service in a production environment is a high-risk operation.
			Condition: func(action *models.FixAction, env string) bool {
				return action.Category == "Restart" && env == "Production"
			},
			Score:    80,
			Severity: models.RiskLevelHigh,
			Description: "Restarting a service in a production environment can cause a service outage.",
		},
		{
			// Rule: Any configuration change carries a moderate risk.
			Condition: func(action *models.FixAction, env string) bool {
				return action.Category == "ConfigChange"
			},
			Score:    40,
			Severity: models.RiskLevelMedium,
			Description: "Changing configuration can have unintended side effects.",
		},
		{
			// Rule: Scaling operations are generally low risk but should be noted.
			Condition: func(action *models.FixAction, env string) bool {
				return action.Category == "Scale"
			},
			Score:    20,
			Severity: models.RiskLevelLow,
			Description: "Scaling operations can affect performance and resource consumption.",
		},
		{
			// Rule: Deleting resources is a high-risk operation.
			Condition: func(action *models.FixAction, env string) bool {
				return action.Category == "Delete"
			},
			Score:    90,
			Severity: models.RiskLevelCritical,
			Description: "Deleting resources can lead to data loss or service unavailability.",
		},
	}
}

// AnalyzeRisk assesses the potential risks of a set of actions based on the default risk rules.
// It calculates a total risk score and determines if manual approval is required.
func AnalyzeRisk(ctx context.Context, actions []*models.FixAction, env string) (*models.RiskAssessment, error) {
	totalScore := 0
	maxSeverity := models.RiskLevelLow
	requiresApproval := false

	rules := DefaultRiskRules()

	for _, action := range actions {
		for _, rule := range rules {
			if rule.Condition(action, env) {
				totalScore += rule.Score
				if rule.Severity > maxSeverity {
					maxSeverity = rule.Severity
				}
			}
		}
	}

	// Example threshold: any score over 70 or any action with High/Critical severity requires approval.
	if totalScore > 70 || maxSeverity >= models.RiskLevelHigh {
		requiresApproval = true
	}

	assessment := &models.RiskAssessment{
		TotalScore:       totalScore,
		MaxSeverity:      maxSeverity,
		RequiresApproval: requiresApproval,
		// Description can be enhanced to list which rules were triggered.
	}

	return assessment, nil
}
