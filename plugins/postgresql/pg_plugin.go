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

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	builtinPG "github.com/kubestack-ai/kubestack-ai/internal/plugins/builtin/postgresql"
	_ "github.com/lib/pq"
)

func init() {
	plugin.RegisterPluginFactory("PostgreSQL", func() plugin.DiagnosticPlugin {
		return &PostgreSQLPlugin{}
	})
}

// PostgreSQLPlugin is the diagnostic plugin implementation for PostgreSQL.
type PostgreSQLPlugin struct {
	db        *sql.DB
	collector *builtinPG.Collector
	analyzer  *builtinPG.Analyzer
	log       logger.Logger
}

func (p *PostgreSQLPlugin) Name() string {
	return "postgresql"
}

func (p *PostgreSQLPlugin) SupportedTypes() []string {
	return []string{"postgresql"}
}

func (p *PostgreSQLPlugin) Version() string {
	return "0.1.0"
}

// Init initializes the plugin with configuration.
func (p *PostgreSQLPlugin) Init(config map[string]interface{}) error {
	p.log = logger.NewLogger("postgresql")

	if config == nil {
		return nil
	}

	dsn, ok := config["dsn"].(string)
	if !ok || dsn == "" {
		return nil
	}

	var err error
	p.db, err = sql.Open("postgres", dsn)
	if err != nil {
		p.log.Warnf("Failed to open postgres connection in Init: %v", err)
		return nil
	}

	// Verify connection
	if err := p.db.Ping(); err != nil {
		p.log.Warnf("Failed to ping postgres in Init: %v", err)
		// We don't return error here to allow lazy connection or transient failures
	}

	p.collector = builtinPG.NewCollector(p.db, p.log)
	p.analyzer = builtinPG.NewAnalyzer(p.log)
	return nil
}

func (p *PostgreSQLPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	var db *sql.DB
	var collector *builtinPG.Collector
	var analyzer *builtinPG.Analyzer
	var log logger.Logger

	if p.log == nil {
		p.log = logger.NewLogger("postgresql")
	}
	log = p.log

	if p.db != nil {
		// Use shared connection
		db = p.db
		collector = p.collector
		analyzer = p.analyzer
	} else {
		// Create temporary connection
		if req.Instance == "" {
			return nil, fmt.Errorf("instance DSN required when not configured globally")
		}

		dsn := req.Instance
		var err error
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open connection: %w", err)
		}
		defer db.Close()

		collector = builtinPG.NewCollector(db, log)
		analyzer = builtinPG.NewAnalyzer(log)
	}

	metricsData, err := collector.CollectMetrics(ctx)
	if err != nil {
		return nil, err
	}

	issues := analyzer.Analyze(metricsData.Data)

	result := &models.DiagnosisResult{
		ID:        fmt.Sprintf("pg-diag-%d", time.Now().Unix()),
		Timestamp: time.Now().UTC(),
		Summary:   fmt.Sprintf("PostgreSQL diagnosis complete. Found %d issues.", len(issues)),
		Issues:    issues,
		Metrics:   metricsData.Data,
	}

	return result, nil
}

func (p *PostgreSQLPlugin) Shutdown() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
