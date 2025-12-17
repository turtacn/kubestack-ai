package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// diagnoseMemory performs memory diagnostics
func (p *RedisEnhancedPlugin) diagnoseMemory(ctx context.Context, result *plugin.DiagnosticResult) error {
	info, err := p.client.Info(ctx, "memory").Result()
	if err != nil {
		return fmt.Errorf("failed to get memory info: %w", err)
	}
	
	metrics := p.parseInfo(info)
	
	// Extract memory metrics
	usedMemory := parseIntMetric(metrics, "used_memory")
	maxMemory := parseIntMetric(metrics, "maxmemory")
	memFragRatio := parseFloatMetric(metrics, "mem_fragmentation_ratio")
	usedMemoryRss := parseIntMetric(metrics, "used_memory_rss")
	
	// Store metrics
	result.Metrics["memory_used_bytes"] = usedMemory
	result.Metrics["memory_max_bytes"] = maxMemory
	result.Metrics["memory_fragmentation_ratio"] = memFragRatio
	result.Metrics["memory_rss_bytes"] = usedMemoryRss
	
	// Check memory usage
	if maxMemory > 0 {
		usagePercent := float64(usedMemory) / float64(maxMemory) * 100
		result.Metrics["memory_usage_percent"] = usagePercent
		
		if usagePercent > 95 {
			result.Findings = append(result.Findings, plugin.Finding{
				Severity:    plugin.SeverityCritical,
				Category:    "memory",
				Title:       "Critical Memory Usage",
				Description: fmt.Sprintf("Memory usage is %.2f%% (used: %d, max: %d)", usagePercent, usedMemory, maxMemory),
				Evidence: map[string]interface{}{
					"used_memory": usedMemory,
					"maxmemory":   maxMemory,
					"usage_percent": usagePercent,
				},
				Remediation: "Consider increasing maxmemory, enabling eviction policy, or scaling Redis instances",
			})
			result.Suggestions = append(result.Suggestions, "Increase maxmemory or enable eviction policy")
		} else if usagePercent > 80 {
			result.Findings = append(result.Findings, plugin.Finding{
				Severity:    plugin.SeverityWarning,
				Category:    "memory",
				Title:       "High Memory Usage",
				Description: fmt.Sprintf("Memory usage is %.2f%%", usagePercent),
				Evidence: map[string]interface{}{
					"usage_percent": usagePercent,
				},
				Remediation: "Monitor memory usage and plan for scaling",
			})
		}
	}
	
	// Check fragmentation ratio
	if memFragRatio < 1.0 {
		result.Findings = append(result.Findings, plugin.Finding{
			Severity:    plugin.SeverityWarning,
			Category:    "memory",
			Title:       "Memory Fragmentation Below 1",
			Description: fmt.Sprintf("Fragmentation ratio is %.2f, indicating swapping or overcommit", memFragRatio),
			Evidence: map[string]interface{}{
				"mem_fragmentation_ratio": memFragRatio,
			},
			Remediation: "Check system memory and swap usage",
		})
	} else if memFragRatio > 1.5 {
		result.Findings = append(result.Findings, plugin.Finding{
			Severity:    plugin.SeverityWarning,
			Category:    "memory",
			Title:       "High Memory Fragmentation",
			Description: fmt.Sprintf("Fragmentation ratio is %.2f", memFragRatio),
			Evidence: map[string]interface{}{
				"mem_fragmentation_ratio": memFragRatio,
			},
			Remediation: "Consider restarting Redis or using activedefrag",
		})
	}
	
	return nil
}

// diagnoseConnections performs connection diagnostics
func (p *RedisEnhancedPlugin) diagnoseConnections(ctx context.Context, result *plugin.DiagnosticResult) error {
	info, err := p.client.Info(ctx, "clients").Result()
	if err != nil {
		return fmt.Errorf("failed to get clients info: %w", err)
	}
	
	metrics := p.parseInfo(info)
	
	connectedClients := parseIntMetric(metrics, "connected_clients")
	blockedClients := parseIntMetric(metrics, "blocked_clients")
	
	result.Metrics["connected_clients"] = connectedClients
	result.Metrics["blocked_clients"] = blockedClients
	
	// Get maxclients from config
	maxClientsResult, err := p.client.ConfigGet(ctx, "maxclients").Result()
	if err == nil {
		if maxClientsStr, ok := maxClientsResult["maxclients"]; ok {
			maxClients := parseIntValue(maxClientsStr)
		if maxClients > 0 {
			usagePercent := float64(connectedClients) / float64(maxClients) * 100
			result.Metrics["connection_usage_percent"] = usagePercent
			
			if usagePercent > 80 {
				result.Findings = append(result.Findings, plugin.Finding{
					Severity:    plugin.SeverityWarning,
					Category:    "connection",
					Title:       "High Connection Usage",
					Description: fmt.Sprintf("Connection usage is %.2f%% (%d/%d)", usagePercent, connectedClients, maxClients),
					Evidence: map[string]interface{}{
						"connected_clients": connectedClients,
						"maxclients":        maxClients,
						"usage_percent":     usagePercent,
					},
					Remediation: "Consider increasing maxclients or investigating connection leaks",
				})
			}
		}
	}
                }
	
	// Check blocked clients
	if blockedClients > 0 {
		result.Findings = append(result.Findings, plugin.Finding{
			Severity:    plugin.SeverityInfo,
			Category:    "connection",
			Title:       "Blocked Clients Detected",
			Description: fmt.Sprintf("%d clients are blocked", blockedClients),
			Evidence: map[string]interface{}{
				"blocked_clients": blockedClients,
			},
			Remediation: "Investigate blocking operations (BLPOP, BRPOP, etc.)",
		})
	}
	
	return nil
}

