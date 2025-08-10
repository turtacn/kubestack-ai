package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/plugins"
)

// NewInstallPluginCmd 创建安装插件命令。NewInstallPluginCmd creates install plugin command.
func NewInstallPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install-plugin [name] [source]",
		Short: "安装中间件插件。Install a middleware plugin",
		Long: `安装指定的中间件插件，用于扩展系统对特定中间件的支持能力。
Install specified middleware plugin to extend system's support capabilities for specific middleware.`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			source := args[1]

			logging.Logger.Infof("Installing plugin: %s from source: %s", name, source)

			// 执行插件安装。Execute plugin installation.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			err := plugins.Manager.Install(ctx, name, source)
			if err != nil {
				logging.Logger.Errorf("Plugin installation failed: %v", err)
				fmt.Printf("插件安装失败: %v\n", err)
				os.Exit(1)
			}

			// 验证安装状态。Verify installation status.
			status := plugins.Manager.GetPluginStatus(name)
			if status == plugins.PluginStatusInstalled || status == plugins.PluginStatusActive {
				fmt.Printf("插件 %s 安装成功。Plugin %s installed successfully.\n", name, name)
				fmt.Println("可以使用以下命令进行诊断:")
				fmt.Printf("  ksa diagnose %s\n", name)
			} else {
				fmt.Printf("插件 %s 安装完成，但状态异常: %s\n", name, status)
				fmt.Println("请检查日志以获取详细信息。Please check logs for details.")
				os.Exit(1)
			}
		},
	}

	// 添加插件安装相关标志。Add plugin installation flags.
	cmd.Flags().BoolP("force", "f", false, "强制重新安装已存在的插件。Force reinstall if plugin exists")
	cmd.Flags().StringP("version", "v", "", "指定插件版本。Specify plugin version")

	return cmd
}

//Personal.AI order the ending
