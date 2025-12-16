package mysql

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// MySQLPlugin implementation
type MySQLPlugin struct {
	db        *sql.DB
	config    *plugin.ConnectionConfig
	connected bool
	collector *MetricsCollector
	executor  *CommandExecutor
}

func NewMySQLPlugin(cfg *plugin.PluginConfig) (plugin.MiddlewarePlugin, error) {
	return &MySQLPlugin{
		collector: NewMetricsCollector(),
		executor:  NewCommandExecutor(),
	}, nil
}

// === Basic Information ===

func (p *MySQLPlugin) Name() string { return "MySQL Plugin" }
func (p *MySQLPlugin) Type() plugin.MiddlewareType { return plugin.MiddlewareMySQL }
func (p *MySQLPlugin) Version() string { return "1.0.0" }

// === Connection Management ===

func (p *MySQLPlugin) Connect(ctx context.Context, config *plugin.ConnectionConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=%s&parseTime=true",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.Timeout.String(),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open mysql: %w", err)
	}

	// Pool settings
	db.SetMaxOpenConns(config.PoolSize)
	db.SetMaxIdleConns(config.PoolSize / 2)
	db.SetConnMaxLifetime(config.Timeout * 10)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("mysql ping failed: %w", err)
	}

	p.db = db
	p.config = config
	p.connected = true
	p.collector.SetDB(db)
	p.executor.SetDB(db)

	return nil
}

func (p *MySQLPlugin) Disconnect(ctx context.Context) error {
	if p.db != nil {
		p.connected = false
		return p.db.Close()
	}
	return nil
}

func (p *MySQLPlugin) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("not connected")
	}
	return p.db.PingContext(ctx)
}

func (p *MySQLPlugin) IsConnected() bool {
	return p.connected
}

// === Metrics Collection ===

func (p *MySQLPlugin) CollectMetrics(ctx context.Context) (*plugin.MetricsSnapshot, error) {
	return p.collector.Collect(ctx)
}

func (p *MySQLPlugin) CollectSpecificMetric(ctx context.Context, metricName string) (interface{}, error) {
	return p.collector.CollectSpecific(ctx, metricName)
}

// === Command Execution ===

func (p *MySQLPlugin) Execute(ctx context.Context, cmd *plugin.Command) (*plugin.CommandResult, error) {
	return p.executor.Execute(ctx, cmd)
}

func (p *MySQLPlugin) SupportedCommands() []plugin.CommandSpec {
	return []plugin.CommandSpec{
		{Name: "SHOW STATUS", Description: "Show server status", RiskLevel: 1},
		{Name: "SHOW VARIABLES", Description: "Show variables", RiskLevel: 1},
		{Name: "SHOW PROCESSLIST", Description: "Show processlist", RiskLevel: 1},
		{Name: "KILL", Description: "Kill connection", RiskLevel: 3},
		{Name: "OPTIMIZE TABLE", Description: "Optimize table", RiskLevel: 3},
	}
}

// === Diagnosis Support ===

func (p *MySQLPlugin) GetDiagnosticData(ctx context.Context) (*plugin.DiagnosticData, error) {
	data := &plugin.DiagnosticData{
		Extra: make(map[string]interface{}),
	}

	metrics, err := p.CollectMetrics(ctx)
	if err != nil {
		return nil, err
	}
	data.Metrics = metrics
	data.Config = p.collector.GetVariables(ctx)
	data.Connections = p.collector.GetProcessList(ctx)
	// Slow logs etc. would be added here

	return data, nil
}

func (p *MySQLPlugin) GetBuiltinRules() []plugin.DiagnosisRule {
	return mysqlBuiltinRules
}

var mysqlBuiltinRules = []plugin.DiagnosisRule{
	{
		ID:          "mysql-connections-high",
		Name:        "High Connection Usage",
		Severity:    plugin.SeverityWarning,
		Condition:   "metrics.Threads_connected / config.max_connections > 0.8",
		Message:     "Connection usage > 80%",
		Suggestion:  "Check for leaks or increase max_connections",
	},
}
