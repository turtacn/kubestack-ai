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

package execution_test

import (
	"context"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/core/execution"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/stretchr/testify/assert"
)

func TestAnalyzeRisk_HighRisk(t *testing.T) {
	planner := execution.NewPlanner()
	actions := []*models.FixAction{
		{Description: "Restart server", Category: "Restart"},
	}

	assessment, err := planner.AnalyzeRisk(context.Background(), actions)

	assert.NoError(t, err)
	assert.NotNil(t, assessment)
	assert.Equal(t, models.RiskLevelHigh, assessment.MaxSeverity)
	assert.True(t, assessment.RequiresApproval)
}

func TestGeneratePlan_Ordering(t *testing.T) {
	planner := execution.NewPlanner()
	issues := []models.Issue{
		{
			Recommendations: []*models.Recommendation{
				{CanAutoFix: true, Fix: models.FixAction{Description: "Restart", Category: "Restart"}},
				{CanAutoFix: true, Fix: models.FixAction{Description: "Change config", Category: "ConfigChange"}},
			},
		},
	}

	plan, err := planner.GeneratePlan(context.Background(), issues)

	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Len(t, plan.Steps, 2)
	// Expect ConfigChange to come before Restart
	assert.Equal(t, "Fix for 'Change config'", plan.Steps[0].Name)
	assert.Equal(t, "Fix for 'Restart'", plan.Steps[1].Name)
}
