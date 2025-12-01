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
	"os"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
	_ "github.com/lib/pq"
)

// postgresPlugin implements the MiddlewarePlugin interface for PostgreSQL.
type postgresPlugin struct {
	base.Plugin
	db        *sql.DB
	collector *Collector
	analyzer  *Analyzer
}

// New creates a new instance of the PostgreSQL plugin.
func New() (interfaces.MiddlewarePlugin, error) {
	p := &postgresPlugin{}
	p.Init("postgresql", "0.1.0", "Provides diagnostics for PostgreSQL databases.")

	// Allow configuration via environment variable, fallback to default for dev
	dsn := os.Getenv("KSA_POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:password@localhost:5432/postgres?sslmode=disable"
		p.Log.Warn("Using default PostgreSQL credentials. Set KSA_POSTGRES_DSN environment variable to configure.")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Set some connection pool defaults
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	p.db = db
	p.collector = NewCollector(db, p.Log)
	p.analyzer = NewAnalyzer(p.Log)

	p.Log.Info("PostgreSQL plugin initialized successfully.")
	return p, nil
}

func (p *postgresPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	p.Log.Info("Starting PostgreSQL diagnosis.")

	metricsData, err := p.collector.CollectMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect postgres metrics: %w", err)
	}

	issues := p.analyzer.Analyze(metricsData.Data)

	result := &models.DiagnosisResult{
		ID:        fmt.Sprintf("pg-diag-%d", time.Now().Unix()),
		Timestamp: time.Now().UTC(),
		Summary:   fmt.Sprintf("PostgreSQL diagnosis complete. Found %d issues.", len(issues)),
		Issues:    issues,
		Metrics:   metricsData.Data,
	}

	return result, nil
}

func (p *postgresPlugin) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	return p.collector.CollectMetrics(ctx)
}

func (p *postgresPlugin) CollectLogs(ctx context.Context, opts *models.LogOptions) (*models.LogData, error) {
	p.Log.Info("PostgreSQL log collection is not yet fully implemented.")
	return &models.LogData{Entries: []string{}}, nil
}

func (p *postgresPlugin) GetConfiguration(ctx context.Context) (*models.ConfigData, error) {
	vars, err := p.collector.CollectVariables(ctx)
	if err != nil {
		return nil, err
	}

	return &models.ConfigData{Data: vars}, nil
}

func (p *postgresPlugin) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	if err := p.Ping(ctx); err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: fmt.Sprintf("Failed to ping PostgreSQL: %v", err)}, nil
	}
	return &models.HealthStatus{IsHealthy: true, Message: "PostgreSQL is responsive."}, nil
}

func (p *postgresPlugin) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *postgresPlugin) SupportedVersions() []string {
	return []string{"12", "13", "14", "15", "16"}
}

func (p *postgresPlugin) CanAutoFix(issue *models.Issue) bool {
	// Placeholder
	return false
}

func (p *postgresPlugin) ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *postgresPlugin) ValidateFix(ctx context.Context, fix *models.FixAction) error {
	return fmt.Errorf("not implemented")
}

func (p *postgresPlugin) Shutdown() error {
	p.Log.Info("Shutting down PostgreSQL plugin.")
	return p.db.Close()
}
