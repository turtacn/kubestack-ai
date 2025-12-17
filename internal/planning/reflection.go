package planning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ReflectionResult represents the result of reflection evaluation
type ReflectionResult struct {
	Success     bool     `json:"success"`
	Summary     string   `json:"summary"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
}

// ReflectionLoop performs post-execution reflection and evaluation
type ReflectionLoop struct {
	llmClient LLMClient
}

// NewReflectionLoop creates a new ReflectionLoop
func NewReflectionLoop(llm LLMClient) *ReflectionLoop {
	return &ReflectionLoop{
		llmClient: llm,
	}
}

// Evaluate performs reflection evaluation on a completed plan execution
func (r *ReflectionLoop) Evaluate(ctx context.Context, plan *Plan, state *ExecutionState) (*ReflectionResult, error) {
	if r.llmClient == nil {
		return r.basicEvaluation(plan, state), nil
	}

	prompt := r.buildReflectionPrompt(plan, state)
	response, err := r.llmClient.Complete(ctx, prompt)
	if err != nil {
		// Fallback to basic evaluation
		return r.basicEvaluation(plan, state), nil
	}

	return r.parseReflectionResponse(response, state)
}

// buildReflectionPrompt constructs the reflection prompt
func (r *ReflectionLoop) buildReflectionPrompt(plan *Plan, state *ExecutionState) string {
	var sb strings.Builder

	sb.WriteString("Please evaluate the following plan execution:\n\n")
	sb.WriteString(fmt.Sprintf("Plan: %s\n", plan.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n\n", plan.Description))

	sb.WriteString("Execution Results:\n")
	for _, step := range plan.Steps {
		stepState, exists := state.StepStates[step.ID]
		if !exists {
			continue
		}

		sb.WriteString(fmt.Sprintf("- Step: %s (%s)\n", step.Name, step.ID))
		sb.WriteString(fmt.Sprintf("  Status: %s\n", stepState.Status))
		if stepState.Error != "" {
			sb.WriteString(fmt.Sprintf("  Error: %s\n", stepState.Error))
		}
		if stepState.Output != nil {
			sb.WriteString(fmt.Sprintf("  Output: %v\n", stepState.Output))
		}
	}

	sb.WriteString("\nPlease provide:\n")
	sb.WriteString("1. Whether the plan execution was successful (true/false)\n")
	sb.WriteString("2. A summary of the execution\n")
	sb.WriteString("3. Any issues identified\n")
	sb.WriteString("4. Suggestions for improvement\n\n")
	sb.WriteString("Respond in JSON format with fields: success, summary, issues, suggestions\n")

	return sb.String()
}

// parseReflectionResponse parses the LLM response into ReflectionResult
func (r *ReflectionLoop) parseReflectionResponse(response string, state *ExecutionState) (*ReflectionResult, error) {
	// Try to parse as JSON
	var result ReflectionResult

	// Extract JSON from response (might be wrapped in markdown code blocks)
	jsonStr := response
	if strings.Contains(response, "```json") {
		start := strings.Index(response, "```json") + 7
		end := strings.LastIndex(response, "```")
		if end > start {
			jsonStr = response[start:end]
		}
	} else if strings.Contains(response, "```") {
		start := strings.Index(response, "```") + 3
		end := strings.LastIndex(response, "```")
		if end > start {
			jsonStr = response[start:end]
		}
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(jsonStr)), &result); err != nil {
		// Fallback: parse text response
		result.Success = !state.HasFailedSteps()
		result.Summary = response
		if !result.Success {
			result.Issues = []string{"Execution had failures"}
		}
	}

	return &result, nil
}

// basicEvaluation performs basic evaluation without LLM
func (r *ReflectionLoop) basicEvaluation(plan *Plan, state *ExecutionState) *ReflectionResult {
	result := &ReflectionResult{
		Success:     state.Status == PlanStatusCompleted,
		Issues:      []string{},
		Suggestions: []string{},
	}

	completedCount := 0
	failedCount := 0
	skippedCount := 0

	for _, stepState := range state.StepStates {
		switch stepState.Status {
		case StepStatusCompleted:
			completedCount++
		case StepStatusFailed:
			failedCount++
			result.Issues = append(result.Issues, fmt.Sprintf("Step %s failed: %s", stepState.StepID, stepState.Error))
		case StepStatusSkipped:
			skippedCount++
		}
	}

	result.Summary = fmt.Sprintf("Plan execution completed with %d/%d steps successful, %d failed, %d skipped",
		completedCount, len(plan.Steps), failedCount, skippedCount)

	if failedCount > 0 {
		result.Suggestions = append(result.Suggestions, "Review failed steps and consider adding retry policies")
	}

	if skippedCount > 0 {
		result.Suggestions = append(result.Suggestions, "Some steps were skipped due to failed dependencies")
	}

	return result
}

// ShouldRetry determines if the plan should be retried based on reflection
func (r *ReflectionLoop) ShouldRetry(result *ReflectionResult) bool {
	if result.Success {
		return false
	}

	// Check if issues are transient
	for _, issue := range result.Issues {
		lowerIssue := strings.ToLower(issue)
		if strings.Contains(lowerIssue, "timeout") ||
			strings.Contains(lowerIssue, "network") ||
			strings.Contains(lowerIssue, "temporary") {
			return true
		}
	}

	return false
}

// GenerateImprovementPlan generates suggestions for plan improvement
func (r *ReflectionLoop) GenerateImprovementPlan(ctx context.Context, plan *Plan, result *ReflectionResult) (string, error) {
	if r.llmClient == nil {
		return r.basicImprovementSuggestion(result), nil
	}

	var sb strings.Builder
	sb.WriteString("Based on the following plan execution issues, suggest improvements:\n\n")
	sb.WriteString(fmt.Sprintf("Plan: %s\n", plan.Name))
	sb.WriteString(fmt.Sprintf("Summary: %s\n\n", result.Summary))

	if len(result.Issues) > 0 {
		sb.WriteString("Issues:\n")
		for _, issue := range result.Issues {
			sb.WriteString(fmt.Sprintf("- %s\n", issue))
		}
	}

	if len(result.Suggestions) > 0 {
		sb.WriteString("\nCurrent Suggestions:\n")
		for _, suggestion := range result.Suggestions {
			sb.WriteString(fmt.Sprintf("- %s\n", suggestion))
		}
	}

	sb.WriteString("\nProvide concrete improvement suggestions for the plan:\n")

	response, err := r.llmClient.Complete(ctx, sb.String())
	if err != nil {
		return r.basicImprovementSuggestion(result), nil
	}

	return response, nil
}

// basicImprovementSuggestion generates basic improvement suggestions
func (r *ReflectionLoop) basicImprovementSuggestion(result *ReflectionResult) string {
	var sb strings.Builder

	sb.WriteString("Improvement Suggestions:\n")

	if len(result.Issues) > 0 {
		sb.WriteString("\n1. Address identified issues:\n")
		for _, issue := range result.Issues {
			sb.WriteString(fmt.Sprintf("   - %s\n", issue))
		}
	}

	if len(result.Suggestions) > 0 {
		sb.WriteString("\n2. Apply suggestions:\n")
		for _, suggestion := range result.Suggestions {
			sb.WriteString(fmt.Sprintf("   - %s\n", suggestion))
		}
	}

	sb.WriteString("\n3. Consider adding:\n")
	sb.WriteString("   - Retry policies for transient failures\n")
	sb.WriteString("   - Rollback actions for critical steps\n")
	sb.WriteString("   - Better error handling and logging\n")
	sb.WriteString("   - Health checks before execution\n")

	return sb.String()
}
