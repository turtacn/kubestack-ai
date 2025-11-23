package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

type MySQLPlugin struct {
	config *MySQLConfig
	db     *sql.DB
	logger *zap.Logger
}

type MySQLConfig struct {
	DSN          string // Data Source Name
	MaxOpenConns int
	MaxIdleConns int
}

func (p *MySQLPlugin) Name() string { return "mysql" }
func (p *MySQLPlugin) Version() string { return "1.0.0" }
func (p *MySQLPlugin) Description() string { return "MySQL diagnostic plugin" }
func (p *MySQLPlugin) SupportedMiddlewareVersions() []string {
	return []string{"5.7", "8.0", "8.1"}
}

func (p *MySQLPlugin) Initialize(config *plugin.PluginConfig) error {
	p.logger = zap.L().With(zap.String("plugin", "mysql"))
	var mysqlConf MySQLConfig
	if err := mapstructure.Decode(config.Settings, &mysqlConf); err != nil {
		return err
	}

	db, err := sql.Open("mysql", mysqlConf.DSN)
	if err != nil {
		return fmt.Errorf("mysql connection failed: %w", err)
	}

	db.SetMaxOpenConns(mysqlConf.MaxOpenConns)
	db.SetMaxIdleConns(mysqlConf.MaxIdleConns)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("mysql ping failed: %w", err)
	}

	p.db = db
	p.config = &mysqlConf
	return nil
}

func (p *MySQLPlugin) Shutdown() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *MySQLPlugin) Collector() plugin.DataCollector {
	return &MySQLDataCollector{plugin: p}
}

func (p *MySQLPlugin) Parser() plugin.MetricParser {
	return &MySQLMetricParser{plugin: p}
}

func (p *MySQLPlugin) HealthChecker() plugin.HealthChecker {
	return &MySQLHealthChecker{plugin: p}
}

// MySQLDataCollector Implementation
type MySQLDataCollector struct {
	plugin *MySQLPlugin
}

func (c *MySQLDataCollector) Collect(ctx context.Context, target *plugin.Target) (*plugin.CollectedData, error) {
	rows, err := c.plugin.db.QueryContext(ctx, "SHOW GLOBAL STATUS")
	if err != nil {
		return nil, err
	}
	statusVars := c.parseKeyValueRows(rows)

	// SHOW PROCESSLIST
	pRows, err := c.plugin.db.QueryContext(ctx, "SHOW FULL PROCESSLIST")
	var processList []map[string]interface{}
	if err == nil {
		processList = c.parseProcessList(pRows)
	}

	// Slow Queries from mysql.slow_log if available
	slowQueries := c.collectSlowQueries(ctx)

	// SHOW VARIABLES for max_connections etc
	vRows, err := c.plugin.db.QueryContext(ctx, "SHOW VARIABLES")
	variables := make(map[string]string)
	if err == nil {
		variables = c.parseKeyValueRows(vRows)
	}

	return &plugin.CollectedData{
		PluginName: "mysql",
		Target:     target,
		Timestamp:  time.Now(),
		RawData: map[string]interface{}{
			"status":       statusVars,
			"variables":    variables,
			"processlist":  processList,
			"slow_queries": slowQueries,
		},
	}, nil
}

func (c *MySQLDataCollector) collectSlowQueries(ctx context.Context) []map[string]interface{} {
	// Try to query mysql.slow_log table. Limit 10 recent.
	// This only works if log_output includes TABLE and user has access.
	query := "SELECT start_time, query_time, rows_sent, rows_examined, sql_text FROM mysql.slow_log ORDER BY start_time DESC LIMIT 10"
	rows, err := c.plugin.db.QueryContext(ctx, query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var startTime time.Time
		var rowsSent, rowsExamined int
		var sqlText string // sql_text is blob/text

		// In some versions query_time is time.Duration (Time type) or float.
		// Actually in mysql.slow_log it is Time type.
		// We'll try to scan into generic vars if types vary, but standard is Time.
		// Let's use []byte for time to be safe and parse string
		var qtBytes []byte

		if err := rows.Scan(&startTime, &qtBytes, &rowsSent, &rowsExamined, &sqlText); err == nil {
			// Parse time "00:00:01.234"
			qtStr := string(qtBytes)
			// Simple heuristic
			results = append(results, map[string]interface{}{
				"start_time":    startTime,
				"query_time_str": qtStr,
				"rows_sent":     rowsSent,
				"rows_examined": rowsExamined,
				"sql":           sqlText,
			})
		}
	}
	return results
}

