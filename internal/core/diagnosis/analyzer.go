// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package diagnosis

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	llm_interfaces "github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// --- Rule-Based Analyzer Implementation ---

// MetricRule defines a simple, declarative rule for evaluating a single system metric
// against a static threshold.
type MetricRule struct {
	// MetricName is the key of the metric to be evaluated (e.g., "cpu_usage_percent").
	MetricName string
	// Threshold is the value to compare the metric against.
	Threshold float64
	// Operator is the comparison operator to use (e.g., ">", "<", "==").
	Operator string
	// Severity is the severity level to assign to the issue if the rule is triggered.
	Severity enum.SeverityLevel
	// IssueTitle is the title of the issue to be created if the rule is triggered.
	IssueTitle string
	// Recommendation is the suggested action to resolve the issue.
	Recommendation string
}

// LogRule defines a simple, declarative rule for identifying issues by matching
// a pattern within log entries.
type LogRule struct {
	// Pattern is the substring to search for within log entries (case-insensitive).
	Pattern string
	// Severity is the severity level to assign to the issue if the pattern is found.
	Severity enum.SeverityLevel
	// IssueTitle is the title of the issue to be created if the pattern is found.
	IssueTitle string
	// Recommendation is the suggested action to resolve the issue.
	Recommendation string
}

// RuleBasedAnalyzer is a concrete implementation of the DiagnosisAnalyzer interface that
// uses a predefined set of metric and log rules to identify common issues. This
// component functions as a simple, stateless rule engine.
type RuleBasedAnalyzer struct {
	log         logger.Logger
	metricRules []MetricRule
	logRules    []LogRule
}

// NewRuleBasedAnalyzer creates a new, configured instance of the RuleBasedAnalyzer.
//
// Parameters:
//   metricRules ([]MetricRule): A slice of metric-based rules to be used for analysis.
//   logRules ([]LogRule): A slice of log-based rules to be used for analysis.
//
// Returns:
//   interfaces.DiagnosisAnalyzer: A new analyzer ready to process data.
func NewRuleBasedAnalyzer(metricRules []MetricRule, logRules []LogRule) interfaces.DiagnosisAnalyzer {
	return &RuleBasedAnalyzer{
		log:         logger.NewLogger("rule-analyzer"),
		metricRules: metricRules,
		logRules:    logRules,
	}
}

// Name returns the unique identifier for this analyzer.
func (a *RuleBasedAnalyzer) Name() string {
	return "RuleBasedAnalyzer"
}

// AnalyzeMetrics iterates through the analyzer's metric rules and evaluates them
// against the provided metric data. For each rule that is triggered, it creates
// a corresponding issue.
//
// Parameters:
//   _ (context.Context): The context for the operation (currently unused).
//   data (*models.MetricsData): The collected metric data to be analyzed.
//
// Returns:
//   []*models.Issue: A slice of issues that were identified from the metrics.
//   error: An error if the analysis fails (nil in this implementation).
func (a *RuleBasedAnalyzer) AnalyzeMetrics(_ context.Context, data *models.MetricsData) ([]*models.Issue, error) {
	var issues []*models.Issue
	for _, rule := range a.metricRules {
		if value, ok := data.Data[rule.MetricName]; ok {
			// This is a simplified check; a real implementation would handle type assertions more gracefully.
			floatValue, isFloat := value.(float64)
			if !isFloat {
				continue
			}

			triggered := false
			switch rule.Operator {
			case ">":
				if floatValue > rule.Threshold {
					triggered = true
				}
			case "<":
				if floatValue < rule.Threshold {
					triggered = true
				}
			}

			if triggered {
				issue := &models.Issue{
					Title:    rule.IssueTitle,
					Severity: rule.Severity,
					Evidence: fmt.Sprintf("Metric '%s' value is %.2f, which violates the threshold of %s %.2f.", rule.MetricName, floatValue, rule.Operator, rule.Threshold),
					Recommendations: []*models.Recommendation{{Description: rule.Recommendation, CanAutoFix: false}},
				}
				issues = append(issues, issue)
			}
		}
	}
	return issues, nil
}

