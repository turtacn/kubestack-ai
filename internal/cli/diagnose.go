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

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

var (
	targetMiddleware string
	namespace        string
	instance         string
	outputJSON       bool
)

// NewDiagnoseCommand creates the `diagnose` command.
func NewDiagnoseCommand(manager interfaces.DiagnosisManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Run a diagnosis on a middleware instance",
		Long:  `Triggers the diagnosis engine to analyze the specified middleware instance and report issues.`,
		Run: func(cmd *cobra.Command, args []string) {
			runDiagnose(manager)
		},
	}

	cmd.Flags().StringVarP(&targetMiddleware, "target", "t", "", "Target middleware type (e.g., redis, mysql)")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().StringVarP(&instance, "instance", "i", "", "Instance name")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output result in JSON format")

	cmd.MarkFlagRequired("target")
	cmd.MarkFlagRequired("instance")

	return cmd
}

func runDiagnose(manager interfaces.DiagnosisManager) {
	mwType, err := enum.ParseMiddlewareType(targetMiddleware)
	if err != nil {
		fmt.Printf("Error: Invalid middleware type '%s'. Allowed: %v\n", targetMiddleware, enum.AllowedMiddlewareTypes())
		os.Exit(1)
	}

	req := &models.DiagnosisRequest{
		TargetMiddleware: mwType,
		Namespace:        namespace,
		Instance:         instance,
		OutputFormat:     "text",
	}

	if outputJSON {
		req.OutputFormat = "json"
	}

	// Progress channel
	progressChan := make(chan interfaces.DiagnosisProgress)

	// Handle progress
	go func() {
		for p := range progressChan {
			if !outputJSON {
				fmt.Printf("[%s] %s: %s\n", p.Step, p.Status, p.Message)
			}
		}
	}()

	ctx := context.Background()
	result, err := manager.RunDiagnosis(ctx, req, progressChan)
	if err != nil {
		fmt.Printf("Diagnosis failed: %v\n", err)
		os.Exit(1)
	}

	if outputJSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		printTextResult(result)
	}
}

func printTextResult(result *models.DiagnosisResult) {
	fmt.Println("\n=== Diagnosis Result ===")
	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Summary: %s\n", result.Summary)

	if len(result.Issues) > 0 {
		fmt.Println("\n--- Issues Found ---")
		for _, issue := range result.Issues {
			fmt.Printf("\nTitle: %s\n", issue.Title)
			fmt.Printf("Severity: %s\n", issue.Severity)
			fmt.Printf("Description: %s\n", issue.Description)
			if len(issue.Recommendations) > 0 {
				fmt.Println("Recommendations:")
				for _, rec := range issue.Recommendations {
					fmt.Printf("  - %s\n", rec.Description)
				}
			}
		}
	} else {
		fmt.Println("\nNo issues found.")
	}
}
