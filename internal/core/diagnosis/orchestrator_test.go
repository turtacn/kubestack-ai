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
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/analysis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// Mock Plugin Manager for testing
type mockPluginManager struct {
	collectDataFunc func(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error)
}

func (m *mockPluginManager) CollectData(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error) {
	if m.collectDataFunc != nil {
		return m.collectDataFunc(ctx, req)
	}
	return &models.CollectedData{
		Metrics: &models.MetricsData{
			Data: map[string]interface{}{
				"cpu_usage":    75.0,
				"memory_usage": 60.0,
			},
		},
		Logs: &models.LogData{
			Entries: []string{"INFO: System healthy"},
		},
	}, nil
}

func (m *mockPluginManager) LoadPlugins() error {
	return nil
}

func (m *mockPluginManager) GetPlugin(name string) (interfaces.DiagnosticPlugin, error) {
	return nil, nil
}

func (m *mockPluginManager) ListPlugins() []interfaces.DiagnosticPlugin {
	return nil
}

func (m *mockPluginManager) Shutdown() {}

func (m *mockPluginManager) LoadPlugin(pluginName string) (interfaces.DiagnosticPlugin, error) {
	return nil, nil
}

func (m *mockPluginManager) UnloadPlugin(pluginName string) error {
	return nil
}

// Mock Analyzer for testing
type mockAnalyzer struct {
	name        string
	analyzeFunc func(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error)
}

func (m *mockAnalyzer) Name() string {
	return m.name
}

func (m *mockAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
	if m.analyzeFunc != nil {
		return m.analyzeFunc(ctx, data)
	}
	result := analysis.NewAnalysisResult(m.name)
	result.Summary = "Mock analysis completed"
	return result, nil
}

// TestOrchestrator_CallOrder verifies the orchestrator calls stages in correct order
func TestOrchestrator_CallOrder(t *testing.T) {
	callOrder := []string{}

	mockPM := &mockPluginManager{
		collectDataFunc: func(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error) {
			callOrder = append(callOrder, "collect")
			return &models.CollectedData{
				Metrics: &models.MetricsData{Data: map[string]interface{}{"test": 1}},
			}, nil
		},
	}

	mockAn := &mockAnalyzer{
		name: "TestAnalyzer",
		analyzeFunc: func(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
			callOrder = append(callOrder, "analyze")
			result := analysis.NewAnalysisResult("TestAnalyzer")
			return result, nil
		},
	}

	orchestrator := NewOrchestrator(mockPM, []analysis.Analyzer{mockAn})

	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         "test-instance",
	}

	progress := make(chan interfaces.DiagnosisProgress, 10)

	report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	// Verify call order
	if len(callOrder) != 2 {
		t.Fatalf("Expected 2 calls, got %d", len(callOrder))
	}

	if callOrder[0] != "collect" {
		t.Errorf("Expected first call to be 'collect', got '%s'", callOrder[0])
	}

	if callOrder[1] != "analyze" {
		t.Errorf("Expected second call to be 'analyze', got '%s'", callOrder[1])
	}

	// Verify progress messages received
	progressMessages := []interfaces.DiagnosisProgress{}
	for msg := range progress {
		progressMessages = append(progressMessages, msg)
	}

	if len(progressMessages) == 0 {
		t.Error("Expected progress messages, got none")
	}
}

