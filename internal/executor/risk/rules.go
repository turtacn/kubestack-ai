package risk

import (
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// RiskRule 风险规则接口
type RiskRule interface {
	Name() string
	Match(plan *models.ExecutionPlan) bool
	Evaluate(plan *models.ExecutionPlan) *RuleResult
}

// RuleResult 规则评估结果
type RuleResult struct {
	Score      int
	Reason     string
	Mitigation string
}

// === 内置规则实现 ===

// DataDeletionRule 数据删除规则
type DataDeletionRule struct{}

func (r *DataDeletionRule) Name() string {
	return "DataDeletionRule"
}

func (r *DataDeletionRule) Match(plan *models.ExecutionPlan) bool {
	// 检查plan.Steps中是否包含DELETE/TRUNCATE/FLUSHALL等操作
	keywords := []string{"DELETE", "TRUNCATE", "FLUSHALL", "FLUSHDB", "DROP", "DEL"}
	for _, step := range plan.Steps {
		if step.Action == nil {
			continue
		}
		cmd := strings.ToUpper(step.Action.Command)
		for _, kw := range keywords {
			if strings.Contains(cmd, kw) {
				return true
			}
		}
	}
	return false
}

func (r *DataDeletionRule) Evaluate(plan *models.ExecutionPlan) *RuleResult {
	return &RuleResult{
		Score:      70,
		Reason:     "操作包含数据删除命令，存在数据丢失风险",
		Mitigation: "建议先执行备份，或使用dry-run模式预览",
	}
}

// ServiceRestartRule 服务重启规则
type ServiceRestartRule struct{}

func (r *ServiceRestartRule) Name() string {
	return "ServiceRestartRule"
}

func (r *ServiceRestartRule) Match(plan *models.ExecutionPlan) bool {
	keywords := []string{"RESTART", "SHUTDOWN", "KILL", "SYSTEMCTL RESTART", "SERVICE RESTART"}
	for _, step := range plan.Steps {
		if step.Action == nil {
			continue
		}
		cmd := strings.ToUpper(step.Action.Command)
		for _, kw := range keywords {
			if strings.Contains(cmd, kw) {
				return true
			}
		}
	}
	return false
}

func (r *ServiceRestartRule) Evaluate(plan *models.ExecutionPlan) *RuleResult {
	return &RuleResult{
		Score:      40,
		Reason:     "操作包含服务重启命令，可能导致短暂服务不可用",
		Mitigation: "建议在低峰期执行，并确保有服务降级预案",
	}
}

// ConfigChangeRule 配置修改规则
type ConfigChangeRule struct{}

func (r *ConfigChangeRule) Name() string {
	return "ConfigChangeRule"
}

func (r *ConfigChangeRule) Match(plan *models.ExecutionPlan) bool {
	keywords := []string{"CONFIG SET", "SET GLOBAL", "SED", "ECHO"}
	for _, step := range plan.Steps {
		if step.Action == nil {
			continue
		}
		cmd := strings.ToUpper(step.Action.Command)
		for _, kw := range keywords {
			if strings.Contains(cmd, kw) {
				return true
			}
		}
	}
	return false
}

func (r *ConfigChangeRule) Evaluate(plan *models.ExecutionPlan) *RuleResult {
	return &RuleResult{
		Score:      30,
		Reason:     "操作包含配置修改，可能影响服务行为",
		Mitigation: "建议确认配置项和值，并备份原配置文件",
	}
}

// BuiltinRules 返回所有内置规则
func BuiltinRules() []RiskRule {
	return []RiskRule{
		&DataDeletionRule{},
		&ServiceRestartRule{},
		&ConfigChangeRule{},
		// 可扩展更多规则
	}
}
