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
	"time"

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
					Recommendations: []*models.Recommendation{
						{
							Description: rule.Recommendation,
							CanAutoFix:  false,
							Fix: models.FixAction{
								Description: rule.Recommendation,
							},
						},
					},
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
					Recommendations: []*models.Recommendation{
						{
							Description: rule.Recommendation,
							CanAutoFix:  false,
							Fix: models.FixAction{
								Description: rule.Recommendation,
							},
						},
					},
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

// --- AI-Based Analyzer Implementation ---

// AIAnalyzer uses a Large Language Model (LLM) to perform advanced, holistic
// analysis of collected data. It implements the DiagnosisAnalyzer interface.
type AIAnalyzer struct {
	log           logger.Logger
	llmClient     llm_interfaces.LLMClient
	promptBuilder *prompt.Builder
}

// NewAIAnalyzer creates a new, configured instance of the AIAnalyzer.
func NewAIAnalyzer(llmClient llm_interfaces.LLMClient) (interfaces.DiagnosisAnalyzer, error) {
	// Define the template locally since GetDefaultTemplates is not available.
	// This template should align with what prompt.Builder expects.
	tmpl := prompt.Template{
		Name:    "generic-diagnosis",
		Content: "Analyze the following system data and identify root causes for any issues. Return the result in JSON format.\n\nContext:\nMiddleware: {{.MiddlewareName}}\nInstance: {{.InstanceName}}\nTime: {{.Timestamp}}\n\nData:\n{{.CollectedData}}",
	}
	pb, err := prompt.NewBuilder(tmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt builder for AIAnalyzer: %w", err)
	}
	return &AIAnalyzer{
		log:           logger.NewLogger("ai-analyzer"),
		llmClient:     llmClient,
		promptBuilder: pb,
	}, nil
}

// Name returns the unique identifier for this analyzer.
func (a *AIAnalyzer) Name() string { return "AIAnalyzer" }

// AnalyzeMetrics is a no-op for the AIAnalyzer. The core analysis is performed
// in CorrelateSystems, which has access to all data sources for a more
// holistic analysis, preventing redundant or partial analyses.
func (a *AIAnalyzer) AnalyzeMetrics(_ context.Context, _ *models.MetricsData) ([]*models.Issue, error) {
	a.log.Debug("Skipping metric analysis; handled by CorrelateSystems.")
	return nil, nil
}

// AnalyzeLogs is a no-op for the AIAnalyzer. The core analysis is performed
// in CorrelateSystems to ensure all data is analyzed together.
func (a *AIAnalyzer) AnalyzeLogs(_ context.Context, _ *models.LogData) ([]*models.Issue, error) {
	a.log.Debug("Skipping log analysis; handled by CorrelateSystems.")
	return nil, nil
}

// CorrelateSystems performs a holistic analysis of all collected data using an LLM.
// It serializes the metrics, logs, and config, builds a prompt, sends it to the LLM,
// and parses the structured JSON response into a list of identified issues.
func (a *AIAnalyzer) CorrelateSystems(ctx context.Context, data *models.SystemCorrelationData) ([]*models.Issue, error) {
	a.log.Info("Starting AI-based correlation analysis...")

	// 1. Serialize the collected data for the prompt
	promptData, err := a.preparePromptData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare data for prompt: %w", err)
	}

	// 2. Build the prompt
	promptStr, err := a.promptBuilder.
		WithData("MiddlewareName", promptData.MiddlewareName).
		WithData("InstanceName", promptData.InstanceName).
		WithData("Timestamp", promptData.Timestamp).
		WithData("CollectedData", promptData.CollectedData).
		Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build diagnosis prompt: %w", err)
	}

	// 3. Send the request to the LLM
	req := &llm_interfaces.LLMRequest{
		Messages: []llm_interfaces.Message{
			{Role: "user", Content: promptStr},
		},
		Temperature: 0.2, // Lower temperature for more deterministic, factual analysis
	}
	a.log.Debug("Sending analysis request to LLM...")
	resp, err := a.llmClient.SendMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %w", err)
	}
	a.log.Debugf("Received LLM response. Prompt tokens: %d, Completion tokens: %d", resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	// 4. Parse the structured JSON response
	return a.parseLLMResponse(resp.Message.Content)
}

