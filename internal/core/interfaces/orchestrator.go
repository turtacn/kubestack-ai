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

// Package interfaces defines the core interfaces that form the backbone of the application's architecture.
// These interfaces decouple the main components, allowing for modularity and testability.
package interfaces

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	llm_interfaces "github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// RequestContext encapsulates all relevant information for a single user request,
// which is passed from the entrypoint (e.g., CLI) down to the core components.
type RequestContext struct {
	// Context is the Go context for the request, used for cancellation and deadlines.
	Context context.Context
	// Command is the name of the command that was invoked (e.g., "diagnose").
	Command string
	// Args is a slice of positional arguments provided by the user.
	Args []string
	// Flags is a map of command-line flags and their values.
	Flags map[string]string
	// RawInput is the unprocessed input string, used specifically for the 'ask' command.
	RawInput string
}

// OrchestratorConfig holds the configuration needed by the core orchestrator to
// initialize its sub-components, such as the diagnosis and execution managers.
type OrchestratorConfig struct {
	// This struct can hold configurations for sub-managers like PluginManager,
	// DiagnosisManager, etc., if they need specific settings not available globally.
}

// Orchestrator defines the contract for the central nervous system of KubeStack-AI.
// It receives requests from the UI layer (e.g., CLI), coordinates the necessary
// sub-components (plugins, diagnosis, execution), and returns the results. Its
// methods are designed to be asynchronous and cancellable via the context parameter.
type Orchestrator interface {
	// ProcessRequest is the main entry point for requests originating from the CLI or API.
	// It inspects the RequestContext and routes the request to the appropriate specialized method.
	ProcessRequest(reqCtx *RequestContext) error

	// LoadPlugin ensures a specific middleware plugin is loaded and ready for use.
	LoadPlugin(ctx context.Context, pluginName string) (MiddlewarePlugin, error)

	// ExecuteDiagnosis coordinates the entire diagnosis process for a given target.
	// It's a long-running operation that streams progress updates to the provided channel.
	ExecuteDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progressChan chan<- DiagnosisProgress) (*models.DiagnosisResult, error)

	// ProcessNaturalLanguage handles a natural language query from the user, providing a single, complete response.
	ProcessNaturalLanguage(ctx context.Context, query string) (string, error)

	// ProcessNaturalLanguageStream handles a natural language query and streams the response
	// back to the caller in real-time chunks.
	ProcessNaturalLanguageStream(ctx context.Context, query string) (<-chan llm_interfaces.StreamingChunk, error)

	// PlanExecution generates a detailed execution plan for a set of proposed actions.
	PlanExecution(ctx context.Context, recommendations []*models.Recommendation) (*models.ExecutionPlan, error)

	// ManageExecution handles the safe, user-confirmed execution of a plan.
	ManageExecution(ctx context.Context, plan *models.ExecutionPlan, confirmFunc ConfirmationFunc) (*models.ExecutionResult, error)

	// ValidateExecution checks if an execution was successful and the original issue is resolved.
	ValidateExecution(ctx context.Context, result *models.ExecutionResult) error

	// GetDiagnosis retrieves a specific diagnosis result by its unique ID.
	GetDiagnosis(ctx context.Context, id string) (*models.DiagnosisResult, error)
}

//Personal.AI order the ending
