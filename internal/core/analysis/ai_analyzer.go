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

package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/core/llm"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// AIAnalyzer implements the Analyzer interface using an LLM for intelligent analysis.
// It serves as the integration point between the orchestrator and the AI/LLM layer.
type AIAnalyzer struct {
	// client is the LLM client used to send analysis requests
	client llm.Client

	// middleware identifies the type of middleware being analyzed
	middleware string

	// namespace is the Kubernetes namespace (if applicable)
	namespace string

	// instance is the specific instance name
	instance string

	// model is the LLM model to use for analysis
	model string

	// temperature controls the randomness of the LLM output
	temperature float32

	// maxTokens limits the response length
	maxTokens int
}

// AIAnalyzerConfig contains configuration for the AIAnalyzer.
type AIAnalyzerConfig struct {
	// Middleware type (e.g., "redis", "mysql")
	Middleware string

	// Namespace for the middleware instance
	Namespace string

	// Instance name
	Instance string

	// LLM model to use (e.g., "gpt-4", "gemini-pro")
	Model string

	// Temperature for LLM generation (0.0 - 1.0)
	Temperature float32

	// MaxTokens for LLM response
	MaxTokens int
}

// NewAIAnalyzer creates a new AIAnalyzer with the given LLM client and configuration.
func NewAIAnalyzer(client llm.Client, config AIAnalyzerConfig) *AIAnalyzer {
	// Set defaults
	if config.Model == "" {
		config.Model = "gpt-4"
	}
	if config.Temperature == 0 {
		config.Temperature = 0.3 // Low temperature for more deterministic, factual output
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 2000
	}

	return &AIAnalyzer{
		client:      client,
		middleware:  config.Middleware,
		namespace:   config.Namespace,
		instance:    config.Instance,
		model:       config.Model,
		temperature: config.Temperature,
		maxTokens:   config.MaxTokens,
	}
}

// Name returns the unique identifier for this analyzer.
func (a *AIAnalyzer) Name() string {
	return "AIAnalyzer"
}

// Analyze processes collected plugin data and returns structured analysis results.
// This is the main entry point that implements the Analyzer interface.
func (a *AIAnalyzer) Analyze(ctx context.Context, data *models.CollectedData) (*AnalysisResult, error) {
	// Step 1: Build AI input from collected data
	aiInput := BuildAIInput(data, a.middleware, a.namespace, a.instance)

	// Step 2: Render prompt template
	template := GetAIAnalysisPromptTemplate()
	userPrompt, err := RenderPrompt(template, aiInput)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	// Step 3: Call LLM client
	llmRequest := &llm.LLMRequest{
		Model: a.model,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: template.SystemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		Temperature: a.temperature,
		MaxTokens:   a.maxTokens,
	}

	llmResponse, err := a.client.SendMessage(ctx, llmRequest)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Step 4: Parse JSON response
	aiOutput, err := a.parseAIOutput(llmResponse.Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI output: %w", err)
	}

	// Step 5: Convert to AnalysisResult
	result := a.convertToAnalysisResult(aiOutput)

	// Add metadata
	result.Metadata["llm_model"] = a.model
	result.Metadata["llm_tokens_used"] = llmResponse.Usage.TotalTokens
	result.Metadata["llm_prompt_tokens"] = llmResponse.Usage.PromptTokens
	result.Metadata["llm_completion_tokens"] = llmResponse.Usage.CompletionTokens

	return result, nil
}

// parseAIOutput parses the JSON response from the LLM into an AIOutput struct.
func (a *AIAnalyzer) parseAIOutput(responseContent string) (*AIOutput, error) {
	// Clean the response - remove markdown code blocks if present
	cleaned := a.cleanJSONResponse(responseContent)

	var output AIOutput
	if err := json.Unmarshal([]byte(cleaned), &output); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w (response: %s)", err, cleaned)
	}

	// Validate the output
	if err := a.validateAIOutput(&output); err != nil {
		return nil, fmt.Errorf("invalid AI output: %w", err)
	}

	return &output, nil
}

// cleanJSONResponse removes markdown code blocks and extra whitespace from the response.
func (a *AIAnalyzer) cleanJSONResponse(response string) string {
	// Remove markdown code blocks
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	return response
}

// validateAIOutput validates the structure of the AI output.
func (a *AIAnalyzer) validateAIOutput(output *AIOutput) error {
	if output.Summary == "" {
		return fmt.Errorf("summary is required")
	}

	for i, issue := range output.Issues {
		if issue.ID == "" {
			return fmt.Errorf("issue[%d]: id is required", i)
		}
		if issue.Title == "" {
			return fmt.Errorf("issue[%d]: title is required", i)
		}
		if issue.Severity == "" {
			return fmt.Errorf("issue[%d]: severity is required", i)
		}
		// Validate severity value
		if !isValidSeverity(issue.Severity) {
			return fmt.Errorf("issue[%d]: invalid severity '%s'", i, issue.Severity)
		}
	}

	return nil
}

// isValidSeverity checks if the severity string is valid.
func isValidSeverity(severity string) bool {
	validSeverities := map[string]bool{
		"Critical": true,
		"critical": true,
		"CRITICAL": true,
		"High":     true,
		"high":     true,
		"HIGH":     true,
		"Medium":   true,
		"medium":   true,
		"MEDIUM":   true,
		"Low":      true,
		"low":      true,
		"LOW":      true,
		"Info":     true,
		"info":     true,
		"INFO":     true,
	}
	return validSeverities[severity]
}

// convertToAnalysisResult converts an AIOutput to the standard AnalysisResult format.
func (a *AIAnalyzer) convertToAnalysisResult(aiOutput *AIOutput) *AnalysisResult {
	result := NewAnalysisResult(a.Name())
	result.Summary = aiOutput.Summary
	result.Issues = aiOutput.ConvertToModelIssues()

	// Add reasoning to metadata if present
	if aiOutput.Reasoning != "" {
		result.Metadata["reasoning"] = aiOutput.Reasoning
	}

	return result
}

// SetMiddlewareContext updates the middleware context for the analyzer.
// This allows the analyzer to be reused for different middleware instances.
func (a *AIAnalyzer) SetMiddlewareContext(middleware, namespace, instance string) {
	a.middleware = middleware
	a.namespace = namespace
	a.instance = instance
}
