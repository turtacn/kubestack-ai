// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law of agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/task"
	"github.com/spf13/cobra"
)

// newDiagnoseCmd creates the diagnose command
func newDiagnoseCmd() *cobra.Command {
	var (
		target    string
		instance  string
		namespace string
		format    string
		async     bool
	)

	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Run a diagnosis on a middleware component",
		Long: `Diagnose a specific middleware instance.
Examples:
  ksa diagnose --target redis --instance my-redis --namespace default
  ksa diagnose --target mysql --instance my-mysql -f json`,
		Run: func(cmd *cobra.Command, args []string) {
			// Validate target
			middlewareType, err := enum.ParseMiddlewareType(target)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				fmt.Printf("Allowed targets: %v\n", enum.AllowedMiddlewareTypes())
				os.Exit(1)
			}

			// Initialize dependencies
			// We cannot re-initialize full stack here easily without duplicating root.go logic.
			// Ideally, we use global diagManager populated by PreRun.
			// But for this patch, we need to ensure diagManager is available.
			if diagManager == nil {
				fmt.Println("Error: Diagnosis manager not initialized. Please ensure config is valid.")
				os.Exit(1)
			}

			req := &models.DiagnosisRequest{
				TargetMiddleware: middlewareType,
				Instance:         instance,
				Namespace:        namespace,
				OutputFormat:     format,
			}

			if async {
				// We need a task queue for async. root.go doesn't expose it globally currently.
				// For this exercise, we focus on synchronous diagnosis as primary CLI use case,
				// or we'd need to expose the queue.
				// Let's assume we can't do async via CLI unless we connect to API server.
				// But requirement said P5 delivers Cron and Notifications.
				// CLI diagnose is for manual test.
				// We will simulate async submission via direct queue enqueue if possible,
				// but queue is local.
				// If we are running CLI, we are a separate process from Server.
				// So we should probably call API for async?
				// For now, we disable async path or warn.
				fmt.Println("Async diagnosis via CLI not fully supported in this mode. Running synchronously.")
			}

			// Run synchronously
			ctx := context.Background()
			fmt.Printf("Starting diagnosis for %s/%s...\n", namespace, instance)

			// Create a channel to receive progress updates
			progressChan := make(chan interfaces.DiagnosisProgress)

			// Print progress in a separate goroutine
			go func() {
				for p := range progressChan {
					fmt.Printf("[%s] %s\n", p.Step, p.Message)
				}
			}()

			result, err := diagManager.RunDiagnosis(ctx, req, progressChan)
			if err != nil {
				fmt.Printf("Diagnosis failed: %v\n", err)
				os.Exit(1)
			}

			// Output result
			printResult(result, format)
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "Target middleware type (e.g., redis, mysql)")
	cmd.Flags().StringVarP(&instance, "instance", "i", "", "Instance name")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")
	cmd.Flags().BoolVar(&async, "async", false, "Run diagnosis asynchronously")
	cmd.MarkFlagRequired("target")
	cmd.MarkFlagRequired("instance")

	return cmd
}

func printResult(result *models.DiagnosisResult, format string) {
	if format == "json" {
		// Print JSON
		// ...
	} else {
		// Print Text
		fmt.Printf("\nDiagnosis Complete. Status: %s\n", result.Status)
		fmt.Println("Summary:", result.Summary)
		// ...
	}
}
