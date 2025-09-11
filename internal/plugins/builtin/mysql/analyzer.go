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

package mysql

import (
	"fmt"
	"strconv"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// analyzer is responsible for analyzing collected MySQL data to identify issues.
type analyzer struct {
	log logger.Logger
}

// newAnalyzer creates a new MySQL data analyzer.
func newAnalyzer(log logger.Logger) *analyzer {
	return &analyzer{log: log}
}

// Analyze is the main entry point for the analyzer. It orchestrates various specialized
// analysis functions and aggregates the issues they find.
func (a *analyzer) Analyze(status, variables map[string]string) []*models.Issue {
	var issues []*models.Issue
	a.log.Info("Analyzing collected MySQL data.")

	issues = append(issues, a.analyzeConnections(status, variables)...)
	issues = append(issues, a.analyzePerformance(status)...)
	issues = append(issues, a.analyzeInnodb(status, variables)...)
	// TODO: Add calls to other analyzers: replication, security, index usage, etc.
	// These would require data from additional collectors (e.g., `SHOW SLAVE STATUS`).

	return issues
}

// analyzeConnections checks for issues related to client connections.
func (a *analyzer) analyzeConnections(status, variables map[string]string) []*models.Issue {
	var issues []*models.Issue

	// Check connection usage against the configured limit.
	if threadsConnectedStr, ok := status["Threads_connected"]; ok {
		if maxConnectionsStr, ok := variables["max_connections"]; ok {
			threadsConnected, _ := strconv.ParseFloat(threadsConnectedStr, 64)
			maxConnections, _ := strconv.ParseFloat(maxConnectionsStr, 64)

			if maxConnections > 0 && (threadsConnected/maxConnections) > 0.85 {
				issues = append(issues, &models.Issue{
					Title:    "High Connection Usage",
					Severity: enum.SeverityWarning,
					Evidence: fmt.Sprintf("Current connections are at %.0f, which is over 85%% of the max_connections limit (%.0f).", threadsConnected, maxConnections),
					Recommendations: []*models.Recommendation{{Description: "Connection usage is high. If this is unexpected, investigate for connection leaks in applications. If this is normal load, consider increasing the 'max_connections' parameter in your my.cnf file."}},
				})
			}
		}
	}

	// Check for a high number of aborted connections.
	if abortedStr, ok := status["Aborted_connects"]; ok {
		if aborted, _ := strconv.ParseInt(abortedStr, 10, 64); aborted > 100 { // Threshold is arbitrary
			issues = append(issues, &models.Issue{
				Title:    "High Number of Aborted Connections",
				Severity: enum.SeverityWarning,
				Evidence: fmt.Sprintf("There have been %d aborted client connections.", aborted),
				Recommendations: []*models.Recommendation{{Description: "A high number of aborted connections can indicate network problems or applications not closing connections properly. Check the 'wait_timeout' and 'interactive_timeout' variables and review application connection handling logic."}},
			})
		}
	}
	return issues
}

// analyzePerformance checks for common performance bottlenecks based on status variables.
func (a *analyzer) analyzePerformance(status map[string]string) []*models.Issue {
	var issues []*models.Issue

	// Check for slow queries.
	if slowQueriesStr, ok := status["Slow_queries"]; ok {
		if slowQueries, _ := strconv.ParseInt(slowQueriesStr, 10, 64); slowQueries > 0 {
			issues = append(issues, &models.Issue{
				Title:    "Slow Queries Detected",
				Severity: enum.SeverityWarning,
				Evidence: fmt.Sprintf("The `Slow_queries` status variable shows that %d slow queries have been logged since startup.", slowQueries),
				Recommendations: []*models.Recommendation{{Description: "Slow queries are impacting performance. Ensure the slow query log is enabled (`slow_query_log = ON`) and analyze it (e.g., with pt-query-digest or mysqldumpslow) to identify and optimize the problematic queries. This often involves adding indexes."}},
			})
		}
	}

	// Check for queries that perform full table scans, which are often inefficient.
	if fullJoinsStr, ok := status["Select_full_join"]; ok {
		if fullJoins, _ := strconv.ParseInt(fullJoinsStr, 10, 64); fullJoins > 10 { // Arbitrary threshold
			issues = append(issues, &models.Issue{
				Title:    "Inefficient Queries (Full Joins) Detected",
				Severity: enum.SeverityWarning,
				Evidence: fmt.Sprintf("The `Select_full_join` status variable shows %d queries have been executed without using an index for joins.", fullJoins),
				Recommendations: []*models.Recommendation{{Description: "Queries performing full joins can be a major performance bottleneck. This usually indicates missing indexes on the columns used in join conditions. Enable `log_queries_not_using_indexes` to find these queries and add appropriate indexes."}},
			})
		}
	}
	return issues
}

// analyzeInnodb checks for common InnoDB storage engine misconfigurations.
func (a *analyzer) analyzeInnodb(status, variables map[string]string) []*models.Issue {
	var issues []*models.Issue

	// Check InnoDB buffer pool size. This is one of the most critical performance parameters.
	if poolSizeStr, ok := variables["innodb_buffer_pool_size"]; ok {
		poolSize, _ := strconv.ParseInt(poolSizeStr, 10, 64)
		// This is a very simplistic check. A real-world check would compare it to system RAM.
		if poolSize < (1024 * 1024 * 1024) { // Less than 1GB
			issues = append(issues, &models.Issue{
				Title:    "Small InnoDB Buffer Pool Size",
				Severity: enum.SeverityWarning,
				Evidence: fmt.Sprintf("innodb_buffer_pool_size is set to %d bytes (~%.2f GB), which may be too small for a production system.", poolSize, float64(poolSize)/1024/1024/1024),
				Recommendations: []*models.Recommendation{{Description: "The InnoDB buffer pool caches data and indexes in memory. A small buffer pool can lead to excessive disk I/O. On a dedicated database server, it's often recommended to set this to 50-75% of the total available RAM."}},
			})
		}
	}
	return issues
}

//Personal.AI order the ending
