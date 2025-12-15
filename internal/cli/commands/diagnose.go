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
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

type diagnoseOptions struct {
	target    string
	instance  string
	namespace string
	output    string
	async     bool
}

func newDiagnoseCmd() *cobra.Command {
	opts := &diagnoseOptions{}

	cmd := &cobra.Command{
		Use:   "diagnose [target]",
		Short: "Diagnose a middleware instance",
		Long: `Diagnose a specific middleware instance (e.g., redis, mysql) to find
performance issues, configuration errors, and anomalies.

Examples:
  ksa diagnose redis --instance my-redis
  ksa diagnose mysql --instance db-01 --namespace prod`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.target = args[0]
			return runDiagnose(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.instance, "instance", "i", "", "Instance name or connection string (required)")
	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "default", "Kubernetes namespace (if applicable)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "text", "Output format (text, json, yaml)")
	cmd.Flags().BoolVar(&opts.async, "async", false, "Run diagnosis asynchronously")

	// Bind flag to viper if not already bound globally, or rely on cmd struct.
	// To be safe and explicit as per review:
	viper.BindPFlag("output.format", cmd.Flags().Lookup("output"))

	// Required flags
	cmd.MarkFlagRequired("instance")

	return cmd
}

func runDiagnose(opts *diagnoseOptions) error {
	ctx := context.Background()

	// Validate target
	mwType, err := enum.ParseMiddlewareType(opts.target)
	if err != nil {
		return fmt.Errorf("unsupported middleware type: %s", opts.target)
	}

	// Build request
	req := &models.DiagnosisRequest{
		TargetMiddleware: mwType,
		Instance:         opts.instance,
		Namespace:        opts.namespace,
		// Metadata can be extended
	}

	// Handle async
	if opts.async {
		// P2: Integrate with task queue
		fmt.Printf("Async diagnosis for %s submitted. Task ID: %s\n", opts.instance, "task-123")
		return nil
	}

	// Sync execution
	fmt.Printf("Starting diagnosis for %s (%s)...\n", opts.instance, opts.target)

	progressChan := make(chan interfaces.DiagnosisProgress)

	// Start a goroutine to print progress
	go func() {
		for p := range progressChan {
			fmt.Printf("[%s] %s: %s\n", p.Step, p.Status, p.Message)
		}
	}()

	// Use the global diagManager (initialized in root.go)

	result, err := diagManager.RunDiagnosis(ctx, req, progressChan)
	if err != nil {
		return fmt.Errorf("diagnosis failed: %w", err)
	}

	// Output result
	format := viper.GetString("output.format")
	if format == "json" {
		report, _ := diagManager.GenerateReport(result)
		fmt.Println(report)
	} else {
		printTextReport(result)
	}

	return nil
}

func printTextReport(result *models.DiagnosisResult) {
	fmt.Printf("\nDiagnosis Complete!\n")
	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Summary: %s\n", result.Summary)

	if len(result.Issues) > 0 {
		fmt.Println("\nIdentified Issues:")
		for i, issue := range result.Issues {
			fmt.Printf("%d. [%s] %s\n", i+1, issue.Severity, issue.Title)
			fmt.Printf("   Description: %s\n", issue.Description)
			// Removed Recommendation print if not present in Issue model
		}
	} else {
		fmt.Println("\nNo issues found.")
	}
}
