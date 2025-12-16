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

package integration_test

import (
	"context"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/analysis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/llm"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// testPluginManager is a mock implementation for testing
type testPluginManager struct {
	data *models.CollectedData
}

func (m *testPluginManager) CollectData(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error) {
	return m.data, nil
}

func (m *testPluginManager) LoadPlugins() error {
	return nil
}

func (m *testPluginManager) LoadPlugin(pluginName string) (interfaces.DiagnosticPlugin, error) {
	return nil, nil
}

func (m *testPluginManager) UnloadPlugin(pluginName string) error {
	return nil
}

func (m *testPluginManager) GetPlugin(name string) (interfaces.DiagnosticPlugin, error) {
	return nil, nil
}

func (m *testPluginManager) ListPlugins() []interfaces.DiagnosticPlugin {
	return nil
}

func (m *testPluginManager) Shutdown() {}

// TestDiagnosis_WithAIAnalyzer tests the diagnosis pipeline with AI analyzer using mock LLM
func TestDiagnosis_WithAIAnalyzer(t *testing.T) {
	// Setup mock LLM client
	mockLLMClient := llm.NewMockClient()
	mockLLMClient.SetResponse(`{
		"summary": "Analysis identified high memory usage and connection issues",
		"reasoning": "Memory usage is at 90% which is above recommended threshold. Multiple connection timeout errors detected in logs.",
		"issues": [
			{
				"id": "ai-issue-001",
				"title": "High Memory Usage Detected",
				"severity": "High",
				"description": "Memory usage is at 90.2%, approaching the configured limit of 2GB",
				"evidence": "memory_usage: 90.2%, maxmemory: 2gb",
				"recommendations": [
					{
						"id": "ai-rec-001",
						"description": "Consider increasing maxmemory configuration or enabling more aggressive eviction policies",
						"canAutoFix": false,
						"priority": 2
					}
				]
			},
			{
				"id": "ai-issue-002",
				"title": "Connection Timeout Errors",
				"severity": "Medium",
				"description": "Multiple connection timeout errors detected in logs, indicating potential network issues",
				"evidence": "ERROR: Connection timeout to client 192.168.1.100",
				"recommendations": [
					{
						"id": "ai-rec-002",
						"description": "Investigate network connectivity and consider increasing timeout configuration",
						"canAutoFix": false,
						"priority": 1
					}
				]
			}
		]
	}`)

	// Setup mock plugin manager
	mockPM := &testPluginManager{
		data: &models.CollectedData{
			Metrics: &models.MetricsData{
				Data: map[string]interface{}{
					"cpu_usage":        85.5,
					"memory_usage":     90.2,
					"connections":      150,
					"queries_per_sec":  1200,
					"response_time_ms": 45.2,
				},
			},
			Logs: &models.LogData{
				Entries: []string{
					"INFO: Redis server started",
					"ERROR: Connection timeout to client 192.168.1.100",
					"ERROR: Memory limit approaching threshold",
					"WARN: Slow query detected: 250ms",
				},
			},
			Config: &models.ConfigData{
				Data: map[string]string{
					"maxmemory":        "2gb",
					"maxmemory-policy": "allkeys-lru",
					"timeout":          "300",
				},
			},
		},
	}

	// Create AI analyzer with mock client
	aiAnalyzer := analysis.NewAIAnalyzer(mockLLMClient, analysis.AIAnalyzerConfig{
		Middleware: "redis",
		Namespace:  "production",
		Instance:   "redis-master-001",
		Model:      "gpt-4",
	})

	// Create orchestrator with AI analyzer
	orchestrator := diagnosis.NewOrchestrator(mockPM, []analysis.Analyzer{aiAnalyzer})

	// Create diagnosis request
	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         "redis-master-001",
		Namespace:        "production",
	}

	// Run diagnosis
	progress := make(chan interfaces.DiagnosisProgress, 20)
	report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)

	// Verify no error
	if err != nil {
		t.Fatalf("Expected no error from diagnosis, got: %v", err)
	}

	// Verify report is not nil
	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	// Verify report structure
	if report.ID == "" {
		t.Error("Report should have a non-empty ID")
	}

	if report.Target.Middleware != enum.Redis {
		t.Errorf("Expected middleware type Redis, got: %v", report.Target.Middleware)
	}

	// Verify issues from AI analyzer
	if len(report.Issues) == 0 {
		t.Fatal("Expected at least one issue from AI analyzer")
	}

	// Verify we have AI-sourced issues
	aiIssueCount := 0
	for _, issue := range report.Issues {
		if issue.Source == "AI" {
			aiIssueCount++
		}
	}

	if aiIssueCount < 2 {
		t.Fatalf("Expected at least 2 AI-sourced issues, got %d", aiIssueCount)
	}

	// Find and verify first AI issue
	foundAIIssue := false
	for _, issue := range report.Issues {
		if issue.Source == "AI" && issue.ID == "ai-issue-001" {
			foundAIIssue = true
			if issue.Title != "High Memory Usage Detected" {
				t.Errorf("Expected issue title 'High Memory Usage Detected', got '%s'", issue.Title)
			}

			if issue.Severity != enum.SeverityHigh {
				t.Errorf("Expected severity High, got %v", issue.Severity)
			}
			break
		}
	}

	if !foundAIIssue {
		t.Error("Expected to find AI issue with ID 'ai-issue-001'")
	}

	// Verify report summary
	if report.Summary == "" {
		t.Error("Expected non-empty summary in report")
	}

	// Verify mock client was called
	if mockLLMClient.CallCount != 1 {
		t.Errorf("Expected 1 LLM call, got %d", mockLLMClient.CallCount)
	}

	// Verify the request sent to LLM contained the correct data
	if mockLLMClient.LastRequest == nil {
		t.Fatal("Expected LastRequest to be set")
	}

	if len(mockLLMClient.LastRequest.Messages) != 2 {
		t.Errorf("Expected 2 messages (system + user), got %d", len(mockLLMClient.LastRequest.Messages))
	}
}

