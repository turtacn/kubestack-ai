package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// MySQLEnhancedPlugin implements the enhanced middleware plugin interface for MySQL
type MySQLEnhancedPlugin struct {
	db        *sql.DB
	target    plugin.MiddlewareTarget
	info      plugin.EnhancedPluginInfo
	config    plugin.PluginConfig
	connected bool
	mu        sync.RWMutex
}

// NewMySQLPlugin creates a new MySQL plugin instance
func NewMySQLPlugin() plugin.Plugin {
	return &MySQLEnhancedPlugin{
		info: plugin.EnhancedPluginInfo{
			ID:          "mysql-diagnostics",
			Name:        "MySQL Diagnostics Plugin",
			Version:     "1.0.0",
			Type:        plugin.PluginTypeMiddleware,
			Description: "Comprehensive MySQL diagnostics and monitoring",
			Author:      "KubeStack AI",
			License:     "Apache-2.0",
			Capabilities: []string{"health-check", "metrics", "diagnose", "slow-query"},
		},
	}
}

func (p *MySQLEnhancedPlugin) Info() plugin.EnhancedPluginInfo                             { return p.info }
func (p *MySQLEnhancedPlugin) Init(ctx context.Context, config plugin.PluginConfig) error { p.config = config; return nil }
func (p *MySQLEnhancedPlugin) Start(ctx context.Context) error                     { return nil }
func (p *MySQLEnhancedPlugin) Stop(ctx context.Context) error                      { return p.Disconnect(ctx) }
func (p *MySQLEnhancedPlugin) HealthCheck(ctx context.Context) error               { 
	if p.db != nil {
		return p.db.PingContext(ctx)
	}
	return fmt.Errorf("not connected")
}
func (p *MySQLEnhancedPlugin) MiddlewareType() string                              { return "mysql" }
func (p *MySQLEnhancedPlugin) SupportedVersions() []string                         { return []string{"5.7", "8.x"} }

func (p *MySQLEnhancedPlugin) Connect(ctx context.Context, target plugin.MiddlewareTarget) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	dsn := p.buildDSN(target)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL connection: %w", err)
	}
	
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping MySQL: %w", err)
	}
	
	p.db = db
	p.target = target
	p.connected = true
	
	return nil
}

func (p *MySQLEnhancedPlugin) Disconnect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.db != nil {
		err := p.db.Close()
		p.db = nil
		p.connected = false
		return err
	}
	return nil
}

func (p *MySQLEnhancedPlugin) Diagnose(ctx context.Context, opts plugin.DiagnoseOptions) (*plugin.DiagnosticResult, error) {
	result := &plugin.DiagnosticResult{
		PluginID:    p.info.ID,
		TargetName:  p.target.Name,
		Status:      plugin.DiagnosticStatusHealthy,
		Findings:    []plugin.Finding{},
		Metrics:     make(map[string]interface{}),
		Suggestions: []string{},
		Timestamp:   time.Now(),
	}
	
	// Basic connection check
	var version string
	err := p.db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	if err == nil {
		result.Metrics["version"] = version
	}
	
	return result, nil
}

func (p *MySQLEnhancedPlugin) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})
	
	var version string
	if err := p.db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err == nil {
		metrics["version"] = version
	}
	
	return metrics, nil
}

func (p *MySQLEnhancedPlugin) Execute(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("action not implemented: %s", action)
}

func (p *MySQLEnhancedPlugin) buildDSN(target plugin.MiddlewareTarget) string {
	user := "root"
	password := ""
	
	if target.Auth != nil {
		if target.Auth.Username != "" {
			user = target.Auth.Username
		}
		if target.Auth.Password != "" {
			password = target.Auth.Password
		}
	}
	
	host := "localhost:3306"
	if len(target.Endpoints) > 0 {
		host = target.Endpoints[0]
	}
	
	return fmt.Sprintf("%s:%s@tcp(%s)/information_schema?parseTime=true&timeout=10s", user, password, host)
}
