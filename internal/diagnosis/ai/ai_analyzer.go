package ai

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/parser"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
)

type Analyzer struct {
	llmClient     interfaces.LLMClient
	promptManager *prompt.FewShotManager
	parser        *parser.StructuredParser
}

func NewAnalyzer(client interfaces.LLMClient, pm *prompt.FewShotManager) *Analyzer {
	return &Analyzer{
		llmClient:     client,
		promptManager: pm,
		parser:        parser.NewStructuredParser(),
	}
}

func (a *Analyzer) Analyze(ctx context.Context, data *models.CollectedData) ([]*models.Issue, error) {
	// 1. Construct Prompt
	promptText := fmt.Sprintf("Analyze the following data: %+v", data)
	req := &interfaces.LLMRequest{
		Model: "gpt-4",
		Messages: []interfaces.Message{
			{Role: "user", Content: promptText},
		},
	}

	// 2. Call LLM
	response, err := a.llmClient.SendMessage(ctx, req)
	if err != nil {
		return nil, err
	}

	// 3. Parse Result
	result, err := a.parser.Parse(response.Message.Content)
	if err != nil {
		return nil, err
	}

	return result.Issues, nil
}

func (a *Analyzer) Name() string {
	return "AIAnalyzer"
}