// Helper struct for serializing data into the prompt template
type diagnosisPromptData struct {
	MiddlewareName string
	InstanceName   string
	Timestamp      string
	CollectedData  string
}

// Helper struct for parsing the JSON response from the LLM
type llmIssuesResponse struct {
	Issues []struct {
		Title           string `json:"title"`
		Severity        string `json:"severity"`
		Description     string `json:"description"`
		Recommendations []struct {
			Description string `json:"description"`
			Command     string `json:"command,omitempty"`
		} `json:"recommendations"`
	} `json:"issues"`
}

func (a *AIAnalyzer) preparePromptData(data *models.SystemCorrelationData) (*diagnosisPromptData, error) {
	// Helper to safely extract and marshal data from the map
	marshalDataSource := func(key string) []byte {
		if source, ok := data.DataSources[key]; ok && source != nil {
			bytes, err := json.MarshalIndent(source, "", "  ")
			if err == nil {
				return bytes
			}
			a.log.Warnf("Failed to marshal data source '%s': %v", key, err)
		}
		return []byte("{}") // Return empty JSON object on failure
	}

	var dataBuilder strings.Builder
	dataBuilder.WriteString("Metrics:\n" + string(marshalDataSource("metrics")) + "\n\n")
	dataBuilder.WriteString("Logs:\n" + string(marshalDataSource("logs")) + "\n\n")
	dataBuilder.WriteString("Configuration:\n" + string(marshalDataSource("config")))

	// Safely extract string and time values with type assertions
	getString := func(key string) string {
		if val, ok := data.DataSources[key].(string); ok {
			return val
		}
		return "N/A"
	}
	getTime := func(key string) time.Time {
		if val, ok := data.DataSources[key].(time.Time); ok {
			return val
		}
		return time.Now()
	}

	return &diagnosisPromptData{
		MiddlewareName: getString("middlewareName"),
		InstanceName:   getString("instanceName"),
		Timestamp:      getTime("timestamp").String(),
		CollectedData:  dataBuilder.String(),
	}, nil
}

func (a *AIAnalyzer) parseLLMResponse(responseContent string) ([]*models.Issue, error) {
	var llmResp llmIssuesResponse
	// The LLM sometimes wraps the JSON in markdown code blocks, so we need to clean it.
	cleanedJSON := strings.Trim(responseContent, " \n\t`json")
	if err := json.Unmarshal([]byte(cleanedJSON), &llmResp); err != nil {
		a.log.Errorf("Failed to unmarshal LLM response JSON. Raw response: %s", responseContent)
		return nil, fmt.Errorf("failed to parse LLM response: %w. Raw response: %s", err, responseContent)
	}

	var issues []*models.Issue
	for _, llmIssue := range llmResp.Issues {
		var recommendations []*models.Recommendation
		for _, llmRec := range llmIssue.Recommendations {
			recommendations = append(recommendations, &models.Recommendation{
				Description: llmRec.Description,
				CanAutoFix:  llmRec.Command != "",
				Fix: models.FixAction{
					Description: llmRec.Description,
					Command:     llmRec.Command,
				},
			})
		}

		issue := &models.Issue{
			Title:           llmIssue.Title,
			Severity:        stringToSeverity(llmIssue.Severity),
			Description:     llmIssue.Description,
			Recommendations: recommendations,
		}
		issues = append(issues, issue)
	}
	return issues, nil
}

// stringToSeverity converts the severity string from the LLM response to the internal enum type.
func stringToSeverity(s string) enum.SeverityLevel {
	switch strings.ToLower(s) {
	case "critical":
		return enum.SeverityCritical
	case "high":
		return enum.SeverityHigh
	case "medium":
		return enum.SeverityMedium
	case "low":
		return enum.SeverityLow
	default:
		// Default to the lowest severity if the LLM provides an unknown value.
		return enum.SeverityLow
	}
}

//Personal.AI order the ending
