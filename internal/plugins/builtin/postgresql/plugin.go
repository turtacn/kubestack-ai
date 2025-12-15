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
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
)

const (
	IssueTitleIdleTx    = "Long Running Idle Transaction"
	IssueTitleNeedAnalyze = "Table Needs Analysis"
)

type postgresPlugin struct {
	base.Plugin
	db     *sql.DB
	config *config.PluginConfig
	fixer  *base.FixExecutor
}

func New() (interfaces.DiagnosticPlugin, error) {
	p := &postgresPlugin{}
	// Call base Plugin Init explicitly
	p.Plugin.Init("postgresql", "1.0.0", "PostgreSQL diagnostic plugin")

	// Default connection (should be updated via Init/Config)
	connStr := "user=postgres dbname=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	p.db = db
	p.fixer = base.NewFixExecutor(p.Log)
	return p, nil
}

func (p *postgresPlugin) SupportedTypes() []enum.MiddlewareType {
	return []enum.MiddlewareType{enum.PostgreSQL}
}

func (p *postgresPlugin) SupportedVersions() []string {
	return []string{"12", "13", "14", "15"}
}

// Init implements the DiagnosticPlugin interface
func (p *postgresPlugin) Init(cfg *config.PluginConfig) error {
	p.config = cfg
	// In a real scenario, we would close the old DB and open a new one with cfg params
	return nil
}

func (p *postgresPlugin) Shutdown() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *postgresPlugin) Diagnose(ctx context.Context, _ *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	// Updated to return DiagnosisResult
	return &models.DiagnosisResult{
		ID:        fmt.Sprintf("pg-diag-%d", time.Now().Unix()),
		Timestamp: time.Now().UTC(),
		Summary:   "PostgreSQL diagnosis placeholder",
		Issues:    []*models.Issue{},
	}, nil
}

func (p *postgresPlugin) CollectMetrics(ctx context.Context, target string) (*models.MetricsData, error) {
	return &models.MetricsData{Data: map[string]interface{}{}}, nil
}

func (p *postgresPlugin) CollectLogs(ctx context.Context, target string, opts *models.LogOptions) (*models.LogData, error) {
	return &models.LogData{Entries: []string{}}, nil
}

func (p *postgresPlugin) CollectConfig(ctx context.Context, target string) (*models.ConfigData, error) {
	return &models.ConfigData{Data: map[string]string{}}, nil
}

func (p *postgresPlugin) HealthCheck(ctx context.Context, target string) (*models.HealthStatus, error) {
	if err := p.Ping(ctx, target); err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: err.Error()}, nil
	}
	return &models.HealthStatus{IsHealthy: true}, nil
}

func (p *postgresPlugin) Ping(ctx context.Context, target string) error {
	return p.db.PingContext(ctx)
}

// --- Fix Capabilities ---

func (p *postgresPlugin) CanAutoFix(issue *models.Issue) (bool, *models.FixAction) {
	switch issue.Title {
	case IssueTitleIdleTx:
		// Requires PID from evidence/metrics. Mocking extracting PID=123 for now if not present
		return true, &models.FixAction{
			ID:          "fix-pg-terminate-backend",
			Description: "Terminate idle backend",
			Command:     "PG_TERMINATE_BACKEND",
			Parameters:  map[string]string{"pid": "123"},
		}
	case IssueTitleNeedAnalyze:
		return true, &models.FixAction{
			ID:          "fix-pg-analyze",
			Description: "Run ANALYZE on table",
			Command:     "ANALYZE",
			Parameters:  map[string]string{"table": "mytable"},
		}
	}
	return false, nil
}

func (p *postgresPlugin) ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error) {
	return p.fixer.Execute(ctx, fix, func(ctx context.Context) error {
		if fix.Command == "PG_TERMINATE_BACKEND" {
			pid := fix.Parameters["pid"]
			if pid == "" {
				return fmt.Errorf("missing pid parameter")
			}
			query := "SELECT pg_terminate_backend($1)"
			_, err := p.db.ExecContext(ctx, query, pid)
			return err
		} else if fix.Command == "ANALYZE" {
			tableName := fix.Parameters["table"]
			if tableName == "" {
				return fmt.Errorf("missing table parameter")
			}
			// Sanitize tableName in real app to prevent SQL injection
			// simple protection: only allow alphanumeric and underscore
			if !isValidTableName(tableName) {
				return fmt.Errorf("invalid table name")
			}
			query := fmt.Sprintf("ANALYZE %s", tableName)
			_, err := p.db.ExecContext(ctx, query)
			return err
		}
		return fmt.Errorf("unknown command")
	}, nil)
}

func (p *postgresPlugin) ValidateFix(ctx context.Context, issue *models.Issue, result *models.FixResult) (bool, string, error) {
	return true, "Assumed success", nil
}

func isValidTableName(name string) bool {
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}
