package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/turtacn/kubestack-ai/internal/pluginmgr"
)

// NewInstallPluginCmd 创建安装插件命令。NewInstallPluginCmd creates install plugin command.
func NewInstallPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install-plugin [name] [source]",
		Short: "Install a plugin",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				fmt.Println("Name and source required")
				return
			}
			err := pluginmgr.Manager.Install(args[0], args[1])
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("Plugin installed")
		},
	}
	return cmd
}

//Personal.AI order the ending
