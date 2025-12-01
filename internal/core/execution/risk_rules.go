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
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// RiskRule defines a condition and a corresponding risk score.
type RiskRule struct {
	Condition func(action *models.FixAction) bool
	Score     int
	Severity  models.RiskLevel
}

// DefaultRiskRules provides a baseline set of rules for risk assessment.
func DefaultRiskRules() []RiskRule {
	return []RiskRule{
		{
			Condition: func(action *models.FixAction) bool {
				return action.Category == "Restart"
			},
			Score:    80,
			Severity: models.RiskLevelHigh,
		},
		{
			Condition: func(action *models.FixAction) bool {
				return action.Category == "ConfigChange"
			},
			Score:    40,
			Severity: models.RiskLevelMedium,
		},
		{
			Condition: func(action *models.FixAction) bool {
				return action.Category == "Scale"
			},
			Score:    60,
			Severity: models.RiskLevelMedium,
		},
		{
			Condition: func(action *models.FixAction) bool {
				return action.Category == "Destructive" // e.g., deleting data
			},
			Score:    100,
			Severity: models.RiskLevelCritical,
		},
	}
}
