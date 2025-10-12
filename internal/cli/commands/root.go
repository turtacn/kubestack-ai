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
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/execution"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/orchestrator"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/client"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/manager"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// cfgFile holds the path to the configuration file provided via a command-line flag.
	cfgFile string

	// rootCmd represents the base command when called without any subcommands.
	// It is the root of the command tree and is responsible for global setup,
	// such as initializing configuration and logging.
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

// Execute is the main entry point for the command-line interface.
// It executes the root command, which in turn handles all subcommand logic.
// This function is called directly by `main.main()` and is the starting point
// for the entire application's command-line functionality. If the root command
// returns an error, it prints the error and exits with a non-zero status code.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra prints the error, so we just need to exit with a non-zero code.
		os.Exit(1)
	}
}

// init is a special Go function that is executed when the package is initialized.
// It sets up global flags and ensures that the dependency injection for the
// orchestrator is configured *after* Cobra has parsed the flags and loaded the config.
func init() {
	// The `cobra.OnInitialize` function allows us to defer the complex setup
	// until after the config file path and other flags have been parsed.
	cobra.OnInitialize(initConfigAndOrchestrator)

	// Add global flags that will apply to all subcommands.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/kubestack-ai/config.yaml or $HOME/.ksa.yaml)")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error, fatal)")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format (text, json, yaml)")

	// Bind flags to Viper to allow them to override config file settings.
	viper.BindPFlag("logger.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("output.format", rootCmd.PersistentFlags().Lookup("output"))

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

// initConfigAndOrchestrator is the main dependency injection container for the application.
// It is responsible for loading the configuration, initializing all core services
// (logger, LLM client, stores, managers), wiring them together, and attaching the
// fully configured commands to the root command.
func initConfigAndOrchestrator() {
	// 1. Load configuration. This is called first to ensure all components get the correct config.
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize logger.
	logCfg := cfg.Logger
	logCfg.Level = viper.GetString("logger.level")
	logger.InitGlobalLogger(&logCfg)
	log := logger.GetLogger()

	// 3. Validate configuration.
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}
	log.Debug("Configuration loaded and validated successfully.")

	// 4. Initialize Core Components (Dependency Injection).
	// This is where we build the object graph.

	// LLM Client
	llmClient, err := client.NewClient(&cfg.LLM)
	if err != nil {
		log.Fatalf("Failed to initialize LLM client: %v", err)
	}
	log.Debugf("Initialized LLM client with provider: %s", cfg.LLM.Provider)

	// Knowledge Base
	vectorStore, err := store.NewVectorStore(&cfg.Knowledge)
	if err != nil {
		log.Fatalf("Failed to initialize vector store: %v", err)
	}
	log.Debugf("Initialized vector store with provider: %s", cfg.Knowledge.Provider)
	// TODO: Initialize and use the DocumentStore and KnowledgeManager here.
	_ = vectorStore // Temporarily use vectorStore to avoid "declared and not used"

	// Analyzers
	ruleAnalyzer := diagnosis.NewRuleBasedAnalyzer(nil, nil) // No rules for now
	aiAnalyzer := diagnosis.NewAIAnalyzer(llmClient)
	analyzers := []interfaces.DiagnosisAnalyzer{ruleAnalyzer, aiAnalyzer}
	log.Debug("Initialized diagnosis analyzers.")

	// Managers
	pluginRegistry, err := manager.NewRegistry([]string{cfg.Plugins.Directory})
	if err != nil {
		log.Fatalf("Failed to initialize plugin registry: %v", err)
	}
	pluginLoader := manager.NewLoader()
	pluginManager := manager.NewManager(pluginRegistry, pluginLoader)
	diagnosisManager := diagnosis.NewManager(pluginManager, analyzers)
	executionManager := execution.NewManager(nil) // No planner for now
	log.Debug("Initialized core managers.")

	// Orchestrator
	orchestrator := orchestrator.NewOrchestrator(cfg, pluginManager, diagnosisManager, executionManager)
	log.Debug("Orchestrator initialized.")

	// 5. Add Subcommands with the fully initialized orchestrator.
	// By passing the orchestrator to the command constructors, we are injecting the
	// application's core logic into the UI layer.
	rootCmd.AddCommand(newDiagnoseCmd(orchestrator))
	rootCmd.AddCommand(newAskCmd(orchestrator))
	rootCmd.AddCommand(newFixCmd(orchestrator))
}

//Personal.AI order the ending
