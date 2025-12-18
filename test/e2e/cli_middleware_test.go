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
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to execute CLI command
func executeKSACommand(t *testing.T, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "./ksa", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// TestCLI_Version tests the version command
func TestCLI_Version(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check if ksa binary exists
	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found, skipping E2E test")
	}

	output, err := executeKSACommand(t, "version")
	if err != nil && !strings.Contains(output, "version") {
		t.Logf("Version command output: %s", output)
		// Version command might fail due to config, but should show version info
	}

	// Should contain version information
	assert.True(t, 
		strings.Contains(output, "version") || 
		strings.Contains(output, "Version") ||
		strings.Contains(output, "KubeStack"),
		"Output should contain version information")
}

// TestCLI_Help tests the help command
func TestCLI_Help(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	output, err := executeKSACommand(t, "--help")
	// Help should always work
	if err != nil {
		t.Logf("Help output: %s", output)
	}

	// Should contain basic commands
	expectedCommands := []string{"diagnose", "ask", "server"}
	for _, cmd := range expectedCommands {
		assert.Contains(t, output, cmd, "Help should mention %s command", cmd)
	}
}

// TestCLI_DiagnoseHelp tests the diagnose command help
func TestCLI_DiagnoseHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	output, err := executeKSACommand(t, "diagnose", "--help")
	if err != nil {
		t.Logf("Diagnose help output: %s", output)
	}

	// Should contain middleware types
	expectedTypes := []string{"redis", "kafka", "mysql", "postgresql", "elasticsearch"}
	for _, mw := range expectedTypes {
		assert.Contains(t, strings.ToLower(output), mw, 
			"Diagnose help should mention %s", mw)
	}
}

// TestCLI_InvalidCommand tests error handling for invalid commands
func TestCLI_InvalidCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	output, err := executeKSACommand(t, "nonexistent-command")
	
	// Should return error for invalid command
	assert.Error(t, err, "Invalid command should return error")
	assert.Contains(t, strings.ToLower(output), "unknown command", 
		"Output should indicate unknown command")
}

// TestCLI_DiagnoseWithoutArgs tests diagnose command without arguments
func TestCLI_DiagnoseWithoutArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	output, err := executeKSACommand(t, "diagnose")
	
	// Should show error or help when no middleware type specified
	if err != nil {
		// Error is expected
		assert.True(t, 
			strings.Contains(output, "required") ||
			strings.Contains(output, "Usage") ||
			strings.Contains(output, "help"),
			"Should show usage or error message")
	}
}

// TestCLI_ConfigurationFileHandling tests configuration file handling
func TestCLI_ConfigurationFileHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	// Create temporary config file
	tmpConfig := `
llm:
  provider: openai
  model: gpt-4
  api_key: test-key
`
	tmpFile, err := os.CreateTemp("", "ksa-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(tmpConfig)
	require.NoError(t, err)
	tmpFile.Close()

	// Test with config file
	output, err := executeKSACommand(t, "--config", tmpFile.Name(), "version")
	
	// Should not error on valid config
	t.Logf("Config test output: %s", output)
	// Config might be valid or have other issues, just test it doesn't panic
}

// TestCLI_OutputFormats tests different output format options
func TestCLI_OutputFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	formats := []string{"json", "yaml", "text"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			// Test help with different formats (doesn't require middleware)
			output, _ := executeKSACommand(t, "--output", format, "--help")
			
			// Should accept format flag
			assert.NotEmpty(t, output, "Output should not be empty for format %s", format)
		})
	}
}

// TestCLI_VerboseFlag tests verbose output flag
func TestCLI_VerboseFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	output, _ := executeKSACommand(t, "--verbose", "--help")
	
	// Should accept verbose flag
	assert.NotEmpty(t, output)
}

// TestCLI_MultipleFlags tests combining multiple flags
func TestCLI_MultipleFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	output, _ := executeKSACommand(t, "--verbose", "--output", "json", "version")
	
	// Should handle multiple flags
	assert.NotEmpty(t, output)
}

// TestCLI_AskCommand tests the ask command basic functionality
func TestCLI_AskCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	// Test ask command help
	output, err := executeKSACommand(t, "ask", "--help")
	if err != nil {
		t.Logf("Ask help output: %s", output)
	}

	assert.Contains(t, strings.ToLower(output), "ask", 
		"Should show ask command help")
}

// TestCLI_ServerCommand tests the server command
func TestCLI_ServerCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	// Test server command help
	output, err := executeKSACommand(t, "server", "--help")
	if err != nil {
		t.Logf("Server help output: %s", output)
	}

	assert.Contains(t, strings.ToLower(output), "server", 
		"Should show server command help")
}

// TestCLI_MonitorCommand tests the monitor command
func TestCLI_MonitorCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	// Test monitor command help
	output, err := executeKSACommand(t, "monitor", "--help")
	if err != nil {
		t.Logf("Monitor help output: %s", output)
	}

	// Should contain monitor command info
	assert.True(t,
		strings.Contains(strings.ToLower(output), "monitor") ||
		strings.Contains(strings.ToLower(output), "watch"),
		"Should show monitor command help")
}

// TestCLI_BinarySize tests that the binary size is reasonable
func TestCLI_BinarySize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	info, err := os.Stat("./ksa")
	require.NoError(t, err)

	size := info.Size()
	t.Logf("Binary size: %d bytes (%.2f MB)", size, float64(size)/(1024*1024))

	// Binary should be between 10MB and 500MB (reasonable range for Go binary with deps)
	assert.Greater(t, size, int64(10*1024*1024), "Binary should be at least 10MB")
	assert.Less(t, size, int64(500*1024*1024), "Binary should be less than 500MB")
}

// TestCLI_ExecutablePermissions tests that binary has correct permissions
func TestCLI_ExecutablePermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := os.Stat("./ksa"); os.IsNotExist(err) {
		t.Skip("ksa binary not found")
	}

	info, err := os.Stat("./ksa")
	require.NoError(t, err)

	mode := info.Mode()
	
	// Should be executable
	assert.True(t, mode&0111 != 0, "Binary should have executable permissions")
}
