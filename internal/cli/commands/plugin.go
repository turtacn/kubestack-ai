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

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// PluginInfo represents simplified plugin information for CLI display
type PluginInfo struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`
}

// newPluginCmd creates the plugin command for managing middleware plugins
func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage KubeStack-AI plugins",
		Long: `Manage middleware diagnostic plugins.
View available plugins, enable/disable plugins, and query plugin information.`,
		Example: `  # List all plugins
  ksa plugin list

  # Show plugin info
  ksa plugin info redis-diagnostics

  # Enable a plugin
  ksa plugin enable redis-diagnostics

  # Disable a plugin
  ksa plugin disable redis-diagnostics`,
	}

	cmd.AddCommand(newPluginListCmd())
	cmd.AddCommand(newPluginInfoCmd())
	cmd.AddCommand(newPluginEnableCmd())
	cmd.AddCommand(newPluginDisableCmd())

	return cmd
}

// newPluginListCmd creates the plugin list subcommand
func newPluginListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available plugins",
		Long:  "Display a list of all available plugins with their current status.",
		Example: `  # List all plugins
  ksa plugin list

  # List plugins in JSON format
  ksa plugin list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get output format from flag
			outputFormat, _ := cmd.Flags().GetString("output")
			
			// Mock plugin list for demonstration
			plugins := getAvailablePlugins()

			if outputFormat == "json" {
				return pluginOutputJSON(plugins)
			} else if outputFormat == "yaml" {
				return pluginOutputYAML(plugins)
			}

			// Text output
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PLUGIN\tTYPE\tVERSION\tDESCRIPTION")
			fmt.Fprintln(w, "------\t----\t-------\t-----------")
			
			for _, plugin := range plugins {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					plugin.Name,
					plugin.Type,
					plugin.Version,
					truncateString(plugin.Description, 50))
			}

			w.Flush()
			return nil
		},
	}

	return cmd
}

// newPluginInfoCmd creates the plugin info subcommand
func newPluginInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <plugin-name>",
		Short: "Show detailed information about a plugin",
		Long:  "Display detailed information about a specific plugin, including its capabilities and configuration.",
		Example: `  # Show plugin information
  ksa plugin info redis-diagnostics

  # Show in JSON format
  ksa plugin info redis-diagnostics -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := args[0]
			outputFormat, _ := cmd.Flags().GetString("output")

			// Find plugin
			plugins := getAvailablePlugins()
			var found *PluginInfo
			for _, p := range plugins {
				if p.Name == pluginName {
					found = &p
					break
				}
			}

			if found == nil {
				return fmt.Errorf("plugin not found: %s", pluginName)
			}

			if outputFormat == "json" {
				return pluginOutputJSON(found)
			} else if outputFormat == "yaml" {
				return pluginOutputYAML(found)
			}

			// Text output
			fmt.Printf("Plugin: %s\n", found.Name)
			fmt.Printf("Type: %s\n", found.Type)
			fmt.Printf("Version: %s\n", found.Version)
			fmt.Printf("Description: %s\n", found.Description)

			return nil
		},
	}

	return cmd
}

// newPluginEnableCmd creates the plugin enable subcommand
func newPluginEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <plugin-name>",
		Short: "Enable a plugin",
		Long:  "Enable a plugin to make it available for use in diagnostics.",
		Example: `  # Enable a plugin
  ksa plugin enable redis-diagnostics`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := args[0]

			// Create enabled marker file
			pluginDir := getPluginDir()
			markerFile := filepath.Join(pluginDir, pluginName+".enabled")
			
			if err := os.MkdirAll(pluginDir, 0755); err != nil {
				return fmt.Errorf("failed to create plugin directory: %w", err)
			}

			if err := os.WriteFile(markerFile, []byte("enabled"), 0644); err != nil {
				return fmt.Errorf("failed to enable plugin: %w", err)
			}

			fmt.Printf("Plugin '%s' enabled successfully\n", pluginName)
			return nil
		},
	}

	return cmd
}

// newPluginDisableCmd creates the plugin disable subcommand
func newPluginDisableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <plugin-name>",
		Short: "Disable a plugin",
		Long:  "Disable a plugin to prevent it from being used in diagnostics.",
		Example: `  # Disable a plugin
  ksa plugin disable redis-diagnostics`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := args[0]

			// Remove enabled marker file
			pluginDir := getPluginDir()
			markerFile := filepath.Join(pluginDir, pluginName+".enabled")
			
			if err := os.Remove(markerFile); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to disable plugin: %w", err)
			}

			fmt.Printf("Plugin '%s' disabled successfully\n", pluginName)
			return nil
		},
	}

	return cmd
}

// getPluginDir returns the plugin directory path
func getPluginDir() string {
	// Try to get from config, fallback to default
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ksa", "plugins")
}

// getAvailablePlugins returns a list of available plugins
func getAvailablePlugins() []PluginInfo {
	// Mock implementation - in production, query the plugin registry
	return []PluginInfo{
		{
			Name:        "redis-diagnostics",
			Type:        "diagnostics",
			Version:     "1.0.0",
			Description: "Redis diagnostics and health checks",
		},
		{
			Name:        "mysql-diagnostics",
			Type:        "diagnostics",
			Version:     "1.0.0",
			Description: "MySQL diagnostics and query analysis",
		},
		{
			Name:        "kafka-diagnostics",
			Type:        "diagnostics",
			Version:     "1.0.0",
			Description: "Kafka cluster monitoring and diagnosis",
		},
		{
			Name:        "elasticsearch-diagnostics",
			Type:        "diagnostics",
			Version:     "1.0.0",
			Description: "Elasticsearch cluster health analysis",
		},
		{
			Name:        "postgresql-diagnostics",
			Type:        "diagnostics",
			Version:     "1.0.0",
			Description: "PostgreSQL performance diagnostics",
		},
	}
}

// pluginOutputJSON outputs data in JSON format
func pluginOutputJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// pluginOutputYAML outputs data in YAML format
func pluginOutputYAML(data interface{}) error {
	enc := yaml.NewEncoder(os.Stdout)
	defer enc.Close()
	return enc.Encode(data)
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
