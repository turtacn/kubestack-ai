package retriever

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/llm"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/prompt"
	"github.com/kubestack-ai/kubestack-ai/internal/rag/models"
)

// ThresholdReranker filters results based on a score threshold.
type ThresholdReranker struct {
	minScore float64
}

func NewThresholdReranker(minScore float64) *ThresholdReranker {
	return &ThresholdReranker{minScore: minScore}
}

func (r *ThresholdReranker) Name() string {
	return "Threshold"
}

func (r *ThresholdReranker) Rerank(ctx context.Context, query string, candidates []models.RetrievalResult) ([]models.RetrievalResult, error) {
	filtered := make([]models.RetrievalResult, 0, len(candidates))
	for _, res := range candidates {
		if res.Score >= r.minScore {
			filtered = append(filtered, res)
		}
	}
	// Sort just in case input wasn't sorted or order matters
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Score > filtered[j].Score
	})
	return filtered, nil
}

// LLMReranker uses an LLM to assess relevance.
type LLMReranker struct {
	llmClient llm.LLMClient
	batchSize int
}

func NewLLMReranker(client llm.LLMClient) *LLMReranker {
	return &LLMReranker{
		llmClient: client,
		batchSize: 5, // Default batch size
	}
}

func (r *LLMReranker) Name() string {
	return "LLM"
}

func (r *LLMReranker) Rerank(ctx context.Context, query string, candidates []models.RetrievalResult) ([]models.RetrievalResult, error) {
	// Re-rank each candidate using the LLM.
	// We will modify the score of each candidate.

	reranked := make([]models.RetrievalResult, len(candidates))
	copy(reranked, candidates)

	// In a real production system, we would batch these or run them concurrently.
	// For this implementation, we will process them sequentially to ensure correctness first.
	for i := range reranked {
		score, err := r.evaluate(ctx, query, reranked[i].Content)
		if err != nil {
			// If LLM fails, we keep the original score or set a neutral one?
			// Let's log error (if we had a logger) and keep original score,
			// or maybe set it to 0 to penalize if we trust the reranker more.
			// Here we keep the original score but maybe degrade it?
			// Let's just continue, assuming the original score (from vector/keyword) is a fallback.
			continue
		}
		reranked[i].Score = score
	}

	// Sort by new score
	sort.Slice(reranked, func(i, j int) bool {
		return reranked[i].Score > reranked[j].Score
	})

	return reranked, nil
}

// Helper to evaluate single doc
func (r *LLMReranker) evaluate(ctx context.Context, query, content string) (float64, error) {
	// Construct prompt to ask for a relevance score 0-10
	tmpl := prompt.Template{
		Name: "rerank",
		Content: `You are a relevance ranking system.
Query: {{.query}}
Document: {{.content}}
Task: Rate the relevance of the document to the query on a scale from 0.0 (irrelevant) to 1.0 (highly relevant).
Output only the number, nothing else.`,
	}

	builder, err := prompt.NewBuilder(tmpl)
	if err != nil {
		return 0, err
	}

	// Truncate content if too long to save tokens
	if len(content) > 1000 {
		content = content[:1000] + "..."
	}

	p, err := builder.WithData("query", query).WithData("content", content).Build()
	if err != nil {
		return 0, err
	}

	resp, err := r.llmClient.Generate(ctx, p)
	if err != nil {
		return 0, err
	}

	// clean response
	resp = strings.TrimSpace(resp)
	val, err := strconv.ParseFloat(resp, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse score: %s", resp)
	}

	// Ensure valid range
	if val < 0 { val = 0 }
	if val > 1 { val = 1 }

	return val, nil
}

// MockLLMReranker for testing purposes without real LLM
type MockLLMReranker struct {
}

func (r *MockLLMReranker) Name() string { return "MockLLM" }
func (r *MockLLMReranker) Rerank(ctx context.Context, query string, candidates []models.RetrievalResult) ([]models.RetrievalResult, error) {
	// Reverse sort for testing
	res := make([]models.RetrievalResult, len(candidates))
	copy(res, candidates)
	if strings.Contains(query, "reverse") {
		sort.Slice(res, func(i, j int) bool {
			return res[i].Score < res[j].Score
		})
	}
	return res, nil
}
