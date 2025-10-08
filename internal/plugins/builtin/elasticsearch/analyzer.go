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

package elasticsearch

import (
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// analyzer is responsible for analyzing collected Elasticsearch data to identify issues.
type analyzer struct {
	log logger.Logger
}

// newAnalyzer creates a new analyzer for Elasticsearch data.
//
// Parameters:
//   log (logger.Logger): A contextualized logger for the analyzer.
//
// Returns:
//   *analyzer: A new instance of the Elasticsearch analyzer.
func newAnalyzer(log logger.Logger) *analyzer {
	return &analyzer{log: log}
}

// AnalyzeClusterHealth is a primary analysis function that checks the cluster
// health status from the `_cluster/health` API endpoint. It creates issues for
// "red" or "yellow" statuses and for any unassigned shards.
//
// Parameters:
//   health (map[string]interface{}): The parsed JSON data from the cluster health API.
//
// Returns:
//   []*models.Issue: A slice of issues identified from the cluster health data.
func (a *analyzer) AnalyzeClusterHealth(health map[string]interface{}) []*models.Issue {
	var issues []*models.Issue
	a.log.Info("Analyzing Elasticsearch cluster health.")

	status, ok := health["status"].(string)
	if !ok {
		a.log.Warn("Could not determine cluster health status from API response.")
		return issues
	}

	if status == "red" {
		issues = append(issues, &models.Issue{
			Title:    "Cluster Health is RED",
			Severity: enum.SeverityCritical,
			Evidence: "The cluster health status is 'red', indicating that at least one primary shard and all its replicas are unavailable.",
			Recommendations: []*models.Recommendation{{
				Description: "A RED cluster status means some data is inaccessible and data loss is possible. Search and indexing will fail for some indices. Use the `_cluster/allocation/explain` API to understand why shards are not being allocated. Investigate logs on all nodes for critical errors like disk full or network partitions.",
			}},
		})
	} else if status == "yellow" {
		issues = append(issues, &models.Issue{
			Title:    "Cluster Health is YELLOW",
			Severity: enum.SeverityHigh,
			Evidence: "The cluster health status is 'yellow', indicating that at least one replica shard is unassigned.",
			Recommendations: []*models.Recommendation{{
				Description: "A YELLOW cluster status means all primary shards are active, so data is available, but fault tolerance is reduced. If a node holding a primary shard fails, data could be lost. This is often caused by a node leaving the cluster or insufficient nodes for replica allocation. Use the `_cluster/allocation/explain` API for details.",
			}},
		})
	}

	if unassigned, ok := health["unassigned_shards"].(float64); ok && unassigned > 0 {
		issues = append(issues, &models.Issue{
			Title:    "Unassigned Shards Detected",
			Severity: enum.SeverityHigh,
			Evidence: fmt.Sprintf("There are %.0f unassigned shards in the cluster.", unassigned),
			Recommendations: []*models.Recommendation{{
				Description: "Unassigned shards reduce data availability and redundancy. Use the `_cat/shards` API to list them and `_cluster/allocation/explain` to diagnose why they are not being allocated to a node.",
			}},
		})
	}
	return issues
}

// AnalyzeNodesStats checks for common issues at the node level by parsing data
// from the `_nodes/stats` API endpoint. It currently checks for high JVM heap
// usage and thread pool rejections.
//
// Parameters:
//   stats (map[string]interface{}): The parsed JSON data from the nodes stats API.
//
// Returns:
//   []*models.Issue: A slice of issues identified from the node-level stats.
func (a *analyzer) AnalyzeNodesStats(stats map[string]interface{}) []*models.Issue {
	var issues []*models.Issue
	a.log.Info("Analyzing Elasticsearch nodes stats.")

	nodes, ok := stats["nodes"].(map[string]interface{})
	if !ok {
		return issues
	}

	for nodeID, nodeData := range nodes {
		nodeMap, ok := nodeData.(map[string]interface{})
		if !ok {
			continue
		}
		nodeName := nodeMap["name"].(string)

		// Analyze JVM Heap Usage
		if jvm, ok := nodeMap["jvm"].(map[string]interface{}); ok {
			if heap, ok := jvm["mem"].(map[string]interface{}); ok {
				if heapUsedPercent, ok := heap["heap_used_percent"].(float64); ok && heapUsedPercent > 85.0 {
					issues = append(issues, &models.Issue{
						Title:    "High JVM Heap Usage",
						Severity: enum.SeverityWarning,
						Evidence: fmt.Sprintf("Node '%s' (%s) has a JVM heap usage of %.2f%%.", nodeName, nodeID, heapUsedPercent),
						Recommendations: []*models.Recommendation{{
							Description: "Sustained high JVM heap usage (>85%) can lead to long garbage collection pauses, performance degradation, and eventually OutOfMemoryError. Consider scaling up the node's memory, optimizing queries/mappings, or adding more nodes to the cluster.",
						}},
					})
				}
			}
		}

		// Analyze Thread Pool Rejections
		if threadPools, ok := nodeMap["thread_pool"].(map[string]interface{}); ok {
			for poolName, poolData := range threadPools {
				poolMap, ok := poolData.(map[string]interface{})
				if !ok {
					continue
				}
				if rejections, ok := poolMap["rejected"].(float64); ok && rejections > 0 {
					issues = append(issues, &models.Issue{
						Title:    "Thread Pool Rejections",
						Severity: enum.SeverityWarning,
						Evidence: fmt.Sprintf("Node '%s' has %.0f rejections in the '%s' thread pool.", nodeName, rejections, poolName),
						Recommendations: []*models.Recommendation{{
							Description: "Thread pool rejections indicate that the node is overloaded and cannot process requests fast enough. This can be caused by expensive queries, high indexing load, or insufficient resources. Investigate the cause of the high load or consider scaling the cluster.",
						}},
					})
				}
			}
		}
	}
	return issues
}

//Personal.AI order the ending
