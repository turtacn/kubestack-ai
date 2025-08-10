package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/turtacn/kubestack-ai/internal/cmd"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/plugins"
)

// 主函数：CLI入口，初始化系统组件并执行命令。Main function: CLI entry point, initializes system components and executes commands.
func main() {
	// 初始化日志记录器。Initialize logger.
	logging.InitLogger()

	// 初始化插件管理器。Initialize plugin manager.
	plugins.InitManager()

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

	// 执行根命令。Execute root command.
	if err := rootCmd.Execute(); err != nil {
		logging.Logger.Error(err)
		os.Exit(1)
	}

}

//Personal.AI order the ending