// diagnoseReplication performs replication diagnostics
func (p *RedisEnhancedPlugin) diagnoseReplication(ctx context.Context, result *plugin.DiagnosticResult) error {
	info, err := p.client.Info(ctx, "replication").Result()
	if err != nil {
		return fmt.Errorf("failed to get replication info: %w", err)
	}
	
	metrics := p.parseInfo(info)
	
	role := getStringMetric(metrics, "role")
	result.Metrics["role"] = role
	
	if role == "master" {
		connectedSlaves := parseIntMetric(metrics, "connected_slaves")
		result.Metrics["connected_slaves"] = connectedSlaves
		
		if connectedSlaves == 0 {
			result.Findings = append(result.Findings, plugin.Finding{
				Severity:    plugin.SeverityInfo,
				Category:    "replication",
				Title:       "No Replicas Connected",
				Description: "Master has no connected replicas",
				Remediation: "Consider setting up replicas for high availability",
			})
		}
	} else if role == "slave" {
		masterLinkStatus := getStringMetric(metrics, "master_link_status")
		masterLastIO := parseIntMetric(metrics, "master_last_io_seconds_ago")
		
		result.Metrics["master_link_status"] = masterLinkStatus
		result.Metrics["master_last_io_seconds_ago"] = masterLastIO
		
		if masterLinkStatus != "up" {
			result.Findings = append(result.Findings, plugin.Finding{
				Severity:    plugin.SeverityCritical,
				Category:    "replication",
				Title:       "Master Link Down",
				Description: fmt.Sprintf("Master link status is %s", masterLinkStatus),
				Evidence: map[string]interface{}{
					"master_link_status": masterLinkStatus,
				},
				Remediation: "Check network connectivity to master and master status",
			})
		} else if masterLastIO > 10 {
			severity := plugin.SeverityWarning
			if masterLastIO > 30 {
				severity = plugin.SeverityError
			}
			result.Findings = append(result.Findings, plugin.Finding{
				Severity:    severity,
				Category:    "replication",
				Title:       "Replication Lag Detected",
				Description: fmt.Sprintf("Last IO from master was %d seconds ago", masterLastIO),
				Evidence: map[string]interface{}{
					"master_last_io_seconds_ago": masterLastIO,
				},
				Remediation: "Check network latency and master load",
			})
		}
	}
	
	return nil
}

// diagnosePersistence performs persistence diagnostics
func (p *RedisEnhancedPlugin) diagnosePersistence(ctx context.Context, result *plugin.DiagnosticResult) error {
	info, err := p.client.Info(ctx, "persistence").Result()
	if err != nil {
		return fmt.Errorf("failed to get persistence info: %w", err)
	}
	
	metrics := p.parseInfo(info)
	
	// Check RDB
	rdbLastSaveStatus := getStringMetric(metrics, "rdb_last_bgsave_status")
	rdbLastSaveTime := parseIntMetric(metrics, "rdb_last_bgsave_time_sec")
	
	result.Metrics["rdb_last_bgsave_status"] = rdbLastSaveStatus
	result.Metrics["rdb_last_bgsave_time_sec"] = rdbLastSaveTime
	
	if rdbLastSaveStatus != "ok" && rdbLastSaveStatus != "" {
		result.Findings = append(result.Findings, plugin.Finding{
			Severity:    plugin.SeverityCritical,
			Category:    "persistence",
			Title:       "RDB Save Failed",
			Description: fmt.Sprintf("Last RDB save status: %s", rdbLastSaveStatus),
			Evidence: map[string]interface{}{
				"rdb_last_bgsave_status": rdbLastSaveStatus,
			},
			Remediation: "Check disk space and permissions, review Redis logs",
		})
	}
	
	if rdbLastSaveTime > 60 {
		result.Findings = append(result.Findings, plugin.Finding{
			Severity:    plugin.SeverityWarning,
			Category:    "persistence",
			Title:       "Long RDB Save Time",
			Description: fmt.Sprintf("Last RDB save took %d seconds", rdbLastSaveTime),
			Evidence: map[string]interface{}{
				"rdb_last_bgsave_time_sec": rdbLastSaveTime,
			},
			Remediation: "Consider optimizing dataset or disk I/O",
		})
	}
	
	// Check AOF if enabled
	aofEnabled := parseIntMetric(metrics, "aof_enabled")
	if aofEnabled == 1 {
		aofLastRewriteStatus := getStringMetric(metrics, "aof_last_rewrite_status")
		result.Metrics["aof_enabled"] = true
		result.Metrics["aof_last_rewrite_status"] = aofLastRewriteStatus
		
		if aofLastRewriteStatus != "ok" && aofLastRewriteStatus != "" {
			result.Findings = append(result.Findings, plugin.Finding{
				Severity:    plugin.SeverityWarning,
				Category:    "persistence",
				Title:       "AOF Rewrite Failed",
				Description: fmt.Sprintf("Last AOF rewrite status: %s", aofLastRewriteStatus),
				Evidence: map[string]interface{}{
					"aof_last_rewrite_status": aofLastRewriteStatus,
				},
				Remediation: "Check disk space and review Redis logs",
			})
		}
	}
	
	return nil
}

