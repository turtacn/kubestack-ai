package ai

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/parser"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
)

type AIAnalyzer struct {
	llmClient         interfaces.LLMClient
	retriever         search.Retriever
	promptTemplate    prompt.PromptTemplate
	parser            *parser.StructuredOutputParser
	knowledgeInjector *KnowledgeInjector
	multiTurnManager  *MultiTurnManager
	fewShotManager    *prompt.FewShotManager
	config            *AIAnalyzerConfig
}

type AIAnalyzerConfig struct {
	Temperature float32
	MaxTokens   int
}

type DiagnosisRequest struct {
	PluginName string
	Query      string
	Logs       string
	Metrics    string
	SessionID  string
}

// This struct is the return type for the Analyze method
type AnalyzeResult struct {
	*parser.DiagnosisResult
	NeedsClarification bool
	ClarifyQuestion    string
}

func NewAIAnalyzer(
	llmClient interfaces.LLMClient,
	retriever search.Retriever,
	template prompt.PromptTemplate,
	parser *parser.StructuredOutputParser,
	knowledgeInjector *KnowledgeInjector,
	multiTurnManager *MultiTurnManager,
	fewShotManager *prompt.FewShotManager,
	config *AIAnalyzerConfig,
) *AIAnalyzer {
	return &AIAnalyzer{
		llmClient:         llmClient,
		retriever:         retriever,
		promptTemplate:    template,
		parser:            parser,
		knowledgeInjector: knowledgeInjector,
		multiTurnManager:  multiTurnManager,
		fewShotManager:    fewShotManager,
		config:            config,
	}
}

func (a *AIAnalyzer) Analyze(ctx context.Context, req *DiagnosisRequest) (*AnalyzeResult, error) {
	// 1. Retrieve knowledge
	docs, err := a.retriever.HybridRetrieve(ctx, req.Query, &search.RetrieveOptions{TopK: 5})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve knowledge: %w", err)
	}

	// 2. Retrieve Few-Shot Examples
	examples, err := a.fewShotManager.RetrieveSimilar(req.Query, req.PluginName, 3)
	if err != nil {
		// Log and continue without examples
		examples = nil
	}

	// 3. Render prompt
	promptData := map[string]interface{}{
		"ServiceType":        req.PluginName,
		"Question":           req.Query,
		"Logs":               []map[string]interface{}{{"Timestamp": "now", "Level": "INFO", "Message": req.Logs}}, // Simplified for now
		"Metrics":            []map[string]interface{}{{"Name": "Metrics", "Value": req.Metrics}},                   // Simplified
		"RetrievedDocuments": docs,
		"FewShotExamples":    examples,
	}

	promptStr, err := a.promptTemplate.Render(promptData)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	// 4. Call LLM
	llmResp, err := a.llmClient.SendMessage(ctx, &interfaces.LLMRequest{
		Messages:       []interfaces.Message{{Role: "user", Content: promptStr}},
		Temperature:    a.config.Temperature,
		MaxTokens:      a.config.MaxTokens,
		ResponseFormat: "json_object",
	})
	if err != nil {
		return nil, fmt.Errorf("llm client failed: %w", err)
	}

	// 5. Parse and validate
	result, err := a.parser.Parse(llmResp.Message.Content)
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	// 6. Multi-turn management (Adaptation needed for new result type)
	// For now, we assume simple return as the MultiTurnManager in memory seems specific to old types.
	// TODO: Update MultiTurnManager to handle parser.DiagnosisResult if needed.
	// turn, needsClarify, err := a.multiTurnManager.ProcessTurn(req.SessionID, req.Query, result)

	return &AnalyzeResult{DiagnosisResult: result}, nil
}
