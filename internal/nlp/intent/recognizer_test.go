package intent_test

import (
	"context"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/nlp/intent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleBasedRecognizer_DiagnoseIntent(t *testing.T) {
	recognizer := intent.NewRuleBasedRecognizer()
	ctx := context.Background()

	cases := []struct {
		input    string
		expected intent.IntentType
		minConf  float64
	}{
		{"帮我诊断一下Redis", intent.IntentDiagnose, 0.8},
		{"看看MySQL有什么问题", intent.IntentDiagnose, 0.8},
		{"Redis怎么了", intent.IntentDiagnose, 0.7},
		{"检查下Kafka", intent.IntentDiagnose, 0.7},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := recognizer.Recognize(ctx, &intent.RecognizeRequest{
				Text: tc.input,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Type)
			assert.GreaterOrEqual(t, result.Confidence, tc.minConf)
		})
	}
}

func TestRuleBasedRecognizer_QueryIntent(t *testing.T) {
	recognizer := intent.NewRuleBasedRecognizer()
	ctx := context.Background()

	cases := []struct {
		input    string
		expected intent.IntentType
	}{
		{"Redis内存使用率是多少", intent.IntentQuery},
		{"查看MySQL连接数", intent.IntentQuery},
		{"当前QPS多高", intent.IntentQuery},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			result, _ := recognizer.Recognize(ctx, &intent.RecognizeRequest{Text: tc.input})
			assert.Equal(t, tc.expected, result.Type)
		})
	}
}

func TestRuleBasedRecognizer_FixIntent(t *testing.T) {
	recognizer := intent.NewRuleBasedRecognizer()
	ctx := context.Background()

	testCases := []string{
		"清理Redis慢日志",
		"杀掉慢查询",
		"释放内存",
	}

	for _, tc := range testCases {
		result, _ := recognizer.Recognize(ctx, &intent.RecognizeRequest{Text: tc})
		assert.Equal(t, intent.IntentFix, result.Type)
	}
}
