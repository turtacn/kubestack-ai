package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/turtacn/kubestack-ai/internal/ai"
	"github.com/turtacn/kubestack-ai/internal/cmd"
	//"github.com/turtacn/kubestack-ai/internal/collectors"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/plugins"
)

// 主函数：CLI入口，初始化系统组件并执行命令。Main function: CLI entry point, initializes system components and executes commands.
func main() {
	// 初始化日志记录器。Initialize logger.
	logging.InitLogger()
	logging.Logger.Info("Starting KubeStack-AI CLI")

	// 初始化插件管理器。Initialize plugin manager.
	plugins.InitManager()

	// 初始化RAG系统。Initialize RAG system.
	rag := ai.NewRAG()
	// 预加载知识库。Preload knowledge base.
	loadKnowledgeBase(rag)

	// 创建根命令。Create root command.
	rootCmd := &cobra.Command{
		Use:   "ksa",
		Short: "KubeStack-AI: AI-powered middleware management CLI",
		Long:  `A unified AI assistant for diagnosing and optimizing middleware on Kubernetes and bare-metal.`,
	}

	// 添加子命令，如诊断命令。Add subcommands, e.g., diagnose command.
	rootCmd.AddCommand(cmd.NewDiagnoseCmd())
	rootCmd.AddCommand(cmd.NewInstallPluginCmd())
	rootCmd.AddCommand(cmd.NewUninstallPluginCmd())
	rootCmd.AddCommand(cmd.NewQueryCmd(rag))

	// 执行根命令。Execute root command.
	if err := rootCmd.Execute(); err != nil {
		logging.Logger.Error("Command execution failed", err)
		os.Exit(1)
	}
}

// 加载知识库数据。Load knowledge base data.
func loadKnowledgeBase(rag ai.RAG) {
	// 示例：加载MySQL和Redis的文档。Example: load MySQL and Redis documents.
	docs := map[string][]string{
		"mysql": {
			"MySQL slow query log can be enabled with slow_query_log = 1",
			"High connections may indicate connection pool issues",
		},
		"redis": {
			"Redis memory usage can be checked with INFO memory",
			"RDB persistence is configured via save directives",
		},
	}
	if err := rag.EmbedAndStore(docs); err != nil {
		logging.Logger.Warn("Failed to load knowledge base", err)
	}
}

//Personal.AI order the ending
