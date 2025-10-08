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
	"fmt"
	"strconv"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// analyzer is responsible for analyzing collected Redis data to identify issues.
type analyzer struct {
	log logger.Logger
}

// newAnalyzer creates a new analyzer for Redis data.
//
// Parameters:
//   log (logger.Logger): A contextualized logger for the analyzer.
//
// Returns:
//   *analyzer: A new instance of the Redis analyzer.
func newAnalyzer(log logger.Logger) *analyzer {
	return &analyzer{log: log}
}

// Analyze is the main entry point for the analyzer. It orchestrates calls to
// various specialized analysis functions (e.g., for memory, persistence) and
// aggregates the issues they find into a single list.
//
// Parameters:
//   info (map[string]string): A map of data from the `INFO` command.
//   config (*models.ConfigData): The structured configuration data.
//   slowlogs (*models.LogData): A list of entries from the slowlog.
//
// Returns:
//   []*models.Issue: A slice of all issues identified from the data.
func (a *analyzer) Analyze(info map[string]string, config *models.ConfigData, slowlogs *models.LogData) []*models.Issue {
	var issues []*models.Issue
	a.log.Info("Analyzing collected Redis data.")

	// Run all specialized analysis functions.
	issues = append(issues, a.analyzeMemory(info, config)...)
	issues = append(issues, a.analyzePersistence(info, config)...)
	issues = append(issues, a.analyzeSecurity(config)...)
	issues = append(issues, a.analyzePerformance(info, slowlogs)...)
	// TODO: Add calls to other analyzers here, e.g., for replication, cluster state, etc.

	return issues
}

// analyzeMemory checks for memory-related issues, such as high fragmentation,
// which can indicate wasted memory, and a risky `noeviction` policy, which can
// cause write failures when the memory limit is reached.
func (a *analyzer) analyzeMemory(info map[string]string, config *models.ConfigData) []*models.Issue {
	var issues []*models.Issue

	// Check memory fragmentation ratio. A high ratio can indicate wasted memory.
	if ratioStr, ok := info["mem_fragmentation_ratio"]; ok {
		if ratio, err := strconv.ParseFloat(ratioStr, 64); err == nil && ratio > 1.5 {
			issues = append(issues, &models.Issue{
				Title:    "High Memory Fragmentation",
				Severity: enum.SeverityWarning,
				Evidence: fmt.Sprintf("mem_fragmentation_ratio is %.2f. A value > 1.5 suggests significant memory fragmentation.", ratio),
				Recommendations: []*models.Recommendation{{
					Description: "High fragmentation can be addressed by restarting the Redis server, which allows the OS to reclaim fragmented memory. Ensure persistence is enabled if data loss is not acceptable. Alternatively, consider using an allocator like jemalloc, which can help mitigate fragmentation.",
				}},
			})
		}
	}

	// Check maxmemory policy. 'noeviction' can cause write failures when memory is full.
	if policy, ok := config.Data["maxmemory-policy"]; ok && policy == "noeviction" {
		issues = append(issues, &models.Issue{
			Title:    "Risky Memory Eviction Policy",
			Severity: enum.SeverityWarning,
			Evidence: "The 'maxmemory-policy' is set to 'noeviction'.",
			Recommendations: []*models.Recommendation{{
				Description: "With the 'noeviction' policy, Redis will return errors on write commands when the memory limit is reached. If this is not the intended behavior, consider using an LRU or LFU policy (e.g., 'allkeys-lru') to allow Redis to evict old data.",
			}},
		})
	}

	return issues
}

// analyzePersistence checks for potential issues with RDB and AOF configurations,
// such as when both persistence methods are disabled, which could lead to total
// data loss on restart.
func (a *analyzer) analyzePersistence(info map[string]string, config *models.ConfigData) []*models.Issue {
	var issues []*models.Issue

	// Check if all persistence is disabled.
	if aof, aofOk := config.Data["appendonly"]; aofOk && aof == "no" {
		if rdbRules, rdbOk := config.Data["save"]; rdbOk && rdbRules == "" {
			issues = append(issues, &models.Issue{
				Title:    "Persistence is Disabled",
				Severity: enum.SeverityHigh,
				Evidence: "Both RDB snapshotting ('save' rules) and AOF logging are disabled.",
				Recommendations: []*models.Recommendation{{
					Description: "Persistence is completely disabled. In case of a server restart or crash, all data will be lost. It is highly recommended to enable either RDB snapshots or AOF logging if data durability is important.",
				}},
			})
		}
	}
	return issues
}

// analyzeSecurity checks for common security misconfigurations, such as disabled
// password protection (`requirepass`) or binding to all network interfaces, which
// can expose the instance to untrusted networks.
func (a *analyzer) analyzeSecurity(config *models.ConfigData) []*models.Issue {
	var issues []*models.Issue

	// Check for disabled password protection.
	if pass, ok := config.Data["requirepass"]; ok && pass == "" {
		issues = append(issues, &models.Issue{
			Title:    "Password Protection Disabled",
			Severity: enum.SeverityCritical,
			Evidence: "The 'requirepass' configuration is empty or not set.",
			Recommendations: []*models.Recommendation{{
				Description: "The Redis instance is not password-protected, allowing any client to connect. It is critical to set a strong password using the 'requirepass' config option to prevent unauthorized access.",
			}},
		})
	}

	// Check if Redis is bound to all network interfaces, which can be a security risk.
	if bind, ok := config.Data["bind"]; ok && (strings.Contains(bind, "0.0.0.0") || strings.Contains(bind, "*")) {
		issues = append(issues, &models.Issue{
			Title:    "Redis Bound to All Network Interfaces",
			Severity: enum.SeverityHigh,
			Evidence: fmt.Sprintf("Redis is bound to '%s', making it potentially accessible from untrusted networks.", bind),
			Recommendations: []*models.Recommendation{{
				Description: "Binding Redis to all interfaces can expose it to external networks. For security, it is strongly recommended to bind Redis only to a specific, trusted network interface (e.g., '127.0.0.1' for local access, or a private IP address).",
			}},
		})
	}

	return issues
}

// analyzePerformance checks for performance issues by inspecting the slowlog for
// any recorded long-running commands.
func (a *analyzer) analyzePerformance(info map[string]string, slowlogs *models.LogData) []*models.Issue {
	var issues []*models.Issue

	if slowlogs != nil && len(slowlogs.Entries) > 0 {
		issues = append(issues, &models.Issue{
			Title:    "Slow Queries Detected",
			Severity: enum.SeverityWarning,
			Evidence: fmt.Sprintf("Found %d entries in the slowlog. The first entry is: %s", len(slowlogs.Entries), slowlogs.Entries[0]),
			Recommendations: []*models.Recommendation{{
				Description: "Slow queries can degrade overall Redis performance by consuming significant CPU time. Analyze the commands in the slowlog (using `SLOWLOG GET`) to identify and optimize these long-running operations. Common causes include commands on large collections (e.g., LREM, SUNION) or complex Lua scripts.",
			}},
		})
	}

	return issues
}

//Personal.AI order the ending
