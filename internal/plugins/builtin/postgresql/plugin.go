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
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// PostgresPlugin is the plugin for diagnosing PostgreSQL instances.
type PostgresPlugin struct {
	db *sql.DB
}

// New creates a new instance of the PostgreSQL plugin.
func New() (interfaces.MiddlewarePlugin, error) {
	return &PostgresPlugin{}, nil
}

func (p *PostgresPlugin) Name() string {
	return "postgresql"
}

func (p *PostgresPlugin) Version() string {
	return "0.1.0"
}

func (p *PostgresPlugin) Description() string {
	return "Provides diagnostics for PostgreSQL instances."
}

func (p *PostgresPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	// This is a simplified diagnosis flow. A real implementation would be more complex.
	data, err := p.CollectAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect data for diagnosis: %w", err)
	}
	issues, err := p.Analyze(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze data for diagnosis: %w", err)
	}
	return &models.DiagnosisResult{
		Issues: issues,
	}, nil
}

func (p *PostgresPlugin) CollectAll(ctx context.Context) (*models.CollectedData, error) {
	metrics, err := p.CollectMetrics(ctx)
	if err != nil {
		return nil, err
	}
	logs, err := p.CollectLogs(ctx, &models.LogOptions{})
	if err != nil {
		return nil, err
	}
	config, err := p.GetConfiguration(ctx)
	if err != nil {
		return nil, err
	}
	return &models.CollectedData{
		Metrics: metrics,
		Logs:    logs,
		Config:  config,
	}, nil
}

func (p *PostgresPlugin) Init(ctx context.Context, config map[string]interface{}) error {
	// In a real implementation, connection details would come from config.
	connStr := "user=postgres password=password dbname=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to postgresql: %w", err)
	}
	p.db = db
	return nil
}

func (p *PostgresPlugin) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	rows, err := p.db.QueryContext(ctx, "SELECT numbackends, xact_commit, xact_rollback, blks_read, blks_hit, tup_returned, tup_fetched, tup_inserted, tup_updated, tup_deleted FROM pg_stat_database WHERE datname = current_database()")
	if err != nil {
		return nil, fmt.Errorf("failed to query pg_stat_database: %w", err)
	}
	defer rows.Close()

	metrics := make(map[string]interface{})
	if rows.Next() {
		var numbackends, xact_commit, xact_rollback, blks_read, blks_hit, tup_returned, tup_fetched, tup_inserted, tup_updated, tup_deleted int64
		if err := rows.Scan(&numbackends, &xact_commit, &xact_rollback, &blks_read, &blks_hit, &tup_returned, &tup_fetched, &tup_inserted, &tup_updated, &tup_deleted); err != nil {
			return nil, fmt.Errorf("failed to scan pg_stat_database row: %w", err)
		}
		metrics["connections"] = float64(numbackends)
		metrics["cache_hit_ratio"] = float64(blks_hit) / float64(blks_hit+blks_read) * 100
		metrics["transactions_committed"] = float64(xact_commit)
		metrics["transactions_rolled_back"] = float64(xact_rollback)
	}

	return &models.MetricsData{Data: metrics}, nil
}

func (p *PostgresPlugin) CollectLogs(ctx context.Context, options *models.LogOptions) (*models.LogData, error) {
	// Log collection from PostgreSQL is complex and depends on the logging setup.
	// This is a placeholder for a future implementation.
	return &models.LogData{Entries: []string{"[placeholder] log collection not implemented for postgresql"}}, nil
}

func (p *PostgresPlugin) GetConfiguration(ctx context.Context) (*models.ConfigData, error) {
	rows, err := p.db.QueryContext(ctx, "SELECT name, setting FROM pg_settings")
	if err != nil {
		return nil, fmt.Errorf("failed to query pg_settings: %w", err)
	}
	defer rows.Close()

	config := make(map[string]string)
	for rows.Next() {
		var name, setting string
		if err := rows.Scan(&name, &setting); err != nil {
			return nil, fmt.Errorf("failed to scan pg_settings row: %w", err)
		}
		config[name] = setting
	}

	return &models.ConfigData{Data: config}, nil
}

func (p *PostgresPlugin) Analyze(ctx context.Context, data *models.CollectedData) ([]*models.Issue, error) {
	analyzer := diagnosis.NewRuleBasedAnalyzer(p.getMetricRules(), p.getLogRules())
	var issues []*models.Issue

	if data.Metrics != nil {
		metricIssues, err := analyzer.AnalyzeMetrics(ctx, data.Metrics)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze postgresql metrics: %w", err)
		}
		issues = append(issues, metricIssues...)
	}

	if data.Logs != nil {
		logIssues, err := analyzer.AnalyzeLogs(ctx, data.Logs)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze postgresql logs: %w", err)
		}
		issues = append(issues, logIssues...)
	}

	return issues, nil
}

func (p *PostgresPlugin) getMetricRules() []diagnosis.MetricRule {
	return []diagnosis.MetricRule{
		{
			MetricName:     "cache_hit_ratio",
			Operator:       "<",
			Threshold:      95,
			Severity:       enum.SeverityWarning,
			IssueTitle:     "Low Cache Hit Ratio",
			Recommendation: "A cache hit ratio below 95% may indicate insufficient memory allocated to PostgreSQL. Consider increasing the 'shared_buffers' parameter in your postgresql.conf file.",
		},
		{
			MetricName:     "connections",
			Operator:       ">",
			Threshold:      100, // This is a generic threshold; a real system might adjust it based on instance size.
			Severity:       enum.SeverityWarning,
			IssueTitle:     "High Number of Connections",
			Recommendation: "The number of active connections is high. This may indicate connection leaks in applications or a need to increase 'max_connections'.",
		},
	}
}

func (p *PostgresPlugin) getLogRules() []diagnosis.LogRule {
	return []diagnosis.LogRule{
		{
			Pattern:        "deadlock detected",
			Severity:       enum.SeverityHigh,
			IssueTitle:     "Deadlock Detected",
			Recommendation: "Deadlocks occur when two or more transactions are waiting for each other to release locks. Review application logic and transaction isolation levels.",
		},
		{
			Pattern:        "slow query",
			Severity:       enum.SeverityWarning,
			IssueTitle:     "Slow Query Logged",
			Recommendation: "A slow query was logged. Use EXPLAIN ANALYZE to investigate the query plan and consider adding indexes to the relevant tables.",
		},
	}
}

func (p *PostgresPlugin) GetHealth(ctx context.Context) (*models.HealthStatus, error) {
	if err := p.db.PingContext(ctx); err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: err.Error()}, nil
	}
	return &models.HealthStatus{IsHealthy: true, Message: "Successfully connected to PostgreSQL."}, nil
}

func (p *PostgresPlugin) CanAutoFix(issue *models.Issue) bool {
	// For now, we'll say that we can't auto-fix anything.
	return false
}

func (p *PostgresPlugin) ExecuteFix(ctx context.Context, action *models.FixAction) (*models.FixResult, error) {
	return nil, fmt.Errorf("auto-fix is not yet implemented for the postgresql plugin")
}

func (p *PostgresPlugin) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	return p.GetHealth(ctx)
}

func (p *PostgresPlugin) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *PostgresPlugin) SupportedVersions() []string {
	return []string{"12", "13", "14", "15"}
}

func (p *PostgresPlugin) ValidateFix(ctx context.Context, action *models.FixAction) error {
	return fmt.Errorf("auto-fix validation is not yet implemented for the postgresql plugin")
}