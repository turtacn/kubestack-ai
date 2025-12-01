package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/parser"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
)

type AIAnalyzer struct {
	llmClient         interfaces.LLMClient
	retriever         search.Retriever
	promptTemplate    prompt.PromptTemplate
	parser            *parser.StructuredParser
	knowledgeInjector *KnowledgeInjector
	multiTurnManager  *MultiTurnManager
	fewShotManager    *prompt.FewShotManager
	queryBuilder      *QueryBuilder
	config            *AIAnalyzerConfig
	log               logger.Logger
}

type AIAnalyzerConfig struct {
	Temperature       float32
	MaxTokens         int
	MinRelevanceScore float64
	MaxContextToken   int
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
	parser *parser.StructuredParser,
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
		queryBuilder:      NewQueryBuilder(),
		config:            config,
		log:               logger.NewLogger("ai_analyzer"),
	}
}

func (a *AIAnalyzer) Analyze(ctx context.Context, req *DiagnosisRequest) (*AnalyzeResult, error) {
	// 1. Build Query
	query := a.queryBuilder.BuildSearchQuery(req.Logs, req.Metrics, req.PluginName)
	if query == "" {
		query = req.Query // Fallback to user query if builder yields nothing
	}
	// Append user query if it's different and not generic
	if req.Query != "" && !strings.Contains(query, req.Query) {
		query += " " + req.Query
	}

	// 2. Retrieve knowledge with fallback
	var knowledgeContext string
	docs, err := a.retriever.HybridRetrieve(ctx, query, &search.RetrieveOptions{
		TopK:     5,
		MinScore: a.config.MinRelevanceScore,
	})
	if err != nil {
		a.log.Warnf("RAG retrieval failed, proceeding without knowledge: %v", err)
		knowledgeContext = ""
	} else {
		knowledgeContext = a.formatDocs(docs)
	}

	// 3. Retrieve Few-Shot Examples
	examples, err := a.fewShotManager.RetrieveSimilar(query, req.PluginName, 3)
	if err != nil {
		// Log and continue without examples
		a.log.Warnf("Few-shot retrieval failed: %v", err)
		examples = nil
	}

	// 4. Render prompt
	// Note: Prompt template expects .SystemLogs and .MetricData as strings for the basic view,
	// but the previous code passed maps.
	// Looking at `prompt_templates.go`:
	// {{.SystemLogs}} -> String expected or something that stringifies well.
	// {{.MetricData}} -> Same.
	// The previous implementation used map structure which might render as map representation in Go templates if not careful.
	// But `DiagnosisPromptTemplate` just prints it.
	// I will pass the raw strings from request as they are likely pre-formatted or just raw text.

	promptData := map[string]interface{}{
		"PluginName":       req.PluginName,
		"Timestamp":        time.Now().Format("2006-01-02 15:04:05"),
		"UserQuery":        req.Query,
		"SystemLogs":       req.Logs,
		"MetricData":       req.Metrics,
		"KnowledgeContext": knowledgeContext,
		"FewShotExamples":  examples,
	}

	promptStr, err := a.promptTemplate.Render(promptData)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	// 5. Call LLM
	llmResp, err := a.llmClient.SendMessage(ctx, &interfaces.LLMRequest{
		Messages:       []interfaces.Message{{Role: "user", Content: promptStr}},
		Temperature:    a.config.Temperature,
		MaxTokens:      a.config.MaxTokens,
		ResponseFormat: "json_object",
	})
	if err != nil {
		return nil, fmt.Errorf("llm client failed: %w", err)
	}

	// 6. Parse and validate
	var diagnosisResult parser.DiagnosisResult
	if err := a.parser.Parse(llmResp.Message.Content, &diagnosisResult); err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	return &AnalyzeResult{DiagnosisResult: &diagnosisResult}, nil
}

func (a *AIAnalyzer) formatDocs(docs []search.Document) string {
	var sb strings.Builder
	totalTokens := 0

	for _, doc := range docs {
		// Simple estimation: 1 token ~ 4 chars
		contentLen := len(doc.Content)
		if totalTokens + contentLen/4 > a.config.MaxContextToken {
			break
		}

		sb.WriteString(fmt.Sprintf("- [%s]: %s\n", doc.Metadata["title"], doc.Content))
		totalTokens += contentLen/4
	}
	return sb.String()
}
