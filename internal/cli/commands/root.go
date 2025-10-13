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
	"github.com/kubestack-ai/kubestack-ai/internal/core/orchestrator"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/client"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/rag"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/manager"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd encapsulates the main command and its dependencies.
type rootCmd struct {
	cmd           *cobra.Command
	cfg           *config.Config
	log           logger.Logger
	orchestrator  interfaces.Orchestrator
}

// NewRootCmd creates a new instance of the application's root command.
func NewRootCmd() *rootCmd {
	root := &rootCmd{}
	root.cmd = &cobra.Command{
		Use:   "ksa",
		Short: "KubeStack-AI is an intelligent SRE assistant for middleware.",
		Long: `KubeStack-AI is a command-line tool that uses AI to help you diagnose,
analyze, and fix issues with your middleware infrastructure, whether it is
running on Kubernetes or bare metal servers.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return root.initConfigAndOrchestrator()
		},
	}

	root.cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kubestack-ai.yaml)")
	root.cmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error, fatal)")
	root.cmd.PersistentFlags().StringP("output", "o", "text", "Output format (text, json, yaml)")
	viper.BindPFlag("logger.level", root.cmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("output.format", root.cmd.PersistentFlags().Lookup("output"))

	return root
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	root := NewRootCmd()
	if err := root.cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func (r *rootCmd) initConfigAndOrchestrator() error {
	// 1. Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	r.cfg = cfg

	// 2. Initialize logger
	logCfg := r.cfg.Logger
	logCfg.Level = viper.GetString("logger.level")
	logger.InitGlobalLogger(&logCfg)
	r.log = logger.GetLogger()

	// 3. Validate configuration
	if err := r.cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	r.log.Debug("Configuration loaded and validated successfully.")

	// 4. Initialize Core Components
	llmClient, err := client.NewClient(&r.cfg.LLM)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	vectorStore, err := store.NewVectorStore(&r.cfg.Knowledge)
	if err != nil {
		return fmt.Errorf("failed to initialize vector store: %w", err)
	}

	documentStore, err := store.NewDocumentStore(&r.cfg.Knowledge)
	if err != nil {
		return fmt.Errorf("failed to initialize document store: %w", err)
	}

	embedder, err := rag.NewEmbedder(llmClient, "")
	if err != nil {
		return fmt.Errorf("failed to initialize embedder: %w", err)
	}

	retriever, err := rag.NewRetriever(embedder, vectorStore)
	if err != nil {
		return fmt.Errorf("failed to initialize retriever: %w", err)
	}

	hybridSearcher, err := search.NewHybridSearcher(documentStore, retriever)
	if err != nil {
		return fmt.Errorf("failed to initialize hybrid searcher: %w", err)
	}

	knowledgeManager, err := knowledge.NewManager(hybridSearcher)
	if err != nil {
		return fmt.Errorf("failed to initialize knowledge manager: %w", err)
	}
	r.log.Debug("Knowledge base initialized successfully.")

	ruleAnalyzer := diagnosis.NewRuleBasedAnalyzer(nil, nil)
	aiAnalyzer := diagnosis.NewAIAnalyzer(llmClient)
	analyzers := []interfaces.DiagnosisAnalyzer{ruleAnalyzer, aiAnalyzer}

	pluginRegistry, err := manager.NewRegistry([]string{r.cfg.Plugins.Directory})
	if err != nil {
		return fmt.Errorf("failed to initialize plugin registry: %w", err)
	}

	pluginLoader := manager.NewLoader()
	pluginManager := manager.NewManager(pluginRegistry, pluginLoader)
	diagnosisManager := diagnosis.NewManager(pluginManager, analyzers)
	executionManager := execution.NewManager(nil)

	r.orchestrator = orchestrator.NewOrchestrator(r.cfg, pluginManager, diagnosisManager, executionManager, knowledgeManager, llmClient)
	r.log.Debug("Orchestrator initialized successfully.")

	// 5. Add Subcommands
	r.cmd.AddCommand(newDiagnoseCmd(r.orchestrator))
	r.cmd.AddCommand(newAskCmd(r.orchestrator))
	r.cmd.AddCommand(newFixCmd(r.orchestrator))
	r.cmd.AddCommand(newVersionCmd())

	return nil
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of KubeStack-AI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("KubeStack-AI v0.1.0")
		},
	}
}