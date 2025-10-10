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
	"errors"

	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// PlaceholderManager is a temporary, non-functional implementation of the
// ExecutionManager interface. It is used to allow the orchestrator and other
// components to be instantiated before the execution engine is fully developed.
// Its methods return an error indicating that the feature is not yet implemented.
type PlaceholderManager struct{}

// PlanExecution returns an error, as this feature is not implemented.
func (p *PlaceholderManager) PlanExecution(ctx context.Context, recommendations []*models.Recommendation) (*models.ExecutionPlan, error) {
	return nil, errors.New("execution planner is not yet implemented")
}

// ExecuteActions returns an error, as this feature is not implemented.
func (p *PlaceholderManager) ExecuteActions(ctx context.Context, plan *models.ExecutionPlan, confirmFunc interfaces.ConfirmationFunc) (*models.ExecutionResult, error) {
	return nil, errors.New("execution engine is not yet implemented")
}

// ValidateExecution returns an error, as this feature is not implemented.
func (p *PlaceholderManager) ValidateExecution(ctx context.Context, result *models.ExecutionResult) error {
	return errors.New("execution validator is not yet implemented")
}