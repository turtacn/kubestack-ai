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

package postgresql

import (
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// Analyzer is responsible for analyzing collected PostgreSQL data to identify issues.
type Analyzer struct {
	log logger.Logger
}

// NewAnalyzer creates a new analyzer for PostgreSQL data.
func NewAnalyzer(log logger.Logger) *Analyzer {
	return &Analyzer{log: log}
}

// Analyze is the main entry point for the analyzer.
func (a *Analyzer) Analyze(metrics map[string]interface{}) []*models.Issue {
	var issues []*models.Issue
	a.log.Info("Analyzing collected PostgreSQL data.")

	issues = append(issues, a.analyzeCacheHitRatio(metrics)...)
	issues = append(issues, a.analyzeConnections(metrics)...)
	issues = append(issues, a.analyzeLongIdleTx(metrics)...)

	return issues
}

func (a *Analyzer) analyzeCacheHitRatio(metrics map[string]interface{}) []*models.Issue {
	var issues []*models.Issue
	if ratio, ok := metrics["cache_hit_ratio"].(float64); ok {
		if ratio < 0.95 {
			issues = append(issues, &models.Issue{
				Title:    "Low Cache Hit Ratio",
				Severity: enum.SeverityWarning,
				Evidence: fmt.Sprintf("Cache hit ratio is %.2f%% (threshold: 95%%).", ratio*100),
				Recommendations: []*models.Recommendation{{
					Description: "Tune shared_buffers to fit more data in memory. Check for sequential scans on large tables.",
					Fix: models.FixAction{
						Description: "Increase shared_buffers",
						Category:    "Configuration",
					},
				}},
			})
		}
	}
	return issues
}

func (a *Analyzer) analyzeConnections(metrics map[string]interface{}) []*models.Issue {
	var issues []*models.Issue
	if usage, ok := metrics["connection_usage_percent"].(float64); ok {
		if usage > 85.0 {
			maxConns := metrics["max_connections"]
			issues = append(issues, &models.Issue{
				Title:    "High Connection Usage",
				Severity: enum.SeverityWarning,
				Evidence: fmt.Sprintf("Connection usage is %.2f%% (Max: %v).", usage, maxConns),
				Recommendations: []*models.Recommendation{{
					Description: "Increase max_connections or use a connection pooler like PgBouncer.",
					Fix: models.FixAction{
						Description: "Adjust max_connections in postgresql.conf",
						Category:    "Configuration",
					},
				}},
			})
		}
	}
	return issues
}

func (a *Analyzer) analyzeLongIdleTx(metrics map[string]interface{}) []*models.Issue {
	var issues []*models.Issue
	if count, ok := metrics["long_idle_tx"].(int64); ok {
		if count > 0 {
			issues = append(issues, &models.Issue{
				Title:    "Long Idle Transactions Detected",
				Severity: enum.SeverityWarning,
				Evidence: fmt.Sprintf("Found %d transactions idle for more than 5 minutes.", count),
				Recommendations: []*models.Recommendation{{
					Description: "Terminate idle transactions to release locks and prevent table bloat. Investigate application logic.",
					Fix: models.FixAction{
						Description: "Terminate idle transactions",
						Category:    "Operation",
					},
				}},
			})
		}
	}
	return issues
}
