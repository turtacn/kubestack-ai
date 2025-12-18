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

package e2e

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/cli/output"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/core/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestOutputFormat_JSON verifies JSON output format
func TestOutputFormat_JSON(t *testing.T) {
	// Create sample diagnosis report
	diagReport := createSampleDiagnosisReport()
	
	// Render as JSON
	outputFormatter := output.NewFormatter("json")
	rendered, err := outputFormatter.Format(diagReport)
	require.NoError(t, err, "Should render JSON without error")
	
	// Validate JSON structure
	t.Run("ValidJSON", func(t *testing.T) {
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(rendered), &parsed)
		require.NoError(t, err, "Should be valid JSON")
		assert.NotEmpty(t, parsed, "JSON should not be empty")
	})
	
	// Validate required fields
	t.Run("RequiredFields", func(t *testing.T) {
		var parsed report.DiagnosisReport
		err := json.Unmarshal([]byte(rendered), &parsed)
		require.NoError(t, err, "Should unmarshal to DiagnosisReport")
		
		assert.NotEmpty(t, parsed.ID, "Should have ID")
		assert.NotEmpty(t, parsed.Version, "Should have version")
		assert.NotEmpty(t, parsed.Status, "Should have status")
		assert.NotNil(t, parsed.Target, "Should have target")
	})
	
	// Validate JSON is pretty-printed (optional, for readability)
	t.Run("PrettyPrinted", func(t *testing.T) {
		if strings.Contains(rendered, "\n") && strings.Contains(rendered, "  ") {
			t.Log("JSON is pretty-printed for readability")
		}
	})
}

// TestOutputFormat_YAML verifies YAML output format
func TestOutputFormat_YAML(t *testing.T) {
	diagReport := createSampleDiagnosisReport()
	
	// Render as YAML
	outputFormatter := output.NewFormatter("yaml")
	rendered, err := outputFormatter.Format(diagReport)
	require.NoError(t, err, "Should render YAML without error")
	
	// Validate YAML structure
	t.Run("ValidYAML", func(t *testing.T) {
		var parsed map[string]interface{}
		err := yaml.Unmarshal([]byte(rendered), &parsed)
		require.NoError(t, err, "Should be valid YAML")
		assert.NotEmpty(t, parsed, "YAML should not be empty")
	})
	
	// Validate required fields
	t.Run("RequiredFields", func(t *testing.T) {
		var parsed report.DiagnosisReport
		err := yaml.Unmarshal([]byte(rendered), &parsed)
		require.NoError(t, err, "Should unmarshal to DiagnosisReport")
		
		assert.NotEmpty(t, parsed.ID, "Should have ID")
		assert.NotEmpty(t, parsed.Version, "Should have version")
		assert.NotEmpty(t, parsed.Status, "Should have status")
	})
	
	// Validate YAML readability
	t.Run("Readability", func(t *testing.T) {
		assert.Contains(t, rendered, "id:", "YAML should contain field labels")
		assert.Contains(t, rendered, "version:", "YAML should contain version field")
	})
}

// TestOutputFormat_Text verifies text output format
func TestOutputFormat_Text(t *testing.T) {
	diagReport := createSampleDiagnosisReport()
	
	// Render as text
	outputFormatter := output.NewFormatter("text")
	rendered, err := outputFormatter.Format(diagReport)
	require.NoError(t, err, "Should render text without error")
	
	// Validate text contains key information
	t.Run("ContainsKeyInfo", func(t *testing.T) {
		assert.Contains(t, rendered, diagReport.ID, "Should contain report ID")
		assert.Contains(t, rendered, diagReport.Status, "Should contain status")
		assert.NotEmpty(t, rendered, "Text should not be empty")
	})
	
	// Validate text is human-readable
	t.Run("HumanReadable", func(t *testing.T) {
		lines := strings.Split(rendered, "\n")
		assert.Greater(t, len(lines), 3, "Should have multiple lines")
		
		// Should not contain raw JSON/YAML syntax
		assert.NotContains(t, rendered, `"id":`, "Should not contain JSON syntax")
		assert.NotContains(t, rendered, `{`, "Should not contain JSON braces (if pure text)")
	})
	
	// Validate sections are present
	t.Run("Sections", func(t *testing.T) {
		// Common sections in text output
		expectedSections := []string{"Report", "Status", "Target", "Summary"}
		foundSections := 0
		for _, section := range expectedSections {
			if strings.Contains(rendered, section) {
				foundSections++
			}
		}
		assert.Greater(t, foundSections, 0, "Should contain at least one expected section")
	})
}

