package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

func init() {
	plugin.RegisterPluginFactory("MySQL", func() plugin.DiagnosticPlugin {
		return &MySQLPlugin{}
	})
}

// MySQLPlugin MySQL诊断插件
type MySQLPlugin struct {
	db *sql.DB
}

func (p *MySQLPlugin) Name() string {
	return "mysql"
}

func (p *MySQLPlugin) SupportedTypes() []string {
	return []string{"mysql", "mariadb"}
}

func (p *MySQLPlugin) Version() string {
	return "1.0.0"
}

func (p *MySQLPlugin) Init(config map[string]interface{}) error {
	dsn, ok := config["dsn"].(string)
	if !ok {
		return fmt.Errorf("config 'dsn' is required")
	}
	var err error
	p.db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("MySQL连接失败: %w", err)
	}
	return p.db.Ping()
}

func (p *MySQLPlugin) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	result := &models.DiagnosisResult{
		Issues: []*models.Issue{},
	}

	// Step 1: 检查慢查询
	slowQueryIssue := p.checkSlowQueries(ctx)
	if slowQueryIssue != nil {
		result.Issues = append(result.Issues, slowQueryIssue)
	}

	// Step 2: 检查死锁
	deadlockIssue := p.checkDeadlocks(ctx)
	if deadlockIssue != nil {
		result.Issues = append(result.Issues, deadlockIssue)
	}

	// Step 3: 检查连接池
	connectionIssue := p.checkConnections(ctx)
	if connectionIssue != nil {
		result.Issues = append(result.Issues, connectionIssue)
	}

	p.attachRecommendations(result.Issues)
	return result, nil
}

func (p *MySQLPlugin) checkSlowQueries(ctx context.Context) *models.Issue {
	var count int
	// Note: This query might fail if user doesn't have permissions or table doesn't exist.
	// We wrap in a simple check.
	err := p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM mysql.slow_log WHERE start_time > NOW() - INTERVAL 1 HOUR").Scan(&count)
	if err != nil {
		// Log error but don't fail diagnosis
		return nil
	}

	if count > 100 {
		return &models.Issue{
			Title:       "MySQL慢查询过多",
			Severity:    enum.SeverityHigh,
			Description: fmt.Sprintf("最近1小时慢查询数: %d", count),
			Source:      "MySQLPlugin",
		}
	}
	return nil
}

func (p *MySQLPlugin) checkDeadlocks(ctx context.Context) *models.Issue {
	var status string
	var trash interface{}
	// SHOW ENGINE INNODB STATUS returns multiple columns, Status is usually the 3rd one (Type, Name, Status)
	row := p.db.QueryRowContext(ctx, "SHOW ENGINE INNODB STATUS")
	err := row.Scan(&trash, &trash, &status)
	if err != nil {
		return nil
	}

	if strings.Contains(status, "LATEST DETECTED DEADLOCK") {
		return &models.Issue{
			Title:       "MySQL检测到死锁",
			Severity:    enum.SeverityHigh,
			Description: "InnoDB引擎状态中包含死锁记录",
			Source:      "MySQLPlugin",
		}
	}
	return nil
}

func (p *MySQLPlugin) checkConnections(ctx context.Context) *models.Issue {
	var current, max int
	var trash interface{}
	p.db.QueryRowContext(ctx, "SHOW STATUS LIKE 'Threads_connected'").Scan(&trash, &current) // Variable_name, Value
	// The Value comes as string usually
	var currentStr, maxStr string
	p.db.QueryRowContext(ctx, "SHOW STATUS LIKE 'Threads_connected'").Scan(&trash, &currentStr)
	p.db.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'max_connections'").Scan(&trash, &maxStr)

	// Simple conversion
	fmt.Sscanf(currentStr, "%d", &current)
	fmt.Sscanf(maxStr, "%d", &max)

	if max > 0 && float64(current)/float64(max) > 0.8 {
		return &models.Issue{
			Title:       "MySQL连接数接近上限",
			Severity:    enum.SeverityMedium,
			Description: fmt.Sprintf("当前连接: %d, 最大连接: %d", current, max),
			Source:      "MySQLPlugin",
		}
	}
	return nil
}

func (p *MySQLPlugin) attachRecommendations(issues []*models.Issue) {
	for _, issue := range issues {
		var recs []*models.Recommendation
		if strings.Contains(issue.Title, "慢查询") {
			recs = append(recs, &models.Recommendation{
				Description: "优化SQL查询，添加合适的索引",
				Fix: models.FixAction{
					Description: "优化SQL查询，添加合适的索引",
					Category:    "Optimization",
				},
			})
		}
		if strings.Contains(issue.Title, "死锁") {
			recs = append(recs, &models.Recommendation{
				Description: "检查事务逻辑，避免长时间持有锁",
				Fix: models.FixAction{
					Description: "检查事务逻辑，避免长时间持有锁",
					Category:    "Application",
				},
			})
		}
		issue.Recommendations = recs
	}
}

func (p *MySQLPlugin) Shutdown() error {
	return p.db.Close()
}
