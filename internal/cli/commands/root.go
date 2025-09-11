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

// Package commands contains all the CLI commands for the application.
package commands

import (
	"fmt"
	"os"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile string

	// rootCmd represents the base command when called without any subcommands.
	rootCmd = &cobra.Command{
		Use:   "ksa",
		Short: "KubeStack-AI is an intelligent SRE assistant for middleware.",
		Long: `KubeStack-AI is a command-line tool that uses AI to help you diagnose,
analyze, and fix issues with your middleware infrastructure, whether it is
running on Kubernetes or bare metal servers.`,
		// PersistentPreRunE is a Cobra hook that runs before any subcommand's Run function.
		// It's the perfect place for global initialization.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// 1. Load configuration from file and environment variables.
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// 2. Initialize the global logger with settings from the config file,
			//    which may have been overridden by command-line flags bound by Viper.
			logCfg := cfg.Logger
			logCfg.Level = viper.GetString("logger.level") // Get the final value after flag parsing.
			logger.InitGlobalLogger(&logCfg)

			log := logger.GetLogger()
			log.Debugf("Logger initialized with level: %s", logCfg.Level)

			// 3. Validate the final configuration.
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("configuration validation failed: %w", err)
			}

			return nil
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra prints the error, so we just need to exit with a non-zero code.
		os.Exit(1)
	}
}

func init() {
	// Add global flags that will apply to all subcommands.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/kubestack-ai/config.yaml or $HOME/.ksa.yaml)")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error, fatal)")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format (text, json, yaml)")

	// Bind flags to Viper to allow them to override config file settings.
	// This makes the flag value available via viper.GetString().
	viper.BindPFlag("logger.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("output.format", rootCmd.PersistentFlags().Lookup("output"))

	// Add subcommands to the root command.
	// These are defined in their own files (e.g., diagnose.go, ask.go).
	rootCmd.AddCommand(newDiagnoseCmd())
	rootCmd.AddCommand(newAskCmd())
	rootCmd.AddCommand(newFixCmd())
	// TODO: Add other commands like `status`, `plugin`, `config`.

	// Add a built-in 'version' command.
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number of KubeStack-AI",
		Run: func(cmd *cobra.Command, args []string) {
			// In a real application, version info would come from the version package and be set at build time.
			fmt.Println("KubeStack-AI v0.1.0")
		},
	})
}

//Personal.AI order the ending
