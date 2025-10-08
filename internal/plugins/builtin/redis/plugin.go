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
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
)

// redisPlugin is the concrete implementation of the MiddlewarePlugin for Redis.
// It embeds base.Plugin to inherit common functionality.
type redisPlugin struct {
	base.Plugin
	client    *redis.Client
	collector *collector
	analyzer  *analyzer
}

// New is the factory function that creates an instance of the Redis plugin.
// It initializes the base plugin, establishes a connection to a Redis instance,
// and wires together the specific collector and analyzer for Redis.
//
// Returns:
//   interfaces.MiddlewarePlugin: A new, fully initialized Redis plugin.
//   error: An error if the Redis client fails to initialize (though the underlying
//          library does not return an error on creation).
func New() (interfaces.MiddlewarePlugin, error) {
	p := &redisPlugin{}
	p.Init("redis", "0.1.0", "Provides diagnostics for Redis instances.")

	// In a real-world plugin, connection info would come from a configuration system,
	// likely passed in during initialization or a 'Connect' method call.
	// For this built-in plugin, we use a placeholder configuration.
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	p.client = rdb
	p.collector = newCollector(rdb, p.Log)
	p.analyzer = newAnalyzer(p.Log)

	p.Log.Info("Redis plugin initialized successfully.")
	return p, nil
}

// Diagnose orchestrates the diagnosis process for Redis. It collects INFO,
// CONFIG, and SLOWLOG data, and then passes it to the analyzer to identify
// potential issues.
func (p *redisPlugin) Diagnose(ctx context.Context, _ *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	p.Log.Info("Starting Redis diagnosis.")

	// 1. Collect all necessary data points.
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

	// 2. Analyze the collected data.
	issues := p.analyzer.Analyze(info, config, slowlogs)

	// 3. Assemble and return the final result.
	result := &models.DiagnosisResult{
		ID:        fmt.Sprintf("redis-diag-%d", time.Now().Unix()),
		Timestamp: time.Now().UTC(),
		Summary:   fmt.Sprintf("Redis diagnosis complete. Found %d potential issues.", len(issues)),
		Issues:    issues,
	}

	return result, nil
}

// --- Interface Method Implementations ---

// CollectMetrics gathers key performance indicators for the Redis instance.
func (p *redisPlugin) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	return p.collector.CollectMetrics(ctx)
}

// GetConfiguration retrieves the live configuration from the Redis instance.
func (p *redisPlugin) GetConfiguration(ctx context.Context) (*models.ConfigData, error) {
	return p.collector.CollectConfig(ctx)
}

// CollectLogs retrieves entries from the Redis slow query log.
func (p *redisPlugin) CollectLogs(ctx context.Context, _ *models.LogOptions) (*models.LogData, error) {
	return p.collector.CollectSlowLog(ctx)
}

// --- Health Checks ---

// HealthCheck performs a basic health check by pinging the Redis instance.
func (p *redisPlugin) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	if err := p.Ping(ctx); err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: fmt.Sprintf("Failed to ping Redis: %v", err)}, nil
	}
	return &models.HealthStatus{IsHealthy: true, Message: "Redis instance is responsive."}, nil
}

// Ping sends a PING command to the Redis server to check for connectivity.
func (p *redisPlugin) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

// Shutdown gracefully closes the Redis client connection to prevent resource leaks.
func (p *redisPlugin) Shutdown() error {
	p.Log.Info("Shutting down Redis plugin and closing client connection.")
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

//Personal.AI order the ending
