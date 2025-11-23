package chain

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/search"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/parser"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
)

// DiagnosisChain manages the end-to-end diagnosis process.
type DiagnosisChain struct {
	retriever      search.Retriever
	llmClient      interfaces.LLMClient
	promptTemplate prompt.PromptTemplate
	parser         *parser.StructuredOutputParser
	fewShotMgr     *prompt.FewShotManager
}

// NewDiagnosisChain creates a new diagnosis chain.
func NewDiagnosisChain(
	retriever search.Retriever,
	llmClient interfaces.LLMClient,
	template prompt.PromptTemplate,
	parser *parser.StructuredOutputParser,
	fewShotMgr *prompt.FewShotManager,
) *DiagnosisChain {
	return &DiagnosisChain{
		retriever:      retriever,
		llmClient:      llmClient,
		promptTemplate: template,
		parser:         parser,
		fewShotMgr:     fewShotMgr,
	}
}

// Execute runs the diagnosis chain.
func (c *DiagnosisChain) Execute(ctx context.Context, question string) (*parser.DiagnosisResult, error) {
	// Step 1: Retrieval
	// Using Retrieve with TopK=10 as requested
	docs, err := c.retriever.Retrieve(ctx, question, 10)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// Step 2: Few-shot retrieval (optional category inference, here simplistic)
	var examples []*prompt.FewShotExample
	if c.fewShotMgr != nil {
		var err error
		examples, err = c.fewShotMgr.RetrieveSimilar(question, "", 3)
		if err != nil {
			// Log error but proceed? Or fail. Let's proceed with empty examples.
			examples = nil
		}
	}

	// Step 3: Prompt Construction
	inputData := map[string]interface{}{
		"Question":           question,
		"RetrievedDocuments": docs,
		"FewShotExamples":    examples,
		// Metrics/Logs would be injected here if available in context, but currently only question is passed.
		// The caller usually passes a "context" object or data, but the signature here is just string.
		// We might need to refactor this signature if we want to pass collected data.
		// For now, we assume the question contains necessary info or prompt template handles it.
	}

	renderedPrompt, err := c.promptTemplate.Render(inputData)
	if err != nil {
		return nil, fmt.Errorf("prompt rendering failed: %w", err)
	}

	// Step 4: LLM Generation
	req := &interfaces.LLMRequest{
		Messages:       []interfaces.Message{{Role: "user", Content: renderedPrompt}},
		ResponseFormat: "json_object",
		Temperature:    0.2, // Low temp for diagnosis
	}
	resp, err := c.llmClient.SendMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llm call failed: %w", err)
	}

	// Step 5: Parsing
	result, err := c.parser.Parse(resp.Message.Content)
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	return result, nil
}
