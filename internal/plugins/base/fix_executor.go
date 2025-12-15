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

package base

import (
	"context"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

const (
	// DryRunContextKey is the context key for dry-run execution.
	DryRunContextKey = "kubestack.dryrun"
)

// FixExecutor encapsulates common logic for executing fix actions, including
// logging, timing, dry-run checks, and rollback handling.
type FixExecutor struct {
	log logger.Logger
}

// NewFixExecutor creates a new FixExecutor.
func NewFixExecutor(log logger.Logger) *FixExecutor {
	return &FixExecutor{
		log: log,
	}
}

// IsDryRun checks if the context specifies a dry-run execution.
func IsDryRun(ctx context.Context) bool {
	val := ctx.Value(DryRunContextKey)
	if val == nil {
		return false
	}
	// Support boolean or string "true"
	if b, ok := val.(bool); ok {
		return b
	}
	if s, ok := val.(string); ok {
		return s == "true"
	}
	return false
}

// Execute handles the standard execution flow for a fix action.
//
// Parameters:
//   ctx: The context, potentially containing the dry-run flag.
//   action: The fix action metadata.
//   execFn: The closure containing the actual fix logic.
//   rollbackFn: Optional closure for rollback logic if execution fails.
func (e *FixExecutor) Execute(
	ctx context.Context,
	action *models.FixAction,
	execFn func(context.Context) error,
	rollbackFn func(context.Context) error,
) (*models.FixResult, error) {
	start := time.Now()

	if IsDryRun(ctx) {
		e.log.Infof("[DryRun] Would execute fix: %s (Command: %s)", action.Description, action.Command)
		return &models.FixResult{
			Success: true,
			Message: fmt.Sprintf("[DryRun] Simulated execution of: %s", action.Description),
		}, nil
	}

	e.log.Infof("Executing fix: %s", action.Description)

	err := execFn(ctx)
	duration := time.Since(start)

	if err != nil {
		e.log.Errorf("Fix execution failed after %s: %v", duration, err)

		// Attempt rollback if provided
		if rollbackFn != nil {
			e.log.Info("Attempting rollback...")
			if rbErr := rollbackFn(ctx); rbErr != nil {
				e.log.Errorf("Rollback failed: %v", rbErr)
				return &models.FixResult{
					Success: false,
					Message: fmt.Sprintf("Execution failed: %v. Rollback also failed: %v", err, rbErr),
				}, err
			}
			e.log.Info("Rollback successful.")
			return &models.FixResult{
				Success: false,
				Message: fmt.Sprintf("Execution failed: %v. Rollback successful.", err),
			}, err
		}

		return &models.FixResult{
			Success: false,
			Message: fmt.Sprintf("Execution failed: %v", err),
		}, err
	}

	e.log.Infof("Fix executed successfully in %s", duration)
	return &models.FixResult{
		Success: true,
		Message: fmt.Sprintf("Fix executed successfully in %s", duration),
	}, nil
}