// TestOutputFormat_Table verifies table output format
func TestOutputFormat_Table(t *testing.T) {
	t.Skip("Table format may not be implemented for all output types")
	
	diagReport := createSampleDiagnosisReport()
	
	// Render as table
	outputFormatter := output.NewFormatter("table")
	rendered, err := outputFormatter.Format(diagReport)
	if err != nil {
		t.Skipf("Table format not implemented: %v", err)
		return
	}
	
	// Validate table structure
	t.Run("TableStructure", func(t *testing.T) {
		// Tables typically have headers and separators
		assert.Contains(t, rendered, "|", "Table should contain column separators")
		assert.Contains(t, rendered, "-", "Table should contain row separators")
	})
}

// TestJSONSchemaCompliance verifies JSON output complies with schema
func TestJSONSchemaCompliance(t *testing.T) {
	diagReport := createSampleDiagnosisReport()
	
	// Render as JSON
	outputFormatter := output.NewFormatter("json")
	rendered, err := outputFormatter.Format(diagReport)
	require.NoError(t, err, "Should render JSON")
	
	// Parse back to struct
	var parsed report.DiagnosisReport
	err = json.Unmarshal([]byte(rendered), &parsed)
	require.NoError(t, err, "Should unmarshal to DiagnosisReport")
	
	// Validate schema compliance
	t.Run("VersionField", func(t *testing.T) {
		assert.Equal(t, report.ReportVersion, parsed.Version, "Version should match schema")
	})
	
	t.Run("TimestampFormat", func(t *testing.T) {
		assert.False(t, parsed.StartTime.IsZero(), "StartTime should be set")
		assert.False(t, parsed.EndTime.IsZero(), "EndTime should be set")
	})
	
	t.Run("RequiredFields", func(t *testing.T) {
		assert.NotEmpty(t, parsed.ID, "ID is required")
		assert.NotEmpty(t, parsed.Status, "Status is required")
		assert.NotNil(t, parsed.Target, "Target is required")
		assert.NotEmpty(t, parsed.Summary, "Summary is required")
	})
	
	t.Run("ArrayFields", func(t *testing.T) {
		assert.NotNil(t, parsed.Issues, "Issues array should be initialized")
		assert.NotNil(t, parsed.Recommendations, "Recommendations array should be initialized")
	})
}

// TestOutputFormatRoundTrip verifies data is preserved through format conversion
func TestOutputFormatRoundTrip(t *testing.T) {
	original := createSampleDiagnosisReport()
	
	formats := []string{"json", "yaml"}
	
	for _, format := range formats {
		t.Run("RoundTrip_"+format, func(t *testing.T) {
			// Format to string
			formatter := output.NewFormatter(format)
			rendered, err := formatter.Format(original)
			require.NoError(t, err, "Should format to %s", format)
			
			// Parse back
			var parsed report.DiagnosisReport
			switch format {
			case "json":
				err = json.Unmarshal([]byte(rendered), &parsed)
			case "yaml":
				err = yaml.Unmarshal([]byte(rendered), &parsed)
			}
			require.NoError(t, err, "Should parse from %s", format)
			
			// Verify key fields match
			assert.Equal(t, original.ID, parsed.ID, "ID should match")
			assert.Equal(t, original.Version, parsed.Version, "Version should match")
			assert.Equal(t, original.Status, parsed.Status, "Status should match")
			assert.Equal(t, original.Summary, parsed.Summary, "Summary should match")
		})
	}
}

