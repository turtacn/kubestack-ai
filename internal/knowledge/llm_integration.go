package knowledge

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/parser"
)

// LLMIntegration handles interactions with the LLM using knowledge base context.
type LLMIntegration struct {
	client     interfaces.LLMClient
	config     config.LLMConfig
	log        logger.Logger
	parser     *parser.StructuredOutputParser
}

// NewLLMIntegration creates a new LLMIntegration.
// Note: In a real scenario, LLMClient should be injected. Here we assume it's passed or created elsewhere.
// For now, we'll accept an interface.
func NewLLMIntegration(client interfaces.LLMClient, cfg config.LLMConfig) *LLMIntegration {
	return &LLMIntegration{
		client: client,
		config: cfg,
		log:    logger.NewLogger("llm-integration"),
		// Initialize a parser for structured output if needed.
		// For now we might just want text, but structured is better.
		// parser: parser.NewStructuredOutputParser(...),
	}
}

// GenerateRecommendations generates diagnosis recommendations using LLM and knowledge base.
func (li *LLMIntegration) GenerateRecommendations(ctx context.Context, diagCtx *DiagnosisContext) ([]*Recommendation, error) {
	prompt, err := li.buildPrompt(diagCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	model := "gpt-4" // Default fallback
	if li.config.Provider == "openai" && li.config.OpenAI.Model != "" {
		model = li.config.OpenAI.Model
	} else if li.config.Provider == "gemini" && li.config.Gemini.Model != "" {
		model = li.config.Gemini.Model
	}

	req := &interfaces.LLMRequest{
		Model: model,
		Messages: []interfaces.Message{
			{Role: "system", Content: "You are an expert middleware reliability engineer."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
	}

	resp, err := li.client.SendMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %w", err)
	}

	return li.parseResponse(resp.Message.Content)
}

func (li *LLMIntegration) buildPrompt(diagCtx *DiagnosisContext) (string, error) {
	const promptTemplate = `
You are a middleware diagnosis expert. Analyze the following situation and provide diagnostic recommendations.

Middleware: {{.MiddlewareType}}
Context:
{{range $key, $value := .Metrics}}
- {{$key}}: {{$value}}
{{end}}

Identified Issues:
{{range .Issues}}
- {{.Title}} (Severity: {{.Severity}})
  Description: {{.Description}}
{{end}}

Relevant Knowledge Base Rules Matched:
{{.KnowledgeContext}}

Please provide a list of concrete recommendations to resolve these issues.
Format your response as a list of actions with priority.
`
	t, err := template.New("diagnosis").Parse(promptTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := t.Execute(&buf, diagCtx); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (li *LLMIntegration) parseResponse(content string) ([]*Recommendation, error) {
	// This is a simplified parser. In production, we should use structured JSON output from LLM.
	// Here we just wrap the content in a single recommendation or try to split by lines.

	rec := &Recommendation{
		Title:      "AI Generated Diagnosis",
		Action:     content,
		Priority:   50,
		Confidence: 0.8,
		Metadata:   map[string]interface{}{"source": "llm"},
	}

	return []*Recommendation{rec}, nil
}
