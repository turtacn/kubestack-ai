package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// RedisEnhancedPlugin implements the enhanced middleware plugin interface for Redis
type RedisEnhancedPlugin struct {
	client    redis.UniversalClient
	target    plugin.MiddlewareTarget
	info      plugin.EnhancedPluginInfo
	config    plugin.PluginConfig
	connected bool
	mu        sync.RWMutex
}

// NewRedisPlugin creates a new Redis plugin instance
func NewRedisPlugin() plugin.Plugin {
	return &RedisEnhancedPlugin{
		info: plugin.EnhancedPluginInfo{
			ID:          "redis-diagnostics",
			Name:        "Redis Diagnostics Plugin",
			Version:     "1.0.0",
			Type:        plugin.PluginTypeMiddleware,
			Description: "Comprehensive Redis diagnostics and monitoring",
			Author:      "KubeStack AI",
			Homepage:    "https://github.com/kubestack-ai/kubestack-ai",
			License:     "Apache-2.0",
			Requires:    []string{},
			Capabilities: []string{
				"health-check",
				"metrics",
				"diagnose",
				"slow-logs",
				"client-list",
				"config",
			},
		},
	}
}

// Info returns plugin metadata
func (p *RedisEnhancedPlugin) Info() plugin.EnhancedPluginInfo {
	return p.info
}

// Init initializes the plugin with configuration
func (p *RedisEnhancedPlugin) Init(ctx context.Context, config plugin.PluginConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.config = config
	return nil
}

// Start starts the plugin
func (p *RedisEnhancedPlugin) Start(ctx context.Context) error {
	// No background processes to start
	return nil
}

// Stop stops the plugin gracefully
func (p *RedisEnhancedPlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.connected && p.client != nil {
		return p.client.Close()
	}
	
	return nil
}

// HealthCheck performs a health check on the plugin
func (p *RedisEnhancedPlugin) HealthCheck(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if !p.connected || p.client == nil {
		return fmt.Errorf("redis client not connected")
	}
	
	return p.client.Ping(ctx).Err()
}

// MiddlewareType returns the type of middleware
func (p *RedisEnhancedPlugin) MiddlewareType() string {
	return "redis"
}

// SupportedVersions returns the list of supported Redis versions
func (p *RedisEnhancedPlugin) SupportedVersions() []string {
	return []string{"5.x", "6.x", "7.x"}
}

// Connect establishes a connection to the Redis instance
func (p *RedisEnhancedPlugin) Connect(ctx context.Context, target plugin.MiddlewareTarget) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.target = target
	
	// Determine Redis mode from options
	mode := target.Options["mode"]
	if mode == "" {
		mode = "standalone"
	}
	
	var client redis.UniversalClient
	
	switch mode {
	case "cluster":
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    target.Endpoints,
			Password: p.getPassword(target.Auth),
		})
	case "sentinel":
		// For sentinel, we need master name
		masterName := target.Options["master_name"]
		if masterName == "" {
			masterName = "mymaster"
		}
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    masterName,
			SentinelAddrs: target.Endpoints,
			Password:      p.getPassword(target.Auth),
		})
	default: // standalone
		addr := target.Endpoints[0]
		if len(target.Endpoints) == 0 {
			return fmt.Errorf("no endpoints specified")
		}
		client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: p.getPassword(target.Auth),
			DB:       0,
		})
	}
	
	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	p.client = client
	p.connected = true
	
	return nil
}

// Disconnect closes the connection
func (p *RedisEnhancedPlugin) Disconnect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.client != nil {
		err := p.client.Close()
		p.client = nil
		p.connected = false
		return err
	}
	
	return nil
}

