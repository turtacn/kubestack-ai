package commands

import (
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/api"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/client"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/manager"
	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the KubeStack-AI REST API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Re-load config to ensure we have everything
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

            // Initialize dependencies (similar to root, but specialized for server if needed)
            // Ideally, we reuse the initialization from root, but cobra's PersistentPreRun
            // has already run.
            // The `orchestrator` global variable in root.go is accessible if we are in same package.
            // But orchestrator doesn't expose DiagnosisManager directly via Getter.
            // We can either expose it or re-init. Re-init is safer for now to avoid coupling with global state hacks.

            llmClient, err := client.NewClientFromConfig(&cfg.LLM)
            if err != nil {
                return fmt.Errorf("failed to create LLM client: %w", err)
            }

            pluginRegistry, err := manager.NewRegistry([]string{cfg.Plugins.Directory})
            if err != nil {
                return fmt.Errorf("failed to create plugin registry: %w", err)
            }
            pluginLoader := manager.NewLoader()
            pluginManager := manager.NewManager(pluginRegistry, pluginLoader)

            ruleAnalyzer := diagnosis.NewRuleBasedAnalyzer(nil, nil)
            aiAnalyzer, err := diagnosis.NewAIAnalyzer(llmClient)
            if err != nil {
                return fmt.Errorf("failed to create AI analyzer: %w", err)
            }
            analyzers := []interfaces.DiagnosisAnalyzer{ruleAnalyzer, aiAnalyzer}
            diagManager := diagnosis.NewManager(pluginManager, analyzers, nil, "reports")

            server := api.NewServer(cfg, diagManager)
            return server.Start(cmd.Context())
		},
	}

    cmd.AddCommand(&cobra.Command{
        Use: "start",
        Short: "Start the server",
        RunE: func(c *cobra.Command, args []string) error {
            return cmd.RunE(c, args)
        },
    })

	return cmd
}
