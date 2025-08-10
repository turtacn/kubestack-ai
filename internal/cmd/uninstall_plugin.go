package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/turtacn/kubestack-ai/internal/plugins"
)

// NewUninstallPluginCmd 创建卸载插件命令。NewUninstallPluginCmd creates uninstall plugin command.
func NewUninstallPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall-plugin [name]",
		Short: "Uninstall a plugin",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Name required")
				return
			}
			err := plugins.Manager.Uninstall(args[0])
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("Plugin uninstalled")
		},
	}
	return cmd
}

//Personal.AI order the ending
