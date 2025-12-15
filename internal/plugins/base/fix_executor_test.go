package base

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/stretchr/testify/assert"
)

func TestFixExecutor_Execute(t *testing.T) {
	log := logger.NewLogger("test")
	exec := NewFixExecutor(log)
	ctx := context.Background()
	action := &models.FixAction{Description: "Test Fix"}

	t.Run("Success", func(t *testing.T) {
		execFn := func(ctx context.Context) error { return nil }
		res, err := exec.Execute(ctx, action, execFn, nil)
		assert.NoError(t, err)
		assert.True(t, res.Success)
	})

	t.Run("Failure", func(t *testing.T) {
		execFn := func(ctx context.Context) error { return fmt.Errorf("boom") }
		res, err := exec.Execute(ctx, action, execFn, nil)
		assert.Error(t, err)
		assert.False(t, res.Success)
	})
}

func TestFixExecutor_Rollback(t *testing.T) {
	log := logger.NewLogger("test")
	exec := NewFixExecutor(log)
	ctx := context.Background()
	action := &models.FixAction{Description: "Test Rollback"}

	t.Run("Rollback Success", func(t *testing.T) {
		execFn := func(ctx context.Context) error { return fmt.Errorf("fail") }
		rollbackFn := func(ctx context.Context) error { return nil }
		res, err := exec.Execute(ctx, action, execFn, rollbackFn)
		assert.Error(t, err)
		assert.False(t, res.Success)
		assert.Contains(t, res.Message, "Rollback successful")
	})

	t.Run("Rollback Failure", func(t *testing.T) {
		execFn := func(ctx context.Context) error { return fmt.Errorf("fail") }
		rollbackFn := func(ctx context.Context) error { return fmt.Errorf("rb fail") }
		res, err := exec.Execute(ctx, action, execFn, rollbackFn)
		assert.Error(t, err)
		assert.False(t, res.Success)
		assert.Contains(t, res.Message, "Rollback also failed")
	})
}

func TestFixExecutor_DryRun(t *testing.T) {
	log := logger.NewLogger("test")
	exec := NewFixExecutor(log)
	ctx := context.WithValue(context.Background(), DryRunContextKey, true)
	action := &models.FixAction{Description: "Test DryRun"}

	execFn := func(ctx context.Context) error { return fmt.Errorf("should not run") }
	res, err := exec.Execute(ctx, action, execFn, nil)
	assert.NoError(t, err)
	assert.True(t, res.Success)
	assert.Contains(t, res.Message, "Simulated execution")
}
