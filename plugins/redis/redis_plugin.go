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
	plugin.RegisterPluginFactory("redis", func() plugin.DiagnosticPlugin {
		return &RedisPlugin{}
	})
}

// RedisPlugin Redis诊断插件
type RedisPlugin struct {
	client *redis.Client
}

func (p *RedisPlugin) Name() string {
	return "redis"
}

func (p *RedisPlugin) SupportedTypes() []string {
	return []string{"redis"}
}

func (p *RedisPlugin) Version() string {
	return "1.0.0"
}

func (p *RedisPlugin) Init(config map[string]interface{}) error {
	addr, ok := config["addr"].(string)
	if !ok {
		return fmt.Errorf("config 'addr' is required and must be a string")
	}
	password, _ := config["password"].(string)
	db, _ := config["db"].(int)

	p.client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	if err := p.client.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("Redis连接失败: %w", err)
	}

	return nil
}

func (p *RedisPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	result := &models.DiagnosisResult{
		Issues: []*models.Issue{},
	}

	// Step 1: 检查内存使用
	memoryIssue := p.checkMemory(ctx)
	if memoryIssue != nil {
		result.Issues = append(result.Issues, memoryIssue)
	}

	// Step 2: 检查连接数
	connectionIssue := p.checkConnections(ctx)
	if connectionIssue != nil {
		result.Issues = append(result.Issues, connectionIssue)
	}

	// Step 3: 检查慢查询
	slowQueryIssue := p.checkSlowLog(ctx)
	if slowQueryIssue != nil {
		result.Issues = append(result.Issues, slowQueryIssue)
	}

	// Step 4: 生成建议
	result.Issues = p.attachRecommendations(result.Issues)

	return result, nil
}

func (p *RedisPlugin) checkMemory(ctx context.Context) *models.Issue {
	info := p.client.Info(ctx, "memory").Val()
	usedMemory := parseMemoryInfo(info, "used_memory")
	maxMemory := parseMemoryInfo(info, "maxmemory")

	if maxMemory > 0 && float64(usedMemory)/float64(maxMemory) > 0.9 {
		return &models.Issue{
			Title:       "Redis内存使用率过高",
			Severity:    enum.SeverityHigh,
			Description: fmt.Sprintf("当前内存使用: %dMB, 最大内存: %dMB", usedMemory/1024/1024, maxMemory/1024/1024),
			Source:      "RedisPlugin",
		}
	}
	return nil
}

func (p *RedisPlugin) checkConnections(ctx context.Context) *models.Issue {
	info := p.client.Info(ctx, "clients").Val()
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

func (p *RedisPlugin) checkSlowLog(ctx context.Context) *models.Issue {
	slowLogs, err := p.client.SlowLogGet(ctx, 10).Result()
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
				Category:    "Configuration",
			})
		}
		if strings.Contains(issue.Title, "连接数") {
			recs = append(recs, &models.Recommendation{
				Description: "检查客户端连接泄漏，优化连接池配置",
				Category:    "Application",
			})
		}
		issue.Recommendations = recs
	}
	return issues
}

func (p *RedisPlugin) Shutdown() error {
	return p.client.Close()
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
