// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not not use this file except in compliance with the License.
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

func TestExecute_Success(t *testing.T) {
	manager := execution.NewManager(nil) // Planner not needed for direct execution
	plan := &models.ExecutionPlan{
		ID: "test-plan",
		Steps: []*models.ExecutionStep{
			{Name: "Step 1", Action: &models.FixAction{Command: "echo 'step 1'"}},
			{Name: "Step 2", Action: &models.FixAction{Command: "echo 'step 2'"}},
		},
	}

	result, err := manager.ExecutePlan(context.Background(), plan)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.ExecutionStatusSuccess, result.Status)
}

func TestExecute_RollbackTrigger(t *testing.T) {
	manager := execution.NewManager(nil)
	plan := &models.ExecutionPlan{
		ID: "test-plan-rollback",
		Steps: []*models.ExecutionStep{
			{Name: "Step 1", Action: &models.FixAction{Command: "echo 'step 1'", RollbackCommand: "echo 'rollback 1'"}},
			{Name: "Step 2", Action: &models.FixAction{Command: "fail"}}, // This command will fail
		},
	}

	result, err := manager.ExecutePlan(context.Background(), plan)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.ExecutionStatusFailedWithRollbackSuccess, result.Status)
}
