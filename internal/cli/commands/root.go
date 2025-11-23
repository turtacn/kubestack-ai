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
	"context"
	"fmt"
	"os"

	"github.com/kubestack-ai/kubestack-ai/internal/cli"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/execution"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
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
	// diagManager is needed for the CLI diagnose command
	diagManager interfaces.DiagnosisManager
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
		diagManager = diagnosis.NewManager(pluginManager, analyzers, nil, "reports")

		// Execution components (using placeholder)
		execManager := &execution.PlaceholderManager{}

		// --- Orchestrator ---
		// Warning: Missing KnowledgeManager and other components for RAG.
		// Passing nil for now as Phase 6 focuses on API/Web.
		// In a real integration, we'd initialize RAGEngine here.
		orchestrator = orch.NewOrchestrator(cfg, pluginManager, diagManager, execManager, nil, llmClient, nil, nil, nil)
		log.Info("Orchestrator and all dependencies initialized successfully.")

		// HACK: Update the diagnose command with the initialized manager.
		// Since cobra commands are static, we need a way to pass the manager.
		// The cli.NewDiagnoseCommand takes manager as arg.
		// But PersistentPreRunE runs AFTER command selection but BEFORE command execution.
		// So we can't swap the command out.
		// Solution: Use a wrapper or global variable in this package that the command uses?
		// Or, better, we reconstruct the command tree? No, that's too late.
		// The `cli.NewDiagnoseCommand` expects manager at construction time.
		// Since we only have manager at runtime (after config load), we need a lazy wrapper.
		// For now, I will assume the command will look up `diagManager` or `orchestrator` from this package if I expose it,
		// OR I'll change `NewDiagnoseCommand` to take a provider function or I'll make `rootCmd` Run logic call a setup function.

		// Actually, the best pattern with Cobra and DI is to have the command struct hold dependencies,
		// and inject them. But since we load config inside PreRun, we are stuck.
		// One way is to let the command access the global `orchestrator` or `diagManager` variable defined here.
		// The `cli` package command I wrote takes `manager` as arg.
		// I will modify `root.go` to NOT add `cli.NewDiagnoseCommand(nil)` at init,
		// but instead use a proxy command that calls the implementation using the global variable.
		// BUT `cli.NewDiagnoseCommand` defines flags. We need those flags registered at init.

		// Let's revert to using `newDiagnoseCmd` (the one I just deleted? Oops).
		// The plan was to "Implement CLI Command in internal/cli/diagnose.go".
		// I did that. And I deleted `internal/cli/commands/diagnose.go`.
		// Now I need to make sure `internal/cli/diagnose.go` works.
		// It takes `manager` in constructor.
		// To make it work with late binding, I will modify `internal/cli/diagnose.go` to accept `nil`
		// and then in `Run` it should check if manager is nil, or I pass a "Getter".

		// However, `internal/cli/diagnose.go` is in `cli` package. `root.go` is in `commands` package.
		// `root.go` calls `cli.NewDiagnoseCommand(...)`.
		// I will add the command in `init()` but pass a *wrapper* that delegates to the real manager.
		// Or simply: The `diagnose` command in `cli` package should use a global or passed-in orchestrator.

		// Let's stick to the global `diagManager` in `root.go`.
		// I will define a wrapper struct implementing `DiagnosisManager` that delegates to `diagManager`.

		return nil
	},
}

// lazyDiagManager delegates to the global diagManager initialized in PreRun
type lazyDiagManager struct{}
func (l *lazyDiagManager) RunDiagnosis(ctx context.Context, req *models.DiagnosisRequest, ch chan<- interfaces.DiagnosisProgress) (*models.DiagnosisResult, error) {
	if diagManager == nil {
		return nil, fmt.Errorf("diagnosis manager not initialized")
	}
	return diagManager.RunDiagnosis(ctx, req, ch)
}
func (l *lazyDiagManager) AnalyzeData(ctx context.Context, req *models.DiagnosisRequest, data *models.CollectedData) ([]*models.Issue, error) {
	if diagManager == nil {
		return nil, fmt.Errorf("diagnosis manager not initialized")
	}
	return diagManager.AnalyzeData(ctx, req, data)
}
func (l *lazyDiagManager) GenerateReport(result *models.DiagnosisResult) (string, error) {
	if diagManager == nil {
		return "", fmt.Errorf("diagnosis manager not initialized")
	}
	return diagManager.GenerateReport(result)
}
func (l *lazyDiagManager) GetDiagnosisResult(id string) (*models.DiagnosisResult, error) {
	if diagManager == nil {
		return nil, fmt.Errorf("diagnosis manager not initialized")
	}
	return diagManager.GetDiagnosisResult(id)
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

	// Use the lazy wrapper
	rootCmd.AddCommand(cli.NewDiagnoseCommand(&lazyDiagManager{}))

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