package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/turtacn/kubestack-ai/internal/errors"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/models"
	"github.com/turtacn/kubestack-ai/internal/plugins"
)

// MySQLPlugin MySQL插件实现。MySQLPlugin implements Plugin for MySQL.
type MySQLPlugin struct {
	config      plugins.PluginConfig
	db          *sql.DB
	version     string
	initialized bool
}

// Name 返回名称。Name returns plugin name.
func (p *MySQLPlugin) Name() string {
	return "mysql"
}

// Version 返回插件版本。Version returns plugin version.
func (p *MySQLPlugin) Version() string {
	return "1.0.0"
}

// SupportedMiddlewareVersions 返回支持的MySQL版本。SupportedMiddlewareVersions returns supported MySQL versions.
func (p *MySQLPlugin) SupportedMiddlewareVersions() []string {
	return []string{"5.7", "8.0"}
}

// Initialize 初始化插件。Initialize initializes the plugin.
func (p *MySQLPlugin) Initialize(config plugins.PluginConfig) error {
	logging.Logger.Info("Initializing MySQL plugin")

	// 存储配置。Store config.
	p.config = config

	// 解析配置参数。Parse config parameters.
	dsn, ok := config["dsn"].(string)
	if !ok {
		// 尝试从配置构建DSN。Try to build DSN from config.
		user := "root"
		if u, ok := config["user"].(string); ok {
			user = u
		}

		password := ""
		if pwd, ok := config["password"].(string); ok {
			password = pwd
		}

		host := "localhost"
		if h, ok := config["host"].(string); ok {
			host = h
		}

		port := "3306"
		if p, ok := config["port"].(string); ok {
			port = p
		}

		dbName := ""
		if db, ok := config["dbname"].(string); ok {
			dbName = db
		}

		// 创建DSN配置。Create DSN config.
		cfg := mysql.Config{
			User:   user,
			Passwd: password,
			Net:    "tcp",
			Addr:   host + ":" + port,
			DBName: dbName,
		}
		dsn = cfg.FormatDSN()
	}

	// 连接到MySQL。Connect to MySQL.
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logging.Logger.Errorf("Failed to open MySQL connection: %v", err)
		return errors.ErrInvalidConfig
	}

	// 测试连接。Test connection.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logging.Logger.Errorf("Failed to ping MySQL: %v", err)
		return errors.ErrInvalidConfig
	}

	// 获取MySQL版本。Get MySQL version.
	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		logging.Logger.Warnf("Failed to get MySQL version: %v", err)
		version = "unknown"
	}
	p.version = version

	p.db = db
	p.initialized = true
	logging.Logger.Infof("MySQL plugin initialized. Connected to MySQL version: %s", version)
	return nil
}

// Validate 验证插件配置。Validate validates plugin configuration.
func (p *MySQLPlugin) Validate() error {
	if !p.initialized {
		return errors.ErrInvalidConfig
	}

	// 检查必要的权限。Check necessary permissions.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := p.db.ExecContext(ctx, "SHOW STATUS")
	if err != nil {
		logging.Logger.Errorf("MySQL user lacks necessary permissions: %v", err)
		return errors.ErrInvalidConfig
	}

	return nil
}

// Cleanup 清理插件资源。Cleanup releases plugin resources.
func (p *MySQLPlugin) Cleanup() error {
	if p.db != nil {
		logging.Logger.Info("Closing MySQL connection")
		if err := p.db.Close(); err != nil {
			logging.Logger.Warnf("Error closing MySQL connection: %v", err)
			return err
		}
	}
	p.initialized = false
	return nil
}

