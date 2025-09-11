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

// RequestContext holds all relevant information for a single user request,
// passed down from the UI layer to the core components.
type RequestContext struct {
	Context  context.Context
	Command  string
	Args     []string
	Flags    map[string]string
	RawInput string // Used specifically for the 'ask' command's natural language input.
}

// OrchestratorConfig holds the configuration needed by the core orchestrator to initialize its components.
type OrchestratorConfig struct {
	// This struct can hold configurations for sub-managers like PluginManager,
	// DiagnosisManager, etc., if they need specific settings not available globally.
}

// Orchestrator is the central nervous system of KubeStack-AI. It receives requests
// from the UI layer (e.g., CLI), coordinates the necessary components (plugins,
// diagnosis, execution), and returns the results. Its methods are designed to be
// asynchronous and cancellable via the context parameter.
type Orchestrator interface {
	// ProcessRequest is the main entry point for requests originating from the CLI or API.
	// It inspects the RequestContext and routes the request to the appropriate specialized method.
	ProcessRequest(reqCtx *RequestContext) error

	// LoadPlugin ensures a specific middleware plugin is loaded and ready for use.
	LoadPlugin(ctx context.Context, pluginName string) (MiddlewarePlugin, error)

	// ExecuteDiagnosis coordinates the entire diagnosis process for a given target.
	// It's a long-running operation that streams progress updates to the provided channel.
	ExecuteDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progressChan chan<- DiagnosisProgress) (*models.DiagnosisResult, error)

	// ProcessNaturalLanguage handles a natural language query from the user.
	// This is the non-streaming version.
	ProcessNaturalLanguage(ctx context.Context, query string) (string, error)

	// ProcessNaturalLanguageStream handles a natural language query and streams the response.
	ProcessNaturalLanguageStream(ctx context.Context, query string) (<-chan llm_interfaces.StreamingChunk, error)

	// PlanExecution generates a detailed execution plan for a set of proposed actions.
	PlanExecution(ctx context.Context, recommendations []*models.Recommendation) (*models.ExecutionPlan, error)

	// ManageExecution handles the planning and execution of fix actions.
	ManageExecution(ctx context.Context, plan *models.ExecutionPlan, confirmFunc ConfirmationFunc) (*models.ExecutionResult, error)

	// ValidateExecution checks if the execution was successful and the original issue is resolved.
	ValidateExecution(ctx context.Context, result *models.ExecutionResult) error
}

//Personal.AI order the ending