// diagnosePerformance performs performance diagnostics
func (p *RedisEnhancedPlugin) diagnosePerformance(ctx context.Context, result *plugin.DiagnosticResult) error {
	// Get stats info
	info, err := p.client.Info(ctx, "stats").Result()
	if err != nil {
		return fmt.Errorf("failed to get stats info: %w", err)
	}
	
	metrics := p.parseInfo(info)
	
	instantaneousOps := parseIntMetric(metrics, "instantaneous_ops_per_sec")
	evictedKeys := parseIntMetric(metrics, "evicted_keys")
	keyspaceHits := parseIntMetric(metrics, "keyspace_hits")
	keyspaceMisses := parseIntMetric(metrics, "keyspace_misses")
	
	result.Metrics["instantaneous_ops_per_sec"] = instantaneousOps
	result.Metrics["evicted_keys"] = evictedKeys
	result.Metrics["keyspace_hits"] = keyspaceHits
	result.Metrics["keyspace_misses"] = keyspaceMisses
	
	// Calculate hit rate
	if keyspaceHits+keyspaceMisses > 0 {
		hitRate := float64(keyspaceHits) / float64(keyspaceHits+keyspaceMisses) * 100
		result.Metrics["hit_rate_percent"] = hitRate
		
		if hitRate < 90 {
			result.Findings = append(result.Findings, plugin.Finding{
				Severity:    plugin.SeverityWarning,
				Category:    "performance",
				Title:       "Low Cache Hit Rate",
				Description: fmt.Sprintf("Cache hit rate is %.2f%%", hitRate),
				Evidence: map[string]interface{}{
					"hit_rate":        hitRate,
					"keyspace_hits":   keyspaceHits,
					"keyspace_misses": keyspaceMisses,
				},
				Remediation: "Review caching strategy and key expiration policies",
			})
		}
	}
	
	// Check evicted keys
	if evictedKeys > 0 {
		result.Findings = append(result.Findings, plugin.Finding{
			Severity:    plugin.SeverityInfo,
			Category:    "performance",
			Title:       "Keys Being Evicted",
			Description: fmt.Sprintf("%d keys have been evicted", evictedKeys),
			Evidence: map[string]interface{}{
				"evicted_keys": evictedKeys,
			},
			Remediation: "Consider increasing maxmemory or adjusting eviction policy",
		})
	}
	
	// Get slow log
	slowLogs, err := p.client.SlowLogGet(ctx, 10).Result()
	if err == nil && len(slowLogs) > 0 {
		result.Metrics["recent_slow_queries"] = len(slowLogs)
		
		result.Findings = append(result.Findings, plugin.Finding{
			Severity:    plugin.SeverityWarning,
			Category:    "performance",
			Title:       "Slow Queries Detected",
			Description: fmt.Sprintf("%d slow queries in recent history", len(slowLogs)),
			Evidence: map[string]interface{}{
				"slow_query_count": len(slowLogs),
			},
			Remediation: "Review slow queries and optimize commands",
		})
	}
	
	return nil
}

// Helper functions for parsing metrics

func parseIntMetric(metrics map[string]interface{}, key string) int64 {
	if val, ok := metrics[key]; ok {
		if str, ok := val.(string); ok {
			if i, err := strconv.ParseInt(str, 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

func parseFloatMetric(metrics map[string]interface{}, key string) float64 {
	if val, ok := metrics[key]; ok {
		if str, ok := val.(string); ok {
			if f, err := strconv.ParseFloat(str, 64); err == nil {
				return f
			}
		}
	}
	return 0
}

func getStringMetric(metrics map[string]interface{}, key string) string {
	if val, ok := metrics[key]; ok {
		if str, ok := val.(string); ok {
			return strings.TrimSpace(str)
		}
	}
	return ""
}

func parseIntValue(val interface{}) int64 {
	if str, ok := val.(string); ok {
		if i, err := strconv.ParseInt(str, 10, 64); err == nil {
			return i
		}
	}
	return 0
}
