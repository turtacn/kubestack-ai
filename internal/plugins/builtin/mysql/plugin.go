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
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/base"
)

// mysqlPlugin is the concrete implementation of the MiddlewarePlugin for MySQL.
type mysqlPlugin struct {
	base.Plugin
	db        *sql.DB
	collector *collector
	analyzer  *analyzer
}

// New is the factory function that creates an instance of the MySQL plugin.
// It initializes the base plugin, establishes a database connection pool, and
// wires together the specific collector and analyzer for MySQL.
//
// Returns:
//   interfaces.MiddlewarePlugin: A new, fully initialized MySQL plugin.
//   error: An error if the database connection fails.
func New() (interfaces.MiddlewarePlugin, error) {
	p := &mysqlPlugin{}
	p.Init("mysql", "0.1.0", "Provides diagnostics for MySQL and compatible databases.")

	// DSN (Data Source Name) format: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	// In a real plugin, this would come from a secure configuration source.
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

	p.Log.Info("MySQL plugin initialized successfully.")
	return p, nil
}

// Diagnose orchestrates the diagnosis process for MySQL by collecting global
// status and configuration variables, and then passing them to the analyzer.
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

// Ping checks the connectivity to the MySQL database by sending a ping request.
func (p *mysqlPlugin) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// HealthCheck performs a basic health check by pinging the database.
func (p *mysqlPlugin) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	if err := p.Ping(ctx); err != nil {
		return &models.HealthStatus{IsHealthy: false, Message: fmt.Sprintf("Failed to ping MySQL: %v", err)}, nil
	}
	return &models.HealthStatus{IsHealthy: true, Message: "MySQL is responsive."}, nil
}

// GetConfiguration retrieves the current server configuration variables.
func (p *mysqlPlugin) GetConfiguration(ctx context.Context) (*models.ConfigData, error) {
	vars, err := p.collector.CollectVariables(ctx)
	if err != nil {
		return nil, err
	}
	return &models.ConfigData{Data: vars}, nil
}

// CollectMetrics gathers key performance indicators from the server.
func (p *mysqlPlugin) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	return p.collector.CollectMetrics(ctx)
}

// CollectLogs provides a placeholder implementation for log collection. A full
// implementation would involve reading the slow query log file or querying the
// `mysql.slow_log` table if `log_output` is set to `TABLE`.
func (p *mysqlPlugin) CollectLogs(ctx context.Context, _ *models.LogOptions) (*models.LogData, error) {
	p.Log.Info("MySQL slow query log collection is a placeholder and not yet fully implemented.")
	return &models.LogData{Entries: []string{}}, nil
}

// Shutdown gracefully closes the underlying database connection pool.
func (p *mysqlPlugin) Shutdown() error {
	p.Log.Info("Shutting down MySQL plugin and closing database connections.")
	return p.db.Close()
}

//Personal.AI order the ending
