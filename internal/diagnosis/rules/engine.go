package rules

import (
	"context"
	"fmt"
	"regexp"

	"github.com/expr-lang/expr"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// RuleEngine evaluates rules
type RuleEngine struct {
	functions map[string]interface{}
}

// EvalContext context for evaluation
type EvalContext struct {
	Metrics     *plugin.MetricsSnapshot
	Config      map[string]interface{}
	SlowLogs    []plugin.SlowLogEntry
	Connections []plugin.ConnectionInfo
	Replication *plugin.ReplicationInfo
	Extra       map[string]interface{}
}

// NewRuleEngine creates a new engine
func NewRuleEngine() *RuleEngine {
	engine := &RuleEngine{
		functions: make(map[string]interface{}),
	}
	engine.registerBuiltinFunctions()
	return engine
}

func (e *RuleEngine) registerBuiltinFunctions() {
	// expr has built-in `any`, `all`, `len`, `filter`, `map` etc.
	// We do not need to override them unless we have specific needs.
	// Overriding them with empty implementation breaks them.
}

// Evaluate evaluates a condition
func (e *RuleEngine) Evaluate(ctx context.Context, condition string, evalCtx *EvalContext) (bool, map[string]interface{}, error) {
	evidence := make(map[string]interface{})

	// Build env
	env := e.buildEnv(evalCtx)

	// Compile
	program, err := expr.Compile(condition, expr.Env(env))
	if err != nil {
		return false, nil, fmt.Errorf("compile condition failed: %w", err)
	}

	// Run
	result, err := expr.Run(program, env)
	if err != nil {
		return false, nil, fmt.Errorf("evaluate condition failed: %w", err)
	}

	matched, ok := result.(bool)
	if !ok {
		return false, nil, fmt.Errorf("condition result is not bool: %T", result)
	}

	if matched {
		evidence = e.collectEvidence(condition, evalCtx)
	}

	return matched, evidence, nil
}

func (e *RuleEngine) buildEnv(evalCtx *EvalContext) map[string]interface{} {
	env := make(map[string]interface{})

	env["metrics"] = e.metricsToMap(evalCtx.Metrics)
	env["config"] = evalCtx.Config
	env["slowlogs"] = evalCtx.SlowLogs
	env["connections"] = evalCtx.Connections
	if evalCtx.Replication != nil {
		// env["replication"] = ... // Convert struct to map if needed or pass struct directly
		// expr can access struct fields if they are exported
		env["replication"] = evalCtx.Replication
	}
	env["extra"] = evalCtx.Extra

	// Built-ins (some are auto-provided by expr, others we can add)
	// expr has `len` built-in, so we don't strictly need to override it unless for custom types

	return env
}

func (e *RuleEngine) metricsToMap(snapshot *plugin.MetricsSnapshot) map[string]float64 {
	if snapshot == nil {
		return make(map[string]float64)
	}
	result := make(map[string]float64)
	for name, metric := range snapshot.Metrics {
		result[name] = metric.Value
	}
	return result
}

func (e *RuleEngine) collectEvidence(condition string, evalCtx *EvalContext) map[string]interface{} {
	evidence := make(map[string]interface{})

	// Regex to find metrics.xxx
	metricsPattern := regexp.MustCompile(`metrics\.(\w+)`)
	matches := metricsPattern.FindAllStringSubmatch(condition, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			name := match[1]
			if val, ok := evalCtx.Metrics.Metrics[name]; ok {
				evidence[name] = val.Value
			}
		}
	}
	return evidence
}
