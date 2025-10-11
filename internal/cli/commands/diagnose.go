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
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/spf13/cobra"
)

// newDiagnoseCmd creates and configures the `diagnose` command.
// This command is responsible for running a comprehensive diagnosis on a specified middleware instance.
// It defines the command's usage, descriptions, examples, arguments, and the main execution logic (`RunE`).
// The RunE function parses arguments, builds a diagnosis request, executes the diagnosis via the
// orchestrator, and prints a formatted report of the findings.
//
// Returns:
//   *cobra.Command: A pointer to the configured cobra.Command object for the `diagnose` command.
func newDiagnoseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnose [middleware-type] [instance-name]",
		Short: "Run a diagnosis on a middleware instance",
		Long: `Runs a comprehensive diagnosis on a specified middleware instance.
It collects data, analyzes it for common issues, and provides a report with actionable recommendations.`,
		Example: `  # Diagnose a Redis instance named 'my-redis' in the 'default' namespace
  ksa diagnose redis my-redis -n default`,
		Args: cobra.ExactArgs(2), // Enforces that exactly two arguments are provided.
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Parse arguments and flags.
			middlewareTypeStr := strings.ToLower(args[0])
			instanceName := args[1]
			namespace, _ := cmd.Flags().GetString("namespace")

			// 2. The orchestrator is now initialized in root.go's PersistentPreRunE.
			if orchestrator == nil {
				return fmt.Errorf("orchestrator not initialized")
			}

			// 3. Validate and build the diagnosis request.
			var middlewareType enum.MiddlewareType
			// This is a simplified validation. A real implementation might use a map or a more robust parser.
			switch middlewareTypeStr {
			case "redis":
				middlewareType = enum.Redis
			case "mysql":
				middlewareType = enum.MySQL
			case "kafka":
				middlewareType = enum.Kafka
			case "elasticsearch":
				middlewareType = enum.Elasticsearch
			default:
				return fmt.Errorf("unsupported middleware type: '%s'. Supported types are: redis, mysql, kafka, elasticsearch", middlewareTypeStr)
			}

			req := &models.DiagnosisRequest{
				TargetMiddleware: middlewareType,
				Instance:         instanceName,
				Namespace:        namespace,
			}

			// 4. Set up UI components and execute the diagnosis.
			fmt.Println("Running diagnosis...")

			progressChan := make(chan interfaces.DiagnosisProgress)
			go func() {
				for p := range progressChan {
					fmt.Printf(" > %s: %s\n", p.Step, p.Message) // Simple text progress
				}
			}()

			result, err := orchestrator.ExecuteDiagnosis(cmd.Context(), req, progressChan)

			if err != nil {
				return fmt.Errorf("diagnosis failed: %w", err)
			}

			// 5. Format and print the result.
			// TODO: Use the formatter from `internal/cli/ui`.
			// formatter := ui.NewFormatter(outputFormat)
			// return formatter.Print(result)
			fmt.Printf("\n--- Diagnosis Report ---\n")
			fmt.Printf("Status: %s\n", result.Status)
			fmt.Printf("Summary: %s\n", result.Summary)
			for _, issue := range result.Issues {
				fmt.Printf("\n[!] %s (%s)\n", issue.Title, issue.Severity)
				fmt.Printf("  ├─ Description: %s\n", issue.Description)
				fmt.Printf("  ├─ Evidence: %s\n", issue.Evidence)
				for _, rec := range issue.Recommendations {
					fmt.Printf("  └─ Recommendation: %s\n", rec.Description)
				}
			}

			return nil
		},
	}

	// Add command-specific flags.
	cmd.Flags().StringP("namespace", "n", "default", "The namespace of the middleware instance (for Kubernetes)")
	return cmd
}

//Personal.AI order the ending
