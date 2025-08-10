package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/plugins"
)

// NewUninstallPluginCmd 创建卸载插件命令。NewUninstallPluginCmd creates uninstall plugin command.
func NewUninstallPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall-plugin [name]",
		Short: "卸载中间件插件。Uninstall a middleware plugin",
		Long: `卸载指定的中间件插件，将从系统中移除该插件及其相关配置。
Uninstall specified middleware plugin, removing it and its configuration from the system.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			logging.Logger.Infof("Uninstalling plugin: %s", name)

			// 检查插件是否已安装。Check if plugin is installed.
			currentStatus := plugins.Manager.GetPluginStatus(name)
			if currentStatus == plugins.PluginStatusUninstalled {
				fmt.Printf("插件 %s 未安装。Plugin %s is not installed.\n", name, name)
				return
			}

			// 确认卸载（如果需要）。Confirm uninstallation if needed.
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				fmt.Printf("确定要卸载插件 %s 吗? [y/N] ", name)
				var confirmation string
				fmt.Scanln(&confirmation)
				if confirmation != "y" && confirmation != "Y" {
					fmt.Println("卸载已取消。Uninstallation cancelled.")
					return
				}
			}

			// 执行卸载。Execute uninstallation.
			err := plugins.Manager.Uninstall(name)
			if err != nil {
				logging.Logger.Errorf("Plugin uninstallation failed: %v", err)
				fmt.Printf("插件卸载失败: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("插件 %s 已成功卸载。Plugin %s has been uninstalled successfully.\n", name, name)
		},
	}

	cmd.Flags().BoolP("force", "f", false, "强制卸载，无需确认。Force uninstall without confirmation")

	return cmd
}

//Personal.AI order the ending
