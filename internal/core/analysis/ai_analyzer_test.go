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

package analysis

import (
	"context"
	"errors"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/llm"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// TestAIAnalyzer_ParseValidJSON tests that the analyzer correctly parses valid JSON responses.
func TestAIAnalyzer_ParseValidJSON(t *testing.T) {
	// Setup mock client with valid response
	mockClient := llm.NewMockClient()
	mockClient.SetResponse(`{
		"summary": "Test analysis completed",
		"reasoning": "Based on the metrics, identified one issue",
		"issues": [
			{
				"id": "test-001",
				"title": "Test Issue",
				"severity": "High",
				"description": "This is a test issue",
				"evidence": "test_metric: 100",
				"recommendations": [
					{
						"id": "rec-001",
						"description": "Test recommendation",
						"canAutoFix": false,
						"priority": 1
					}
				]
			}
		]
	}`)

	// Create analyzer
	analyzer := NewAIAnalyzer(mockClient, AIAnalyzerConfig{
		Middleware: "redis",
		Namespace:  "test-ns",
		Instance:   "test-instance",
	})

	// Create test data
	testData := &models.CollectedData{
		Metrics: &models.MetricsData{
			Data: map[string]interface{}{
				"test_metric": 100,
			},
		},
		Logs: &models.LogData{
			Entries: []string{"test log 1", "test log 2"},
		},
		Config: &models.ConfigData{
			Data: map[string]string{
				"test_config": "value",
			},
		},
	}

	// Execute analysis
	result, err := analyzer.Analyze(context.Background(), testData)

	// Assertions
	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	if result == nil {
		t.Fatal("Analyze() returned nil result")
	}

	if result.AnalyzerName != "AIAnalyzer" {
		t.Errorf("Expected AnalyzerName 'AIAnalyzer', got '%s'", result.AnalyzerName)
	}

	if result.Summary != "Test analysis completed" {
		t.Errorf("Expected summary 'Test analysis completed', got '%s'", result.Summary)
	}

	if len(result.Issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.ID != "test-001" {
		t.Errorf("Expected issue ID 'test-001', got '%s'", issue.ID)
	}

	if issue.Title != "Test Issue" {
		t.Errorf("Expected issue title 'Test Issue', got '%s'", issue.Title)
	}

	if issue.Severity != enum.SeverityHigh {
		t.Errorf("Expected severity 'High', got '%s'", issue.Severity)
	}

	if issue.Source != "AI" {
		t.Errorf("Expected source 'AI', got '%s'", issue.Source)
	}

	if len(issue.Recommendations) != 1 {
		t.Fatalf("Expected 1 recommendation, got %d", len(issue.Recommendations))
	}

	// Check metadata
	if _, ok := result.Metadata["llm_model"]; !ok {
		t.Error("Expected llm_model in metadata")
	}

	if _, ok := result.Metadata["reasoning"]; !ok {
		t.Error("Expected reasoning in metadata")
	}

	// Verify mock client was called
	if mockClient.CallCount != 1 {
		t.Errorf("Expected 1 LLM call, got %d", mockClient.CallCount)
	}
}

// TestAIAnalyzer_InvalidJSON tests error handling when LLM returns invalid JSON.
func TestAIAnalyzer_InvalidJSON(t *testing.T) {
	mockClient := llm.NewMockClient()
	mockClient.SetResponse("This is not valid JSON")

	analyzer := NewAIAnalyzer(mockClient, AIAnalyzerConfig{
		Middleware: "redis",
		Instance:   "test-instance",
	})

	testData := &models.CollectedData{
		Metrics: &models.MetricsData{Data: map[string]interface{}{}},
	}

	_, err := analyzer.Analyze(context.Background(), testData)

	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}

	if mockClient.CallCount != 1 {
		t.Errorf("Expected 1 LLM call, got %d", mockClient.CallCount)
	}
}

// TestAIAnalyzer_LLMError tests error handling when LLM client returns an error.
func TestAIAnalyzer_LLMError(t *testing.T) {
	mockClient := llm.NewMockClient()
	mockClient.SetError(errors.New("LLM service unavailable"))

	analyzer := NewAIAnalyzer(mockClient, AIAnalyzerConfig{
		Middleware: "redis",
		Instance:   "test-instance",
	})

	testData := &models.CollectedData{
		Metrics: &models.MetricsData{Data: map[string]interface{}{}},
	}

	_, err := analyzer.Analyze(context.Background(), testData)

	if err == nil {
		t.Fatal("Expected error when LLM fails, got nil")
	}

	if mockClient.CallCount != 1 {
		t.Errorf("Expected 1 LLM call, got %d", mockClient.CallCount)
	}
}

// TestAIAnalyzer_EmptyIssues tests handling of analysis with no issues found.
func TestAIAnalyzer_EmptyIssues(t *testing.T) {
	mockClient := llm.NewMockClient()
	mockClient.SetResponse(`{
		"summary": "No issues found, system is healthy",
		"issues": []
	}`)

	analyzer := NewAIAnalyzer(mockClient, AIAnalyzerConfig{
		Middleware: "redis",
		Instance:   "test-instance",
	})

	testData := &models.CollectedData{
		Metrics: &models.MetricsData{Data: map[string]interface{}{}},
	}

	result, err := analyzer.Analyze(context.Background(), testData)

	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	if len(result.Issues) != 0 {
		t.Errorf("Expected 0 issues, got %d", len(result.Issues))
	}

	if result.Summary != "No issues found, system is healthy" {
		t.Errorf("Unexpected summary: %s", result.Summary)
	}
}

// TestAIAnalyzer_MultipleIssues tests handling of multiple issues.
func TestAIAnalyzer_MultipleIssues(t *testing.T) {
	mockClient := llm.NewMockClient()
	mockClient.SetResponse(`{
		"summary": "Found multiple issues",
		"issues": [
			{
				"id": "issue-001",
				"title": "Issue 1",
				"severity": "Critical",
				"description": "Critical issue",
				"evidence": "evidence 1",
				"recommendations": []
			},
			{
				"id": "issue-002",
				"title": "Issue 2",
				"severity": "Medium",
				"description": "Medium issue",
				"evidence": "evidence 2",
				"recommendations": []
			},
			{
				"id": "issue-003",
				"title": "Issue 3",
				"severity": "Low",
				"description": "Low issue",
				"evidence": "evidence 3",
				"recommendations": []
			}
		]
	}`)

	analyzer := NewAIAnalyzer(mockClient, AIAnalyzerConfig{
		Middleware: "mysql",
		Instance:   "test-instance",
	})

	testData := &models.CollectedData{
		Metrics: &models.MetricsData{Data: map[string]interface{}{}},
	}

	result, err := analyzer.Analyze(context.Background(), testData)

	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	if len(result.Issues) != 3 {
		t.Fatalf("Expected 3 issues, got %d", len(result.Issues))
	}

	// Verify severities
	expectedSeverities := []enum.SeverityLevel{enum.SeverityCritical, enum.SeverityMedium, enum.SeverityLow}
	for i, expected := range expectedSeverities {
		if result.Issues[i].Severity != expected {
			t.Errorf("Issue %d: expected severity '%s', got '%s'",
				i, expected.String(), result.Issues[i].Severity.String())
		}
	}
}

// TestAIAnalyzer_CleanJSONResponse tests JSON cleaning functionality.
func TestAIAnalyzer_CleanJSONResponse(t *testing.T) {
	analyzer := &AIAnalyzer{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean JSON",
			input:    `{"test": "value"}`,
			expected: `{"test": "value"}`,
		},
		{
			name:     "JSON with markdown",
			input:    "```json\n{\"test\": \"value\"}\n```",
			expected: `{"test": "value"}`,
		},
		{
			name:     "JSON with extra whitespace",
			input:    "  \n  {\"test\": \"value\"}  \n  ",
			expected: `{"test": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.cleanJSONResponse(tt.input)
			if result != tt.expected {
				t.Errorf("cleanJSONResponse() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestAIAnalyzer_ValidateSeverity tests severity validation.
func TestAIAnalyzer_ValidateSeverity(t *testing.T) {
	tests := []struct {
		severity string
		valid    bool
	}{
		{"Critical", true},
		{"High", true},
		{"Medium", true},
		{"Low", true},
		{"Info", true},
		{"critical", true},
		{"CRITICAL", true},
		{"Invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			result := isValidSeverity(tt.severity)
			if result != tt.valid {
				t.Errorf("isValidSeverity(%q) = %v, want %v", tt.severity, result, tt.valid)
			}
		})
	}
}

// TestAIAnalyzer_SetMiddlewareContext tests context updates.
func TestAIAnalyzer_SetMiddlewareContext(t *testing.T) {
	mockClient := llm.NewMockClient()
	analyzer := NewAIAnalyzer(mockClient, AIAnalyzerConfig{
		Middleware: "redis",
		Namespace:  "ns1",
		Instance:   "instance1",
	})

	// Verify initial values
	if analyzer.middleware != "redis" {
		t.Errorf("Expected middleware 'redis', got '%s'", analyzer.middleware)
	}

	// Update context
	analyzer.SetMiddlewareContext("mysql", "ns2", "instance2")

	// Verify updated values
	if analyzer.middleware != "mysql" {
		t.Errorf("Expected middleware 'mysql', got '%s'", analyzer.middleware)
	}
	if analyzer.namespace != "ns2" {
		t.Errorf("Expected namespace 'ns2', got '%s'", analyzer.namespace)
	}
	if analyzer.instance != "instance2" {
		t.Errorf("Expected instance 'instance2', got '%s'", analyzer.instance)
	}
}

// TestBuildAIInput tests the AIInput builder function.
func TestBuildAIInput(t *testing.T) {
	data := &models.CollectedData{
		Metrics: &models.MetricsData{
			Data: map[string]interface{}{
				"metric1": 100,
				"metric2": "value",
			},
		},
		Logs: &models.LogData{
			Entries: []string{"log1", "log2", "log3"},
		},
		Config: &models.ConfigData{
			Data: map[string]string{
				"config1": "value1",
				"config2": "value2",
			},
		},
	}

	input := BuildAIInput(data, "redis", "test-ns", "test-instance")

	if input.Context.Middleware != "redis" {
		t.Errorf("Expected middleware 'redis', got '%s'", input.Context.Middleware)
	}

	if input.Context.Namespace != "test-ns" {
		t.Errorf("Expected namespace 'test-ns', got '%s'", input.Context.Namespace)
	}

	if input.Context.Instance != "test-instance" {
		t.Errorf("Expected instance 'test-instance', got '%s'", input.Context.Instance)
	}

	if len(input.Data.Metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(input.Data.Metrics))
	}

	if len(input.Data.Logs) != 3 {
		t.Errorf("Expected 3 logs, got %d", len(input.Data.Logs))
	}

	if len(input.Data.Config) != 2 {
		t.Errorf("Expected 2 config entries, got %d", len(input.Data.Config))
	}
}
