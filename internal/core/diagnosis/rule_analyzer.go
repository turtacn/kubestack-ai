package diagnosis

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/analysis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// RuleBasedAnalyzer implements simple rule-based checks.
// This is a v1 implementation that provides basic threshold-based analysis.
// Future versions will be enhanced with more sophisticated rules and ML-based detection.
type RuleBasedAnalyzer struct {
	logger logger.Logger
}

// NewRuleBasedAnalyzer creates a new instance.
// Args are placeholders for now to match caller signature in root.go (nil, nil)
func NewRuleBasedAnalyzer(ruleStore interface{}, log interface{}) interfaces.DiagnosisAnalyzer {
	return &RuleBasedAnalyzer{
		logger: logger.NewLogger("rule-analyzer"),
	}
}

// NewRuleAnalyzer creates a new RuleBasedAnalyzer that implements the analysis.Analyzer interface
func NewRuleAnalyzer() analysis.Analyzer {
	return &RuleBasedAnalyzer{
		logger: logger.NewLogger("rule-analyzer"),
	}
}

func (a *RuleBasedAnalyzer) Name() string { return "RuleBasedAnalyzer" }

// Analyze implements the analysis.Analyzer interface
func (a *RuleBasedAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
	result := analysis.NewAnalysisResult(a.Name())

	var allIssues []*models.Issue

	// Analyze metrics if available
	if data.Metrics != nil {
		issues, err := a.analyzeMetricsInternal(ctx, data.Metrics)
		if err == nil {
			allIssues = append(allIssues, issues...)
		} else {
			a.logger.Warnf("Metrics analysis failed: %v", err)
		}
	}

	// Analyze logs if available
	if data.Logs != nil {
		issues, err := a.analyzeLogsInternal(ctx, data.Logs)
		if err == nil {
			allIssues = append(allIssues, issues...)
		} else {
			a.logger.Warnf("Logs analysis failed: %v", err)
		}
	}

	// Analyze config if available
	if data.Config != nil {
		issues, err := a.analyzeConfigInternal(ctx, data.Config)
		if err == nil {
			allIssues = append(allIssues, issues...)
		} else {
			a.logger.Warnf("Config analysis failed: %v", err)
		}
	}

	result.Issues = allIssues
	result.Summary = fmt.Sprintf("Rule-based analysis completed. Found %d issue(s).", len(allIssues))
	result.Metadata["metrics_analyzed"] = data.Metrics != nil
	result.Metadata["logs_analyzed"] = data.Logs != nil
	result.Metadata["config_analyzed"] = data.Config != nil

	return result, nil
}

// Legacy interface methods for backward compatibility

func (a *RuleBasedAnalyzer) AnalyzeMetrics(ctx context.Context, data *models.MetricsData) ([]*models.Issue, error) {
	return a.analyzeMetricsInternal(ctx, data)
}

func (a *RuleBasedAnalyzer) AnalyzeLogs(ctx context.Context, data *models.LogData) ([]*models.Issue, error) {
	return a.analyzeLogsInternal(ctx, data)
}

func (a *RuleBasedAnalyzer) CorrelateSystems(ctx context.Context, data *models.SystemCorrelationData) ([]*models.Issue, error) {
	// Correlation not yet implemented in v1
	return nil, nil
}

// Internal analysis methods

func (a *RuleBasedAnalyzer) analyzeMetricsInternal(ctx context.Context, data *models.MetricsData) ([]*models.Issue, error) {
	if data == nil || data.Data == nil {
		return nil, nil
	}

	var issues []*models.Issue

	// Example rule: Check for high CPU usage
	if cpuUsage, ok := data.Data["cpu_usage"]; ok {
		if cpuVal, ok := cpuUsage.(float64); ok && cpuVal > 80.0 {
			issue := &models.Issue{
				ID:          fmt.Sprintf("rule-metric-cpu-%d", len(issues)+1),
				Source:      "RuleBasedAnalyzer",
				Title:       "High CPU Usage Detected",
				Severity:    enum.SeverityHigh,
				Description: fmt.Sprintf("CPU usage is at %.2f%%, which exceeds the recommended threshold of 80%%", cpuVal),
				Evidence:    fmt.Sprintf("cpu_usage: %.2f%%", cpuVal),
			}
			issues = append(issues, issue)
		}
	}

	// Example rule: Check for high memory usage
	if memUsage, ok := data.Data["memory_usage"]; ok {
		if memVal, ok := memUsage.(float64); ok && memVal > 85.0 {
			issue := &models.Issue{
				ID:          fmt.Sprintf("rule-metric-mem-%d", len(issues)+1),
				Source:      "RuleBasedAnalyzer",
				Title:       "High Memory Usage Detected",
				Severity:    enum.SeverityHigh,
				Description: fmt.Sprintf("Memory usage is at %.2f%%, which exceeds the recommended threshold of 85%%", memVal),
				Evidence:    fmt.Sprintf("memory_usage: %.2f%%", memVal),
			}
			issues = append(issues, issue)
		}
	}

	return issues, nil
}

func (a *RuleBasedAnalyzer) analyzeLogsInternal(ctx context.Context, data *models.LogData) ([]*models.Issue, error) {
	if data == nil || len(data.Entries) == 0 {
		return nil, nil
	}

	var issues []*models.Issue

	// Example rule: Look for error patterns in logs
	errorCount := 0
	for _, entry := range data.Entries {
		// Simple pattern matching for demonstration
		// In production, this would use more sophisticated regex or NLP
		if contains(entry, "ERROR") || contains(entry, "FATAL") || contains(entry, "Exception") {
			errorCount++
		}
	}

	if errorCount > 10 {
		issue := &models.Issue{
			ID:          "rule-log-errors",
			Source:      "RuleBasedAnalyzer",
			Title:       "High Error Rate in Logs",
			Severity:    enum.SeverityMedium,
			Description: fmt.Sprintf("Found %d error entries in recent logs, which may indicate system instability", errorCount),
			Evidence:    fmt.Sprintf("error_count: %d", errorCount),
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

func (a *RuleBasedAnalyzer) analyzeConfigInternal(ctx context.Context, data *models.ConfigData) ([]*models.Issue, error) {
	if data == nil || data.Data == nil {
		return nil, nil
	}

	var issues []*models.Issue

	// Example rule: Check for insecure configurations
	// This is a placeholder for future rule-based config validation

	return issues, nil
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
