package redis

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

type RedisPlugin struct {
	config *RedisConfig
	client redis.UniversalClient
	logger *zap.Logger
}

type RedisConfig struct {
	Address  string
	Password string
	DB       int
	PoolSize int
	Timeout  time.Duration
}

// 实现Plugin接口
func (p *RedisPlugin) Name() string { return "redis" }
func (p *RedisPlugin) Version() string { return "1.0.0" }
func (p *RedisPlugin) Description() string {
	return "Redis middleware diagnostic plugin"
}
func (p *RedisPlugin) SupportedMiddlewareVersions() []string {
	return []string{"5.x", "6.x", "7.x"}
}

func (p *RedisPlugin) Initialize(config *plugin.PluginConfig) error {
	p.logger = zap.L().With(zap.String("plugin", "redis"))
	var redisConf RedisConfig
	if err := mapstructure.Decode(config.Settings, &redisConf); err != nil {
		return err
	}

	opts := &redis.UniversalOptions{
		Addrs:    []string{redisConf.Address},
		Password: redisConf.Password,
		DB:       redisConf.DB,
		PoolSize: redisConf.PoolSize,
	}

	if redisConf.Timeout > 0 {
		opts.DialTimeout = redisConf.Timeout
		opts.ReadTimeout = redisConf.Timeout
		opts.WriteTimeout = redisConf.Timeout
	}

	p.client = redis.NewUniversalClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}
	p.config = &redisConf
	return nil
}

func (p *RedisPlugin) Shutdown() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

func (p *RedisPlugin) Collector() plugin.DataCollector {
	return &RedisDataCollector{plugin: p}
}

func (p *RedisPlugin) Parser() plugin.MetricParser {
	return &RedisMetricParser{plugin: p}
}

func (p *RedisPlugin) HealthChecker() plugin.HealthChecker {
	return &RedisHealthChecker{plugin: p}
}

// RedisDataCollector 实现
type RedisDataCollector struct {
	plugin *RedisPlugin
}

func (c *RedisDataCollector) Collect(ctx context.Context, target *plugin.Target) (*plugin.CollectedData, error) {
	// If target is provided and different from internal config, we might need to create a new client.
	// For now, we assume the plugin is initialized for a specific target, or we reuse the client.
	// The prompt implies we use p.client.

	infoStr, err := c.plugin.client.Info(ctx).Result()
	if err != nil {
		return nil, err
	}

	// Try to get slowlogs, ignore error as it might be disabled
	slowlogs, _ := c.plugin.client.Do(ctx, "SLOWLOG", "GET", 100).Result()

	// Try to get config, ignore error as it might be restricted
	configs, _ := c.plugin.client.Do(ctx, "CONFIG", "GET", "*").Result()

	data := &plugin.CollectedData{
		PluginName: "redis",
		Target:     target,
		Timestamp:  time.Now(),
		RawData: map[string]interface{}{
			"info":    infoStr,
			"slowlog": slowlogs,
			"config":  configs,
		},
	}
	return data, nil
}

func (c *RedisDataCollector) SupportedDataSources() []plugin.DataSourceType {
	return []plugin.DataSourceType{plugin.DataSourceCommand, plugin.DataSourceLog}
}

// RedisMetricParser 实现
type RedisMetricParser struct {
	plugin *RedisPlugin
}

func (p *RedisMetricParser) Parse(ctx context.Context, data *plugin.CollectedData) (*plugin.ParsedMetrics, error) {
	infoStr, ok := data.RawData["info"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid info data")
	}

	metrics := make(map[string]*plugin.MetricValue)

	// 示例：提取used_memory
	if matches := regexp.MustCompile(`used_memory:(\d+)`).FindStringSubmatch(infoStr); len(matches) > 1 {
		usedMem, _ := strconv.ParseInt(matches[1], 10, 64)
		metrics["memory_used_bytes"] = &plugin.MetricValue{
			Name:  "memory_used_bytes",
			Value: usedMem,
			Unit:  "bytes",
		}
	}

	// 解析更多指标
	if matches := regexp.MustCompile(`connected_clients:(\d+)`).FindStringSubmatch(infoStr); len(matches) > 1 {
		val, _ := strconv.ParseInt(matches[1], 10, 64)
		metrics["connected_clients"] = &plugin.MetricValue{
			Name:  "connected_clients",
			Value: val,
			Unit:  "count",
		}
	}

	// keyspace_hits
	var hits, misses int64
	if matches := regexp.MustCompile(`keyspace_hits:(\d+)`).FindStringSubmatch(infoStr); len(matches) > 1 {
		hits, _ = strconv.ParseInt(matches[1], 10, 64)
		metrics["keyspace_hits"] = &plugin.MetricValue{
			Name:  "keyspace_hits",
			Value: hits,
			Unit:  "count",
		}
	}

	if matches := regexp.MustCompile(`keyspace_misses:(\d+)`).FindStringSubmatch(infoStr); len(matches) > 1 {
		misses, _ = strconv.ParseInt(matches[1], 10, 64)
		metrics["keyspace_misses"] = &plugin.MetricValue{
			Name:  "keyspace_misses",
			Value: misses,
			Unit:  "count",
		}
	}

	if hits+misses > 0 {
		hitRate := float64(hits) / float64(hits+misses)
		metrics["hit_rate"] = &plugin.MetricValue{Name: "hit_rate", Value: hitRate, Unit: "ratio"}
	}

	return &plugin.ParsedMetrics{
		PluginName: "redis",
		Timestamp:  time.Now(),
		Metrics:    metrics,
	}, nil
}

