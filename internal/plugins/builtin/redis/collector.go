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

package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
)

// collector is responsible for gathering raw data and metrics from a Redis instance.
type collector struct {
	client *redis.Client
	log    logger.Logger
	base   *base.CollectorBase
}

// newCollector creates a new Redis data collector.
func newCollector(client *redis.Client, log logger.Logger) *collector {
	return &collector{
		client: client,
		log:    log,
		base:   base.NewCollectorBase(log, nil), // Use default retry/timeout config
	}
}

// CollectInfo retrieves and parses the output of the Redis INFO command.
// This is a primary source of diagnostic data.
func (c *collector) CollectInfo(ctx context.Context) (map[string]string, error) {
	c.log.Info("Collecting Redis INFO ALL.")
	res, err := c.base.Retry("Redis_INFO", func() (interface{}, error) {
		return c.client.Info(ctx, "all").Result()
	})
	if err != nil {
		return nil, err
	}

	infoStr := res.(string)
	infoMap := make(map[string]string)
	lines := strings.Split(infoStr, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			infoMap[parts[0]] = parts[1]
		}
	}
	return infoMap, nil
}

// CollectConfig retrieves the Redis configuration using the CONFIG GET command.
func (c *collector) CollectConfig(ctx context.Context) (*models.ConfigData, error) {
	c.log.Info("Collecting Redis CONFIG.")
	res, err := c.base.Retry("Redis_CONFIG_GET", func() (interface{}, error) {
		return c.client.ConfigGet(ctx, "*").Result()
	})
	if err != nil {
		return nil, err
	}

	configSlice := res.([]interface{})
	configMap := make(map[string]string)
	for i := 0; i < len(configSlice); i += 2 {
		key, okKey := configSlice[i].(string)
		value, okVal := configSlice[i+1].(string)
		if okKey && okVal {
			configMap[key] = value
		}
	}
	return &models.ConfigData{Data: configMap}, nil
}

// CollectSlowLog retrieves entries from the Redis slow query log.
func (c *collector) CollectSlowLog(ctx context.Context) (*models.LogData, error) {
	c.log.Info("Collecting Redis SLOWLOG.")
	// Get up to 128 of the most recent slowlog entries.
	res, err := c.base.Retry("Redis_SLOWLOG_GET", func() (interface{}, error) {
		return c.client.SlowLogGet(ctx, 128).Result()
	})
	if err != nil {
		return nil, err
	}

	slowLogs := res.([]redis.SlowLog)
	logEntries := make([]string, len(slowLogs))
	for i, sl := range slowLogs {
		logEntries[i] = fmt.Sprintf("id=%d, timestamp=%d, duration_us=%d, command=%q",
			sl.ID, sl.Time.Unix(), sl.Duration.Microseconds(), sl.Args)
	}
	return &models.LogData{Entries: logEntries}, nil
}

// CollectMetrics derives key performance indicators from the INFO command's data.
func (c *collector) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	c.log.Info("Collecting and deriving Redis metrics.")
	info, err := c.CollectInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not collect info for metrics: %w", err)
	}

	metrics := make(map[string]interface{})
	stringToIntMetrics := []string{
		"connected_clients", "used_memory", "used_memory_rss", "mem_fragmentation_ratio",
		"total_commands_processed", "instantaneous_ops_per_sec", "keyspace_hits", "keyspace_misses",
	}

	for _, key := range stringToIntMetrics {
		if valStr, ok := info[key]; ok {
			// Some values like mem_fragmentation_ratio can be floats
			if val, err := strconv.ParseFloat(valStr, 64); err == nil {
				metrics[key] = val
			}
		}
	}

	// Calculate derived metrics like hit rate
	hits, okH := metrics["keyspace_hits"].(float64)
	misses, okM := metrics["keyspace_misses"].(float64)
	if okH && okM {
		total := hits + misses
		if total > 0 {
			metrics["keyspace_hit_rate_percent"] = (hits / total) * 100.0
		} else {
			metrics["keyspace_hit_rate_percent"] = 100.0 // No misses means 100% hit rate
		}
	}

	return &models.MetricsData{Data: metrics}, nil
}

// TODO: Implement CollectKeyspaceAnalysis. This would involve using the SCAN command to
// iterate through keys without blocking the server, which can be a slow and intensive process.

// TODO: Implement CollectClusterInfo for Redis Cluster mode, using the CLUSTER INFO and CLUSTER NODES commands.

//Personal.AI order the ending