func (c *MySQLDataCollector) parseKeyValueRows(rows *sql.Rows) map[string]string {
	defer rows.Close()
	result := make(map[string]string)
	var key, value string
	for rows.Next() {
		if err := rows.Scan(&key, &value); err == nil {
			result[key] = value
		}
	}
	return result
}

func (c *MySQLDataCollector) parseProcessList(rows *sql.Rows) []map[string]interface{} {
	defer rows.Close()
	columns, _ := rows.Columns()
	var result []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err == nil {
			entry := make(map[string]interface{})
			for i, col := range columns {
				var v interface{}
				val := values[i]
				if b, ok := val.([]byte); ok {
					v = string(b)
				} else {
					v = val
				}
				entry[col] = v
			}
			result = append(result, entry)
		}
	}
	return result
}

func (c *MySQLDataCollector) SupportedDataSources() []plugin.DataSourceType {
	return []plugin.DataSourceType{plugin.DataSourceCommand}
}

// MySQLMetricParser Implementation
type MySQLMetricParser struct {
	plugin *MySQLPlugin
}

func (p *MySQLMetricParser) Parse(ctx context.Context, data *plugin.CollectedData) (*plugin.ParsedMetrics, error) {
	status := data.RawData["status"].(map[string]string)
	variables := data.RawData["variables"].(map[string]string)

	queries, _ := strconv.ParseInt(status["Queries"], 10, 64)
	uptime, _ := strconv.ParseInt(status["Uptime"], 10, 64)
	var qps float64
	if uptime > 0 {
		qps = float64(queries) / float64(uptime)
	}

	threadsConnected, _ := strconv.ParseInt(status["Threads_connected"], 10, 64)
	maxConnections, _ := strconv.ParseInt(variables["max_connections"], 10, 64)

	connUsage := 0.0
	if maxConnections > 0 {
		connUsage = float64(threadsConnected) / float64(maxConnections)
	}

	metrics := map[string]*plugin.MetricValue{
		"qps":               {Value: qps, Unit: "queries/s"},
		"threads_connected": {Value: threadsConnected, Unit: "count"},
		"connection_usage":  {Value: connUsage, Unit: "ratio"},
	}

	return &plugin.ParsedMetrics{
		PluginName: "mysql",
		Timestamp:  time.Now(),
		Metrics:    metrics,
	}, nil
}

func (p *MySQLMetricParser) AvailableMetrics() []plugin.MetricDefinition {
	return []plugin.MetricDefinition{
		{Name: "qps"},
		{Name: "threads_connected"},
		{Name: "connection_usage"},
	}
}

// MySQLHealthChecker Implementation
type MySQLHealthChecker struct {
	plugin *MySQLPlugin
}

func (c *MySQLHealthChecker) Check(ctx context.Context, target *plugin.Target) (*plugin.HealthStatus, error) {
	pingResult := &plugin.HealthCheckResult{Name: "connectivity"}
	if err := c.plugin.db.PingContext(ctx); err != nil {
		pingResult.Status = plugin.UnhealthyLevel
		pingResult.Message = "DB ping failed: " + err.Error()
	} else {
		pingResult.Status = plugin.HealthyLevel
	}

	connResult := c.checkConnections(ctx)
	replResult := c.checkReplication(ctx)

	items := []*plugin.HealthCheckResult{pingResult, connResult, replResult}
	overall := c.calculateOverallHealth(items)

	return &plugin.HealthStatus{
		PluginName: "mysql",
		Overall:    overall,
		Items:      items,
		Timestamp:  time.Now(),
	}, nil
}