// TestOrchestrator_ErrorPropagation tests that errors are properly handled
func TestOrchestrator_ErrorPropagation(t *testing.T) {
	t.Run("Collection Error", func(t *testing.T) {
		mockPM := &mockPluginManager{
			collectDataFunc: func(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error) {
				return nil, &testError{"collection failed"}
			},
		}

		orchestrator := NewOrchestrator(mockPM, []analysis.Analyzer{})
		req := &models.DiagnosisRequest{
			TargetMiddleware: enum.Redis,
			Instance:         "test-instance",
		}

		progress := make(chan interfaces.DiagnosisProgress, 10)
		report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)

		if err == nil {
			t.Error("Expected error from collection failure")
		}

		if report != nil {
			t.Error("Expected nil report on collection failure")
		}

		// Verify error message sent via progress
		foundError := false
		for msg := range progress {
			if msg.Status == "Failed" && msg.Step == "Collection" {
				foundError = true
				break
			}
		}

		if !foundError {
			t.Error("Expected error progress message for collection failure")
		}
	})

	t.Run("Analyzer Error - Should Continue", func(t *testing.T) {
		mockPM := &mockPluginManager{}

		failingAnalyzer := &mockAnalyzer{
			name: "FailingAnalyzer",
			analyzeFunc: func(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
				return nil, &testError{"analyzer failed"}
			},
		}

		successAnalyzer := &mockAnalyzer{
			name: "SuccessAnalyzer",
			analyzeFunc: func(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
				result := analysis.NewAnalysisResult("SuccessAnalyzer")
				result.Issues = []*models.Issue{
					{
						ID:       "test-1",
						Source:   "SuccessAnalyzer",
						Title:    "Test Issue",
						Severity: enum.SeverityLow,
					},
				}
				return result, nil
			},
		}

		orchestrator := NewOrchestrator(mockPM, []analysis.Analyzer{failingAnalyzer, successAnalyzer})
		req := &models.DiagnosisRequest{
			TargetMiddleware: enum.Redis,
			Instance:         "test-instance",
		}

		progress := make(chan interfaces.DiagnosisProgress, 10)
		report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)

		if err != nil {
			t.Fatalf("Expected no error even with failing analyzer, got: %v", err)
		}

		if report == nil {
			t.Fatal("Expected report to be generated despite analyzer failure")
		}

		// Should have issue from successful analyzer
		if len(report.Issues) != 1 {
			t.Errorf("Expected 1 issue from successful analyzer, got %d", len(report.Issues))
		}
	})
}

// TestOrchestrator_ReportGeneration verifies report structure
func TestOrchestrator_ReportGeneration(t *testing.T) {
	mockPM := &mockPluginManager{
		collectDataFunc: func(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error) {
			return &models.CollectedData{
				Metrics: &models.MetricsData{
					Data: map[string]interface{}{
						"cpu_usage":    85.5,
						"memory_usage": 90.2,
					},
				},
				Logs: &models.LogData{
					Entries: []string{"ERROR: Something went wrong"},
				},
			}, nil
		},
	}

	mockAn := &mockAnalyzer{
		name: "TestAnalyzer",
		analyzeFunc: func(ctx context.Context, data *models.CollectedData) (*analysis.AnalysisResult, error) {
			result := analysis.NewAnalysisResult("TestAnalyzer")
			result.Issues = []*models.Issue{
				{
					ID:          "issue-1",
					Source:      "TestAnalyzer",
					Title:       "Test Issue 1",
					Severity:    enum.SeverityHigh,
					Description: "This is a test issue",
					Evidence:    "cpu_usage: 85.5",
				},
			}
			return result, nil
		},
	}

	orchestrator := NewOrchestrator(mockPM, []analysis.Analyzer{mockAn})

	req := &models.DiagnosisRequest{
		TargetMiddleware: enum.Redis,
		Instance:         "test-redis-001",
		Namespace:        "production",
	}

	progress := make(chan interfaces.DiagnosisProgress, 10)
	report, err := orchestrator.RunDiagnosis(context.Background(), req, progress)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify report structure
	if report.ID == "" {
		t.Error("Expected report to have an ID")
	}

	if report.Target.Middleware != enum.Redis {
		t.Errorf("Expected middleware Redis, got %v", report.Target.Middleware)
	}

	if report.Target.Instance != "test-redis-001" {
		t.Errorf("Expected instance 'test-redis-001', got '%s'", report.Target.Instance)
	}

	if len(report.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(report.Issues))
	}

	if len(report.Metrics) == 0 {
		t.Error("Expected metrics to be populated")
	}

	if report.Summary == "" {
		t.Error("Expected summary to be non-empty")
	}
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
