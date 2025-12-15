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

package base_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
	"github.com/stretchr/testify/assert"
)

func TestFixExecutor_Execute(t *testing.T) {
	log := logger.NewLogger("test")
	executor := base.NewFixExecutor(log)
	action := &models.FixAction{
		Description: "Test Fix",
		Command:     "echo test",
	}

	t.Run("Success", func(t *testing.T) {
		executed := false
		res, err := executor.Execute(context.Background(), action, func(ctx context.Context) error {
			executed = true
			return nil
		}, nil)

		assert.NoError(t, err)
		assert.True(t, res.Success)
		assert.True(t, executed)
	})

	t.Run("FailureWithRollback", func(t *testing.T) {
		rolledBack := false
		res, err := executor.Execute(context.Background(), action, func(ctx context.Context) error {
			return errors.New("exec failed")
		}, func(ctx context.Context) error {
			rolledBack = true
			return nil
		})

		assert.Error(t, err)
		assert.False(t, res.Success)
		assert.True(t, rolledBack)
		assert.Contains(t, res.Message, "Rollback successful")
	})

	t.Run("DryRun", func(t *testing.T) {
		executed := false
		ctx := context.WithValue(context.Background(), base.DryRunContextKey, true)
		res, err := executor.Execute(ctx, action, func(ctx context.Context) error {
			executed = true
			return nil
		}, nil)

		assert.NoError(t, err)
		assert.True(t, res.Success)
		assert.False(t, executed)
		assert.Contains(t, res.Message, "[DryRun]")
	})
}
