package chain

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/llm/parser"
)

// ChainStep defines a single step in the chain.
type ChainStep interface {
	Execute(ctx context.Context, input interface{}) (interface{}, error)
}

// ChainExecutor executes a sequence of steps.
type ChainExecutor struct {
	steps []ChainStep
}

// NewChainExecutor creates a new chain executor.
func NewChainExecutor(steps ...ChainStep) *ChainExecutor {
	return &ChainExecutor{
		steps: steps,
	}
}

// Execute runs the chain sequentially.
func (e *ChainExecutor) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	currentInput := input
	var err error

	for i, step := range e.steps {
		currentInput, err = step.Execute(ctx, currentInput)
		if err != nil {
			return nil, fmt.Errorf("step %d failed: %w", i, err)
		}
	}

	return currentInput, nil
}

// ChainContext holds data passing through the diagnosis chain.
type ChainContext struct {
	Question        string
	ServiceType     string
	Documents       interface{} // Retained from knowledge retrieval
	Metrics         interface{}
	Logs            interface{}
	DiagnosisResult *parser.DiagnosisResult
	Solution        interface{}
}
