package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/turtacn/kubestack-ai/internal/diagnosis"
	// 假设engine实例。Assume engine instance.
)

// NewDiagnoseCmd 创建诊断命令。NewDiagnoseCmd creates diagnose command.
func NewDiagnoseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnose [middleware]",
		Short: "Diagnose middleware issues",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Middleware required")
				return
			}
			// 示例engine。Example engine.
			engine := diagnosis.NewEngine(nil, nil, nil)
			result, err := engine.Diagnose(args[0])
			if err != nil {
				fmt.Println(err)
				return
			}
			jsonOutput, _ := json.Marshal(result)
			fmt.Println(string(jsonOutput))
		},
	}
	return cmd
}

//Personal.AI order the ending
