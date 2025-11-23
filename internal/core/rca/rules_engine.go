package rca

import (
	"context"
	"sort"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/core/detection/models"
)

// Condition represents a condition for a rule match.
type Condition struct {
	AnomalyType string
	Severity    string // Optional, if empty matches any severity
}

// Rule represents a mapping from symptoms (conditions) to a root cause.
type Rule struct {
	Name       string
	Conditions []Condition
	RootCause  string
	Priority   int
	Actions    []string
}

// RCAResult represents the outcome of Root Cause Analysis.
type RCAResult struct {
	RootCause       string   `json:"root_cause"`
	Recommendations []string `json:"recommendations"`
	Confidence      float64  `json:"confidence"`
	MatchedRule     string   `json:"matched_rule,omitempty"`
}

// Engine is the RCA Rules Engine.
type RulesEngine struct {
	rules []Rule
}

// NewRulesEngine creates a new RulesEngine.
func NewRulesEngine(cfg *config.Config) *RulesEngine {
	var rules []Rule

	if cfg != nil && len(cfg.RCA.Rules) > 0 {
		// Convert config rules to internal rules
		for _, rc := range cfg.RCA.Rules {
			var conditions []Condition
			for _, cc := range rc.Conditions {
				conditions = append(conditions, Condition{
					AnomalyType: cc.AnomalyType,
					Severity:    cc.Severity,
				})
			}
			rules = append(rules, Rule{
				Name:       rc.Name,
				Conditions: conditions,
				RootCause:  rc.RootCause,
				Priority:   rc.Priority,
				Actions:    rc.Actions,
			})
		}
	} else {
		// Initialize with some default rules for now.
		rules = []Rule{
			{
				Name: "High CPU from Connections",
				Conditions: []Condition{
					{AnomalyType: models.AnomalyTypeHighCPU},
					{AnomalyType: models.AnomalyTypeHighConnections},
				},
				RootCause: "Connection Storm",
				Priority:  100,
				Actions:   []string{"Check for client connection leaks", "Increase connection limit if capacity allows", "Implement connection pooling"},
			},
			{
				Name: "High Memory",
				Conditions: []Condition{
					{AnomalyType: models.AnomalyTypeHighMemory},
				},
				RootCause: "Memory Leak or OOM Risk",
				Priority:  50,
				Actions:   []string{"Analyze memory dump", "Check for large keys (Redis)"},
			},
			{
				Name: "Slow Query",
				Conditions: []Condition{
					{AnomalyType: models.AnomalyTypeSlowQuery},
				},
				RootCause: "Unoptimized Query",
				Priority:  80,
				Actions:   []string{"Explain query plan", "Add missing index"},
			},
			// Fallback generic rules
			{
				Name: "High CPU Generic",
				Conditions: []Condition{
					{AnomalyType: models.AnomalyTypeHighCPU},
				},
				RootCause: "High CPU Usage",
				Priority:  10,
				Actions:   []string{"Check top consumers", "Scale up CPU"},
			},
		}
	}

	return &RulesEngine{
		rules: rules,
	}
}

// SetRules allows updating rules dynamically (e.g., from config).
func (e *RulesEngine) SetRules(rules []Rule) {
	e.rules = rules
}

// Analyze performs root cause analysis based on anomalies.
func (e *RulesEngine) Analyze(ctx context.Context, anomalies []models.Anomaly) (*RCAResult, error) {
	var matchedRules []Rule

	for _, rule := range e.rules {
		if e.matchRule(rule, anomalies) {
			matchedRules = append(matchedRules, rule)
		}
	}

	// Sort by priority descending
	sort.Slice(matchedRules, func(i, j int) bool {
		return matchedRules[i].Priority > matchedRules[j].Priority
	})

	if len(matchedRules) > 0 {
		topRule := matchedRules[0]
		return &RCAResult{
			RootCause:       topRule.RootCause,
			Recommendations: topRule.Actions,
			Confidence:      e.calculateConfidence(topRule, anomalies),
			MatchedRule:     topRule.Name,
		}, nil
	}

	return &RCAResult{
		RootCause:  "Unknown",
		Confidence: 0.0,
	}, nil
}

func (e *RulesEngine) matchRule(rule Rule, anomalies []models.Anomaly) bool {
	// All conditions in the rule must be met (AND logic)
	for _, condition := range rule.Conditions {
		conditionMet := false
		for _, anomaly := range anomalies {
			if anomaly.Type == condition.AnomalyType {
				if condition.Severity == "" || anomaly.Severity == condition.Severity {
					conditionMet = true
					break
				}
			}
		}
		if !conditionMet {
			return false
		}
	}
	return true
}

func (e *RulesEngine) calculateConfidence(rule Rule, anomalies []models.Anomaly) float64 {
	// Basic confidence calculation based on rule priority and number of conditions
	// Higher priority rules generally imply higher confidence
	baseConfidence := 0.7
	if rule.Priority > 80 {
		baseConfidence = 0.9
	} else if rule.Priority > 50 {
		baseConfidence = 0.8
	}

	// Add a little boost if we matched multiple specific conditions
	if len(rule.Conditions) > 1 {
		baseConfidence += 0.05
	}

	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	return baseConfidence
}
