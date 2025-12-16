//go:build redis_legacy
// +build redis_legacy

package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// MetricsCollector Redis metrics collector
type MetricsCollector struct {
	client *redis.Client
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

func (c *MetricsCollector) SetClient(client *redis.Client) {
	c.client = client
}

// Collect collects all metrics
func (c *MetricsCollector) Collect(ctx context.Context) (*plugin.MetricsSnapshot, error) {
	snapshot := &plugin.MetricsSnapshot{
		Timestamp: time.Now(),
		Metrics:   make(map[string]plugin.MetricValue),
		RawData:   make(map[string]interface{}),
	}

	// 1. INFO ALL
	infoResult, err := c.client.Info(ctx, "all").Result()
	if err != nil {
		return nil, err
	}

	// 2. Parse INFO
	infoMap := c.parseInfo(infoResult)
	snapshot.RawData["info"] = infoMap

	// 3. Extract key metrics
	keyMetrics := []struct {
		name  string
		field string
		unit  string
	}{
		{"used_memory", "used_memory", "bytes"},
		{"used_memory_rss", "used_memory_rss", "bytes"},
		{"maxmemory", "maxmemory", "bytes"},
		{"mem_fragmentation_ratio", "mem_fragmentation_ratio", "ratio"},
		{"connected_clients", "connected_clients", "count"},
		{"blocked_clients", "blocked_clients", "count"},
		{"keyspace_hits", "keyspace_hits", "count"},
		{"keyspace_misses", "keyspace_misses", "count"},
		{"evicted_keys", "evicted_keys", "count"},
		{"expired_keys", "expired_keys", "count"},
	}

	for _, m := range keyMetrics {
		if val, ok := infoMap[m.field]; ok {
			floatVal, _ := strconv.ParseFloat(val, 64)
			snapshot.Metrics[m.name] = plugin.MetricValue{
				Name:      m.name,
				Value:     floatVal,
				Unit:      m.unit,
				Timestamp: snapshot.Timestamp,
			}
		}
	}

	// 4. Calculate derived metrics
	c.calculateDerivedMetrics(snapshot)

	return snapshot, nil
}

// CollectSpecific collects specific metric
func (c *MetricsCollector) CollectSpecific(ctx context.Context, metricName string) (interface{}, error) {
	switch metricName {
	case "memory":
		return c.client.Info(ctx, "memory").Result()
	default:
		return nil, fmt.Errorf("metric not found: %s", metricName)
	}
}

// parseInfo parses INFO output
func (c *MetricsCollector) parseInfo(info string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(info, "
")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return result
}

// calculateDerivedMetrics calculates derived metrics
func (c *MetricsCollector) calculateDerivedMetrics(snapshot *plugin.MetricsSnapshot) {
	// Memory usage ratio
	if usedMem, ok := snapshot.Metrics["used_memory"]; ok {
		if maxMem, ok := snapshot.Metrics["maxmemory"]; ok && maxMem.Value > 0 {
			snapshot.Metrics["memory_usage_ratio"] = plugin.MetricValue{
				Name:      "memory_usage_ratio",
				Value:     usedMem.Value / maxMem.Value,
				Unit:      "ratio",
				Timestamp: snapshot.Timestamp,
			}
		}
	}

	// Hit rate
	if hits, ok := snapshot.Metrics["keyspace_hits"]; ok {
		if misses, ok := snapshot.Metrics["keyspace_misses"]; ok {
			total := hits.Value + misses.Value
			if total > 0 {
				snapshot.Metrics["hit_rate"] = plugin.MetricValue{
					Name:      "hit_rate",
					Value:     hits.Value / total,
					Unit:      "ratio",
					Timestamp: snapshot.Timestamp,
				}
			}
		}
	}
}
