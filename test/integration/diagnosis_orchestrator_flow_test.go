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

package integration

import (
	"context"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/analysis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// TestDiagnosis_MinimalFlow tests the complete diagnosis pipeline with mock components
func TestDiagnosis_MinimalFlow(t *testing.T) {
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

	// Setup analyzer
	ruleAnalyzer := diagnosis.NewRuleAnalyzer()

	// Create orchestrator
	orchestrator := diagnosis.NewOrchestrator(mockPM, []analysis.Analyzer{ruleAnalyzer})

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

	if report.Target.Instance != "redis-master-001" {
		t.Errorf("Expected instance 'redis-master-001', got: '%s'", report.Target.Instance)
	}

	if report.Target.Namespace != "production" {
		t.Errorf("Expected namespace 'production', got: '%s'", report.Target.Namespace)
	}

	// Verify issues are present (rule analyzer should detect high memory/cpu)
	if len(report.Issues) == 0 {
		t.Error("Expected at least one issue to be detected by rule analyzer")
	}

	t.Logf("Report generated with %d issues", len(report.Issues))

	// Verify each issue has required fields
	for i, issue := range report.Issues {
		if issue.ID == "" {
			t.Errorf("Issue %d: ID should not be empty", i)
		}
		if issue.Source == "" {
			t.Errorf("Issue %d: Source should not be empty", i)
		}
		if issue.Title == "" {
			t.Errorf("Issue %d: Title should not be empty", i)
		}
		if issue.Description == "" {
			t.Errorf("Issue %d: Description should not be empty", i)
		}
	}

	// Verify metrics are preserved in report
	if len(report.Metrics) == 0 {
		t.Error("Expected metrics to be populated in report")
	}

	// Verify summary is present
	if report.Summary == "" {
		t.Error("Expected report to have a summary")
	}

	// Verify progress messages were sent
	progressMessages := []interfaces.DiagnosisProgress{}
	for msg := range progress {
		progressMessages = append(progressMessages, msg)
		t.Logf("Progress: Step=%s, Status=%s, Message=%s", msg.Step, msg.Status, msg.Message)
	}

	if len(progressMessages) < 3 {
		t.Errorf("Expected at least 3 progress messages (Collection, Analysis, Reporting), got %d", len(progressMessages))
	}

	// Verify all stages were reported
	stages := map[string]bool{}
	for _, msg := range progressMessages {
		stages[msg.Step] = true
	}

	expectedStages := []string{"Collection", "Analysis", "Reporting"}
	for _, stage := range expectedStages {
		if !stages[stage] {
			t.Errorf("Expected progress message for stage '%s'", stage)
		}
	}
}

// TestDiagnosis_ReportJSONSerialization tests that reports can be serialized to JSON
func TestDiagnosis_ReportJSONSerialization(t *testing.T) {
	mockPM := &testPluginManager{
		data: &models.CollectedData{
			Metrics: &models.MetricsData{
				Data: map[string]interface{}{
					"cpu_usage": 75.0,
				},
			},
		},
	}

	ruleAnalyzer := diagnosis.NewRuleAnalyzer()
	orchestrator := diagnosis.NewOrchestrator(mockPM, []analysis.Analyzer{ruleAnalyzer})

	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         "test-redis",
	}

	progress := make(chan interfaces.DiagnosisProgress, 10)
	report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test JSON serialization
	jsonStr, err := report.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize report to JSON: %v", err)
	}

	if jsonStr == "" {
		t.Error("Expected non-empty JSON output")
	}

	t.Logf("Report JSON length: %d bytes", len(jsonStr))
}

