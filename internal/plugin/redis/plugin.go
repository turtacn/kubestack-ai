package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// RedisPlugin implementation
type RedisPlugin struct {
	client    *redis.Client
	config    *plugin.ConnectionConfig
	connected bool
	collector *MetricsCollector
	executor  *CommandExecutor
}

// NewRedisPlugin creates a new Redis plugin
func NewRedisPlugin(cfg *plugin.PluginConfig) (plugin.MiddlewarePlugin, error) {
	return &RedisPlugin{
		collector: NewMetricsCollector(),
		executor:  NewCommandExecutor(),
	}, nil
}

// === Basic Information ===

func (p *RedisPlugin) Name() string { return "Redis Plugin" }
func (p *RedisPlugin) Type() plugin.MiddlewareType { return plugin.MiddlewareRedis }
func (p *RedisPlugin) Version() string { return "1.0.0" }

// === Connection Management ===

func (p *RedisPlugin) Connect(ctx context.Context, config *plugin.ConnectionConfig) error {
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:     config.Password,
		DB:           0,
		DialTimeout:  config.Timeout,
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
		PoolSize:     config.PoolSize,
	}

	// TLS
	if config.TLS != nil {
		opts.TLSConfig = config.TLS.ToTLSConfig()
	}

	p.client = redis.NewClient(opts)

	// Verify connection
	if err := p.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	p.config = config
	p.connected = true
	p.collector.SetClient(p.client)
	p.executor.SetClient(p.client)

	return nil
}

func (p *RedisPlugin) Disconnect(ctx context.Context) error {
	if p.client != nil {
		p.connected = false
		return p.client.Close()
	}
	return nil
}

func (p *RedisPlugin) Ping(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("not connected")
	}
	return p.client.Ping(ctx).Err()
}

func (p *RedisPlugin) IsConnected() bool {
	return p.connected
}

// === Metrics Collection ===

func (p *RedisPlugin) CollectMetrics(ctx context.Context) (*plugin.MetricsSnapshot, error) {
	return p.collector.Collect(ctx)
}

func (p *RedisPlugin) CollectSpecificMetric(ctx context.Context, metricName string) (interface{}, error) {
	return p.collector.CollectSpecific(ctx, metricName)
}

// === Command Execution ===

func (p *RedisPlugin) Execute(ctx context.Context, cmd *plugin.Command) (*plugin.CommandResult, error) {
	return p.executor.Execute(ctx, cmd)
}

func (p *RedisPlugin) SupportedCommands() []plugin.CommandSpec {
	return []plugin.CommandSpec{
		{Name: "INFO", Description: "Get server info", RiskLevel: 1},
		{Name: "CONFIG GET", Description: "Get config", RiskLevel: 1},
		{Name: "CONFIG SET", Description: "Set config", RiskLevel: 3},
		{Name: "SLOWLOG", Description: "Slowlog operations", RiskLevel: 1},
		{Name: "CLIENT LIST", Description: "List clients", RiskLevel: 1},
		{Name: "CLIENT KILL", Description: "Kill client", RiskLevel: 3},
		{Name: "MEMORY DOCTOR", Description: "Memory doctor", RiskLevel: 1},
		{Name: "MEMORY PURGE", Description: "Memory purge", RiskLevel: 2},
		{Name: "DEBUG SLEEP", Description: "Debug sleep", RiskLevel: 4},
		{Name: "FLUSHDB", Description: "Flush DB", RiskLevel: 5},
		{Name: "FLUSHALL", Description: "Flush All", RiskLevel: 5},
	}
}

// === Diagnosis Support ===

func (p *RedisPlugin) GetDiagnosticData(ctx context.Context) (*plugin.DiagnosticData, error) {
	data := &plugin.DiagnosticData{
		Extra: make(map[string]interface{}),
	}

	// 1. Collect metrics
	metrics, err := p.CollectMetrics(ctx)
	if err != nil {
		return nil, err
	}
	data.Metrics = metrics

	// 2. Get config
	configResult := p.client.ConfigGet(ctx, "*")
	if configResult.Err() == nil {
		// Convert map[string]string to map[string]interface{}
		configMap := make(map[string]interface{})
		for k, v := range configResult.Val() {
			configMap[k] = v
		}
		data.Config = configMap
	}

	// 3. Get slow logs
	slowLogs, _ := p.client.SlowLogGet(ctx, 100).Result()
	data.SlowLogs = p.convertSlowLogs(slowLogs)

	// 4. Get connection info
	clientList, _ := p.client.ClientList(ctx).Result()
	data.Connections = p.parseClientList(clientList)

	// 5. Get replication info
	infoRepl, _ := p.client.Info(ctx, "replication").Result()
	data.Replication = p.parseReplicationInfo(infoRepl)

	// 6. Extra data
	// MemoryDoctor is not available in go-redis v9 directly sometimes depending on version or command support
	// We can use Do command
	res, _ := p.client.Do(ctx, "MEMORY", "DOCTOR").Result()
	data.Extra["memory_doctor"] = res

	return data, nil
}

func (p *RedisPlugin) GetBuiltinRules() []plugin.DiagnosisRule {
	return redisBuiltinRules
}

// Helper methods

func (p *RedisPlugin) convertSlowLogs(logs []redis.SlowLog) []plugin.SlowLogEntry {
	entries := make([]plugin.SlowLogEntry, len(logs))
	for i, log := range logs {
		entries[i] = plugin.SlowLogEntry{
			ID:       fmt.Sprintf("%d", log.ID),
			Time:     log.Time,
			Duration: log.Duration,
			Command:  fmt.Sprintf("%v", log.Args),
			ClientIP: log.ClientAddr,
		}
	}
	return entries
}

func (p *RedisPlugin) parseClientList(clientList string) []plugin.ConnectionInfo {
	// Simple mock parsing or real parsing could be implemented
	// For now we return empty or simple parse
	return []plugin.ConnectionInfo{}
}

func (p *RedisPlugin) parseReplicationInfo(info string) *plugin.ReplicationInfo {
	return &plugin.ReplicationInfo{
		Details: map[string]interface{}{"raw": info},
	}
}

// Built-in rules
var redisBuiltinRules = []plugin.DiagnosisRule{
	{
		ID:          "redis-memory-high",
		Name:        "Memory Usage High",
		Severity:    plugin.SeverityWarning,
		Condition:   "metrics.used_memory_rss / metrics.maxmemory > 0.8",
		Message:     "Redis memory usage reached {{.usage}}%",
		Suggestion:  "Check for big keys or increase memory",
	},
}