// Diagnose performs diagnostic checks
func (p *RedisEnhancedPlugin) Diagnose(ctx context.Context, opts plugin.DiagnoseOptions) (*plugin.DiagnosticResult, error) {
	p.mu.RLock()
	if !p.connected {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected to Redis")
	}
	p.mu.RUnlock()
	
	startTime := time.Now()
	result := &plugin.DiagnosticResult{
		PluginID:    p.info.ID,
		TargetName:  p.target.Name,
		Status:      plugin.DiagnosticStatusHealthy,
		Findings:    []plugin.Finding{},
		Metrics:     make(map[string]interface{}),
		Suggestions: []string{},
		Timestamp:   startTime,
	}
	
	// Determine which categories to diagnose
	categories := opts.Categories
	if len(categories) == 0 {
		// Default: all categories
		categories = []string{"memory", "connection", "persistence", "replication", "performance"}
	}
	
	// Run diagnostics for each category
	for _, category := range categories {
		switch category {
		case "memory":
			if err := p.diagnoseMemory(ctx, result); err != nil {
				return nil, fmt.Errorf("memory diagnosis failed: %w", err)
			}
		case "connection":
			if err := p.diagnoseConnections(ctx, result); err != nil {
				return nil, fmt.Errorf("connection diagnosis failed: %w", err)
			}
		case "persistence":
			if err := p.diagnosePersistence(ctx, result); err != nil {
				return nil, fmt.Errorf("persistence diagnosis failed: %w", err)
			}
		case "replication":
			if err := p.diagnoseReplication(ctx, result); err != nil {
				return nil, fmt.Errorf("replication diagnosis failed: %w", err)
			}
		case "performance":
			if err := p.diagnosePerformance(ctx, result); err != nil {
				return nil, fmt.Errorf("performance diagnosis failed: %w", err)
			}
		}
	}
	
	result.Duration = time.Since(startTime)
	
	// Determine overall status based on findings
	result.Status = p.determineOverallStatus(result.Findings)
	
	return result, nil
}

// GetMetrics retrieves current metrics
func (p *RedisEnhancedPlugin) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if !p.connected {
		return nil, fmt.Errorf("not connected to Redis")
	}
	
	info, err := p.client.Info(ctx, "all").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}
	
	metrics := p.parseInfo(info)
	return metrics, nil
}

// Execute performs an action on the middleware
func (p *RedisEnhancedPlugin) Execute(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if !p.connected {
		return nil, fmt.Errorf("not connected to Redis")
	}
	
	switch action {
	case "get-slow-logs":
		count := int64(10)
		if c, ok := params["count"].(int64); ok {
			count = c
		}
		return p.client.SlowLogGet(ctx, count).Result()
		
	case "get-client-list":
		return p.client.ClientList(ctx).Result()
		
	case "get-config":
		pattern := "*"
		if p, ok := params["pattern"].(string); ok {
			pattern = p
		}
		return p.client.ConfigGet(ctx, pattern).Result()
		
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// Helper methods

func (p *RedisEnhancedPlugin) getPassword(auth *plugin.AuthConfig) string {
	if auth == nil {
		return ""
	}
	if auth.Password != "" {
		return auth.Password
	}
	if auth.Token != "" {
		return auth.Token
	}
	return ""
}

func (p *RedisEnhancedPlugin) parseInfo(info string) map[string]interface{} {
	metrics := make(map[string]interface{})
	
	lines := splitLines(info)
	for _, line := range lines {
		if line == "" || line[0] == '#' {
			continue
		}
		
		parts := splitKeyValue(line)
		if len(parts) == 2 {
			metrics[parts[0]] = parts[1]
		}
	}
	
	return metrics
}

func (p *RedisEnhancedPlugin) determineOverallStatus(findings []plugin.Finding) plugin.DiagnosticStatus {
	hasCritical := false
	hasWarning := false
	
	for _, finding := range findings {
		switch finding.Severity {
		case plugin.SeverityCritical:
			hasCritical = true
		case plugin.SeverityWarning, plugin.SeverityError:
			hasWarning = true
		}
	}
	
	if hasCritical {
		return plugin.DiagnosticStatusCritical
	}
	if hasWarning {
		return plugin.DiagnosticStatusWarning
	}
	
	return plugin.DiagnosticStatusHealthy
}

// Utility functions

func splitLines(s string) []string {
	lines := make([]string, 0)
	current := ""
	for _, c := range s {
		if c == '\n' || c == '\r' {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func splitKeyValue(s string) []string {
	idx := -1
	for i, c := range s {
		if c == ':' {
			idx = i
			break
		}
	}
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}