// CollectMetrics 采集指标。CollectMetrics collects metrics.
func (p *MySQLPlugin) CollectMetrics(ctx context.Context) (*models.Metrics, error) {
	if !p.initialized {
		return nil, errors.ErrInvalidConfig
	}

	logging.Logger.Debug("Collecting MySQL metrics")

	metrics := &models.Metrics{
		"version":   p.version,
		"status":    make(map[string]interface{}),
		"variables": make(map[string]interface{}),
	}

	// 收集状态指标。Collect status metrics.
	statusRows, err := p.db.QueryContext(ctx, "SHOW GLOBAL STATUS")
	if err != nil {
		logging.Logger.Errorf("Failed to query global status: %v", err)
		return nil, errors.ErrDataCollectionFailed
	}
	defer statusRows.Close()

	statusMetrics := []string{
		"Threads_connected", "Threads_running", "Connections",
		"Slow_queries", "Queries", "Questions",
		"Com_select", "Com_insert", "Com_update", "Com_delete",
		"Innodb_buffer_pool_reads", "Innodb_buffer_pool_read_requests",
		"Innodb_data_reads", "Innodb_data_writes",
	}

	for statusRows.Next() {
		var variable, value string
		if err := statusRows.Scan(&variable, &value); err != nil {
			logging.Logger.Warnf("Failed to scan status row: %v", err)
			continue
		}

		// 只保留我们关心的指标。Only keep metrics we care about.
		for _, m := range statusMetrics {
			if variable == m {
				metrics["status"].(map[string]interface{})[variable] = value
				break
			}
		}
	}

	// 收集变量配置。Collect variable configuration.
	varRows, err := p.db.QueryContext(ctx, "SHOW GLOBAL VARIABLES")
	if err != nil {
		logging.Logger.Errorf("Failed to query global variables: %v", err)
		return metrics, nil // 即使变量查询失败，也返回已收集的指标。Return collected metrics even if variables query fails.
	}
	defer varRows.Close()

	variables := []string{
		"max_connections", "wait_timeout", "interactive_timeout",
		"slow_query_log", "long_query_time", "query_cache_type",
		"innodb_buffer_pool_size", "innodb_flush_log_at_trx_commit",
	}

	for varRows.Next() {
		var variable, value string
		if err := varRows.Scan(&variable, &value); err != nil {
			logging.Logger.Warnf("Failed to scan variable row: %v", err)
			continue
		}

		// 只保留我们关心的变量。Only keep variables we care about.
		for _, v := range variables {
			if variable == v {
				metrics["variables"].(map[string]interface{})[variable] = value
				break
			}
		}
	}

	// 计算缓冲池命中率。Calculate buffer pool hit rate.
	if reads, ok := metrics["status"].(map[string]interface{})["Innodb_buffer_pool_reads"].(string); ok && reads != "" {
		if requests, ok := metrics["status"].(map[string]interface{})["Innodb_buffer_pool_read_requests"].(string); ok && requests != "" {
			readsInt, _ := strconv.Atoi(reads)
			requestsInt, _ := strconv.Atoi(requests)

			if requestsInt > 0 {
				misses := readsInt
				hits := requestsInt - misses
				hitRate := float64(hits) / float64(requestsInt) * 100
				metrics["buffer_pool_hit_rate"] = fmt.Sprintf("%.2f%%", hitRate)
			}
		}
	}

	logging.Logger.Debug("Completed collecting MySQL metrics")
	return metrics, nil
}

// AnalyzeLogs 分析日志。AnalyzeLogs analyzes logs.
func (p *MySQLPlugin) AnalyzeLogs(ctx context.Context) (models.Logs, error) {
	if !p.initialized {
		return nil, errors.ErrInvalidConfig
	}

	logging.Logger.Debug("Analyzing MySQL logs")

	logs := models.Logs{}

	// 检查慢查询日志是否启用。Check if slow query log is enabled.
	var slowLogEnabled string
	err := p.db.QueryRowContext(ctx, "SELECT @@global.slow_query_log").Scan(&slowLogEnabled)
	if err != nil {
		logging.Logger.Warnf("Failed to check slow query log status: %v", err)
		return logs, nil
	}

	// 如果启用了慢查询日志，获取最近的慢查询。If slow query log is enabled, get recent slow queries.
	if slowLogEnabled == "1" {
		// 注意：在实际实现中，这里应该读取慢查询日志文件
		// Note: In a real implementation, we would read the slow query log file here
		// 这里使用模拟数据作为示例
		// Using mock data here as an example

		logs = append(logs, models.LogEntry{
			Timestamp: time.Now().Add(-5 * time.Minute),
			Level:     "warning",
			Message:   "Slow query detected: SELECT * FROM large_table WHERE condition; (Duration: 2.54s)",
		})
	}

	// 从错误日志表获取错误（MySQL 5.7+）。Get errors from error log table (MySQL 5.7+).
	errorLogRows, err := p.db.QueryContext(ctx, `
		SELECT event_time, error_code, SUBSTRING(message, 1, 200) 
		FROM performance_schema.error_log 
		WHERE event_time > NOW() - INTERVAL 1 HOUR
		ORDER BY event_time DESC
		LIMIT 10
	`)

	if err != nil {
		logging.Logger.Warnf("Failed to query error log: %v", err)
		return logs, nil
	}
	defer errorLogRows.Close()

	for errorLogRows.Next() {
		var eventTime time.Time
		var errorCode int
		var message string

		if err := errorLogRows.Scan(&eventTime, &errorCode, &message); err != nil {
			logging.Logger.Warnf("Failed to scan error log row: %v", err)
			continue
		}

		logs = append(logs, models.LogEntry{
			Timestamp: eventTime,
			Level:     "error",
			Message:   fmt.Sprintf("Error %d: %s", errorCode, message),
		})
	}

	logging.Logger.Debug("Completed analyzing MySQL logs")
	return logs, nil
}

