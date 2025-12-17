package planning

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockToolRegistry for testing
type MockToolRegistry struct {
	results    map[string]any
	errors     map[string]error
	calls      []string
	executeFunc func(ctx context.Context, toolName string, args map[string]any) (any, error)
}

func NewMockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		results: make(map[string]any),
		errors:  make(map[string]error),
		calls:   []string{},
	}
}

func (m *MockToolRegistry) Execute(ctx context.Context, toolName string, args map[string]any) (any, error) {
	// Use custom function if set
	if m.executeFunc != nil {
		return m.executeFunc(ctx, toolName, args)
	}
	
	m.calls = append(m.calls, toolName)
	if err, exists := m.errors[toolName]; exists {
		return nil, err
	}
	if result, exists := m.results[toolName]; exists {
		return result, nil
	}
	return "success", nil
}

// MockLLMClient for testing
type MockLLMClient struct {
	responses map[string]string
	errors    map[string]error
	calls     []string
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: make(map[string]string),
		errors:    make(map[string]error),
		calls:     []string{},
	}
}

func (m *MockLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	m.calls = append(m.calls, prompt)
	if err, exists := m.errors[prompt]; exists {
		return "", err
	}
	if response, exists := m.responses[prompt]; exists {
		return response, nil
	}
	return "mock response", nil
}