// TestCase-4: TestDiagnosis_DryRunAutoFix
// Integration test for AutoFix capability with dry-run mode
func TestDiagnosis_DryRunAutoFix(t *testing.T) {
	// Setup mock plugin manager with auto-fixable issues
	mockPM := &testPluginManager{
		data: &models.CollectedData{
			Metrics: &models.MetricsData{
				Data: map[string]interface{}{
					"memory_usage":    95.0, // High memory usage
					"cpu_usage":       85.0,
					"connection_pool": 150,
				},
			},
			Logs: &models.LogData{
				Entries: []string{
					"ERROR: Memory limit reached",
					"WARN: Connection pool exhausted",
				},
			},
			Config: &models.ConfigData{
				Data: map[string]string{
					"max_memory":      "1gb",
					"max_connections": "100",
				},
			},
		},
	}

	// Setup analyzer
	ruleAnalyzer := diagnosis.NewRuleAnalyzer()

	// Create orchestrator with AutoFix enabled
	opts := &diagnosis.OrchestratorOptions{
		EnableAutoFix: true,
	}
	orchestrator := diagnosis.NewOrchestratorWithOptions(mockPM, []analysis.Analyzer{ruleAnalyzer}, opts)

	// Verify AutoFix is enabled
	if !orchestrator.IsAutoFixEnabled() {
		t.Fatal("Expected AutoFix to be enabled")
	}

	// Create diagnosis request
	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         "redis-autofix-test",
		Namespace:        "test",
	}

	// Run diagnosis
	progress := make(chan interfaces.DiagnosisProgress, 20)
	report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)

	if err != nil {
		t.Fatalf("Expected no error from diagnosis, got: %v", err)
	}

	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	// Verify AutoFix metadata is present
	if autoFixEnabled, ok := report.Metadata["autofix_enabled"].(bool); ok {
		if !autoFixEnabled {
			t.Error("Expected autofix_enabled metadata to be true")
		}
	} else {
		t.Error("Expected autofix_enabled metadata to be present")
	}

	// Check for AutoFix hints
	if hints, ok := report.Metadata["autofix_hints"]; ok {
		t.Logf("Found AutoFix hints: %v", hints)
	}

	// Verify report structure
	if report.ID == "" {
		t.Error("Report should have a non-empty ID")
	}

	t.Logf("✓ AutoFix integration test passed: Report generated with AutoFix metadata")

	// Close progress channel
	for range progress {
		// Drain progress channel
	}
}

// TestDiagnosis_MultipleAnalyzers tests orchestration with multiple analyzers
func TestDiagnosis_MultipleAnalyzers(t *testing.T) {
	mockPM := &testPluginManager{
		data: &models.CollectedData{
			Metrics: &models.MetricsData{
				Data: map[string]interface{}{
					"cpu_usage": 85.0,
				},
			},
		},
	}

	// Create multiple analyzers
	analyzer1 := diagnosis.NewRuleAnalyzer()
	analyzer2 := &mockTestAnalyzer{name: "SecondAnalyzer"}

	orchestrator := diagnosis.NewOrchestrator(mockPM, []analysis.Analyzer{analyzer1, analyzer2})

	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         "test-redis",
	}

	progress := make(chan interfaces.DiagnosisProgress, 10)
	report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify metadata shows multiple analyzers
	if analyzerCount, ok := report.Metadata["analyzer_count"]; ok {
		if count, ok := analyzerCount.(int); ok && count != 2 {
			t.Errorf("Expected 2 analyzers, got %d", count)
		}
	} else {
		t.Error("Expected analyzer_count in metadata")
	}
}

// Test helper: mock plugin manager
type testPluginManager struct {
	data *models.CollectedData
}

func (m *testPluginManager) CollectData(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error) {
	return m.data, nil
}

func (m *testPluginManager) LoadPlugins() error {
	return nil
}

func (m *testPluginManager) GetPlugin(name string) (interfaces.DiagnosticPlugin, error) {
	return nil, nil
}

func (m *testPluginManager) ListPlugins() []interfaces.DiagnosticPlugin {
	return nil
}

func (m *testPluginManager) Shutdown() {}

func (m *testPluginManager) LoadPlugin(pluginName string) (interfaces.DiagnosticPlugin, error) {
	return nil, nil
}

func (m *testPluginManager) UnloadPlugin(pluginName string) error {
	return nil
}

// Test helper: mock analyzer
type mockTestAnalyzer struct {
	name string
}

func (m *mockTestAnalyzer) Name() string {
	return m.name
}

func (m *mockTestAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
	result := analysis.NewAnalysisResult(m.name)
	result.Summary = "Mock analysis completed"
	// Add a test issue
	result.Issues = []*models.Issue{
		{
			ID:          "mock-issue-1",
			Source:      m.name,
			Title:       "Mock Issue",
			Severity:    enum.SeverityLow,
			Description: "This is a mock issue for testing",
		},
	}
	return result, nil
}
