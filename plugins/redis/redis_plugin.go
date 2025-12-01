package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

func init() {
	plugin.RegisterPluginFactory("Redis", func() plugin.DiagnosticPlugin {
		return &RedisPlugin{}
	})
}

// RedisPlugin Redis诊断插件. This plugin is stateless.
type RedisPlugin struct{}

func (p *RedisPlugin) Name() string {
	return "redis"
}

func (p *RedisPlugin) SupportedTypes() []string {
	return []string{"redis"}
}

func (p *RedisPlugin) Version() string {
	return "1.0.0"
}

// Init is a no-op for the stateless Redis plugin.
func (p *RedisPlugin) Init(config map[string]interface{}) error {
	return nil // Stateless, no init needed.
}

func (p *RedisPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	if req.Instance == "" {
		return nil, fmt.Errorf("redis diagnosis requires an instance endpoint in the request")
	}

	addr := req.Instance
	// NOTE: The current DiagnosisRequest model does not support passing credentials.
	// Assuming no password for now.
	var password string

	// TODO: Add support for specifying the database index in the request.
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at '%s': %w", addr, err)
	}

	result := &models.DiagnosisResult{
		Issues:  []*models.Issue{},
		Metrics: make(map[string]interface{}),
	}

	// Step 1: 检查内存使用
	memoryIssue, metrics := p.checkMemory(ctx, client)
	if memoryIssue != nil {
		result.Issues = append(result.Issues, memoryIssue)
	}
	for k, v := range metrics {
		result.Metrics[k] = v
	}

	// Step 2: 检查连接数
	connectionIssue := p.checkConnections(ctx, client)
	if connectionIssue != nil {
		result.Issues = append(result.Issues, connectionIssue)
	}

	// Step 3: 检查慢查询
	slowQueryIssue := p.checkSlowLog(ctx, client)
	if slowQueryIssue != nil {
		result.Issues = append(result.Issues, slowQueryIssue)
	}

	// Step 4: 生成建议
	result.Issues = p.attachRecommendations(result.Issues)

	return result, nil
}

func (p *RedisPlugin) checkMemory(ctx context.Context, client *redis.Client) (*models.Issue, map[string]interface{}) {
	metrics := make(map[string]interface{})
	info, err := client.Info(ctx, "memory").Result()
	if err != nil {
		return nil, metrics // Or return a specific error issue
	}

	usedMemory := parseMemoryInfo(info, "used_memory")
	maxMemory := parseMemoryInfo(info, "maxmemory")
	usedMemoryMB := usedMemory / 1024 / 1024

	metrics["used_memory"] = usedMemory
	metrics["maxmemory"] = maxMemory
	metrics["used_memory_mb"] = usedMemoryMB

	var issue *models.Issue
	if maxMemory > 0 && float64(usedMemory)/float64(maxMemory) > 0.9 {
		issue = &models.Issue{
			Title:       "Redis内存使用率过高",
			Severity:    enum.SeverityHigh,
			Description: fmt.Sprintf("当前内存使用: %dMB, 最大内存: %dMB", usedMemoryMB, maxMemory/1024/1024),
			Source:      "RedisPlugin",
		}
	}
	return issue, metrics
}

func (p *RedisPlugin) checkConnections(ctx context.Context, client *redis.Client) *models.Issue {
	info, err := client.Info(ctx, "clients").Result()
	if err != nil {
		return nil // Or return a specific error issue
	}
	connectedClients := parseClientsInfo(info, "connected_clients")

	if connectedClients > 10000 {
		return &models.Issue{
			Title:       "Redis连接数过多",
			Severity:    enum.SeverityMedium,
			Description: fmt.Sprintf("当前连接数: %d", connectedClients),
			Source:      "RedisPlugin",
		}
	}
	return nil
}

func (p *RedisPlugin) checkSlowLog(ctx context.Context, client *redis.Client) *models.Issue {
	slowLogs, err := client.SlowLogGet(ctx, 10).Result()
	if err != nil {
		return nil // Ignore error or return warning
	}

	if len(slowLogs) > 0 {
		return &models.Issue{
			Title:       "Redis存在慢查询",
			Severity:    enum.SeverityMedium,
			Description: fmt.Sprintf("最近10条慢查询中有 %d 条", len(slowLogs)),
			Source:      "RedisPlugin",
		}
	}
	return nil
}

func (p *RedisPlugin) attachRecommendations(issues []*models.Issue) []*models.Issue {
	for _, issue := range issues {
		var recs []*models.Recommendation
		if strings.Contains(issue.Title, "内存") {
			recs = append(recs, &models.Recommendation{
				Description: "设置maxmemory-policy为allkeys-lru",
				Fix: models.FixAction{
					Description: "设置maxmemory-policy为allkeys-lru",
					Category:    "Configuration",
				},
			})
		}
		if strings.Contains(issue.Title, "连接数") {
			recs = append(recs, &models.Recommendation{
				Description: "检查客户端连接泄漏，优化连接池配置",
				Fix: models.FixAction{
					Description: "检查客户端连接泄漏，优化连接池配置",
					Category:    "Application",
				},
			})
		}
		issue.Recommendations = recs
	}
	return issues
}

// Shutdown is a no-op for the stateless Redis plugin.
func (p *RedisPlugin) Shutdown() error {
	return nil
}

// Helpers
func parseMemoryInfo(info string, key string) int64 {
	// Simple parser
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, key+":") {
			valStr := strings.TrimPrefix(line, key+":")
			val, _ := strconv.ParseInt(valStr, 10, 64)
			return val
		}
	}
	return 0
}

func parseClientsInfo(info string, key string) int64 {
	return parseMemoryInfo(info, key)
}
