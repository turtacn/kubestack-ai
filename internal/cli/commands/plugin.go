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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/kubestack-ai/kubestack-ai/internal/plugins/manager"
	"github.com/spf13/cobra"
)

// newPluginCmd creates the plugin command for managing middleware plugins
func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage KubeStack-AI plugins",
		Long: `Manage plugins for middleware diagnostics and operations.
Plugins provide capabilities for diagnosing and managing different middleware types
such as Redis, MySQL, Kafka, Elasticsearch, and PostgreSQL.`,
		Example: `  # List all available plugins
  ksa plugin list

  # Show detailed information about a plugin
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

// newPluginListCmd lists all available plugins
func newPluginListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available plugins",
		Long:  "Display a list of all available plugins with their current status.",
		Example: `  # List all plugins
  ksa plugin list

  # List plugins in JSON format
  ksa plugin list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := cmd.Context().Value("config").(*GlobalConfig)
			
			// Initialize plugin registry
			pluginDirs := []string{cfg.Config.Plugins.Directory}
			registry, err := manager.NewRegistry(pluginDirs)
			if err != nil {
				return fmt.Errorf("failed to create plugin registry: %w", err)
			}

			// Get all plugins
			plugins := registry.ListAll()

			if cfg.OutputFormat == "json" {
				return outputJSON(plugins)
			} else if cfg.OutputFormat == "yaml" {
				return outputYAML(plugins)
			}

			// Text output
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PLUGIN\tTYPE\tVERSION\tDESCRIPTION")
			fmt.Fprintln(w, "------\t----\t-------\t-----------")
			
			for _, plugin := range plugins {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					plugin.Name(),
					plugin.Type(),
					plugin.Version(),
					truncateString(plugin.Description(), 50))
			}
			
			w.Flush()
			return nil
		},
	}
}

// newPluginInfoCmd shows detailed information about a plugin
func newPluginInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <plugin-name>",
		Short: "Show detailed information about a plugin",
		Long:  "Display detailed information about a specific plugin including capabilities and configuration.",
		Example: `  # Show plugin information
  ksa plugin info redis-diagnostics

  # Show information in JSON format
  ksa plugin info redis-diagnostics -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := cmd.Context().Value("config").(*GlobalConfig)
			pluginName := args[0]

			// Initialize plugin registry
			pluginDirs := []string{cfg.Config.Plugins.Directory}
			registry, err := manager.NewRegistry(pluginDirs)
			if err != nil {
				return fmt.Errorf("failed to create plugin registry: %w", err)
			}

			// Get plugin
			plugin := registry.Get(pluginName)
			if plugin == nil {
				return fmt.Errorf("plugin not found: %s", pluginName)
			}

			// Create info structure
			info := map[string]interface{}{
				"name":        plugin.Name(),
				"type":        plugin.Type(),
				"version":     plugin.Version(),
				"description": plugin.Description(),
			}

			if cfg.OutputFormat == "json" {
				return outputJSON(info)
			} else if cfg.OutputFormat == "yaml" {
				return outputYAML(info)
			}

			// Text output
			fmt.Printf("Plugin: %s\n", plugin.Name())
			fmt.Printf("Type: %s\n", plugin.Type())
			fmt.Printf("Version: %s\n", plugin.Version())
			fmt.Printf("Description: %s\n", plugin.Description())

			return nil
		},
	}
}

// newPluginEnableCmd enables a plugin
func newPluginEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <plugin-name>",
		Short: "Enable a plugin",
		Long:  "Enable a plugin to make it available for use.",
		Example: `  # Enable a plugin
  ksa plugin enable redis-diagnostics`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := cmd.Context().Value("config").(*GlobalConfig)
			pluginName := args[0]

			// Initialize plugin registry
			pluginDirs := []string{cfg.Config.Plugins.Directory}
			registry, err := manager.NewRegistry(pluginDirs)
			if err != nil {
				return fmt.Errorf("failed to create plugin registry: %w", err)
			}

			// Check if plugin exists
			plugin := registry.Get(pluginName)
			if plugin == nil {
				return fmt.Errorf("plugin not found: %s", pluginName)
			}

			// Create enabled marker file
			enabledFile := filepath.Join(cfg.Config.Plugins.Directory, pluginName+".enabled")
			if err := os.WriteFile(enabledFile, []byte("enabled"), 0644); err != nil {
				return fmt.Errorf("failed to enable plugin: %w", err)
			}

			fmt.Printf("Plugin '%s' enabled successfully\n", pluginName)
			return nil
		},
	}
}

// newPluginDisableCmd disables a plugin
func newPluginDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <plugin-name>",
		Short: "Disable a plugin",
		Long:  "Disable a plugin to prevent it from being used.",
		Example: `  # Disable a plugin
  ksa plugin disable redis-diagnostics`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := cmd.Context().Value("config").(*GlobalConfig)
			pluginName := args[0]

			// Initialize plugin registry
			pluginDirs := []string{cfg.Config.Plugins.Directory}
			registry, err := manager.NewRegistry(pluginDirs)
			if err != nil {
				return fmt.Errorf("failed to create plugin registry: %w", err)
			}

			// Check if plugin exists
			plugin := registry.Get(pluginName)
			if plugin == nil {
				return fmt.Errorf("plugin not found: %s", pluginName)
			}

			// Remove enabled marker file
			enabledFile := filepath.Join(cfg.Config.Plugins.Directory, pluginName+".enabled")
			if err := os.Remove(enabledFile); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to disable plugin: %w", err)
			}

			fmt.Printf("Plugin '%s' disabled successfully\n", pluginName)
			return nil
		},
	}
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	// Plugin command will be registered in root.go
}
