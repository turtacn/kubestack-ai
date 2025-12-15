package entity_test

import (
	"context"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/nlp/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityExtractor_MiddlewareType(t *testing.T) {
	extractor := entity.BuildDefaultExtractor()
	ctx := context.Background()

	cases := []struct {
		input    string
		expected string
		tokens   []string
	}{
		{"帮我看看Redis", "redis", []string{"帮", "我", "看看", "Redis"}},
		{"MySQL连接数", "mysql", []string{"MySQL", "连接数"}},
		{"ES集群状态", "elasticsearch", []string{"ES", "集群", "状态"}},
		{"pg数据库", "postgresql", []string{"pg", "数据库"}},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			entities, err := extractor.Extract(ctx, tc.input, tc.tokens)
			require.NoError(t, err)

			found := false
			for _, e := range entities {
				if e.Type == entity.EntityMiddlewareType {
					assert.Equal(t, tc.expected, e.NormValue)
					found = true
					break
				}
			}
			assert.True(t, found, "should find middleware type entity")
		})
	}
}

func TestEntityExtractor_MetricName(t *testing.T) {
	extractor := entity.BuildDefaultExtractor()
	ctx := context.Background()

	cases := []struct {
		input    string
		expected string
		tokens   []string
	}{
		{"内存使用率是多少", "memory_usage", []string{"内存使用率", "是", "多少"}},
		{"连接数告警", "connections", []string{"连接数", "告警"}},
		{"QPS很低", "qps", []string{"QPS", "很", "低"}},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			entities, err := extractor.Extract(ctx, tc.input, tc.tokens)
			require.NoError(t, err)

			found := false
			for _, e := range entities {
				if e.Type == entity.EntityMetricName {
					assert.Equal(t, tc.expected, e.NormValue)
					found = true
					break
				}
			}
			assert.True(t, found, "should find metric name entity")
		})
	}
}

func TestEntityExtractor_TimeRange(t *testing.T) {
	extractor := entity.BuildDefaultExtractor()
	ctx := context.Background()

	cases := []struct {
		input       string
		expectedVal string
	}{
		{"最近1小时的数据", "1h"},
		{"昨天的日志", "昨天"},
		{"最近30分钟", "30m"},
	}

	for _, tc := range cases {
		entities, err := extractor.Extract(ctx, tc.input, nil)
		require.NoError(t, err)

		var timeEntity *entity.Entity
		for _, e := range entities {
			if e.Type == entity.EntityTimeRange {
				timeEntity = &e
				break
			}
		}

		require.NotNil(t, timeEntity)
		assert.Equal(t, tc.expectedVal, timeEntity.NormValue)
	}
}

func TestEntityExtractor_InstanceID(t *testing.T) {
	extractor := entity.BuildDefaultExtractor()
	ctx := context.Background()

	cases := []string{
		"redis-cluster-01的状态",
		"检查10.0.1.100:6379",
		"mysql-master-prod连接数",
	}

	for _, tc := range cases {
		entities, err := extractor.Extract(ctx, tc, nil)
		require.NoError(t, err)

		found := false
		for _, e := range entities {
			if e.Type == entity.EntityInstanceID {
				found = true
				break
			}
		}
		assert.True(t, found, "should find instance ID in: %s", tc)
	}
}
