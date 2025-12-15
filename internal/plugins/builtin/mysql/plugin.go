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

// Package mysql implements the built-in plugin for diagnosing MySQL databases.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
)

const (
	IssueTitleSlowQuery = "Slow Query Running"
	IssueTitleConnFull  = "Connection Pool Exhaustion"
	IssueTitleLockWait  = "Lock Wait Timeout"
)

// mysqlPlugin is the concrete implementation of the MiddlewarePlugin for MySQL.
type mysqlPlugin struct {
	base.Plugin
	db        *sql.DB
	collector *collector
	analyzer  *analyzer
	fixer     *base.FixExecutor
}

// New is the factory function that creates an instance of the MySQL plugin.
func New() (interfaces.MiddlewarePlugin, error) {
	p := &mysqlPlugin{}
	// Use base.Plugin Init
	p.Plugin.Init("mysql", "0.1.0", "Provides diagnostics for MySQL and compatible databases.")

	dsn := "root:password@tcp(127.0.0.1:3306)/?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	p.db = db
	p.collector = newCollector(db, p.Log)
	p.analyzer = newAnalyzer(p.Log)
	p.fixer = base.NewFixExecutor(p.Log)

	p.Log.Info("MySQL plugin initialized successfully.")
	return p, nil
}

// Init shadows base.Plugin.Init to satisfy interface
func (p *mysqlPlugin) Init(cfg *config.PluginConfig) error {
	// In real world, update DSN from config
	return nil
}

// Diagnose orchestrates the diagnosis process for MySQL.
func (p *mysqlPlugin) Diagnose(ctx context.Context, _ *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	p.Log.Info("Starting MySQL diagnosis.")

	// 1. Collect data
	globalStatus, err := p.collector.CollectGlobalStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect mysql global status: %w", err)
	}
	variables, err := p.collector.CollectVariables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect mysql variables: %w", err)
	}

	// 2. Analyze data
	issues := p.analyzer.Analyze(globalStatus, variables)

	// 3. Assemble result
	result := &models.DiagnosisResult{
		ID:        fmt.Sprintf("mysql-diag-%d", time.Now().Unix()),
		Timestamp: time.Now().UTC(),
		Summary:   fmt.Sprintf("MySQL diagnosis complete. Found %d potential issues.", len(issues)),
		Issues:    issues,
	}

	return result, nil
}

// --- Interface Method Implementations ---

func (p *mysqlPlugin) Ping(ctx context.Context, target string) error {
	return p.db.PingContext(ctx)
}

func (p *mysqlPlugin) HealthCheck(ctx context.Context, target string) (*models.HealthStatus, error) {
	if err := p.Ping(ctx, target); err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: fmt.Sprintf("Failed to ping MySQL: %v", err)}, nil
	}
	return &models.HealthStatus{IsHealthy: true, Message: "MySQL is responsive."}, nil
}

func (p *mysqlPlugin) CollectConfig(ctx context.Context, target string) (*models.ConfigData, error) {
	vars, err := p.collector.CollectVariables(ctx)
	if err != nil {
		return nil, err
	}
	return &models.ConfigData{Data: vars}, nil
}

func (p *mysqlPlugin) CollectMetrics(ctx context.Context, target string) (*models.MetricsData, error) {
	return p.collector.CollectMetrics(ctx)
}

func (p *mysqlPlugin) CollectLogs(ctx context.Context, target string, _ *models.LogOptions) (*models.LogData, error) {
	p.Log.Info("MySQL slow query log collection is a placeholder.")
	return &models.LogData{Entries: []string{}}, nil
}

func (p *mysqlPlugin) Shutdown() error {
	p.Log.Info("Shutting down MySQL plugin and closing database connections.")
	return p.db.Close()
}

// --- Fix Capabilities ---

func (p *mysqlPlugin) CanAutoFix(issue *models.Issue) (bool, *models.FixAction) {
	switch issue.Title {
	case IssueTitleSlowQuery:
		// Expect Evidence to contain ID like "ID: 123"
		return true, &models.FixAction{
			ID:          "fix-mysql-kill-query",
			Description: "Kill slow query process",
			Command:     "KILL QUERY ?",
			Category:    "Query",
			Parameters:  map[string]string{"process_id": extractIDFromEvidence(issue.Evidence)},
		}
	case IssueTitleConnFull:
		return true, &models.FixAction{
			ID:          "fix-mysql-kill-sleep",
			Description: "Kill sleeping connections",
			Command:     "KILL_SLEEP_CONNECTIONS", // Custom meta-command
			Category:    "Connection",
		}
	}
	return false, nil
}

func (p *mysqlPlugin) ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error) {
	return p.fixer.Execute(ctx, fix, func(ctx context.Context) error {
		if fix.Command == "KILL_SLEEP_CONNECTIONS" {
			// Logic to kill sleep connections
			rows, err := p.db.QueryContext(ctx, "SELECT ID FROM information_schema.processlist WHERE Command = 'Sleep' AND Time > 60")
			if err != nil {
				return err
			}
			defer rows.Close()
			var ids []int
			for rows.Next() {
				var id int
				if err := rows.Scan(&id); err == nil {
					ids = append(ids, id)
				}
			}
			for _, id := range ids {
				p.db.ExecContext(ctx, fmt.Sprintf("KILL %d", id))
			}
			return nil
		} else if strings.HasPrefix(fix.Command, "KILL QUERY") {
			pidStr := fix.Parameters["process_id"]
			if pidStr == "" {
				return fmt.Errorf("missing process_id parameter")
			}
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				return fmt.Errorf("invalid process_id: %v", err)
			}
			_, err = p.db.ExecContext(ctx, fmt.Sprintf("KILL QUERY %d", pid))
			return err
		}
		return fmt.Errorf("unknown command: %s", fix.Command)
	}, nil)
}

func (p *mysqlPlugin) ValidateFix(ctx context.Context, issue *models.Issue, result *models.FixResult) (bool, string, error) {
	if !result.Success {
		return false, "Fix execution failed", nil
	}
	// Simplified validation
	return true, "Assumed success based on no error", nil
}

func extractIDFromEvidence(evidence string) string {
	// Simple placeholder extractor. Real impl would use regex.
	// For now, assume format "Process ID: 123 ..."
	if strings.Contains(evidence, "Process ID: ") {
		parts := strings.Split(evidence, "Process ID: ")
		if len(parts) > 1 {
			idPart := strings.Split(parts[1], " ")[0]
			return idPart
		}
	}
	return "0"
}
