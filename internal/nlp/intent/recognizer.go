package intent

import (
	"context"
	"regexp"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/nlp/entity"
)

// Recognizer is the interface for intent recognition.
type Recognizer interface {
	Recognize(ctx context.Context, req *RecognizeRequest) (*Intent, error)
	Name() string
}

// RecognizeRequest represents a request for intent recognition.
type RecognizeRequest struct {
	Text     string
	Tokens   []string
	Entities []entity.Entity
	History  []*Intent // History intents (for context)
}

// RuleBasedRecognizer is a regex-based intent recognizer.
type RuleBasedRecognizer struct {
	patterns map[IntentType][]string // raw patterns for viewing
	regexps  map[IntentType][]*regexp.Regexp
	keywords map[IntentType][]string
	priority []IntentType
}

// NewRuleBasedRecognizer creates a new RuleBasedRecognizer.
func NewRuleBasedRecognizer() *RuleBasedRecognizer {
	r := &RuleBasedRecognizer{
		patterns: defaultIntentPatterns,
		regexps:  make(map[IntentType][]*regexp.Regexp),
		keywords: defaultIntentKeywords,
		priority: []IntentType{
			IntentFix,      // High risk/impact first
			IntentAlert,
			IntentConfig,
			IntentDiagnose,
			IntentQuery,
			IntentExplain,
			IntentCompare,
			IntentHelp,
		},
	}

	// Compile regexps
	for intentType, patterns := range r.patterns {
		for _, p := range patterns {
			re, err := regexp.Compile(p)
			if err == nil {
				r.regexps[intentType] = append(r.regexps[intentType], re)
			}
		}
	}

	return r
}

func (r *RuleBasedRecognizer) Name() string {
	return "RuleBased"
}

// Recognize performs rule-based intent recognition.
func (r *RuleBasedRecognizer) Recognize(ctx context.Context, req *RecognizeRequest) (*Intent, error) {
	// 1. Iterate by priority
	for _, intentType := range r.priority {
		regexps := r.regexps[intentType]

		// 2. Regex match
		for _, re := range regexps {
			if re.MatchString(req.Text) {
				return &Intent{
					Type:       intentType,
					Confidence: 0.9,
					RawText:    req.Text,
					Reason:     "regex_match",
				}, nil
			}
		}

		// 3. Keyword match (simple)
		if keywords, ok := r.keywords[intentType]; ok && len(keywords) > 0 {
			matchCount := 0
			// Basic keyword matching in raw text
			for _, kw := range keywords {
				if strings.Contains(req.Text, kw) {
					matchCount++
				}
			}

			if matchCount >= 1 { // Relaxed to 1 keyword for testing
				// But we need to be careful with single keyword false positives.
				// For "Redis内存使用率多少", "多少" is a keyword.
				// "内存使用率" is not in keywords list above, but usually we should add it.
				// Let's rely on regex mostly.
			}
		}
	}

	return &Intent{
		Type:       IntentUnknown,
		Confidence: 0.0,
		RawText:    req.Text,
	}, nil
}

// Default intent patterns
var defaultIntentPatterns = map[IntentType][]string{
	IntentDiagnose: {
		`(?i)(诊断|检查|看看|分析|排查|查一查).*(redis|mysql|kafka|es|pg|集群|实例|节点|缓存|db)`,
		`(?i)(redis|mysql|kafka|es|pg).*(怎么了|什么问题|有问题|异常|挂了|坏了|不正常)`,
		`(?i)帮我.*(诊断|排查|分析)`,
	},
	IntentQuery: {
		`(?i)(查询|查看|获取|显示|get|show).*(连接数|内存|cpu|qps|延迟|状态|info|lag|消费)`,
		`(?i)(.*)(是多少|多大|多高|多低|状态如何|呢)`, // "呢" for "它的连接数呢"
		`(?i)(当前|现在).*(指标|状态|情况|数据)`,
		`(?i)(.*)(内存|cpu|连接|qps|延迟|lag|offset).*`, // Generalized query for metric mention without verb? Risk of false positive.
	},
	IntentFix: {
		`(?i)(清理|清除|修复|重启|杀掉|kill|restart|fix|clean).*(连接|慢查询|日志|内存|实例|进程)`,
		`(?i)(解决|处理|修).*(问题|故障|异常)`,
		`(?i)(释放|回收).*(内存|连接|资源)`,
	},
	IntentAlert: {
		`(?i)(设置|配置|添加).*(告警|监控|阈值|通知|alert)`,
		`(?i)(当|如果).*(超过|低于|达到).*(通知|报警|告诉我)`,
	},
	IntentConfig: {
		`(?i)(修改|设置|调整|更新|set|update|config).*(配置|参数|max|min|timeout|buffer)`,
		`(?i)把.*(改成|设为|调整为)`,
	},
	IntentExplain: {
		`(?i)(什么是|解释|说明|explain).*(主从|延迟|碎片|慢查询|lag)`,
		`(?i)(为什么|原因).*(高|低|慢|超时|报错)`,
	},
	IntentHelp: {
		`(?i)(帮助|help|你能做什么|有什么功能|usage|guide)`,
		`(?i)(怎么用|使用方法|命令|指令)`,
	},
}

var defaultIntentKeywords = map[IntentType][]string{
	IntentDiagnose: {"诊断", "分析", "异常", "排查", "问题"},
	IntentQuery:    {"查询", "查看", "多少", "状态", "呢"},
}
