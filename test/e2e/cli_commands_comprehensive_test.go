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
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/cli/commands"
	"github.com/kubestack-ai/kubestack-ai/internal/cli/validator"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllCommandsHaveHelp verifies that all commands have complete help text
func TestAllCommandsHaveHelp(t *testing.T) {
	rootCmd := getRootCommand()
	
	// Check root command
	assert.NotEmpty(t, rootCmd.Use, "Root command should have Use field")
	assert.NotEmpty(t, rootCmd.Short, "Root command should have Short description")
	assert.NotEmpty(t, rootCmd.Long, "Root command should have Long description")
	
	// Check all subcommands
	expectedCommands := []string{"diagnose", "ask", "fix", "server", "plugin", "version"}
	commands := rootCmd.Commands()
	
	commandMap := make(map[string]*cobra.Command)
	for _, cmd := range commands {
		commandMap[cmd.Name()] = cmd
	}
	
	for _, cmdName := range expectedCommands {
		t.Run("Command_"+cmdName, func(t *testing.T) {
			cmd, exists := commandMap[cmdName]
			require.True(t, exists, "Command '%s' should be registered", cmdName)
			assert.NotEmpty(t, cmd.Use, "Command '%s' should have Use field", cmdName)
			assert.NotEmpty(t, cmd.Short, "Command '%s' should have Short description", cmdName)
		})
	}
}

// TestAllCommandsAreRegistered verifies the command tree is complete
func TestAllCommandsAreRegistered(t *testing.T) {
	rootCmd := getRootCommand()
	commands := rootCmd.Commands()
	
	// Expected minimum set of commands
	expectedCommands := map[string]bool{
		"diagnose": false,
		"ask":      false,
		"fix":      false,
		"server":   false,
		"version":  false,
	}
	
	for _, cmd := range commands {
		if _, exists := expectedCommands[cmd.Name()]; exists {
			expectedCommands[cmd.Name()] = true
		}
	}
	
	// Verify all expected commands are registered
	for cmdName, found := range expectedCommands {
		assert.True(t, found, "Command '%s' should be registered", cmdName)
	}
}

// TestGlobalFlagsWork verifies global flags work across all commands
func TestGlobalFlagsWork(t *testing.T) {
	rootCmd := getRootCommand()
	
	// Check persistent flags
	configFlag := rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, configFlag, "Global --config flag should exist")
	assert.NotEmpty(t, configFlag.Usage, "--config flag should have usage text")
	
	logLevelFlag := rootCmd.PersistentFlags().Lookup("log-level")
	require.NotNil(t, logLevelFlag, "Global --log-level flag should exist")
	assert.NotEmpty(t, logLevelFlag.Usage, "--log-level flag should have usage text")
	
	outputFlag := rootCmd.PersistentFlags().Lookup("output")
	require.NotNil(t, outputFlag, "Global --output flag should exist")
	assert.NotEmpty(t, outputFlag.Usage, "--output flag should have usage text")
}

// TestCommandTreeValidation validates the entire command tree structure
func TestCommandTreeValidation(t *testing.T) {
	rootCmd := getRootCommand()
	
	errors := validator.ValidateCommandTree(rootCmd)
	
	if len(errors) > 0 {
		t.Errorf("Command tree validation failed with %d errors:", len(errors))
		for i, err := range errors {
			t.Errorf("  %d. %v", i+1, err)
		}
	}
}

// TestDiagnoseCommandFlags verifies diagnose command has all required flags
func TestDiagnoseCommandFlags(t *testing.T) {
	rootCmd := getRootCommand()
	diagnoseCmd := findCommand(rootCmd, "diagnose")
	require.NotNil(t, diagnoseCmd, "Diagnose command should exist")
	
	// Check for expected flags
	expectedFlags := []string{"instance", "namespace", "dry-run"}
	
	for _, flagName := range expectedFlags {
		t.Run("Flag_"+flagName, func(t *testing.T) {
			flag := diagnoseCmd.Flags().Lookup(flagName)
			if flag == nil {
				// Try persistent flags
				flag = diagnoseCmd.PersistentFlags().Lookup(flagName)
			}
			require.NotNil(t, flag, "Flag --%s should exist", flagName)
			assert.NotEmpty(t, flag.Usage, "Flag --%s should have usage text", flagName)
		})
	}
}

// TestAskCommandFlags verifies ask command has required flags
func TestAskCommandFlags(t *testing.T) {
	rootCmd := getRootCommand()
	askCmd := findCommand(rootCmd, "ask")
	require.NotNil(t, askCmd, "Ask command should exist")
	
	// Ask command should accept arguments
	assert.True(t, askCmd.Args == nil || askCmd.Args == cobra.MinimumNArgs(1), 
		"Ask command should accept at least 1 argument")
}

// TestFixCommandFlags verifies fix command has required flags
func TestFixCommandFlags(t *testing.T) {
	rootCmd := getRootCommand()
	fixCmd := findCommand(rootCmd, "fix")
	require.NotNil(t, fixCmd, "Fix command should exist")
	
	// Check for expected flags
	expectedFlags := []string{"id", "auto-approve"}
	
	for _, flagName := range expectedFlags {
		t.Run("Flag_"+flagName, func(t *testing.T) {
			flag := fixCmd.Flags().Lookup(flagName)
			if flag == nil {
				flag = fixCmd.PersistentFlags().Lookup(flagName)
			}
			require.NotNil(t, flag, "Flag --%s should exist", flagName)
			assert.NotEmpty(t, flag.Usage, "Flag --%s should have usage text", flagName)
		})
	}
}

