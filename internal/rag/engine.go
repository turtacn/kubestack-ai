package rag

import (
	"context"
	"fmt"
	"sort"

	"github.com/kubestack-ai/kubestack-ai/internal/llm"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
	"github.com/kubestack-ai/kubestack-ai/internal/rag/models"
	"github.com/kubestack-ai/kubestack-ai/internal/rag/query"
	"github.com/kubestack-ai/kubestack-ai/internal/rag/retriever"
)

// RAGEngine orchestrates the Retrieval-Augmented Generation (RAG) pipeline.
type RAGEngine struct {
	multiStageRetriever *retriever.MultiStageRetriever
	queryRewriter       *query.QueryRewriter
	queryExpander       *query.QueryExpander
	llmClient           llm.LLMClient
	fusionStrategy      retriever.FusionStrategy
	finalTopK           int
}

// NewRAGEngine creates a new RAGEngine.
func NewRAGEngine(
	retriever *retriever.MultiStageRetriever,
	rewriter *query.QueryRewriter,
	expander *query.QueryExpander,
	llmClient llm.LLMClient,
) *RAGEngine {
	return &RAGEngine{
		multiStageRetriever: retriever,
		queryRewriter:       rewriter,
		queryExpander:       expander,
		llmClient:           llmClient,
		// Default to RRF for query fusion
		fusionStrategy:      nil, // Set dynamically or via method
		finalTopK:           10,  // Default limit
	}
}

// SetFusionStrategy allows setting the fusion strategy for merging results from expanded queries.
func (e *RAGEngine) SetFusionStrategy(f retriever.FusionStrategy) {
	e.fusionStrategy = f
}

// SetFinalTopK allows configuring the context window size.
func (e *RAGEngine) SetFinalTopK(k int) {
	if k > 0 {
		e.finalTopK = k
	}
}

// Query performs the end-to-end RAG process.
func (e *RAGEngine) Query(ctx context.Context, question string) (string, error) {
	// 1. Query Rewriting
	rewrittenQuery, err := e.queryRewriter.Rewrite(ctx, question)
	if err != nil {
		// Log warning, use original
		rewrittenQuery = question
	}

	// 2. Query Expansion
	queries := []string{rewrittenQuery}
	if e.queryExpander.Enabled() {
		expanded, err := e.queryExpander.Expand(ctx, rewrittenQuery)
		if err == nil {
			queries = expanded
		}
		// If expand fails, just use the rewritten query
	}

	// 3. Multi-Stage Retrieval
	var allResults [][]models.RetrievalResult
	for _, q := range queries {
		results, err := e.multiStageRetriever.Retrieve(ctx, q)
		if err != nil {
			// Log error
			continue
		}
		allResults = append(allResults, results)
	}

	if len(allResults) == 0 {
		return "", fmt.Errorf("no results retrieved for query: %s", question)
	}

	// 4. Fusion
	// If we have multiple query results, fuse them.
	// If only one, just take it.
	var finalResults []models.RetrievalResult
	if len(allResults) == 1 {
		finalResults = allResults[0]
	} else if e.fusionStrategy != nil {
		finalResults, err = e.fusionStrategy.Fuse(ctx, allResults)
		if err != nil {
			return "", fmt.Errorf("fusion of query results failed: %w", err)
		}
	} else {
		// Fallback to simple dedup if no strategy configured
		finalResultsMap := make(map[string]models.RetrievalResult)
		for _, resSet := range allResults {
			for _, res := range resSet {
				if existing, ok := finalResultsMap[res.DocID]; ok {
					if res.Score > existing.Score {
						finalResultsMap[res.DocID] = res
					}
				} else {
					finalResultsMap[res.DocID] = res
				}
			}
		}
		finalResults = make([]models.RetrievalResult, 0, len(finalResultsMap))
		for _, res := range finalResultsMap {
			finalResults = append(finalResults, res)
		}
		// Sort desc
		sort.Slice(finalResults, func(i, j int) bool {
			return finalResults[i].Score > finalResults[j].Score
		})
	}

	// Limit context window
	if len(finalResults) > e.finalTopK {
		finalResults = finalResults[:e.finalTopK]
	}

	// 5. Build Prompt
	docContents := make([]string, len(finalResults))
	for i, doc := range finalResults {
		docContents[i] = doc.Content
	}

	tmpl := prompt.Template{
		Name:    "rag",
		Content: "Answer the following question based on the provided context:\n\n{{range .context}}Context: {{.}}\n{{end}}\nQuestion: {{.question}}",
	}
	builder, err := prompt.NewBuilder(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to create prompt builder: %w", err)
	}

	promptData, err := builder.
		WithData("context", docContents).
		WithData("question", question).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build prompt: %w", err)
	}

	// 6. Generate Answer
	answer, err := e.llmClient.Generate(ctx, promptData)
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}

	return answer, nil
}
