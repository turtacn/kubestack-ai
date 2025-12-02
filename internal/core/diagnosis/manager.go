package diagnosis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/diagnosis/ai"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge"
	llm_interfaces "github.com/kubestack-ai/kubestack-ai/internal/llm/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
)

// Manager coordinates the end-to-end diagnosis process.
type Manager struct {
	pluginManager interfaces.PluginManager
	analyzers     []interfaces.DiagnosisAnalyzer
	execManager   interfaces.ExecutionManager
	reportDir     string
	knowledgeBase *knowledge.KnowledgeBase
	logger        logger.Logger
}

// NewManager creates a new diagnosis manager instance.
func NewManager(pm interfaces.PluginManager, analyzers []interfaces.DiagnosisAnalyzer, em interfaces.ExecutionManager, reportDir string, kb *knowledge.KnowledgeBase) *Manager {
	return &Manager{
		pluginManager: pm,
		analyzers:     analyzers,
		execManager:   em,
		reportDir:     reportDir,
		knowledgeBase: kb,
		logger:        logger.NewLogger("diagnosis-manager"),
	}
}

func (m *Manager) RunDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progress chan<- interfaces.DiagnosisProgress) (*models.DiagnosisResult, error) {
	defer close(progress)

	m.logger.Infof("Starting new diagnosis for %s on instance %s", req.TargetMiddleware, req.Instance)

	// 1. Data Collection
	progress <- interfaces.DiagnosisProgress{Step: "Collection", Status: "InProgress", Message: "Gathering metrics and logs..."}

	data, err := m.pluginManager.CollectData(ctx, req)
	if err != nil {
		m.logger.Errorf("Data collection failed: %v", err)
		return nil, fmt.Errorf("collection failed: %w", err)
	}

	// 2. Analysis
	progress <- interfaces.DiagnosisProgress{Step: "Analysis", Status: "InProgress", Message: "Analyzing collected data..."}

	issues, err := m.AnalyzeData(ctx, req, data)
	if err != nil {
		m.logger.Errorf("Analysis failed: %v", err)
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// 3. Result Compilation
	progress <- interfaces.DiagnosisProgress{Step: "Reporting", Status: "InProgress", Message: "Generating final report..."}

	result := &models.DiagnosisResult{
		ID:        fmt.Sprintf("%s-%d", req.Instance, time.Now().Unix()),
		Timestamp: time.Now(),
		Status:    calculateOverallStatus(issues),
		Summary:   fmt.Sprintf("Diagnosis completed for %s. Found %d issues.", req.TargetMiddleware, len(issues)),
		Issues:    issues,
	}

	m.logger.Infof("Diagnosis completed for %s. Found %d issues. Report ID: %s", req.TargetMiddleware, len(issues), result.ID)
	return result, nil
}

func (m *Manager) AnalyzeData(ctx context.Context, req *models.DiagnosisRequest, data *models.CollectedData) ([]*models.Issue, error) {
	var allIssues []*models.Issue

	for _, analyzer := range m.analyzers {
		// Since DiagnosisAnalyzer interface is broken into AnalyzeMetrics/Logs/Correlate,
		// and manager is generic, we need to adapt or call specific methods if we know what data we have.
		// For now, assuming analyzer has a generic Analyze method OR we call specific ones if data available.
		// BUT the interface DEFINITION in Step 59 shows NO Analyze method. It shows AnalyzeMetrics, AnalyzeLogs, CorrelateSystems.
		// So `analyzer.Analyze` call was invalid.

		if data.Metrics != nil {
			issues, err := analyzer.AnalyzeMetrics(ctx, data.Metrics)
			if err == nil {
				allIssues = append(allIssues, issues...)
			}
		}
		if data.Logs != nil {
			issues, err := analyzer.AnalyzeLogs(ctx, data.Logs)
			if err == nil {
				allIssues = append(allIssues, issues...)
			}
		}
		// SystemCorrelationData is complex to construct from CollectedData in this context without specific logic.
		// Skipping for now.
	}

	return allIssues, nil
}

func (m *Manager) GenerateReport(result *models.DiagnosisResult) (string, error) {
	bytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (m *Manager) GetDiagnosisResult(id string) (*models.DiagnosisResult, error) {
	// TODO: Implement storage retrieval
	return nil, fmt.Errorf("not implemented")
}

func (m *Manager) GetKnowledgeBase() *knowledge.KnowledgeBase {
	return m.knowledgeBase
}

// Helpers

func calculateOverallStatus(issues []*models.Issue) enum.DiagnosisStatus {
	if len(issues) == 0 {
		return enum.StatusHealthy
	}
	// Logic to determine status based on max severity of issues
	// For now, return Warning if any issues exist
	return enum.StatusWarning
}

// RuleBasedAnalyzer implementation placeholder (Satisfying DiagnosisAnalyzer interface)
type RuleBasedAnalyzer struct {}
func NewRuleBasedAnalyzer(re interface{}, kb *knowledge.KnowledgeBase) *RuleBasedAnalyzer { return &RuleBasedAnalyzer{} }
func (r *RuleBasedAnalyzer) Name() string { return "RuleBased" }
func (r *RuleBasedAnalyzer) AnalyzeMetrics(ctx context.Context, data *models.MetricsData) ([]*models.Issue, error) { return nil, nil }
func (r *RuleBasedAnalyzer) AnalyzeLogs(ctx context.Context, data *models.LogData) ([]*models.Issue, error) { return nil, nil }
func (r *RuleBasedAnalyzer) CorrelateSystems(ctx context.Context, data *models.SystemCorrelationData) ([]*models.Issue, error) { return nil, nil }

// AIAnalyzer implementation placeholder wrapper
func NewAIAnalyzer(client llm_interfaces.LLMClient) (*ai.Analyzer, error) {
	return ai.NewAnalyzer(client, nil), nil
}
