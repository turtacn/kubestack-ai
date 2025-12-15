package plugin_test

import (
	"context"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/builtin/elasticsearch"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/builtin/kafka"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/builtin/mysql"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/builtin/postgresql"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/builtin/redis"
	"github.com/stretchr/testify/assert"
)

func TestAllPlugins_ImplementFixInterface(t *testing.T) {
	// Initialize all plugins
	redisP, _ := redis.New()
	mysqlP, _ := mysql.New()
	kafkaP, kafkaErr := kafka.New()
	esP, esErr := elasticsearch.New()
	pgP, _ := postgresql.New()

	plugins := []struct {
		Name   string
		Plugin interface{}
		Err    error
	}{
		{"redis", redisP, nil},
		{"mysql", mysqlP, nil},
		{"kafka", kafkaP, kafkaErr},
		{"elasticsearch", esP, esErr},
		{"postgresql", pgP, nil},
	}

	for _, p := range plugins {
		t.Run(p.Name, func(t *testing.T) {
			if p.Err != nil {
				t.Skipf("Skipping %s due to init error (likely missing dependency): %v", p.Name, p.Err)
			}
			assert.NotNil(t, p.Plugin)
		})
	}
}

func TestRedisPlugin_FixCapability(t *testing.T) {
	p, err := redis.New()
	assert.NoError(t, err)

	ctx := context.WithValue(context.Background(), base.DryRunContextKey, true)

	t.Run("CanAutoFix_MemoryHigh", func(t *testing.T) {
		issue := &models.Issue{Title: redis.IssueTitleMemoryHigh}
		ok, fix := p.CanAutoFix(issue)
		assert.True(t, ok)
		assert.NotNil(t, fix)
		assert.Equal(t, "MEMORY PURGE", fix.Command)

		res, err := p.ExecuteFix(ctx, fix)
		assert.NoError(t, err)
		assert.True(t, res.Success)
		assert.Contains(t, res.Message, "[DryRun]")
	})

	t.Run("CanAutoFix_SlowLog", func(t *testing.T) {
		issue := &models.Issue{Title: redis.IssueTitleSlowLog}
		ok, fix := p.CanAutoFix(issue)
		assert.True(t, ok)
		assert.Equal(t, "SLOWLOG RESET", fix.Command)
	})
}

func TestMySQLPlugin_FixCapability(t *testing.T) {
	p, err := mysql.New()
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), base.DryRunContextKey, true)

	t.Run("CanAutoFix_SlowQuery", func(t *testing.T) {
		issue := &models.Issue{
			Title:    mysql.IssueTitleSlowQuery,
			Evidence: "Process ID: 12345 executing...",
		}
		ok, fix := p.CanAutoFix(issue)
		assert.True(t, ok)
		assert.Equal(t, "KILL QUERY ?", fix.Command)
		assert.Equal(t, "12345", fix.Parameters["process_id"])

		res, err := p.ExecuteFix(ctx, fix)
		assert.NoError(t, err)
		assert.True(t, res.Success)
	})
}

func TestKafkaPlugin_FixCapability(t *testing.T) {
	p, err := kafka.New()
	if err != nil {
		t.Skip("Kafka not available")
	}
	ctx := context.WithValue(context.Background(), base.DryRunContextKey, true)

	t.Run("CanAutoFix_ConsumerLag", func(t *testing.T) {
		issue := &models.Issue{Title: kafka.IssueTitleConsumerLag}
		ok, fix := p.CanAutoFix(issue)
		assert.True(t, ok)
		assert.Equal(t, "KAFKA_RESET_OFFSET", fix.Command)

		res, err := p.ExecuteFix(ctx, fix)
		assert.NoError(t, err)
		assert.True(t, res.Success)
	})
}

func TestESPlugin_FixCapability(t *testing.T) {
	p, err := elasticsearch.New()
	if err != nil {
		t.Skip("ES not available")
	}
	ctx := context.WithValue(context.Background(), base.DryRunContextKey, true)

	t.Run("CanAutoFix_ReadOnly", func(t *testing.T) {
		issue := &models.Issue{Title: elasticsearch.IssueTitleIndexReadOnly}
		ok, fix := p.CanAutoFix(issue)
		assert.True(t, ok)
		assert.Equal(t, "ES_UNLOCK_INDEX", fix.Command)

		res, err := p.ExecuteFix(ctx, fix)
		assert.NoError(t, err)
		assert.True(t, res.Success)
	})
}

func TestPostgresPlugin_FixCapability(t *testing.T) {
	p, err := postgresql.New()
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), base.DryRunContextKey, true)

	t.Run("CanAutoFix_IdleTx", func(t *testing.T) {
		issue := &models.Issue{Title: postgresql.IssueTitleIdleTx}
		ok, fix := p.CanAutoFix(issue)
		assert.True(t, ok)
		assert.Equal(t, "PG_TERMINATE_BACKEND", fix.Command)

		res, err := p.ExecuteFix(ctx, fix)
		assert.NoError(t, err)
		assert.True(t, res.Success)
	})
}
