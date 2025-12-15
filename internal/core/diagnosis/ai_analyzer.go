package diagnosis

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	llminterfaces "github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
)

// AIAnalyzer implements LLM-based diagnosis.
type AIAnalyzer struct {
	logger logger.Logger
	client llminterfaces.LLMClient
}

// NewAIAnalyzer creates a new AI analyzer.
func NewAIAnalyzer(client llminterfaces.LLMClient) (interfaces.DiagnosisAnalyzer, error) {
	return &AIAnalyzer{
		logger: logger.NewLogger("ai-analyzer"),
		client: client,
	}, nil
}

func (a *AIAnalyzer) Name() string { return "AIAnalyzer" }

func (a *AIAnalyzer) AnalyzeMetrics(ctx context.Context, data *models.MetricsData) ([]*models.Issue, error) {
	// Minimal stub
	return nil, nil
}

func (a *AIAnalyzer) AnalyzeLogs(ctx context.Context, data *models.LogData) ([]*models.Issue, error) {
	// Minimal stub
	return nil, nil
}

func (a *AIAnalyzer) CorrelateSystems(ctx context.Context, data *models.SystemCorrelationData) ([]*models.Issue, error) {
	return nil, nil
}
