package entity

import (
	"regexp"
	"strings"
)

// EntityPattern defines a regex pattern for entity extraction with an optional normalizer.
type EntityPattern struct {
	Regex      *regexp.Regexp
	Normalizer func(match string) string
}

// Middleware type dictionary
var middlewareTypeDict = map[string]string{
	"redis":         "redis",
	"mysql":         "mysql",
	"kafka":         "kafka",
	"elasticsearch": "elasticsearch",
	"es":            "elasticsearch",
	"postgresql":    "postgresql",
	"postgres":      "postgresql",
	"pg":            "postgresql",
	"mongodb":       "mongodb",
	"mongo":         "mongodb",
	"zookeeper":     "zookeeper",
	"zk":            "zookeeper",
	"nginx":         "nginx",
}

// Metric name dictionary
var metricNameDict = map[string]string{
	"内存":      "memory_usage",
	"内存使用率":   "memory_usage",
	"memory":    "memory_usage",
	"连接数":     "connections",
	"connection": "connections",
	"qps":       "qps",
	"tps":       "tps",
	"延迟":      "latency",
	"latency":   "latency",
	"响应时间":    "latency",
	"cpu":       "cpu",
	"cpu使用率":  "cpu",
	"磁盘":      "disk_usage",
	"磁盘使用率":   "disk_usage",
	"disk":      "disk_usage",
	"慢查询":     "slow_query",
	"slowlog":   "slow_query",
	"lag":       "consumer_lag",
	"堆积":      "consumer_lag",
}

// Time range patterns
var timeRangePatterns = []*EntityPattern{
	{
		Regex: regexp.MustCompile(`最近(\d+)(小时|分钟|天|周|h|m|d|w)`),
		Normalizer: func(match string) string {
			// Basic normalization, e.g., "最近1小时" -> "1h"
			match = strings.Replace(match, "最近", "", 1)
			match = strings.Replace(match, "小时", "h", 1)
			match = strings.Replace(match, "分钟", "m", 1)
			match = strings.Replace(match, "天", "d", 1)
			match = strings.Replace(match, "周", "w", 1)
			return match
		},
	},
	{
		Regex: regexp.MustCompile(`(今天|昨天|本周|上周|today|yesterday)`),
		Normalizer: func(match string) string {
			return match // Keep as is, resolved at runtime
		},
	},
}

// Threshold patterns
var thresholdPatterns = []*EntityPattern{
	{
		Regex: regexp.MustCompile(`(\d+(\.\d+)?)\s*(%|百分比)`),
		Normalizer: func(match string) string {
			return strings.TrimSpace(match)
		},
	},
	{
		Regex: regexp.MustCompile(`(>|<|>=|<=|超过|低于|大于|小于)\s*(\d+)`),
		Normalizer: func(match string) string {
			return match
		},
	},
}

// Instance ID patterns
var instanceIDPatterns = []*EntityPattern{
	{
		// Matches: redis-cluster-01, mysql-master-prod, pod-name-123
		Regex: regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9-]*-[a-zA-Z0-9]+`),
		Normalizer: nil,
	},
	{
		// Matches IP:Port like 10.0.1.100:6379
		Regex: regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d+`),
		Normalizer: nil,
	},
}
