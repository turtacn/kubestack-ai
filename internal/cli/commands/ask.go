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

	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/spf13/cobra"
)

// newAskCmd creates and configures the `ask` command.
// This command allows users to ask questions in natural language to the KubeStack-AI assistant.
func newAskCmd(orchestrator interfaces.Orchestrator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask [question...]",
		Short: "Ask KubeStack-AI a question in natural language",
		Long: `Send a natural language question to the KubeStack-AI assistant.
The assistant will use its knowledge base and diagnostic capabilities to provide an answer.
The response will be streamed to the console in real-time.`,
		Example: `  # Ask a general question about a technology
  ksa ask what is redis persistence?

  # Ask for advice on a specific issue
  ksa ask why is my redis memory fragmentation high?`,
		Args: cobra.MinimumNArgs(1), // Requires at least one word for the question.
		RunE: func(cmd *cobra.Command, args []string) error {
			question := strings.Join(args, " ")

			fmt.Print("🤖 KubeStack-AI: ")

			// Get the streaming channel from the orchestrator.
			responseChan, err := orchestrator.ProcessNaturalLanguageStream(cmd.Context(), question)
			if err != nil {
				return err
			}

			// Read from the channel and print chunks as they arrive.
			// TODO: Use a markdown renderer here for a richer UI (e.g., bubbles, glamour).
			for chunk := range responseChan {
				if chunk.Err != nil {
					return fmt.Errorf("\n\nerror from AI stream: %w", chunk.Err)
				}
				fmt.Print(chunk.Content)
			}
			fmt.Println() // Add a final newline for clean terminal output.

			// TODO: Implement session/history management.
			// The conversation could be saved here to provide context for future `ask` commands.

			return nil
		},
	}
	return cmd
}

//Personal.AI order the ending