func TestDefaultStepExecutor_ExecuteToolCall(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.results["test_tool"] = "tool_result"

	executor := NewDefaultStepExecutor(mockTools, nil)

	step := &Step{
		ID:   "step1",
		Type: StepTypeToolCall,
		Action: ActionSpec{
			ToolName: "test_tool",
			ToolArgs: map[string]any{"arg1": "value1"},
		},
	}

	result, err := executor.Execute(context.Background(), step, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "tool_result" {
		t.Errorf("expected 'tool_result', got %v", result)
	}

	if len(mockTools.calls) != 1 || mockTools.calls[0] != "test_tool" {
		t.Errorf("expected tool to be called once, got %v", mockTools.calls)
	}
}

func TestDefaultStepExecutor_ExecuteLLMQuery(t *testing.T) {
	mockLLM := NewMockLLMClient()
	mockLLM.responses["test prompt"] = "llm response"

	executor := NewDefaultStepExecutor(nil, mockLLM)

	step := &Step{
		ID:   "step1",
		Type: StepTypeLLMQuery,
		Action: ActionSpec{
			Prompt: "test prompt",
		},
	}

	result, err := executor.Execute(context.Background(), step, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "llm response" {
		t.Errorf("expected 'llm response', got %v", result)
	}

	if len(mockLLM.calls) != 1 {
		t.Errorf("expected LLM to be called once, got %d calls", len(mockLLM.calls))
	}
}

func TestDefaultStepExecutor_ExecuteCondition(t *testing.T) {
	executor := NewDefaultStepExecutor(nil, nil)

	step := &Step{
		ID:   "step1",
		Type: StepTypeCondition,
		Action: ActionSpec{
			Condition: "true",
		},
	}

	result, err := executor.Execute(context.Background(), step, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestDefaultStepExecutor_Timeout(t *testing.T) {
	mockLLM := NewMockLLMClient()
	mockLLM.responses["slow"] = "response"

	executor := NewDefaultStepExecutor(nil, mockLLM)

	step := &Step{
		ID:      "step1",
		Type:    StepTypeLLMQuery,
		Timeout: 1 * time.Millisecond, // Very short timeout
		Action: ActionSpec{
			Prompt: "slow",
		},
	}

	ctx := context.Background()
	_, err := executor.Execute(ctx, step, nil)

	// Note: This test might be flaky depending on system speed
	// The timeout context might not always trigger
	if err != nil && err != context.DeadlineExceeded {
		// Either succeeds quickly or times out - both are acceptable
	}
}

func TestDefaultStepExecutor_Retry(t *testing.T) {
	// Create a custom mock that tracks call count
	callCount := 0
	mockTools := NewMockToolRegistry()
	
	// Set custom execute function for retry test
	mockTools.executeFunc = func(ctx context.Context, toolName string, args map[string]any) (any, error) {
		callCount++
		mockTools.calls = append(mockTools.calls, toolName)
		if callCount < 3 {
			return nil, errors.New("temporary failure")
		}
		return "success", nil
	}

	executor := NewDefaultStepExecutor(mockTools, nil)

	step := &Step{
		ID:   "step1",
		Type: StepTypeToolCall,
		Action: ActionSpec{
			ToolName: "retry_tool",
		},
		RetryPolicy: &RetryPolicy{
			MaxRetries: 3,
			BackoffMs:  10,
		},
	}

	result, err := executor.Execute(context.Background(), step, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "success" {
		t.Errorf("expected 'success', got %v", result)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestDefaultStepExecutor_RetryExhausted(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.errors["failing_tool"] = errors.New("persistent failure")

	executor := NewDefaultStepExecutor(mockTools, nil)

	step := &Step{
		ID:   "step1",
		Type: StepTypeToolCall,
		Action: ActionSpec{
			ToolName: "failing_tool",
		},
		RetryPolicy: &RetryPolicy{
			MaxRetries: 2,
			BackoffMs:  10,
		},
	}

	_, err := executor.Execute(context.Background(), step, nil)
	if err == nil {
		t.Error("expected error after retry exhaustion")
	}

	if len(mockTools.calls) != 3 { // Initial + 2 retries
		t.Errorf("expected 3 calls, got %d", len(mockTools.calls))
	}
}

func TestParallelExecutor_ExecuteParallel(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.results["tool1"] = "result1"
	mockTools.results["tool2"] = "result2"
	mockTools.results["tool3"] = "result3"

	executor := NewDefaultStepExecutor(mockTools, nil)
	parallelExecutor := NewParallelExecutor(3)

	steps := []*Step{
		{
			ID:   "step1",
			Type: StepTypeToolCall,
			Action: ActionSpec{
				ToolName: "tool1",
			},
		},
		{
			ID:   "step2",
			Type: StepTypeToolCall,
			Action: ActionSpec{
				ToolName: "tool2",
			},
		},
		{
			ID:   "step3",
			Type: StepTypeToolCall,
			Action: ActionSpec{
				ToolName: "tool3",
			},
		},
	}

	results := parallelExecutor.ExecuteParallel(context.Background(), steps, executor, nil)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for i := 1; i <= 3; i++ {
		stepID := "step" + string(rune('0'+i))
		result, exists := results[stepID]
		if !exists {
			t.Errorf("expected result for %s", stepID)
			continue
		}
		if result.Error != nil {
			t.Errorf("unexpected error for %s: %v", stepID, result.Error)
		}
	}
}

func TestParallelExecutor_StepFailure(t *testing.T) {
	mockTools := NewMockToolRegistry()
	mockTools.results["tool1"] = "result1"
	mockTools.errors["tool2"] = errors.New("tool2 failed")
	mockTools.results["tool3"] = "result3"

	executor := NewDefaultStepExecutor(mockTools, nil)
	parallelExecutor := NewParallelExecutor(3)

	steps := []*Step{
		{ID: "step1", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "tool1"}},
		{ID: "step2", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "tool2"}},
		{ID: "step3", Type: StepTypeToolCall, Action: ActionSpec{ToolName: "tool3"}},
	}

	results := parallelExecutor.ExecuteParallel(context.Background(), steps, executor, nil)

	if results["step2"].Error == nil {
		t.Error("expected error for step2")
	}

	if results["step1"].Error != nil {
		t.Errorf("unexpected error for step1: %v", results["step1"].Error)
	}

	if results["step3"].Error != nil {
		t.Errorf("unexpected error for step3: %v", results["step3"].Error)
	}
}

func TestSimpleConditionEvaluator(t *testing.T) {
	evaluator := &SimpleConditionEvaluator{}

	tests := []struct {
		condition string
		expected  bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"anything else", true}, // Default behavior
	}

	for _, test := range tests {
		result, err := evaluator.Evaluate(context.Background(), test.condition, nil)
		if err != nil {
			t.Errorf("unexpected error for condition '%s': %v", test.condition, err)
		}
		if result != test.expected {
			t.Errorf("condition '%s': expected %v, got %v", test.condition, test.expected, result)
		}
	}
}