// AnalyzeLogs iterates through the analyzer's log rules and searches for the
// specified pattern within the collected log entries. If a match is found, it
// creates a corresponding issue.
//
// Parameters:
//   _ (context.Context): The context for the operation (currently unused).
//   data (*models.LogData): The collected log data to be analyzed.
//
// Returns:
//   []*models.Issue: A slice of issues that were identified from the logs.
//   error: An error if the analysis fails (nil in this implementation).
func (a *RuleBasedAnalyzer) AnalyzeLogs(_ context.Context, data *models.LogData) ([]*models.Issue, error) {
	var issues []*models.Issue
	for _, rule := range a.logRules {
		for _, logEntry := range data.Entries {
			if strings.Contains(strings.ToLower(logEntry), strings.ToLower(rule.Pattern)) {
				issues = append(issues, &models.Issue{
					Title:    rule.IssueTitle,
					Severity: rule.Severity,
					Evidence: fmt.Sprintf("Found pattern '%s' in log entry: %s", rule.Pattern, logEntry),
					Recommendations: []*models.Recommendation{{Description: rule.Recommendation, CanAutoFix: false}},
				})
				// To avoid flooding with similar issues, we only report the first occurrence of a pattern.
				// A more advanced implementation could aggregate findings.
				break
			}
		}
	}
	return issues, nil
}

// CorrelateSystems is a placeholder implementation. The RuleBasedAnalyzer does
// not support cross-system correlation, as this typically requires more advanced
// logic than simple, independent rules.
func (a *RuleBasedAnalyzer) CorrelateSystems(_ context.Context, _ *models.SystemCorrelationData) ([]*models.Issue, error) {
	a.log.Info("Cross-system correlation is not implemented in the RuleBasedAnalyzer.")
	return nil, nil
}

// --- AI-Based Analyzer (Conceptual Placeholder) ---

// AIAnalyzer is a conceptual placeholder for a more advanced, AI-powered analyzer.
// It demonstrates how a component that leverages a Large Language Model (LLM) would
// fit into the diagnosis framework by also implementing the DiagnosisAnalyzer interface.
// This approach would enable features like anomaly detection, advanced pattern
// recognition, and sophisticated root cause analysis.
type AIAnalyzer struct {
	log       logger.Logger
	llmClient llm_interfaces.LLMClient // Assumes an LLMClient interface exists.
}

// Name returns the unique identifier for this analyzer.
func (a *AIAnalyzer) Name() string { return "AIAnalyzer" }

// AnalyzeMetrics is a placeholder for an AI-driven metric analysis implementation.
// It would involve sending the metric data to an LLM and parsing the response to identify issues.
func (a *AIAnalyzer) AnalyzeMetrics(ctx context.Context, data *models.MetricsData) ([]*models.Issue, error) {
	// 1. Serialize metric data into a format the LLM can understand.
	// 2. Create a prompt asking the LLM to analyze the metrics for anomalies, trends, or known bad patterns.
	// 3. Call the llmClient.
	// 4. Parse the structured response (e.g., JSON) from the LLM back into []*models.Issue.
	a.log.Info("AI-based metric analysis is a placeholder and not yet implemented.")
	return nil, nil
}

// AnalyzeLogs is a placeholder for an AI-driven log analysis implementation.
// It would involve sending log summaries or filtered entries to an LLM to identify complex error patterns.
func (a *AIAnalyzer) AnalyzeLogs(ctx context.Context, data *models.LogData) ([]*models.Issue, error) {
	a.log.Info("AI-based log analysis is a placeholder and not yet implemented.")
	return nil, nil
}

// CorrelateSystems is a placeholder for an AI-driven correlation analysis implementation.
// This is where an LLM could be particularly powerful, finding non-obvious connections
// between different data sources (e.g., a metric spike and a specific log error) to
// perform advanced root cause analysis.
func (a *AIAnalyzer) CorrelateSystems(ctx context.Context, data *models.SystemCorrelationData) ([]*models.Issue, error) {
	// This is where an LLM would excel, finding non-obvious connections between different data sources
	// to perform advanced root cause analysis.
	a.log.Info("AI-based correlation analysis is a placeholder and not yet implemented.")
	return nil, nil
}

//Personal.AI order the ending