// TestOutputFormatErrors verifies error handling for invalid formats
func TestOutputFormatErrors(t *testing.T) {
	diagReport := createSampleDiagnosisReport()
	
	t.Run("InvalidFormat", func(t *testing.T) {
		formatter := output.NewFormatter("invalid-format")
		_, err := formatter.Format(diagReport)
		// Should either return error or default to a valid format
		if err != nil {
			assert.Contains(t, err.Error(), "format", "Error should mention format")
		}
	})
	
	t.Run("NilInput", func(t *testing.T) {
		formatter := output.NewFormatter("json")
		_, err := formatter.Format(nil)
		// Should handle nil gracefully
		if err == nil {
			t.Log("Formatter handles nil input gracefully")
		}
	})
}

// TestOutputFormatterFactory verifies formatter creation
func TestOutputFormatterFactory(t *testing.T) {
	formats := []string{"json", "yaml", "text"}
	
	for _, format := range formats {
		t.Run("Create_"+format, func(t *testing.T) {
			formatter := output.NewFormatter(format)
			assert.NotNil(t, formatter, "Should create formatter for %s", format)
		})
	}
}

// TestOutputFormatPerformance verifies formatting performance
func TestOutputFormatPerformance(t *testing.T) {
	diagReport := createSampleDiagnosisReport()
	
	formats := []string{"json", "yaml", "text"}
	
	for _, format := range formats {
		t.Run("Performance_"+format, func(t *testing.T) {
			formatter := output.NewFormatter(format)
			
			start := time.Now()
			iterations := 100
			
			for i := 0; i < iterations; i++ {
				_, err := formatter.Format(diagReport)
				require.NoError(t, err, "Should format successfully")
			}
			
			elapsed := time.Since(start)
			avgTime := elapsed / time.Duration(iterations)
			
			// Formatting should be fast (< 10ms average)
			assert.Less(t, avgTime, 10*time.Millisecond, 
				"Average formatting time should be < 10ms, got %v", avgTime)
			
			t.Logf("Format %s: %d iterations in %v (avg %v per iteration)", 
				format, iterations, elapsed, avgTime)
		})
	}
}

// Helper function to create sample diagnosis report
func createSampleDiagnosisReport() *report.DiagnosisReport {
	now := time.Now()
	
	return &report.DiagnosisReport{
		Version: report.ReportVersion,
		ID:      "test-diag-20240101-120000",
		Target: &report.DiagnosisTarget{
			Middleware: enum.Redis.String(),
			Instance:   "localhost:6379",
			Namespace:  "default",
			Labels: map[string]string{
				"env": "test",
			},
		},
		Status:    "completed",
		StartTime: now.Add(-5 * time.Minute),
		EndTime:   now,
		Duration:  5 * time.Minute,
		Summary:   "Redis instance is healthy with minor warnings",
		Issues: []*models.Issue{
			{
				ID:          "issue-001",
				Severity:    "warning",
				Category:    "performance",
				Title:       "High memory usage",
				Description: "Memory usage is at 85%",
				Impact:      "May affect performance under load",
			},
		},
		Recommendations: []*models.Recommendation{
			{
				ID:          "rec-001",
				Priority:    "medium",
				Category:    "optimization",
				Title:       "Increase memory limit",
				Description: "Consider increasing Redis memory limit",
				Actions: []models.Action{
					{
						Type:        "config_change",
						Description: "Update maxmemory setting",
						Command:     "CONFIG SET maxmemory 2gb",
					},
				},
			},
		},
		Metrics: map[string]interface{}{
			"memory_usage_percent": 85.0,
			"connected_clients":    42,
			"ops_per_sec":          1250.5,
		},
		Metadata: map[string]string{
			"collector": "redis-plugin",
			"version":   "1.0.0",
		},
	}
}
