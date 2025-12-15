package context_test

import (
	"context"
	"testing"
	"time"

	ncontext "github.com/kubestack-ai/kubestack-ai/internal/nlp/context"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextManager_MultiTurn(t *testing.T) {
	manager := ncontext.NewInMemoryContextManager(5, 1*time.Minute, nil)
	ctx := context.Background()
	sessionID := "test-session"

	// 1. Get initial context
	c, err := manager.GetContext(ctx, sessionID)
	require.NoError(t, err)
	assert.Empty(t, c.Turns)

	// 2. Add a turn
	c.AddTurn(&ncontext.Turn{
		Text: "Hello",
		Time: time.Now(),
	})
	err = manager.SaveContext(ctx, sessionID, c)
	require.NoError(t, err)

	// 3. Retrieve and verify
	c2, err := manager.GetContext(ctx, sessionID)
	require.NoError(t, err)
	assert.Len(t, c2.Turns, 1)
	assert.Equal(t, "Hello", c2.Turns[0].Text)
}

func TestContextManager_EntityCarryOver(t *testing.T) {
	manager := ncontext.NewInMemoryContextManager(10, 30*time.Minute, nil)
	ctx := context.Background()
	sessionID := "carry-over-test"

	// Create initial context with Redis entity
	convCtx, _ := manager.GetContext(ctx, sessionID)
	convCtx.AddTurn(&ncontext.Turn{
		Entities: []entity.Entity{
			{Type: entity.EntityMiddlewareType, Value: "redis", NormValue: "redis"},
		},
	})
	manager.SaveContext(ctx, sessionID, convCtx)

	// Retrieve again
	loadedCtx, _ := manager.GetContext(ctx, sessionID)
	activeEntity, found := loadedCtx.GetActiveEntity(entity.EntityMiddlewareType)

	assert.True(t, found)
	assert.Equal(t, "redis", activeEntity.NormValue)
}

func TestContextManager_SessionTimeout(t *testing.T) {
	// Set very short TTL
	manager := ncontext.NewInMemoryContextManager(10, 10*time.Millisecond, nil)
	ctx := context.Background()
	sessionID := "timeout-test"

	c, _ := manager.GetContext(ctx, sessionID)
	c.AddTurn(&ncontext.Turn{Text: "hi"})
	manager.SaveContext(ctx, sessionID, c)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	c2, _ := manager.GetContext(ctx, sessionID)
	assert.Empty(t, c2.Turns, "Context should be empty (reset) after timeout")
}
