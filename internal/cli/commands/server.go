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
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/server"
	"github.com/spf13/cobra"
)

// newServerCmd creates the `server` command.
func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the KubeStack-AI API server",
		Long:  `Starts the HTTP server to expose the KubeStack-AI API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// The orchestrator is initialized in root.go's PersistentPreRunE.
			if orchestrator == nil {
				return fmt.Errorf("orchestrator not initialized")
			}

			// Get the server config from the global config object.
			cfg := config.GetConfig()
			apiServer := server.NewServer(orchestrator, &cfg.Server)
			return apiServer.Start()
		},
	}
	return cmd
}