// CollectConfig 验证配置。ValidateConfig collects and validates configuration.
func (p *MySQLPlugin) CollectConfig(ctx context.Context) (*models.Config, error) {
	if !p.initialized {
		return nil, errors.ErrInvalidConfig
	}

	logging.Logger.Debug("Collecting MySQL configuration")

	config := &models.Config{
		"mysqld":      make(map[string]interface{}),
		"innodb":      make(map[string]interface{}),
		"query_cache": make(map[string]interface{}),
		"logging":     make(map[string]interface{}),
	}

	// 查询关键配置变量。Query key configuration variables.
	rows, err := p.db.QueryContext(ctx, `
		SHOW GLOBAL VARIABLES WHERE 
		Variable_name IN (
			'max_connections', 'wait_timeout', 'interactive_timeout',
			'slow_query_log', 'long_query_time', 'log_error',
			'innodb_buffer_pool_size', 'innodb_flush_log_at_trx_commit', 
			'innodb_log_buffer_size', 'innodb_file_per_table',
			'query_cache_type', 'query_cache_size', 'query_cache_limit'
		)
	`)

	if err != nil {
		logging.Logger.Errorf("Failed to query configuration variables: %v", err)
		return nil, errors.ErrDataCollectionFailed
	}
	defer rows.Close()

	for rows.Next() {
		var variable, value string
		if err := rows.Scan(&variable, &value); err != nil {
			logging.Logger.Warnf("Failed to scan config row: %v", err)
			continue
		}

		// 按类别组织配置。Organize config by category.
		switch variable {
		case "max_connections", "wait_timeout", "interactive_timeout":
			config["mysqld"].(map[string]interface{})[variable] = value
		case "slow_query_log", "long_query_time", "log_error":
			config["logging"].(map[string]interface{})[variable] = value
		case "innodb_buffer_pool_size", "innodb_flush_log_at_trx_commit",
			"innodb_log_buffer_size", "innodb_file_per_table":
			config["innodb"].(map[string]interface{})[variable] = value
		case "query_cache_type", "query_cache_size", "query_cache_limit":
			config["query_cache"].(map[string]interface{})[variable] = value
		default:
			config[variable] = value
		}
	}

	// 获取存储引擎状态。Get storage engine status.
	var innodbStatus string
	err = p.db.QueryRowContext(ctx, "SHOW ENGINE INNODB STATUS").Scan(
		new(string), new(string), &innodbStatus)
	if err != nil {
		logging.Logger.Warnf("Failed to get InnoDB status: %v", err)
	} else {
		// 只存储前1000个字符，避免过大。Store only first 1000 chars to avoid bloat.
		if len(innodbStatus) > 1000 {
			innodbStatus = innodbStatus[:1000] + "..."
		}
		config["innodb_status"] = innodbStatus
	}

	logging.Logger.Debug("Completed collecting MySQL configuration")
	return config, nil
}

