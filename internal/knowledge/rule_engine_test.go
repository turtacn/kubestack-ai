package knowledge_test

import (
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/knowledge"
	"github.com/stretchr/testify/assert"
)

func TestRuleMatching(t *testing.T) {
	// Setup: 创建规则引擎和测试规则
	kb := knowledge.NewKnowledgeBase()
	engine := knowledge.NewRuleEngine(kb)

	rule := &knowledge.Rule{
		ID:             "redis-mem-001",
		Name:           "High Memory",
		MiddlewareType: "redis",
		Condition:      "memory_usage > 80",
		Priority:       10,
		Recommendation: "Increase memory or enable eviction",
		Severity:       "HIGH",
	}
	err := kb.AddRule(rule)
	assert.NoError(t, err)

	// Setup: 创建诊断上下文
	ctx := &knowledge.DiagnosisContext{
		MiddlewareType: "redis",
		Metrics: map[string]interface{}{
			"memory_usage": 85.0,
		},
	}

	// Action: 执行规则匹配
	matches, err := engine.Match(ctx)
	assert.NoError(t, err)

	// Assert: 验证匹配结果
	assert.Len(t, matches, 1)
	assert.Equal(t, "redis-mem-001", matches[0].Rule.ID)
	assert.Equal(t, 10, matches[0].Rule.Priority)
}

func TestConditionEvaluation(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		context   map[string]interface{}
		expected  bool
	}{
		{
			name:      "Simple >",
			condition: "cpu_usage > 80",
			context:   map[string]interface{}{"cpu_usage": 85},
			expected:  true,
		},
		{
			name:      "Complex AND",
			condition: "cpu_usage > 80 && memory_usage > 70",
			context:   map[string]interface{}{"cpu_usage": 85, "memory_usage": 75},
			expected:  true,
		},
		{
			name:      "Complex OR",
			condition: "error_rate > 0.05 || latency > 1000",
			context:   map[string]interface{}{"error_rate": 0.03, "latency": 1200},
			expected:  true,
		},
		{
			name:      "False condition",
			condition: "disk_usage > 90",
			context:   map[string]interface{}{"disk_usage": 50},
			expected:  false,
		},
	}

	evaluator := knowledge.NewConditionEvaluator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(tt.condition, tt.context)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKnowledgeBaseQuery(t *testing.T) {
	// Setup: 初始化知识库
	kb := knowledge.NewKnowledgeBase()

	rules := []*knowledge.Rule{
		{
			ID:             "r1",
			Name:           "R1",
			MiddlewareType: "redis",
			Severity:       "HIGH",
			Condition:      "true",
			Tags:           []string{"tag1"},
			Priority:       10,
		},
		{
			ID:             "r2",
			Name:           "R2",
			MiddlewareType: "redis",
			Severity:       "LOW",
			Condition:      "true",
			Tags:           []string{"tag2"},
			Priority:       5,
		},
		{
			ID:             "r3",
			Name:           "R3",
			MiddlewareType: "mysql",
			Severity:       "HIGH",
			Condition:      "true",
			Tags:           []string{"tag1"},
			Priority:       8,
		},
	}

	for _, r := range rules {
		kb.AddRule(r)
	}

	// Action: 按中间件类型查询
	t.Run("Query by Middleware", func(t *testing.T) {
		res, err := kb.QueryRules(knowledge.QueryOptions{
			MiddlewareType: "redis",
		})
		assert.NoError(t, err)
		assert.Len(t, res, 2)
	})

	// Action: 按严重级别过滤
	t.Run("Query by Severity", func(t *testing.T) {
		res, err := kb.QueryRules(knowledge.QueryOptions{
			Severity: []string{"HIGH"},
		})
		assert.NoError(t, err)
		assert.Len(t, res, 2) // r1 and r3
	})

	// Action: 按Tags过滤
	t.Run("Query by Tags", func(t *testing.T) {
		res, err := kb.QueryRules(knowledge.QueryOptions{
			Tags: []string{"tag1"},
		})
		assert.NoError(t, err)
		assert.Len(t, res, 2) // r1 and r3
	})
}
