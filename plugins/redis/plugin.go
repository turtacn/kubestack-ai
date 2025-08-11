package redis

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/turtacn/kubestack-ai/internal/errors"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/models"
	"github.com/turtacn/kubestack-ai/internal/plugins"
)

// RedisPlugin Redis插件实现。RedisPlugin implements Plugin for Redis.
type RedisPlugin struct {
	client      *redis.Client
	config      plugins.PluginConfig
	version     string
	initialized bool
}

// NewRedisPlugin 创建Redis插件实例。NewRedisPlugin creates a new Redis plugin instance.
func NewRedisPlugin() *RedisPlugin {
	return &RedisPlugin{}
}

// Name 返回插件名称。Name returns plugin name.
func (p *RedisPlugin) Name() string {
	return "redis"
}

// Version 返回插件版本。Version returns plugin version.
func (p *RedisPlugin) Version() string {
	return "1.0.0"
}

// SupportedVersions 返回支持的Redis版本。SupportedVersions returns supported Redis versions.
func (p *RedisPlugin) SupportedVersions() []string {
	return []string{"6.x", "7.x", "8.x"}
}

// Initialize 初始化插件。Initialize initializes the plugin.
func (p *RedisPlugin) Initialize(config plugins.PluginConfig) error {
	logging.Logger.Info("Initializing Redis plugin")

	// 存储配置。Store configuration.
	p.config = config

	// 解析配置参数。Parse configuration parameters.
	addr := "localhost:6379"
	if a, ok := config["address"].(string); ok {
		addr = a
	}

	password := ""
	if pwd, ok := config["password"].(string); ok {
		password = pwd
	}

	db := 0
	if d, ok := config["db"].(int); ok {
		db = d
	}

	// 创建Redis客户端。Create Redis client.
	p.client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// 测试连接。Test connection.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := p.client.Ping(ctx).Result()
	if err != nil {
		logging.Logger.Errorf("Failed to connect to Redis: %v", err)
		return errors.ErrInvalidConfig
	}

	// 获取Redis版本。Get Redis version.
	info, err := p.client.Info(ctx, "server").Result()
	if err != nil {
		logging.Logger.Warnf("Failed to get Redis info: %v", err)
		p.version = "unknown"
	} else {
		for _, line := range strings.Split(info, "\r\n") {
			if strings.HasPrefix(line, "redis_version:") {
				p.version = strings.TrimPrefix(line, "redis_version:")
				break
			}
		}
	}

	p.initialized = true
	logging.Logger.Infof("Redis plugin initialized. Connected to Redis version: %s", p.version)
	return nil
}