// TestDiagnosis_WithMultipleAnalyzers tests combining AI analyzer with rule-based analyzer
func TestDiagnosis_WithMultipleAnalyzers(t *testing.T) {
	// Setup mock LLM client
	mockLLMClient := llm.NewMockClient()
	mockLLMClient.SetResponse(`{
		"summary": "AI analysis: System shows signs of resource pressure",
		"issues": [
			{
				"id": "ai-multi-001",
				"title": "AI Detected Resource Pressure",
				"severity": "Medium",
				"description": "Combined metrics indicate resource pressure",
				"evidence": "cpu: 85.5%, memory: 90.2%",
				"recommendations": []
			}
		]
	}`)

	// Setup mock plugin manager
	mockPM := &testPluginManager{
		data: &models.CollectedData{
			Metrics: &models.MetricsData{
				Data: map[string]interface{}{
					"cpu_usage":    85.5,
					"memory_usage": 90.2,
				},
			},
			Logs:   &models.LogData{Entries: []string{}},
			Config: &models.ConfigData{Data: map[string]string{}},
		},
	}

	// Create both analyzers
	ruleAnalyzer := diagnosis.NewRuleAnalyzer()
	aiAnalyzer := analysis.NewAIAnalyzer(mockLLMClient, analysis.AIAnalyzerConfig{
		Middleware: "redis",
		Instance:   "test-instance",
	})

	// Create orchestrator with both analyzers
	orchestrator := diagnosis.NewOrchestrator(mockPM, []analysis.Analyzer{ruleAnalyzer, aiAnalyzer})

	// Create diagnosis request
	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         "test-instance",
	}

	// Run diagnosis
	progress := make(chan interfaces.DiagnosisProgress, 20)
	report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)

	// Verify no error
	if err != nil {
		t.Fatalf("Expected no error from diagnosis, got: %v", err)
	}

	// Verify report contains issues
	if len(report.Issues) == 0 {
		t.Fatal("Expected at least one issue in report")
	}

	// Verify both analyzer types are present in issue sources
	hasRuleSource := false
	hasAISource := false

	for _, issue := range report.Issues {
		if issue.Source == "Rule" {
			hasRuleSource = true
		}
		if issue.Source == "AI" {
			hasAISource = true
		}
	}

	// We expect AI issues from the AI analyzer
	if !hasAISource {
		t.Error("Expected AI-sourced issues in report")
	}

	t.Logf("Report generated with %d issues (Rule: %v, AI: %v)", len(report.Issues), hasRuleSource, hasAISource)
}
