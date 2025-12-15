package query

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/llm"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
)

type QueryExpander struct {
	llmClient llm.LLMClient
	enabled   bool
}

func NewQueryExpander(client llm.LLMClient, enabled bool) *QueryExpander {
	return &QueryExpander{
		llmClient: client,
		enabled:   enabled,
	}
}

func (e *QueryExpander) Enabled() bool {
	return e.enabled
}

func (e *QueryExpander) Expand(ctx context.Context, query string) ([]string, error) {
	if !e.enabled || e.llmClient == nil {
		return []string{query}, nil
	}

	// HyDE Strategy: Generate a hypothetical document/answer
	tmpl := prompt.Template{
		Name: "hyde",
		Content: `Please write a short passage to answer the question: "{{.query}}".
The passage should be technical and detailed.`,
	}

	builder, _ := prompt.NewBuilder(tmpl)
	p, _ := builder.WithData("query", query).Build()

	hypotheticalDoc, err := e.llmClient.Generate(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("failed to generate hypothetical document: %w", err)
	}

	// Return original query and the hypothetical document as queries
	return []string{query, hypotheticalDoc}, nil
}