// Validate 验证插件配置。Validate validates plugin configuration.
func (p *RedisPlugin) Validate() error {
	if !p.initialized {
		return errors.ErrInvalidConfig
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 验证连接是否有效。Verify connection is valid.
	_, err := p.client.Ping(ctx).Result()
	if err != nil {
		return errors.ErrInvalidConfig
	}

	return nil
}

// Cleanup 清理资源。Cleanup releases resources.
func (p *RedisPlugin) Cleanup() error {
	if p.client != nil {
		logging.Logger.Info("Cleaning up Redis plugin resources")
		return p.client.Close()
	}
	return nil
}

// CollectMetrics 采集Redis指标。CollectMetrics collects Redis metrics.
func (p *RedisPlugin) CollectMetrics() (models.Metrics, error) {
	if !p.initialized {
		return nil, errors.ErrInvalidConfig
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics := make(models.Metrics)

	// 获取基本统计信息。Get basic stats.
	infoStats, err := p.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}
	p.parseStats(infoStats, metrics)

	// 获取内存信息。Get memory info.
	infoMemory, err := p.client.Info(ctx, "memory").Result()
	if err != nil {
		return nil, err
	}
	p.parseMemory(infoMemory, metrics)

	// 获取持久化信息。Get persistence info.
	infoPersistence, err := p.client.Info(ctx, "persistence").Result()
	if err != nil {
		return nil, err
	}
	p.parsePersistence(infoPersistence, metrics)

	return metrics, nil
}

// AnalyzeLogs 分析Redis日志。AnalyzeLogs analyzes Redis logs.
func (p *RedisPlugin) AnalyzeLogs() (models.Logs, error) {
	if !p.initialized {
		return nil, errors.ErrInvalidConfig
	}

	// 在实际实现中，这里会从日志文件或日志系统获取并分析日志
	// In a real implementation, this would fetch and analyze logs from log files or logging systems
	logs := models.Logs{
		"Log analysis not implemented in this version",
	}

	return logs, nil
}

// CollectConfig 收集Redis配置。CollectConfig collects Redis configuration.
func (p *RedisPlugin) CollectConfig() (models.Config, error) {
	if !p.initialized {
		return nil, errors.ErrInvalidConfig
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := make(models.Config)

	// 获取配置参数。Get configuration parameters.
	configInfo, err := p.client.ConfigGet(ctx, "*").Result()
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(configInfo); i += 2 {
		key := configInfo[i]
		value := configInfo[i+1]
		config[key] = value
	}

	return config, nil
}

// Diagnose 执行Redis诊断。Diagnose performs Redis diagnosis.
func (p *RedisPlugin) Diagnose() ([]models.Finding, error) {
	if !p.initialized {
		return nil, errors.ErrInvalidConfig
	}

	findings := []models.Finding{}

	// 采集指标用于诊断。Collect metrics for diagnosis.
	metrics, err := p.CollectMetrics()
	if err != nil {
		return nil, err
	}

	// 检查内存使用。Check memory usage.
	if memUsage, ok := metrics["used_memory_percent"].(float64); ok {
		if memUsage > 90 {
			findings = append(findings, models.Finding{
				Type:     "memory",
				Title:    "High Memory Usage",
				Detail:   "Redis is using more than 90% of available memory",
				Evidence: []string{strconv.FormatFloat(memUsage, 'f', 2, 64) + "% memory used"},
				Severity: "high",
				Recommendations: []models.Recommendation{
					{
						Description: "Identify and remove large keys",
						Command:     "redis-cli --bigkeys",
						AutoFix:     false,
					},
					{
						Description: "Increase memory limit or add more nodes",
						AutoFix:     false,
					},
				},
			})
		}
	}

	// 检查命中率。Check hit rate.
	if hitRate, ok := metrics["keyspace_hit_rate"].(float64); ok {
		if hitRate < 0.8 {
			findings = append(findings, models.Finding{
				Type:     "performance",
				Title:    "Low Cache Hit Rate",
				Detail:   "Redis cache hit rate is below 80%",
				Evidence: []string{strconv.FormatFloat(hitRate*100, 'f', 2, 64) + "% hit rate"},
				Severity: "medium",
				Recommendations: []models.Recommendation{
					{
						Description: "Review cache eviction policy",
						Command:     "redis-cli config get maxmemory-policy",
						AutoFix:     false,
					},
				},
			})
		}
	}

	// 检查持久化问题。Check persistence issues.
	if rdbLastErr, ok := metrics["rdb_last_bgsave_status"].(string); ok && rdbLastErr != "ok" {
		findings = append(findings, models.Finding{
			Type:     "persistence",
			Title:    "RDB Persistence Failure",
			Detail:   "Last RDB save operation failed",
			Evidence: []string{"rdb_last_bgsave_status: " + rdbLastErr},
			Severity: "high",
			Recommendations: []models.Recommendation{
				{
					Description: "Check disk space and permissions",
					Command:     "df -h && ls -la /var/lib/redis",
					AutoFix:     false,
				},
			},
		})
	}

	return findings, nil
}

// 解析统计信息。parseStats parses stats information into metrics.
func (p *RedisPlugin) parseStats(info string, metrics models.Metrics) {
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || !strings.Contains(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		key, value := parts[0], parts[1]

		switch key {
		case "total_connections_received":
			if val, err := strconv.ParseInt(value, 10, 64); err == nil {
				metrics["total_connections"] = val
			}
		case "keyspace_hits":
			if val, err := strconv.ParseInt(value, 10, 64); err == nil {
				metrics["keyspace_hits"] = val
			}
		case "keyspace_misses":
			if val, err := strconv.ParseInt(value, 10, 64); err == nil {
				metrics["keyspace_misses"] = val

				// 计算命中率。Calculate hit rate.
				if hits, ok := metrics["keyspace_hits"].(int64); ok && hits+val > 0 {
					metrics["keyspace_hit_rate"] = float64(hits) / float64(hits+val)
				}
			}
		}
	}
}

// 解析内存信息。parseMemory parses memory information into metrics.
func (p *RedisPlugin) parseMemory(info string, metrics models.Metrics) {
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || !strings.Contains(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		key, value := parts[0], parts[1]

		switch key {
		case "used_memory":
			if val, err := strconv.ParseInt(value, 10, 64); err == nil {
				metrics["used_memory"] = val
			}
		case "used_memory_human":
			metrics["used_memory_human"] = value
		case "maxmemory":
			if val, err := strconv.ParseInt(value, 10, 64); err == nil {
				metrics["max_memory"] = val

				// 计算内存使用率。Calculate memory usage percentage.
				if used, ok := metrics["used_memory"].(int64); ok && val > 0 {
					metrics["used_memory_percent"] = float64(used) / float64(val) * 100
				}
			}
		case "mem_fragmentation_ratio":
			if val, err := strconv.ParseFloat(value, 64); err == nil {
				metrics["mem_fragmentation_ratio"] = val
			}
		}
	}
}

// 解析持久化信息。parsePersistence parses persistence information into metrics.
func (p *RedisPlugin) parsePersistence(info string, metrics models.Metrics) {
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || !strings.Contains(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		key, value := parts[0], parts[1]

		switch key {
		case "rdb_last_save_time":
			if val, err := strconv.ParseInt(value, 10, 64); err == nil {
				metrics["rdb_last_save_time"] = val
				metrics["rdb_last_save_time_human"] = time.Unix(val, 0).Format(time.RFC3339)
			}
		case "rdb_last_bgsave_status":
			metrics["rdb_last_bgsave_status"] = value
		case "aof_enabled":
			if val, err := strconv.ParseBool(value); err == nil {
				metrics["aof_enabled"] = val
			}
		case "aof_last_write_status":
			metrics["aof_last_write_status"] = value
		}
	}
}

// 确保RedisPlugin实现了Plugin接口。Ensure RedisPlugin implements Plugin interface.
var _ plugins.Plugin = (*RedisPlugin)(nil)

// 新增：通过 init 函数注册插件
func init() {
	plugins.RegisterPlugin("redis", func() plugins.Plugin {
		return &RedisPlugin{}
	})
}

//Personal.AI order the ending
