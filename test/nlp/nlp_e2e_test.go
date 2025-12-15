package nlp_test

import (
	"context"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/nlp"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp/entity"
	"github.com/kubestack-ai/kubestack-ai/internal/nlp/intent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNLPProcessor_FullPipeline(t *testing.T) {
	// === Setup ===
	processor := nlp.NewNLPProcessor(&nlp.Config{
		TokenizerType: "simple",
		EnableLLM:     false,
		MaxTurns:      10,
		SessionTTL:    30 * time.Minute,
	}, nil)
	ctx := context.Background()

	// === Test Case 1: Full Pipeline ===
	// Use explicit separation to ensure both middleware type and instance id are extracted without overlap
	result, err := processor.Process(ctx, &nlp.ProcessRequest{
		Text:      "帮我看看Redis实例redis-cluster-01的内存使用率",
		SessionID: "session-1",
	})

	require.NoError(t, err)

	// Verify Intent
	assert.Equal(t, intent.IntentDiagnose, result.Intent.Type)

	// Verify Entities
	entityTypes := make([]entity.EntityType, 0)
	for _, e := range result.Entities {
		entityTypes = append(entityTypes, e.Type)
	}
	assert.Contains(t, entityTypes, entity.EntityMiddlewareType, "Should contain middleware type (Redis)")
	assert.Contains(t, entityTypes, entity.EntityInstanceID, "Should contain instance ID (redis-cluster-01)")
	assert.Contains(t, entityTypes, entity.EntityMetricName, "Should contain metric name (memory_usage)")
}

func TestNLPProcessor_MultiTurnConversation(t *testing.T) {
	processor := nlp.NewNLPProcessor(nlp.DefaultConfig(), nil)
	ctx := context.Background()
	sessionID := "multi-turn-test"

	// === Turn 1: Mention Redis ===
	result1, _ := processor.Process(ctx, &nlp.ProcessRequest{
		Text:      "Redis内存使用率多少",
		SessionID: sessionID,
	})
	assert.Equal(t, intent.IntentQuery, result1.Intent.Type)

	// === Turn 2: Use pronoun "它" (it) or just implied context ===
	result2, _ := processor.Process(ctx, &nlp.ProcessRequest{
		Text:      "它的连接数呢", // "It" refers to Redis
		SessionID: sessionID,
	})

	// Verify Intent
	assert.Equal(t, intent.IntentQuery, result2.Intent.Type)

	// Verify Redis context is available in result.Context
	activeMiddleware, found := result2.Context.GetActiveEntity(entity.EntityMiddlewareType)
	assert.True(t, found)
	assert.Equal(t, "redis", activeMiddleware.NormValue)

	// === Turn 3: Follow up ===
	result3, _ := processor.Process(ctx, &nlp.ProcessRequest{
		Text:      "帮我清理下慢日志",
		SessionID: sessionID,
	})

	// Intent changes to Fix
	assert.Equal(t, intent.IntentFix, result3.Intent.Type)

	activeMiddleware3, found3 := result3.Context.GetActiveEntity(entity.EntityMiddlewareType)
	assert.True(t, found3)
	assert.Equal(t, "redis", activeMiddleware3.NormValue)
}