func (c *MySQLHealthChecker) checkConnections(ctx context.Context) *plugin.HealthCheckResult {
	// We need 2 queries: Threads_connected and max_connections
	// It's cleaner to query them explicitly here for the check rather than relying on Collector data
	// to ensure HealthCheck is standalone (though Collector reuse is efficient, the interface pattern allows separate).

	var connected, max int

	// Get Threads_connected
	row := c.plugin.db.QueryRowContext(ctx, "SHOW GLOBAL STATUS LIKE 'Threads_connected'")
	var name, value string
	if err := row.Scan(&name, &value); err != nil {
		return &plugin.HealthCheckResult{Name: "connections", Status: plugin.UnhealthyLevel, Message: "Failed to query status"}
	}
	connected, _ = strconv.Atoi(value)

	// Get max_connections
	row = c.plugin.db.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'max_connections'")
	if err := row.Scan(&name, &value); err != nil {
		return &plugin.HealthCheckResult{Name: "connections", Status: plugin.UnhealthyLevel, Message: "Failed to query variables"}
	}
	max, _ = strconv.Atoi(value)

	if max > 0 {
		usage := float64(connected) / float64(max)
		if usage > 0.90 {
			return &plugin.HealthCheckResult{
				Name: "connections",
				Status: plugin.UnhealthyLevel,
				Message: fmt.Sprintf("Connection usage high: %d/%d (%.2f%%)", connected, max, usage*100),
			}
		}
	}

	return &plugin.HealthCheckResult{Name: "connections", Status: plugin.HealthyLevel}
}

func (c *MySQLHealthChecker) checkReplication(ctx context.Context) *plugin.HealthCheckResult {
	rows, err := c.plugin.db.QueryContext(ctx, "SHOW SLAVE STATUS")
	if err != nil {
		return &plugin.HealthCheckResult{Name: "replication", Status: plugin.UnhealthyLevel, Message: err.Error()}
	}
	defer rows.Close()

	if !rows.Next() {
		// No rows means it's not a slave (or master)
		// We consider this Healthy (Not Applicable)
		return &plugin.HealthCheckResult{Name: "replication", Status: plugin.HealthyLevel, Message: "Not a slave"}
	}

	cols, _ := rows.Columns()
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	rows.Scan(valuePtrs...)

	rowMap := make(map[string]string)
	for i, col := range cols {
		if b, ok := values[i].([]byte); ok {
			rowMap[col] = string(b)
		} else if values[i] != nil {
			rowMap[col] = fmt.Sprintf("%v", values[i])
		}
	}

	ioRunning := rowMap["Slave_IO_Running"]
	sqlRunning := rowMap["Slave_SQL_Running"]

	if ioRunning != "Yes" || sqlRunning != "Yes" {
		return &plugin.HealthCheckResult{
			Name: "replication",
			Status: plugin.UnhealthyLevel,
			Message: fmt.Sprintf("Replication broken. IO: %s, SQL: %s", ioRunning, sqlRunning),
		}
	}

	secondsBehind := rowMap["Seconds_Behind_Master"]
	if sec, err := strconv.Atoi(secondsBehind); err == nil && sec > 60 {
		return &plugin.HealthCheckResult{
			Name: "replication",
			Status: plugin.DegradedLevel,
			Message: fmt.Sprintf("Replication lag: %d seconds", sec),
		}
	}

	return &plugin.HealthCheckResult{Name: "replication", Status: plugin.HealthyLevel}
}

func (c *MySQLHealthChecker) calculateOverallHealth(items []*plugin.HealthCheckResult) plugin.HealthLevel {
	overall := plugin.HealthyLevel
	for _, item := range items {
		if item.Status > overall {
			overall = item.Status
		}
	}
	return overall
}

func (c *MySQLHealthChecker) CheckItems() []plugin.HealthCheckItem {
	return []plugin.HealthCheckItem{
		{Name: "connectivity"},
		{Name: "connections"},
		{Name: "replication"},
	}
}

func init() {
	plugin.RegisterPlugin(&MySQLPluginFactory{})
}

type MySQLPluginFactory struct{}

func (f *MySQLPluginFactory) Create() plugin.Plugin {
	return &MySQLPlugin{}
}

func (f *MySQLPluginFactory) Metadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:       "mysql",
		Version:    "1.0.0",
		APIVersion: "v1",
		Description: "MySQL plugin",
	}
}
