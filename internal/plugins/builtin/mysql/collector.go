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
	"context"
	"database/sql"
	"strconv"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// collector is responsible for gathering raw data and metrics from a MySQL database.
type collector struct {
	db  *sql.DB
	log logger.Logger
}

// newCollector creates a new data collector for MySQL.
//
// Parameters:
//   db (*sql.DB): An active database connection pool.
//   log (logger.Logger): A contextualized logger for the collector.
//
// Returns:
//   *collector: A new instance of the MySQL collector.
func newCollector(db *sql.DB, log logger.Logger) *collector {
	return &collector{db: db, log: log}
}

// queryToMap is a helper function to execute a simple two-column query (key, value)
// and return the result as a map. This is used for `SHOW STATUS` and `SHOW VARIABLES`.
func (c *collector) queryToMap(ctx context.Context, query string) (map[string]string, error) {
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			c.log.Warnf("Failed to scan row for query '%s': %v", query, err)
			continue
		}
		results[key] = value
	}
	return results, nil
}

// CollectGlobalStatus executes `SHOW GLOBAL STATUS` to retrieve MySQL's performance
// counters and state variables.
//
// Returns:
//   map[string]string: A map of status variable names to their string values.
//   error: An error if the query fails.
func (c *collector) CollectGlobalStatus(ctx context.Context) (map[string]string, error) {
	c.log.Info("Collecting MySQL global status.")
	return c.queryToMap(ctx, "SHOW GLOBAL STATUS")
}

// CollectVariables executes `SHOW GLOBAL VARIABLES` to retrieve the current
// server configuration settings.
//
// Returns:
//   map[string]string: A map of configuration variable names to their string values.
//   error: An error if the query fails.
func (c *collector) CollectVariables(ctx context.Context) (map[string]string, error) {
	c.log.Info("Collecting MySQL global variables.")
	return c.queryToMap(ctx, "SHOW GLOBAL VARIABLES")
}

// CollectMetrics derives a standardized set of key performance indicators from the
// raw data collected from `SHOW GLOBAL STATUS`. It converts key status variables
// from strings to numeric types for easier analysis.
//
// Returns:
//   *models.MetricsData: A structured representation of the key metrics.
//   error: An error if the underlying data collection fails.
func (c *collector) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	c.log.Info("Collecting and deriving MySQL metrics.")
	status, err := c.CollectGlobalStatus(ctx)
	if err != nil {
		return nil, err
	}

	metrics := make(map[string]interface{})
	// A selection of important metrics to convert from string to a numeric type.
	// In a real system, this list would be much more extensive and configurable.
	numericMetrics := []string{
		"Threads_connected",
		"Threads_running",
		"Connections",
		"Aborted_connects",
		"Uptime",
		"Innodb_buffer_pool_wait_free",
		"Innodb_log_waits",
		"Slow_queries",
		"Select_full_join",
		"Created_tmp_disk_tables",
		"Created_tmp_files",
	}

	for _, key := range numericMetrics {
		if valStr, ok := status[key]; ok {
			if val, err := strconv.ParseFloat(valStr, 64); err == nil {
				metrics[key] = val
			}
		}
	}

	// Example of a derived metric: Connection usage percentage
	if threadsConnected, tcOK := metrics["Threads_connected"].(float64); tcOK {
		if maxConnectionsStr, mcOK := status["max_connections"]; mcOK {
			if maxConnections, err := strconv.ParseFloat(maxConnectionsStr, 64); err == nil && maxConnections > 0 {
				metrics["Connection_usage_percent"] = (threadsConnected / maxConnections) * 100.0
			}
		}
	}

	return &models.MetricsData{Data: metrics}, nil
}

// TODO: Implement CollectProcessList (`SHOW FULL PROCESSLIST`) to check for long-running queries or lock waits.
// TODO: Implement CollectSlaveStatus (`SHOW SLAVE STATUS`) to check replication health.
// TODO: Implement CollectInnodbStatus (`SHOW ENGINE INNODB STATUS`) to debug InnoDB-specific issues.
// TODO: Implement CollectTableStats (`SELECT table_schema, table_name, data_length, index_length FROM information_schema.tables`) for table size and fragmentation analysis.
// TODO: Implement slow query log collection, which depends on the `slow_query_log` and `log_output` variables (can be FILE or TABLE).

//Personal.AI order the ending