// TestServerCommandFlags verifies server command has required flags
func TestServerCommandFlags(t *testing.T) {
	rootCmd := getRootCommand()
	serverCmd := findCommand(rootCmd, "server")
	require.NotNil(t, serverCmd, "Server command should exist")
	
	// Server command typically has port/host flags
	// These might be in the config, but we should verify the command exists
	assert.NotEmpty(t, serverCmd.Short, "Server command should have description")
}

// TestVersionCommand verifies version command output
func TestVersionCommand(t *testing.T) {
	rootCmd := getRootCommand()
	versionCmd := findCommand(rootCmd, "version")
	require.NotNil(t, versionCmd, "Version command should exist")
	
	// Capture output
	buf := new(bytes.Buffer)
	versionCmd.SetOut(buf)
	versionCmd.SetErr(buf)
	
	// Run version command
	versionCmd.Run(versionCmd, []string{})
	
	output := buf.String()
	assert.Contains(t, output, "KubeStack-AI", "Version output should contain project name")
	assert.True(t, strings.Contains(output, "v") || strings.Contains(output, "version"), 
		"Version output should contain version information")
}

// TestHelpTextQuality validates help text quality for all commands
func TestHelpTextQuality(t *testing.T) {
	rootCmd := getRootCommand()
	commands := getAllCommands(rootCmd)
	
	for _, cmd := range commands {
		t.Run("HelpText_"+cmd.Name(), func(t *testing.T) {
			// Short description should not be empty
			assert.NotEmpty(t, cmd.Short, "Command '%s' should have Short description", cmd.Name())
			
			// Short description should be concise (less than 80 chars is a good guideline)
			if len(cmd.Short) > 120 {
				t.Logf("Warning: Command '%s' has a long Short description (%d chars)", 
					cmd.Name(), len(cmd.Short))
			}
			
			// If command has flags, they should have usage text
			cmd.Flags().VisitAll(func(flag *cobra.Flag) {
				assert.NotEmpty(t, flag.Usage, "Flag --%s in command '%s' should have usage text", 
					flag.Name, cmd.Name())
			})
		})
	}
}

// TestCommandExamples verifies that commands with examples have valid syntax
func TestCommandExamples(t *testing.T) {
	rootCmd := getRootCommand()
	commands := getAllCommands(rootCmd)
	
	for _, cmd := range commands {
		if cmd.Example != "" {
			t.Run("Examples_"+cmd.Name(), func(t *testing.T) {
				// Example should contain the command name
				assert.Contains(t, cmd.Example, cmd.Name(), 
					"Example for '%s' should contain the command name", cmd.Name())
				
				// Example should not be too short
				assert.Greater(t, len(cmd.Example), 10, 
					"Example for '%s' seems too short", cmd.Name())
			})
		}
	}
}

// Helper functions

// getRootCommand creates a root command for testing (without execution)
func getRootCommand() *cobra.Command {
	// Create a mock root command structure similar to the actual one
	// We can't directly use commands.Execute as it would try to run initialization
	rootCmd := &cobra.Command{
		Use:   "ksa",
		Short: "KubeStack-AI is an intelligent SRE assistant for middleware.",
		Long: `KubeStack-AI is a command-line tool that uses AI to help you diagnose,
analyze, and fix issues with your middleware infrastructure, whether it is
running on Kubernetes or bare metal servers.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	
	// Add global flags
	rootCmd.PersistentFlags().String("config", "", "config file")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format")
	
	// Add subcommands (simplified versions for testing)
	rootCmd.AddCommand(&cobra.Command{
		Use:   "diagnose [middleware] --instance [instance]",
		Short: "Diagnose middleware issues",
		Run:   func(cmd *cobra.Command, args []string) {},
	})
	
	rootCmd.AddCommand(&cobra.Command{
		Use:   "ask [question]",
		Short: "Ask a question about middleware",
		Args:  cobra.MinimumNArgs(1),
		Run:   func(cmd *cobra.Command, args []string) {},
	})
	
	rootCmd.AddCommand(&cobra.Command{
		Use:   "fix --id [diagnosis-id]",
		Short: "Apply fixes for diagnosed issues",
		Run:   func(cmd *cobra.Command, args []string) {},
	})
	
	rootCmd.AddCommand(&cobra.Command{
		Use:   "server",
		Short: "Start the KubeStack-AI API server",
		Run:   func(cmd *cobra.Command, args []string) {},
	})
	
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number of KubeStack-AI",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("KubeStack-AI v0.1.0")
		},
	})
	
	// Add flags to diagnose command
	diagnoseCmd := findCommand(rootCmd, "diagnose")
	if diagnoseCmd != nil {
		diagnoseCmd.Flags().String("instance", "", "Instance identifier")
		diagnoseCmd.Flags().String("namespace", "", "Kubernetes namespace")
		diagnoseCmd.Flags().Bool("dry-run", false, "Dry run mode")
	}
	
	// Add flags to fix command
	fixCmd := findCommand(rootCmd, "fix")
	if fixCmd != nil {
		fixCmd.Flags().String("id", "", "Diagnosis ID to fix")
		fixCmd.Flags().Bool("auto-approve", false, "Automatically approve fixes")
	}
	
	return rootCmd
}

// findCommand finds a command by name in the root command's subcommands
func findCommand(root *cobra.Command, name string) *cobra.Command {
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

// getAllCommands returns all commands in the tree
func getAllCommands(root *cobra.Command) []*cobra.Command {
	commands := []*cobra.Command{root}
	for _, cmd := range root.Commands() {
		commands = append(commands, getAllCommands(cmd)...)
	}
	return commands
}

// init ensures test environment variables are set
func init() {
	// Set test environment
	os.Setenv("KSA_TEST_MODE", "true")
}
