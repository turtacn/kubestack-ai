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
	"strconv"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	_ "github.com/lib/pq"
)

// Collector is responsible for gathering raw data and metrics from a PostgreSQL database.
type Collector struct {
	db  *sql.DB
	log logger.Logger
}

// NewCollector creates a new data collector for PostgreSQL.
func NewCollector(db *sql.DB, log logger.Logger) *Collector {
	return &Collector{db: db, log: log}
}

// CollectMetrics gathers key performance indicators from PostgreSQL.
func (c *Collector) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	c.log.Info("Collecting PostgreSQL metrics.")
	metrics := make(map[string]interface{})

	// 1. Connection stats from pg_stat_activity
	connQuery := `
		SELECT
			count(*) as total_connections,
			count(*) FILTER (WHERE state = 'active') as active_connections,
			count(*) FILTER (WHERE state = 'idle in transaction') as idle_in_transaction,
			count(*) FILTER (WHERE state = 'idle in transaction' AND state_change < NOW() - INTERVAL '5 minutes') as long_idle_tx
		FROM pg_stat_activity
	`
	var total, active, idleTx, longIdleTx sql.NullInt64
	err := c.db.QueryRowContext(ctx, connQuery).Scan(&total, &active, &idleTx, &longIdleTx)
	if err != nil {
		c.log.Warnf("Failed to query pg_stat_activity: %v", err)
		return nil, fmt.Errorf("failed to query pg_stat_activity: %w", err)
	}

	metrics["total_connections"] = total.Int64
	metrics["active_connections"] = active.Int64
	metrics["idle_in_transaction"] = idleTx.Int64
	metrics["long_idle_tx"] = longIdleTx.Int64

	// 2. Database stats from pg_stat_database
	dbQuery := `
		SELECT
			sum(blks_hit) as blks_hit,
			sum(blks_read) as blks_read,
			sum(xact_commit) as xact_commit,
			sum(xact_rollback) as xact_rollback
		FROM pg_stat_database
	`
	var blksHit, blksRead, xactCommit, xactRollback sql.NullInt64
	err = c.db.QueryRowContext(ctx, dbQuery).Scan(&blksHit, &blksRead, &xactCommit, &xactRollback)
	if err != nil {
		c.log.Warnf("Failed to query pg_stat_database: %v", err)
		// Don't fail completely, just log
	} else {
		metrics["blks_hit"] = blksHit.Int64
		metrics["blks_read"] = blksRead.Int64
		metrics["xact_commit"] = xactCommit.Int64
		metrics["xact_rollback"] = xactRollback.Int64

		totalOps := blksHit.Int64 + blksRead.Int64
		if totalOps > 0 {
			metrics["cache_hit_ratio"] = float64(blksHit.Int64) / float64(totalOps)
		} else {
			metrics["cache_hit_ratio"] = 0.0
		}
	}

	// 3. Get max_connections
	var maxConns string
	err = c.db.QueryRowContext(ctx, "SHOW max_connections").Scan(&maxConns)
	if err == nil {
		if val, err := strconv.Atoi(maxConns); err == nil {
			metrics["max_connections"] = float64(val) // Use float64 for consistency in metrics
			if val > 0 {
				metrics["connection_usage_percent"] = (float64(total.Int64) / float64(val)) * 100.0
			}
		}
	} else {
		c.log.Warnf("Failed to get max_connections: %v", err)
	}

	return &models.MetricsData{Data: metrics}, nil
}

// CollectVariables collects configuration variables from pg_settings.
func (c *Collector) CollectVariables(ctx context.Context) (map[string]string, error) {
	rows, err := c.db.QueryContext(ctx, "SELECT name, setting FROM pg_settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vars := make(map[string]string)
	for rows.Next() {
		var name, setting string
		if err := rows.Scan(&name, &setting); err != nil {
			continue
		}
		vars[name] = setting
	}
	return vars, nil
}
