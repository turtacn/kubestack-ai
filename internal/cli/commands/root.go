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

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/execution"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	orch "github.com/kubestack-ai/kubestack-ai/internal/core/orchestrator"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/client"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/manager"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	// orchestrator is the central component that coordinates all application logic.
	// It is initialized in PersistentPreRunE to ensure all dependencies are ready.
	orchestrator interfaces.Orchestrator
)

var rootCmd = &cobra.Command{
	Use:   "ksa",
	Short: "KubeStack-AI is an intelligent SRE assistant for middleware.",
	Long: `KubeStack-AI is a command-line tool that uses AI to help you diagnose,
analyze, and fix issues with your middleware infrastructure, whether it is
running on Kubernetes or bare metal servers.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 1. Load configuration
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// 2. Initialize logger
		logCfg := cfg.Logger
		logCfg.Level = viper.GetString("logger.level")
		logger.InitGlobalLogger(&logCfg)
		log := logger.GetLogger()
		log.Debugf("Logger initialized with level: %s", logCfg.Level)

		// 3. Validate configuration
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		// 4. Initialize all core components (Dependency Injection)
		llmClient, err := client.NewClientFromConfig(&cfg.LLM)
		if err != nil {
			return fmt.Errorf("failed to create LLM client: %w", err)
		}

		// Plugin components
		pluginRegistry, err := manager.NewRegistry([]string{cfg.Plugins.Directory})
		if err != nil {
			return fmt.Errorf("failed to create plugin registry: %w", err)
		}
		pluginLoader := manager.NewLoader()
		pluginManager := manager.NewManager(pluginRegistry, pluginLoader)

		// Diagnosis components
		ruleAnalyzer := diagnosis.NewRuleBasedAnalyzer(nil, nil)
		aiAnalyzer, err := diagnosis.NewAIAnalyzer(llmClient)
		if err != nil {
			return fmt.Errorf("failed to create AI analyzer: %w", err)
		}
		analyzers := []interfaces.DiagnosisAnalyzer{ruleAnalyzer, aiAnalyzer}
		diagManager := diagnosis.NewManager(pluginManager, analyzers, "reports")

		// Execution components (using placeholder)
		execManager := &execution.PlaceholderManager{}

		// --- Orchestrator ---
		// Warning: Missing KnowledgeManager and other components for RAG.
		// Passing nil for now as Phase 6 focuses on API/Web.
		// In a real integration, we'd initialize RAGEngine here.
		orchestrator = orch.NewOrchestrator(cfg, pluginManager, diagManager, execManager, nil, llmClient, nil, nil, nil)
		log.Info("Orchestrator and all dependencies initialized successfully.")

		return nil
	},
}

// Execute is the main entry point for the command-line interface.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// init is a special Go function that is executed when the package is initialized.
func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/kubestack-ai/config.yaml or $HOME/.ksa.yaml)")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error, fatal)")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format (text, json, yaml)")

	viper.BindPFlag("logger.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("output.format", rootCmd.PersistentFlags().Lookup("output"))

	rootCmd.AddCommand(newDiagnoseCmd())
	rootCmd.AddCommand(newAskCmd())
	rootCmd.AddCommand(newFixCmd())
	rootCmd.AddCommand(newServerCmd())

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number of KubeStack-AI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("KubeStack-AI v0.1.0")
		},
	})
}