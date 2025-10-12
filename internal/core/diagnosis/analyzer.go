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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	llm_interfaces "github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
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

// --- AI-Based Analyzer ---

// AIAnalyzer is a concrete implementation of the DiagnosisAnalyzer interface that leverages
// a Large Language Model (LLM) to perform advanced root cause analysis. It excels at
// correlating various data sources (metrics, logs, configuration) to identify complex
// and non-obvious issues that rule-based systems might miss.
type AIAnalyzer struct {
	log           logger.Logger
	llmClient     llm_interfaces.LLMClient
	promptBuilder *prompt.Builder
}

// NewAIAnalyzer creates a new, configured instance of the AIAnalyzer.
//
// Parameters:
//   llmClient (llm_interfaces.LLMClient): A client for interacting with an LLM provider.
//
// Returns:
//   interfaces.DiagnosisAnalyzer: A new analyzer ready to process data.
func NewAIAnalyzer(llmClient llm_interfaces.LLMClient) interfaces.DiagnosisAnalyzer {
	pb, err := prompt.NewBuilder(prompt.AllTemplates)
	if err != nil {
		// This is a panic because the templates are statically compiled into the binary.
		// If they fail to parse, it's a critical, unrecoverable developer error.
		panic(fmt.Sprintf("failed to create prompt builder: %v", err))
	}
	return &AIAnalyzer{
		log:           logger.NewLogger("ai-analyzer"),
		llmClient:     llmClient,
		promptBuilder: pb,
	}
}

// Name returns the unique identifier for this analyzer.
func (a *AIAnalyzer) Name() string { return "AIAnalyzer" }

// AnalyzeMetrics defers to the more powerful CorrelateSystems method. The primary
// strength of the AIAnalyzer is in correlation, not isolated data analysis.
func (a *AIAnalyzer) AnalyzeMetrics(ctx context.Context, data *models.MetricsData) ([]*models.Issue, error) {
	correlationData := &models.SystemCorrelationData{
		DataSources: map[string]interface{}{"metrics": data},
	}
	return a.CorrelateSystems(ctx, correlationData)
}

// AnalyzeLogs defers to the more powerful CorrelateSystems method. The primary
// strength of the AIAnalyzer is in correlation, not isolated data analysis.
func (a *AIAnalyzer) AnalyzeLogs(ctx context.Context, data *models.LogData) ([]*models.Issue, error) {
	correlationData := &models.SystemCorrelationData{
		DataSources: map[string]interface{}{"logs": data},
	}
	return a.CorrelateSystems(ctx, correlationData)
}

// CorrelateSystems is the core of the AIAnalyzer. It serializes all provided data,
// constructs a detailed prompt for the LLM, sends the request, and parses the
// structured JSON response back into a list of identified issues.
func (a *AIAnalyzer) CorrelateSystems(ctx context.Context, data *models.SystemCorrelationData) ([]*models.Issue, error) {
	if data == nil || len(data.DataSources) == 0 {
		a.log.Debug("No data provided to AIAnalyzer for correlation.")
		return nil, nil
	}

	// 1. Serialize the collected data to JSON.
	jsonData, err := json.MarshalIndent(data.DataSources, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize data for LLM prompt: %w", err)
	}

	// 2. Build the prompt using the prompt builder.
	messages, err := a.promptBuilder.Build(prompt.TemplateDiagnosisID, map[string]string{"context_data": string(jsonData)}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build diagnosis prompt: %w", err)
	}

	a.log.Infof("Sending diagnosis request to LLM...")

	// 3. Call the LLM client.
	llmRequest := &llm_interfaces.LLMRequest{Messages: messages}
	llmResponse, err := a.llmClient.SendMessage(ctx, llmRequest)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	a.log.Infof("Received diagnosis response from LLM.")
	a.log.Debugf("LLM Response: %s", llmResponse.Message.Content)

	// 4. Parse the structured response from the LLM.
	var analysisResult models.AIAnalysisResult
	if err := json.Unmarshal([]byte(llmResponse.Message.Content), &analysisResult); err != nil {
		// This can happen if the LLM response is not valid JSON.
		// We can add more robust parsing/fallback logic here in the future.
		return nil, fmt.Errorf("failed to parse structured response from LLM: %w. Response: %s", err, llmResponse.Message.Content)
	}

	a.log.Infof("Successfully parsed %d issues from LLM response.", len(analysisResult.Issues))
	return analysisResult.Issues, nil
}

//Personal.AI order the ending
