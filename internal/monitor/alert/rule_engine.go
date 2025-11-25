package alert

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/types"
)

// RuleEngine manages alert rules
type RuleEngine struct {
	rules []*types.AlertRule
	mu    sync.RWMutex
	log   logger.Logger
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(log logger.Logger) *RuleEngine {
	return &RuleEngine{
		rules: make([]*types.AlertRule, 0),
		log:   log,
	}
}

// LoadRules loads rules from config
func (e *RuleEngine) LoadRules(rulesConfig []config.AlertRuleConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules = make([]*types.AlertRule, 0)
	for _, ruleData := range rulesConfig {
		if err := e.validateExpr(ruleData.Expr); err != nil {
			e.log.Warnf("Skipping invalid rule %s: %v", ruleData.Name, err)
			continue
		}

		rule := &types.AlertRule{
			Name:        ruleData.Name,
			Expr:        ruleData.Expr,
			For:         ruleData.For,
			Severity:    ruleData.Severity,
			Labels:      ruleData.Labels,
			Annotations: ruleData.Annotations,
			Notifiers:   ruleData.Notifiers,
		}
		e.rules = append(e.rules, rule)
	}

	e.log.Infof("Loaded %d alert rules", len(e.rules))
	return nil
}

// validateExpr validates the rule expression
func (e *RuleEngine) validateExpr(expr string) error {
	// Regex match: metric_name operator value
	// Example: cpu_usage > 80, redis_memory_percent >= 90
	// Simplified regex for MVP
	pattern := `^([\w_]+)\s*(>|>=|<|<=|==|!=)\s*([\d.]+)$`
	matched, _ := regexp.MatchString(pattern, expr)
	if !matched {
		return fmt.Errorf("unsupported expression format: %s", expr)
	}
	return nil
}

// GetRules returns all rules
func (e *RuleEngine) GetRules() []*types.AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.rules
}
