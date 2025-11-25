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
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/core/execution"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/stretchr/testify/assert"
)

func TestAnalyzeRisk_HighRisk(t *testing.T) {
	actions := []*models.FixAction{
		{Category: "Restart"},
	}
	assessment, err := execution.AnalyzeRisk(context.Background(), actions, "Production")
	assert.NoError(t, err)
	assert.Equal(t, models.RiskLevelHigh, assessment.MaxSeverity)
	assert.True(t, assessment.RequiresApproval)
}

func TestGeneratePlan_Ordering(t *testing.T) {
	recommendations := []*models.Recommendation{
		{ID: "rec-restart", CanAutoFix: true, Command: "systemctl restart myservice", Category: "Restart"},
		{ID: "rec-config", CanAutoFix: true, Command: "echo 'timeout=500' >> /etc/myservice.conf", Category: "ConfigChange"},
	}

	planner := execution.NewPlanner()
	plan, err := planner.GeneratePlan(context.Background(), recommendations)

	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Len(t, plan.Steps, 2)
	assert.Equal(t, "ConfigChange", plan.Steps[0].Action.Category)
	assert.Equal(t, "Restart", plan.Steps[1].Action.Category)
}
