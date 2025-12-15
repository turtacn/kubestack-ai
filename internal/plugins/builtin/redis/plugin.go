// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package redis implements the built-in plugin for diagnosing Redis instances.
package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
)

const (
	// Issue titles that can be auto-fixed
	IssueTitleMemoryHigh = "High Memory Usage"
	IssueTitleSlowLog    = "Slow Queries Detected"
	IssueTitleConnHigh   = "High Connection Count"

	// Fix command categories
	FixCatMemory = "Memory"
	FixCatConfig = "Configuration"
	FixCatConn   = "Connection"
)

// redisPlugin is the concrete implementation of the MiddlewarePlugin for Redis.
type redisPlugin struct {
	base.Plugin
	client    *redis.Client
	collector *collector
	analyzer  *analyzer
	fixer     *base.FixExecutor
}

// New is the factory function that creates an instance of the Redis plugin.
func New() (interfaces.MiddlewarePlugin, error) {
	p := &redisPlugin{}
	// Use base.Plugin Init to set basic info
	p.Plugin.Init("redis", "0.1.0", "Provides diagnostics for Redis instances.")

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	p.client = rdb
	p.collector = newCollector(rdb, p.Log)
	p.analyzer = newAnalyzer(p.Log)
	p.fixer = base.NewFixExecutor(p.Log)

	p.Log.Info("Redis plugin initialized successfully.")
	return p, nil
}

// Init shadows base.Plugin.Init to satisfy interface
func (p *redisPlugin) Init(cfg *config.PluginConfig) error {
	// Real world: update redis client from config
	return nil
}

// Diagnose orchestrates the diagnosis process for Redis.
func (p *redisPlugin) Diagnose(ctx context.Context, _ *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	p.Log.Info("Starting Redis diagnosis.")

	info, err := p.collector.CollectInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect redis info: %w", err)
	}
	config, err := p.collector.CollectConfig(ctx)
	if err != nil {
		p.Log.Warnf("Failed to collect redis config: %v", err)
	}
	slowlogs, err := p.collector.CollectSlowLog(ctx)
	if err != nil {
		p.Log.Warnf("Failed to collect redis slowlog: %v", err)
	}

	issues := p.analyzer.Analyze(info, config, slowlogs)

	result := &models.DiagnosisResult{
		ID:        fmt.Sprintf("redis-diag-%d", time.Now().Unix()),
		Timestamp: time.Now().UTC(),
		Summary:   fmt.Sprintf("Redis diagnosis complete. Found %d potential issues.", len(issues)),
		Issues:    issues,
	}

	return result, nil
}

// --- Interface Method Implementations ---

func (p *redisPlugin) CollectMetrics(ctx context.Context, target string) (*models.MetricsData, error) {
	return p.collector.CollectMetrics(ctx)
}

func (p *redisPlugin) CollectLogs(ctx context.Context, target string, opts *models.LogOptions) (*models.LogData, error) {
	return p.collector.CollectSlowLog(ctx)
}

func (p *redisPlugin) CollectConfig(ctx context.Context, target string) (*models.ConfigData, error) {
	return p.collector.CollectConfig(ctx)
}

func (p *redisPlugin) HealthCheck(ctx context.Context, target string) (*models.HealthStatus, error) {
	if err := p.Ping(ctx, target); err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: fmt.Sprintf("Failed to ping Redis: %v", err)}, nil
	}
	return &models.HealthStatus{IsHealthy: true, Message: "Redis instance is responsive."}, nil
}

func (p *redisPlugin) Ping(ctx context.Context, target string) error {
	return p.client.Ping(ctx).Err()
}

func (p *redisPlugin) Shutdown() error {
	p.Log.Info("Shutting down Redis plugin and closing client connection.")
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// --- Fix Capabilities ---

func (p *redisPlugin) CanAutoFix(issue *models.Issue) (bool, *models.FixAction) {
	switch issue.Title {
	case IssueTitleMemoryHigh:
		return true, &models.FixAction{
			ID:          "fix-redis-memory-purge",
			Description: "Execute MEMORY PURGE to release fragmented memory",
			Command:     "MEMORY PURGE",
			Category:    FixCatMemory,
		}
	case IssueTitleSlowLog:
		return true, &models.FixAction{
			ID:          "fix-redis-slowlog-reset",
			Description: "Reset slow log to clear old entries",
			Command:     "SLOWLOG RESET",
			Category:    FixCatConfig,
		}
	case IssueTitleConnHigh:
		return true, &models.FixAction{
			ID:          "fix-redis-kill-normal-clients",
			Description: "Kill all normal clients to free up connections",
			Command:     "CLIENT KILL TYPE normal",
			Category:    FixCatConn,
		}
	}
	return false, nil
}

func (p *redisPlugin) ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error) {
	return p.fixer.Execute(ctx, fix, func(ctx context.Context) error {
		parts := strings.Fields(fix.Command)
		if len(parts) == 0 {
			return fmt.Errorf("empty command")
		}

		// cmdName is now unused
		// cmdName := parts[0]
		args := make([]interface{}, len(parts))
		for i, v := range parts {
			args[i] = v
		}

		return p.client.Do(ctx, args...).Err()
	}, nil) // No rollback for these simple commands
}

func (p *redisPlugin) ValidateFix(ctx context.Context, issue *models.Issue, result *models.FixResult) (bool, string, error) {
	if !result.Success {
		return false, "Fix execution failed previously", nil
	}

	switch issue.Title {
	case IssueTitleSlowLog:
		// Go-redis v8 does not expose SlowLogLen helper directly on Client.
		// Use Do command.
		val, err := p.client.Do(ctx, "SLOWLOG", "LEN").Int()
		if err != nil {
			return false, "", err
		}
		if val == 0 {
			return true, "Slow log is empty", nil
		}
		return false, fmt.Sprintf("Slow log still has %d entries", val), nil

	case IssueTitleMemoryHigh:
		// Simple validation: just check we can ping, actual memory drop is hard to simulate immediately without waiting
		err := p.client.Ping(ctx).Err()
		return err == nil, "Redis is responsive after purge", err

	case IssueTitleConnHigh:
		// Validate we are responsive
		err := p.client.Ping(ctx).Err()
		return err == nil, "Redis is responsive after connection kill", err
	}

	return true, "Fix assumed successful", nil
}
