package diagnosis

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/diagnosis/rules"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// DiagnosisEngine
type DiagnosisEngine struct {
	pluginRegistry *plugin.PluginRegistry
	ruleEngine     *rules.RuleEngine
	analyzers      []Analyzer
	log            logger.Logger
}

// DiagnosisRequest
type DiagnosisRequest struct {
	MiddlewareType plugin.MiddlewareType
	InstanceID     string
	CustomRules    []plugin.DiagnosisRule
}

// DiagnosisResult
type DiagnosisResult struct {
	RequestID      string
	MiddlewareType plugin.MiddlewareType
	InstanceID     string
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	Issues         []Issue
	Summary        string
	HealthScore    int
	DiagnosticData *plugin.DiagnosticData
}

// Issue
type Issue struct {
	RuleID      string
	Name        string
	Severity    plugin.Severity
	Description string
	Suggestion  string
	Evidence    map[string]interface{}
	DetectedAt  time.Time
}

// Option
type EngineOption func(*DiagnosisEngine)

// NewDiagnosisEngine
func NewDiagnosisEngine(registry *plugin.PluginRegistry, opts ...EngineOption) *DiagnosisEngine {
	engine := &DiagnosisEngine{
		pluginRegistry: registry,
		ruleEngine:     rules.NewRuleEngine(),
		analyzers:      make([]Analyzer, 0),
		log:            logger.NewLogger("DiagnosisEngine"),
	}

	// Note: History store is nil for now as it's not implemented yet
	engine.analyzers = append(engine.analyzers,
		NewThresholdAnalyzer(),
		NewAnomalyAnalyzer(nil),
		NewTrendAnalyzer(nil),
	)

	for _, opt := range opts {
		opt(engine)
	}

	return engine
}

// Diagnose
func (e *DiagnosisEngine) Diagnose(ctx context.Context, req *DiagnosisRequest) (*DiagnosisResult, error) {
	result := &DiagnosisResult{
		RequestID:      uuid.New().String(),
		MiddlewareType: req.MiddlewareType,
		InstanceID:     req.InstanceID,
		StartTime:      time.Now(),
		Issues:         make([]Issue, 0),
	}

	// 1. Get Plugin
	p, err := e.pluginRegistry.GetPlugin(req.MiddlewareType)
	if err != nil {
		return nil, fmt.Errorf("plugin not found: %w", err)
	}

	// 2. Collect Data
	diagData, err := p.GetDiagnosticData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect data: %w", err)
	}
	result.DiagnosticData = diagData

	// 3. Get Rules
	allRules := p.GetBuiltinRules()
	if len(req.CustomRules) > 0 {
		allRules = append(allRules, req.CustomRules...)
	}

	// 4. Rule Evaluation
	ruleContext := &rules.EvalContext{
		Metrics:     diagData.Metrics,
		Config:      diagData.Config,
		SlowLogs:    diagData.SlowLogs,
		Connections: diagData.Connections,
		Replication: diagData.Replication,
		Extra:       diagData.Extra,
	}

	for _, rule := range allRules {
		matched, evidence, err := e.ruleEngine.Evaluate(ctx, rule.Condition, ruleContext)
		if err != nil {
			e.log.Warn("rule evaluation failed", "ruleID", rule.ID, "error", err)
			continue
		}

		if matched {
			issue := Issue{
				RuleID:      rule.ID,
				Name:        rule.Name,
				Severity:    rule.Severity,
				Description: e.renderTemplate(rule.Message, evidence),
				Suggestion:  e.renderTemplate(rule.Suggestion, evidence),
				Evidence:    evidence,
				DetectedAt:  time.Now(),
			}
			result.Issues = append(result.Issues, issue)
		}
	}

	// 5. Analyzers
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, analyzer := range e.analyzers {
		wg.Add(1)
		go func(a Analyzer) {
			defer wg.Done()
			issues, err := a.Analyze(ctx, diagData)
			if err != nil {
				e.log.Warn("analyzer failed", "analyzer", a.Name(), "error", err)
				return
			}
			if len(issues) > 0 {
				mu.Lock()
				result.Issues = append(result.Issues, issues...)
				mu.Unlock()
			}
		}(analyzer)
	}
	wg.Wait()

	// 6. Finalize
	e.sortIssuesBySeverity(result.Issues)
	result.HealthScore = e.calculateHealthScore(result.Issues)
	result.Summary = e.generateSummary(result)
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

func (e *DiagnosisEngine) sortIssuesBySeverity(issues []Issue) {
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Severity > issues[j].Severity
	})
}

func (e *DiagnosisEngine) calculateHealthScore(issues []Issue) int {
	score := 100
	for _, issue := range issues {
		switch issue.Severity {
		case plugin.SeverityCritical:
			score -= 30
		case plugin.SeverityError:
			score -= 20
		case plugin.SeverityWarning:
			score -= 10
		case plugin.SeverityInfo:
			score -= 2
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}

func (e *DiagnosisEngine) generateSummary(result *DiagnosisResult) string {
	if len(result.Issues) == 0 {
		return "System is healthy."
	}
	counts := make(map[plugin.Severity]int)
	for _, issue := range result.Issues {
		counts[issue.Severity]++
	}
	return fmt.Sprintf("Found %d Critical, %d Error, %d Warning issues.",
		counts[plugin.SeverityCritical], counts[plugin.SeverityError], counts[plugin.SeverityWarning])
}

func (e *DiagnosisEngine) renderTemplate(tmpl string, data map[string]interface{}) string {
	t, err := template.New("msg").Parse(tmpl)
	if err != nil {
		return tmpl
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return tmpl
	}
	return buf.String()
}
