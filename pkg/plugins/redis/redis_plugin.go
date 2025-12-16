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

// RedisPlugin Redis诊断插件.
type RedisPlugin struct {
	client *redis.Client
}

func (p *RedisPlugin) Name() string {
	return "redis"
}

func (p *RedisPlugin) Type() plugin.MiddlewareType {
	return plugin.MiddlewareRedis
}

func (p *RedisPlugin) Version() string {
	return "1.0.0"
}

// Connect implements MiddlewarePlugin
func (p *RedisPlugin) Connect(ctx context.Context, config *plugin.ConnectionConfig) error {
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       0,
	}
	if config.PoolSize > 0 {
		opts.PoolSize = config.PoolSize
	}
	p.client = redis.NewClient(opts)
	return p.client.Ping(ctx).Err()
}

// Disconnect implements MiddlewarePlugin
func (p *RedisPlugin) Disconnect(ctx context.Context) error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// Ping implements MiddlewarePlugin
func (p *RedisPlugin) Ping(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("not connected")
	}
	return p.client.Ping(ctx).Err()
}

// IsConnected implements MiddlewarePlugin
func (p *RedisPlugin) IsConnected() bool {
	return p.client != nil
}

// CollectMetrics implements MiddlewarePlugin
func (p *RedisPlugin) CollectMetrics(ctx context.Context) (*plugin.MetricsSnapshot, error) {
	// Not implemented for this patch
	return nil, nil
}

// CollectSpecificMetric implements MiddlewarePlugin
func (p *RedisPlugin) CollectSpecificMetric(ctx context.Context, metricName string) (interface{}, error) {
	return nil, nil
}

// Execute implements MiddlewarePlugin
func (p *RedisPlugin) Execute(ctx context.Context, cmd *plugin.Command) (*plugin.CommandResult, error) {
	return nil, nil
}

// SupportedCommands implements MiddlewarePlugin
func (p *RedisPlugin) SupportedCommands() []plugin.CommandSpec {
	return nil
}

// GetDiagnosticData implements MiddlewarePlugin
func (p *RedisPlugin) GetDiagnosticData(ctx context.Context) (*plugin.DiagnosticData, error) {
	return nil, nil
}

// GetBuiltinRules implements MiddlewarePlugin
func (p *RedisPlugin) GetBuiltinRules() []plugin.DiagnosisRule {
	return nil
}


// Legacy Interface Methods (DiagnosticPlugin)
func (p *RedisPlugin) Init(config map[string]interface{}) error {
	return nil
}

func (p *RedisPlugin) SupportedTypes() []string {
	return []string{"redis"}
}

func (p *RedisPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	if req.Instance == "" {
		return nil, fmt.Errorf("redis diagnosis requires an instance endpoint in the request")
	}

	addr := req.Instance
	var password string

	// Create a temporary client if not connected or address differs
	client := p.client
	if client == nil || client.Options().Addr != addr {
		client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       0,
		})
		defer client.Close()
	}

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
		return nil, metrics
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
		return nil
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
		return nil
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

func (p *RedisPlugin) Shutdown() error {
	return p.Disconnect(context.Background())
}

// Helpers
func parseMemoryInfo(info string, key string) int64 {
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
