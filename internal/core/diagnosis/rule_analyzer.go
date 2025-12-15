package diagnosis

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// RuleBasedAnalyzer implements simple rule-based checks.
type RuleBasedAnalyzer struct {
	logger logger.Logger
}

// NewRuleBasedAnalyzer creates a new instance.
// Args are placeholders for now to match caller signature in root.go (nil, nil)
func NewRuleBasedAnalyzer(ruleStore interface{}, log interface{}) interfaces.DiagnosisAnalyzer {
	return &RuleBasedAnalyzer{
		logger: logger.NewLogger("rule-analyzer"),
	}
}

func (a *RuleBasedAnalyzer) Name() string { return "RuleBasedAnalyzer" }

func (a *RuleBasedAnalyzer) AnalyzeMetrics(ctx context.Context, data *models.MetricsData) ([]*models.Issue, error) {
	// Minimal stub
	return nil, nil
}

func (a *RuleBasedAnalyzer) AnalyzeLogs(ctx context.Context, data *models.LogData) ([]*models.Issue, error) {
	// Minimal stub
	return nil, nil
}

func (a *RuleBasedAnalyzer) CorrelateSystems(ctx context.Context, data *models.SystemCorrelationData) ([]*models.Issue, error) {
	return nil, nil
}