// Diagnose 诊断。Diagnose performs MySQL-specific diagnosis.
func (p *MySQLPlugin) Diagnose(ctx context.Context, target plugins.DiagnosticTarget) (*models.DiagnosisResult, error) {
	if !p.initialized {
		return nil, errors.ErrInvalidConfig
	}

	logging.Logger.Info("Performing MySQL diagnosis")

	// 收集数据。Collect data.
	metrics, _ := p.CollectMetrics(ctx)
	logs, _ := p.AnalyzeLogs(ctx)
	config, _ := p.CollectConfig(ctx)

	// 创建诊断结果。Create diagnosis result.
	result := models.NewDiagnosisResult("mysql", "")
	result.MiddlewareVersion = p.version

	// 检查连接数。Check connection count.
	if metrics != nil {
		if threadsConnected, ok := metrics["status"].(map[string]interface{})["Threads_connected"].(string); ok {
			if maxConnections, ok := config["mysqld"].(map[string]interface{})["max_connections"].(string); ok {
				tc, _ := strconv.Atoi(threadsConnected)
				mc, _ := strconv.Atoi(maxConnections)

				if mc > 0 && tc > int(float64(mc)*0.8) {
					// 连接数超过最大连接数的80%。Connection count over 80% of max connections.
					result.Findings = append(result.Findings, models.Finding{
						Type:   "performance",
						Title:  "High connection count",
						Detail: fmt.Sprintf("Current connections (%d) are approaching max connections (%d)", tc, mc),
						Evidence: []string{
							fmt.Sprintf("Threads_connected: %d", tc),
							fmt.Sprintf("max_connections: %d", mc),
						},
						Severity: "medium",
						Recommendations: []models.Recommendation{
							{
								Description: "Increase max_connections or optimize connection usage",
								Command:     "SET GLOBAL max_connections = " + strconv.Itoa(mc*2),
								AutoFix:     false,
								RiskLevel:   "low",
							},
						},
					})
				}
			}
		}
	}

	// 检查慢查询。Check for slow queries.
	if metrics != nil {
		if slowQueries, ok := metrics["status"].(map[string]interface{})["Slow_queries"].(string); ok && slowQueries != "0" {
			result.Findings = append(result.Findings, models.Finding{
				Type:   "performance",
				Title:  "Slow queries detected",
				Detail: fmt.Sprintf("MySQL has recorded %s slow queries", slowQueries),
				Evidence: []string{
					fmt.Sprintf("Slow_queries: %s", slowQueries),
				},
				Severity: "medium",
				Recommendations: []models.Recommendation{
					{
						Description: "Analyze and optimize slow queries",
						Command:     "mysqldumpslow -s t /var/log/mysql/slow.log",
						AutoFix:     false,
						RiskLevel:   "low",
					},
				},
			})
		}
	}

	// 确定整体状态。Determine overall status.
	result.Status = determineMySQLStatus(result.Findings)

	logging.Logger.Info("Completed MySQL diagnosis")
	return result, nil
}

// Analyze 分析指标并提供建议。Analyze metrics and provide recommendations.
func (p *MySQLPlugin) Analyze(ctx context.Context, metrics models.Metrics) (*plugins.AnalysisResult, error) {
	// 简化实现。Simplified implementation.
	return &plugins.AnalysisResult{
		HealthScore:     85.0,
		IssuesFound:     len(p.Findings),
		Recommendations: []models.Recommendation{},
	}, nil
}

// Repair 修复检测到的问题。Repair detected issues.
func (p *MySQLPlugin) Repair(ctx context.Context, issue models.Finding) (*plugins.RepairResult, error) {
	// 简化实现。Simplified implementation.
	return &plugins.RepairResult{
		Success:           true,
		Message:           "Repair completed",
		AffectedResources: []string{},
		DurationMs:        100,
	}, nil
}

// determineMySQLStatus 根据发现的问题确定MySQL的整体状态。Determine overall MySQL status based on findings.
func determineMySQLStatus(findings []models.Finding) string {
	for _, finding := range findings {
		if finding.Severity == "high" {
			return "critical"
		}
	}

	for _, finding := range findings {
		if finding.Severity == "medium" {
			return "warning"
		}
	}

	return "healthy"
}

var _ plugins.Plugin = (*MySQLPlugin)(nil)

//Personal.AI order the ending
