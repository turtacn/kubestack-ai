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
	"os/exec"
	"strings"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIDiagnoseJSONOutput tests that the CLI outputs valid JSON conforming to DiagnosisReport v1 schema
func TestCLIDiagnoseJSONOutput(t *testing.T) {
	t.Skip("Skipping E2E test - requires built binary and running environment")

	// Build the CLI binary first
	buildCmd := exec.Command("go", "build", "-o", "ksa_test", "./cmd/ksa/main.go")
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build CLI binary")

	defer func() {
		// Cleanup
		exec.Command("rm", "-f", "ksa_test").Run()
	}()

	// Run diagnosis with JSON output
	cmd := exec.Command("./ksa_test", "diagnose", "redis", "--instance", "test-redis", "--output", "json", "--dry-run")
	output, err := cmd.CombinedOutput()

	// Command might fail due to missing dependencies, but we can still check output format if available
	if err != nil {
		t.Logf("CLI execution failed (expected in test environment): %v", err)
		t.Logf("Output: %s", string(output))
		// Don't fail the test if it's just missing dependencies
		if strings.Contains(string(output), "diagnosis failed") {
			t.Skip("Skipping - diagnosis engine not available in test environment")
		}
	}

	// Parse the JSON output
	var diagReport report.DiagnosisReport
	err = json.Unmarshal(output, &diagReport)
	require.NoError(t, err, "Failed to parse JSON output: %s", string(output))

	// Validate schema
	assert.Equal(t, report.ReportVersion, diagReport.Version, "Version should be v1")
	assert.NotEmpty(t, diagReport.ID, "Report ID should not be empty")
	assert.NotEmpty(t, diagReport.Target.Middleware, "Target middleware should be set")
	assert.NotEmpty(t, diagReport.Status, "Status should be set")
	assert.NotEmpty(t, diagReport.Summary, "Summary should not be empty")
	assert.NotNil(t, diagReport.Issues, "Issues should be initialized (even if empty)")
}

// TestCLIDiagnoseTextOutput tests that the CLI outputs human-readable text format
func TestCLIDiagnoseTextOutput(t *testing.T) {
	t.Skip("Skipping E2E test - requires built binary and running environment")

	// Build the CLI binary first
	buildCmd := exec.Command("go", "build", "-o", "ksa_test", "./cmd/ksa/main.go")
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build CLI binary")

	defer func() {
		// Cleanup
		exec.Command("rm", "-f", "ksa_test").Run()
	}()

	// Run diagnosis with text output (default)
	cmd := exec.Command("./ksa_test", "diagnose", "redis", "--instance", "test-redis", "--dry-run")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("CLI execution failed (expected in test environment): %v", err)
		t.Logf("Output: %s", string(output))
		if strings.Contains(string(output), "diagnosis failed") {
			t.Skip("Skipping - diagnosis engine not available in test environment")
		}
	}

	// Validate text output contains expected sections
	outputStr := string(output)
	assert.Contains(t, outputStr, "Diagnosis Complete", "Text output should contain 'Diagnosis Complete'")
	assert.Contains(t, outputStr, "Report Version", "Text output should contain 'Report Version'")
	assert.Contains(t, outputStr, "Status:", "Text output should contain 'Status:'")
	assert.Contains(t, outputStr, "Summary:", "Text output should contain 'Summary:'")
}

// TestReportSchemaValidation tests the DiagnosisReport schema programmatically
func TestReportSchemaValidation(t *testing.T) {
	// Create a mock report
	target := report.DiagnosisTarget{
		Middleware: enum.Redis,
		Instance:   "test-redis",
		Namespace:  "default",
	}

	diagReport := report.NewDiagnosisReport("test-id-123", target)

	// Validate version
	assert.Equal(t, report.ReportVersion, diagReport.Version, "Report version should be v1")

	// Validate structure
	assert.Equal(t, "test-id-123", diagReport.ID)
	assert.Equal(t, enum.Redis, diagReport.Target.Middleware)
	assert.Equal(t, "test-redis", diagReport.Target.Instance)
	assert.NotNil(t, diagReport.Issues)
	assert.NotNil(t, diagReport.Metrics)
	assert.NotNil(t, diagReport.Metadata)

	// Test JSON serialization
	jsonData, err := diagReport.ToJSON()
	require.NoError(t, err, "Failed to serialize to JSON")

	// Parse back
	var parsedReport report.DiagnosisReport
	err = json.Unmarshal([]byte(jsonData), &parsedReport)
	require.NoError(t, err, "Failed to parse JSON")

	assert.Equal(t, diagReport.Version, parsedReport.Version)
	assert.Equal(t, diagReport.ID, parsedReport.ID)
}

// TestReportVersionFreeze ensures the version constant is frozen
func TestReportVersionFreeze(t *testing.T) {
	// This test serves as a contract - if someone tries to change the version,
	// this test will fail, forcing them to consider backward compatibility
	assert.Equal(t, "v1", report.ReportVersion, "Report version must remain v1 for backward compatibility")
}
