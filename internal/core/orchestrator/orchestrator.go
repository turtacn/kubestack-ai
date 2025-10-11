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

// Package orchestrator implements the central coordinator for KubeStack-AI.
package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/constants"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	llm_interfaces "github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// orchestrator is the concrete implementation of the interfaces.Orchestrator interface.
// It holds references to all the major components (managers) and delegates tasks to them.
type orchestrator struct {
	cfg              *config.Config
	log              logger.Logger
	pluginManager    interfaces.PluginManager
	diagnosisManager interfaces.DiagnosisManager
	executionManager interfaces.ExecutionManager
	knowledgeManager interfaces.KnowledgeManager
}

// NewOrchestrator creates a new instance of the core orchestrator. It acts as a
// dependency injection hub, taking in all the major manager components and wiring
// them together.
//
// Parameters:
//   cfg (*config.Config): The global application configuration.
//   pm (interfaces.PluginManager): The manager responsible for the plugin lifecycle.
//   dm (interfaces.DiagnosisManager): The manager for running diagnoses.
//   em (interfaces.ExecutionManager): The manager for executing fix plans.
//   km (interfaces.KnowledgeManager): The manager for accessing the knowledge base.
//
// Returns:
//   interfaces.Orchestrator: A new, fully initialized orchestrator.
func NewOrchestrator(
	cfg *config.Config,
	pm interfaces.PluginManager,
	dm interfaces.DiagnosisManager,
	em interfaces.ExecutionManager,
	km interfaces.KnowledgeManager,
) interfaces.Orchestrator {
	return &orchestrator{
		cfg:              cfg,
		log:              logger.NewLogger("orchestrator"),
		pluginManager:    pm,
		diagnosisManager: dm,
		executionManager: em,
		knowledgeManager: km,
	}
}

// ProcessRequest serves as the main entry point for requests originating from a UI
// layer (like the CLI). It inspects the request context and routes the request to
// the appropriate specialized method within the orchestrator.
//
// NOTE: This is a simplified router. A production implementation would use a more
// robust command dispatch mechanism, likely integrated with the CLI framework.
//
// Parameters:
//   reqCtx (*interfaces.RequestContext): The context containing all information about the user's request.
//
// Returns:
//   error: An error if the command is unknown or if processing fails.
func (o *orchestrator) ProcessRequest(reqCtx *interfaces.RequestContext) error {
	o.log.Infof("Processing request for command: %s with args: %v", reqCtx.Command, reqCtx.Args)

	// This is a simplified router. A real implementation would use a more robust
	// command dispatch mechanism, likely integrated with the CLI framework (e.g., Cobra).
	switch reqCtx.Command {
	case constants.CommandDiagnose:
		// In a real implementation, args and flags would be parsed to create this request.
		diagReq := &models.DiagnosisRequest{
			// ... populate from reqCtx ...
		}
		// The progress channel would be created and managed by the CLI command.
		progressChan := make(chan interfaces.DiagnosisProgress, 10)
		_, err := o.ExecuteDiagnosis(reqCtx.Context, diagReq, progressChan)
		return err
	case constants.CommandAsk:
		resp, err := o.ProcessNaturalLanguage(reqCtx.Context, reqCtx.RawInput)
		if err != nil {
			return err
		}
		fmt.Println(resp) // For CLI, print response directly. For API, this would be a structured response.
		return nil
	case constants.CommandFix:
		// Logic for 'fix' command would go here.
		return fmt.Errorf("command '%s' not implemented yet", reqCtx.Command)
	default:
		return fmt.Errorf("unknown command: %s", reqCtx.Command)
	}
}

// LoadPlugin delegates the task of loading a plugin to the plugin manager.
func (o *orchestrator) LoadPlugin(ctx context.Context, pluginName string) (interfaces.MiddlewarePlugin, error) {
	o.log.Infof("Orchestrating load for plugin: %s", pluginName)
	return o.pluginManager.LoadPlugin(pluginName)
}

// ExecuteDiagnosis delegates the entire diagnosis process to the diagnosis manager.
func (o *orchestrator) ExecuteDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progressChan chan<- interfaces.DiagnosisProgress) (*models.DiagnosisResult, error) {
	o.log.Infof("Executing diagnosis for middleware: %s on instance: %s", req.TargetMiddleware.String(), req.Instance)
	return o.diagnosisManager.RunDiagnosis(ctx, req, progressChan)
}

// ProcessNaturalLanguage orchestrates the handling of a natural language query.
// NOTE: This is a placeholder. A full implementation would involve complex prompt
// engineering, retrieval-augmented generation (RAG) from a knowledge base, and
// interaction with the LLM client.
func (o *orchestrator) ProcessNaturalLanguage(ctx context.Context, query string) (string, error) {
	o.log.Infof("Processing natural language query: %s", query)
	return fmt.Sprintf("AI Response to '%s' (full implementation pending)", query), nil
}

// ManageExecution delegates the safe, user-confirmed execution of a fix plan
// to the execution manager.
func (o *orchestrator) ManageExecution(ctx context.Context, plan *models.ExecutionPlan, confirmFunc interfaces.ConfirmationFunc) (*models.ExecutionResult, error) {
	o.log.Infof("Managing execution for plan ID: %s", plan.ID)
	return o.executionManager.ExecuteActions(ctx, plan, confirmFunc)
}

// ProcessNaturalLanguageStream handles a streaming 'ask' command query.
// NOTE: This is a placeholder. A full implementation would call the LLM client's
// streaming method and pipe the results to the returned channel.
func (o *orchestrator) ProcessNaturalLanguageStream(ctx context.Context, query string) (<-chan llm_interfaces.StreamingChunk, error) {
	o.log.Infof("Processing streaming natural language query: %s", query)
	chunkChan := make(chan llm_interfaces.StreamingChunk)
	go func() {
		defer close(chunkChan)
		time.Sleep(500 * time.Millisecond)
		chunkChan <- llm_interfaces.StreamingChunk{Content: "This is a streamed response... "}
		time.Sleep(500 * time.Millisecond)
		chunkChan <- llm_interfaces.StreamingChunk{Content: "(not fully implemented yet)."}
	}()
	return chunkChan, nil
}

// PlanExecution delegates the task of generating an execution plan to the execution manager.
func (o *orchestrator) PlanExecution(ctx context.Context, recommendations []*models.Recommendation) (*models.ExecutionPlan, error) {
	o.log.Info("Orchestrating execution planning.")
	return o.executionManager.PlanExecution(ctx, recommendations)
}

// ValidateExecution delegates the task of validating a completed execution
// to the execution manager.
func (o *orchestrator) ValidateExecution(ctx context.Context, result *models.ExecutionResult) error {
	o.log.Info("Orchestrating execution validation.")
	return o.executionManager.ValidateExecution(ctx, result)
}

// GetDiagnosis delegates the retrieval of a diagnosis report to the diagnosis manager.
func (o *orchestrator) GetDiagnosis(ctx context.Context, id string) (*models.DiagnosisResult, error) {
	o.log.Infof("Retrieving diagnosis report with ID: %s", id)
	return o.diagnosisManager.GetDiagnosis(ctx, id)
}

//Personal.AI order the ending
