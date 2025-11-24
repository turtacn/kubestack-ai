package knowledge

import (
	"fmt"
	"sort"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// ConditionEvaluator evaluates rule conditions against metrics.
type ConditionEvaluator struct{}

// NewConditionEvaluator creates a new ConditionEvaluator.
func NewConditionEvaluator() *ConditionEvaluator {
	return &ConditionEvaluator{}
}

// Evaluate checks if the condition expression evaluates to true given the context.
func (ce *ConditionEvaluator) Evaluate(condition string, context map[string]interface{}) (bool, error) {
	expression, err := govaluate.NewEvaluableExpression(condition)
	if err != nil {
		return false, fmt.Errorf("failed to parse condition '%s': %w", condition, err)
	}

	result, err := expression.Evaluate(context)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate condition '%s': %w", condition, err)
	}

	boolResult, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("condition result is not boolean: %v", result)
	}

	return boolResult, nil
}

// RuleMatch represents a successful match of a rule.
type RuleMatch struct {
	Rule       *Rule
	Confidence float64
	Context    *DiagnosisContext
}

// Recommendation represents a diagnostic recommendation.
type Recommendation struct {
	Title      string                 `json:"title"`
	Action     string                 `json:"action"`
	Priority   int                    `json:"priority"`
	Confidence float64                `json:"confidence"`
	RuleID     string                 `json:"rule_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// DiagnosisContext holds the context for rule matching.
type DiagnosisContext struct {
	MiddlewareType   string
	Namespace        string
	Metrics          map[string]interface{}
	Issues           []*models.Issue
	KnowledgeContext string
}

// RuleEngine orchestrates rule matching and execution.
type RuleEngine struct {
	kb        *KnowledgeBase
	evaluator *ConditionEvaluator
	log       logger.Logger
}

// NewRuleEngine creates a new RuleEngine.
func NewRuleEngine(kb *KnowledgeBase) *RuleEngine {
	return &RuleEngine{
		kb:        kb,
		evaluator: NewConditionEvaluator(),
		log:       logger.NewLogger("rule-engine"),
	}
}

// Match finds all rules that match the current context.
func (re *RuleEngine) Match(ctx *DiagnosisContext) ([]*RuleMatch, error) {
	// 1. Query potentially relevant rules
	rules, err := re.kb.QueryRules(QueryOptions{
		MiddlewareType: ctx.MiddlewareType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}

	var matches []*RuleMatch
	for _, rule := range rules {
		// 2. Evaluate condition
		matched, err := re.evaluator.Evaluate(rule.Condition, ctx.Metrics)
		if err != nil {
			// Log warning but continue with other rules
			re.log.Warnf("Rule %s evaluation failed: %v", rule.ID, err)
			continue
		}

		if matched {
			matches = append(matches, &RuleMatch{
				Rule:       rule,
				Confidence: 1.0, // Default confidence for hard rules
				Context:    ctx,
			})
		}
	}

	// 3. Sort by Priority
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Rule.Priority > matches[j].Rule.Priority
	})

	return matches, nil
}

// Execute processes matches and generates recommendations.
func (re *RuleEngine) Execute(matches []*RuleMatch) ([]*Recommendation, error) {
	var recommendations []*Recommendation

	for _, match := range matches {
		rec := &Recommendation{
			Title:      match.Rule.Name,
			Action:     match.Rule.Recommendation,
			Priority:   match.Rule.Priority,
			Confidence: match.Confidence,
			RuleID:     match.Rule.ID,
			Metadata: map[string]interface{}{
				"rule_version": match.Rule.Version,
				"matched_at":   time.Now(),
			},
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}
