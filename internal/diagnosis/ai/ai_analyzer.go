package ai

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

type AIAnalyzer struct {
	llmClient         interfaces.LLMClient
	retriever         search.Retriever
	promptRenderer    *PromptRenderer
	parser            *StructuredParser
	knowledgeInjector *KnowledgeInjector
	multiTurnManager  *MultiTurnManager
	validator         *OutputValidator
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
	*DiagnosisResult
	NeedsClarification bool
	ClarifyQuestion    string
}

func NewAIAnalyzer(
	llmClient interfaces.LLMClient,
	retriever search.Retriever,
	promptRenderer *PromptRenderer,
	parser *StructuredParser,
	knowledgeInjector *KnowledgeInjector,
	multiTurnManager *MultiTurnManager,
	validator *OutputValidator,
	config *AIAnalyzerConfig,
) *AIAnalyzer {
	return &AIAnalyzer{
		llmClient:         llmClient,
		retriever:         retriever,
		promptRenderer:    promptRenderer,
		parser:            parser,
		knowledgeInjector: knowledgeInjector,
		multiTurnManager:  multiTurnManager,
		validator:         validator,
		config:            config,
	}
}

func (a *AIAnalyzer) Analyze(ctx context.Context, req *DiagnosisRequest) (*AnalyzeResult, error) {
	// 1. Retrieve knowledge
	docs, err := a.retriever.HybridRetrieve(ctx, req.Query, &search.RetrieveOptions{TopK: 5})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve knowledge: %w", err)
	}

	// 2. Inject knowledge
	knowledgeCtx := a.knowledgeInjector.InjectKnowledge(docs, req.Query)

	// 3. Render prompt
	promptData := map[string]interface{}{
		"PluginName":       req.PluginName,
		"Timestamp":        "now", // In a real scenario, you'd pass the actual timestamp
		"UserQuery":        req.Query,
		"SystemLogs":       req.Logs,
		"MetricData":       req.Metrics,
		"KnowledgeContext": knowledgeCtx,
	}
	prompt, err := a.promptRenderer.Render("diagnosis", promptData)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	// 4. Call LLM
	llmResp, err := a.llmClient.SendMessage(ctx, &interfaces.LLMRequest{
		Messages:    []interfaces.Message{{Role: "user", Content: prompt}},
		Temperature: a.config.Temperature,
		MaxTokens:   a.config.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("llm client failed: %w", err)
	}

	// 5. Parse and validate
	result, err := a.parser.ParseDiagnosisResult(llmResp.Message.Content)
	if err != nil {
		// Retry logic
		return a.retryWithStricterPrompt(ctx, prompt)
	}
	if err := a.validator.Validate(result); err != nil {
		return nil, fmt.Errorf("invalid AI output: %w", err)
	}

	// 6. Multi-turn management
	turn, needsClarify, err := a.multiTurnManager.ProcessTurn(req.SessionID, req.Query, result)
	if err != nil {
		return nil, fmt.Errorf("failed to process turn: %w", err)
	}

	if needsClarify {
		return &AnalyzeResult{NeedsClarification: true, ClarifyQuestion: turn.Content}, nil
	}

	return &AnalyzeResult{DiagnosisResult: result}, nil
}

func (a *AIAnalyzer) retryWithStricterPrompt(ctx context.Context, originalPrompt string) (*AnalyzeResult, error) {
	stricterPrompt := originalPrompt + "\n\nCRITICAL: You MUST respond ONLY with valid JSON matching the schema above. Do not include any explanatory text."

	llmResp, err := a.llmClient.SendMessage(ctx, &interfaces.LLMRequest{
		Messages:    []interfaces.Message{{Role: "user", Content: stricterPrompt}},
		Temperature: 0.1, // Lower temperature for stricter adherence
		MaxTokens:   a.config.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("llm client failed on retry: %w", err)
	}

	result, err := a.parser.ParseDiagnosisResult(llmResp.Message.Content)
	if err != nil {
		return nil, fmt.Errorf("parsing failed even on retry: %w", err)
	}

	if err := a.validator.Validate(result); err != nil {
		return nil, fmt.Errorf("validation failed even on retry: %w", err)
	}

	return &AnalyzeResult{DiagnosisResult: result}, nil
}