func (p *RedisMetricParser) AvailableMetrics() []plugin.MetricDefinition {
	return []plugin.MetricDefinition{
		{Name: "memory_used_bytes", Unit: "bytes", Description: "Used memory"},
		{Name: "connected_clients", Unit: "count", Description: "Number of connected clients"},
		{Name: "hit_rate", Unit: "ratio", Description: "Keyspace hit rate"},
	}
}

// RedisHealthChecker 实现
type RedisHealthChecker struct {
	plugin *RedisPlugin
}

func (c *RedisHealthChecker) Check(ctx context.Context, target *plugin.Target) (*plugin.HealthStatus, error) {
	// 1. PING检查
	pingResult := &plugin.HealthCheckResult{Name: "ping"}
	if err := c.plugin.client.Ping(ctx).Err(); err != nil {
		pingResult.Status = plugin.UnhealthyLevel
		pingResult.Message = "PING failed: " + err.Error()
	} else {
		pingResult.Status = plugin.HealthyLevel
	}

	// 2. 主从延迟检查（若配置了主从）
	replResult := c.checkReplication(ctx)

	// 3. 内存使用率检查
	memResult := c.checkMemory(ctx)

	items := []*plugin.HealthCheckResult{pingResult, replResult, memResult}
	overall := c.calculateOverallHealth(items)

	return &plugin.HealthStatus{
		PluginName: "redis",
		Overall:    overall,
		Items:      items,
		Timestamp:  time.Now(),
	}, nil
}

func (c *RedisHealthChecker) checkReplication(ctx context.Context) *plugin.HealthCheckResult {
	info, err := c.plugin.client.Info(ctx, "replication").Result()
	if err != nil {
		return &plugin.HealthCheckResult{
			Name:    "replication",
			Status:  plugin.UnhealthyLevel,
			Message: "Failed to get replication info: " + err.Error(),
		}
	}

	result := &plugin.HealthCheckResult{Name: "replication", Status: plugin.HealthyLevel}

	role := ""
	if matches := regexp.MustCompile(`role:(\w+)`).FindStringSubmatch(info); len(matches) > 1 {
		role = matches[1]
	}

	if role == "slave" {
		linkStatus := ""
		if matches := regexp.MustCompile(`master_link_status:(\w+)`).FindStringSubmatch(info); len(matches) > 1 {
			linkStatus = matches[1]
		}

		if linkStatus != "up" {
			result.Status = plugin.UnhealthyLevel
			result.Message = fmt.Sprintf("Master link status is %s", linkStatus)
			return result
		}

		lastIO := -1
		if matches := regexp.MustCompile(`master_last_io_seconds_ago:(\d+)`).FindStringSubmatch(info); len(matches) > 1 {
			lastIO, _ = strconv.Atoi(matches[1])
		}

		if lastIO > 10 {
			result.Status = plugin.DegradedLevel
			result.Message = fmt.Sprintf("High replication lag: %d seconds", lastIO)
		}
	}

	return result
}

func (c *RedisHealthChecker) checkMemory(ctx context.Context) *plugin.HealthCheckResult {
	info, err := c.plugin.client.Info(ctx, "memory").Result()
	if err != nil {
		return &plugin.HealthCheckResult{Name: "memory", Status: plugin.UnhealthyLevel, Message: err.Error()}
	}

	var used, max int64
	if matches := regexp.MustCompile(`used_memory:(\d+)`).FindStringSubmatch(info); len(matches) > 1 {
		used, _ = strconv.ParseInt(matches[1], 10, 64)
	}
	if matches := regexp.MustCompile(`maxmemory:(\d+)`).FindStringSubmatch(info); len(matches) > 1 {
		max, _ = strconv.ParseInt(matches[1], 10, 64)
	}

	if max > 0 {
		usage := float64(used) / float64(max)
		if usage > 0.95 {
			return &plugin.HealthCheckResult{Name: "memory", Status: plugin.UnhealthyLevel, Message: fmt.Sprintf("Memory usage critical: %.2f%%", usage*100)}
		} else if usage > 0.90 {
			return &plugin.HealthCheckResult{Name: "memory", Status: plugin.DegradedLevel, Message: fmt.Sprintf("Memory usage high: %.2f%%", usage*100)}
		}
	}

	return &plugin.HealthCheckResult{Name: "memory", Status: plugin.HealthyLevel}
}

func (c *RedisHealthChecker) calculateOverallHealth(items []*plugin.HealthCheckResult) plugin.HealthLevel {
	overall := plugin.HealthyLevel
	for _, item := range items {
		if item.Status > overall {
			overall = item.Status
		}
	}
	return overall
}

func (c *RedisHealthChecker) CheckItems() []plugin.HealthCheckItem {
	return []plugin.HealthCheckItem{
		{Name: "ping", Description: "Redis responsiveness"},
		{Name: "replication", Description: "Master-slave sync status"},
		{Name: "memory", Description: "Memory usage"},
	}
}

func init() {
	// Register the plugin factory
	plugin.RegisterPlugin(&RedisPluginFactory{})
}

type RedisPluginFactory struct{}

func (f *RedisPluginFactory) Create() plugin.Plugin {
	return &RedisPlugin{}
}

func (f *RedisPluginFactory) Metadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:       "redis",
		Version:    "1.0.0",
		APIVersion: "v1",
		Description: "Redis plugin",
	}
}
