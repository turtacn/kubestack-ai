// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law of agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package diagnosis implements the core logic for the diagnosis engine.
package diagnosis

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/detection"
	detection_models "github.com/kubestack-ai/kubestack-ai/internal/core/detection/models"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/core/rca"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/chain"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/client"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/parser"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// manager is the concrete implementation of the interfaces.DiagnosisManager.
type manager struct {
	log             logger.Logger
	pluginRegistry  *plugin.Registry
	analyzers       []interfaces.DiagnosisAnalyzer
	diagnosisChain  *chain.DiagnosisChain
	cache           *diagnosisCache
	reportDir       string
	anomalyDetector *detection.AnomalyDetector
	rcaEngine       *rca.RulesEngine

	// Knowledge Base Components
	knowledgeBase   *knowledge.KnowledgeBase
	ruleEngine      *knowledge.RuleEngine
	ruleLoader      *knowledge.RuleLoader
	llmIntegration  *knowledge.LLMIntegration
	config          *config.Config
}

// NewManager creates a new instance of the diagnosis manager.
// kb is optional, if provided it uses the existing knowledge base, otherwise it creates a new one.
func NewManager(
	registry *plugin.Registry,
	analyzers []interfaces.DiagnosisAnalyzer,
	diagnosisChain *chain.DiagnosisChain,
	reportDir string,
	kb *knowledge.KnowledgeBase, // Optional injection
) interfaces.DiagnosisManager {
	// Ensure the report directory exists
	if reportDir == "" {
		reportDir = "reports"
	}
	if _, err := os.Stat(reportDir); os.IsNotExist(err) {
		os.MkdirAll(reportDir, 0755)
	}

	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		cfg = &config.Config{} // Empty config
	}

	// Initialize Knowledge Base Components
	var ruleEngine *knowledge.RuleEngine
	var ruleLoader *knowledge.RuleLoader

	if kb == nil {
		kb = knowledge.NewKnowledgeBase()
	}

	ruleEngine = knowledge.NewRuleEngine(kb)
	ruleLoader = knowledge.NewRuleLoader(kb)

	if len(cfg.Knowledge.RuleFiles) > 0 {
		for _, file := range cfg.Knowledge.RuleFiles {
			// Ignore errors for now or log them
			_ = ruleLoader.LoadFromFile(file)
		}
	} else {
		// Try loading from default directory
		_ = ruleLoader.LoadFromDirectory("internal/knowledge/repository")
	}

	// Initialize LLM Integration
	var llmInt *knowledge.LLMIntegration
	if cfg.Knowledge.EnableLLMEnhancement {
		c, err := client.NewClientFromConfig(&cfg.LLM)
		if err == nil {
			llmInt = knowledge.NewLLMIntegration(c, cfg.LLM)
		} else {
			fmt.Printf("Failed to initialize LLM client: %v\n", err)
		}
	}

	return &manager{
		log:             logger.NewLogger("diagnosis-manager"),
		pluginRegistry:  registry,
		analyzers:       analyzers,
		diagnosisChain:  diagnosisChain,
		cache:           newDiagnosisCache(10 * time.Minute),
		reportDir:       reportDir,
		anomalyDetector: detection.NewAnomalyDetector(cfg),
		rcaEngine:       rca.NewRulesEngine(cfg),

		knowledgeBase:   kb,
		ruleEngine:      ruleEngine,
		ruleLoader:      ruleLoader,
		llmIntegration:  llmInt,
		config:          cfg,
	}
}

// RunDiagnosis executes the full, end-to-end diagnosis workflow.
func (m *manager) RunDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progressChan chan<- interfaces.DiagnosisProgress) (*models.DiagnosisResult, error) {
	if result, found := m.cache.Get(req); found {
		m.log.Info("Returning diagnosis result from cache.")
		sendProgress(progressChan, "Cache", "Completed", "Found valid result in cache.")
		return result, nil
	}

	m.log.Infof("Starting new diagnosis for %s on instance %s", req.TargetMiddleware, req.Instance)
	sendProgress(progressChan, "Initialization", "InProgress", "Finding plugins...")

	// Step 1: 根据中间件类型查找插件
	plugins := m.pluginRegistry.FindByType(req.TargetMiddleware.String())
	if len(plugins) == 0 {
		return nil, fmt.Errorf("未找到支持 %s 的插件", req.TargetMiddleware)
	}
	sendProgress(progressChan, "Initialization", "Completed", fmt.Sprintf("Found %d plugins.", len(plugins)))

	// Step 2: Anomaly Detection
	sendProgress(progressChan, "Detection", "InProgress", "Running anomaly detection...")

	detectionInput := &detection_models.DetectionInput{
		Context: map[string]string{
			"middleware": req.TargetMiddleware.String(),
			"instance":   req.Instance,
		},
	}

	detectionResult, err := m.anomalyDetector.Detect(ctx, detectionInput)
	if err != nil {
		m.log.Warnf("Anomaly detection failed: %v", err)
	} else {
		m.log.Infof("Anomaly detection completed. Found %d anomalies.", len(detectionResult.Anomalies))
	}

	// Add anomalies to context so plugins or subsequent steps can use them
	ctx = context.WithValue(ctx, "anomalies", detectionResult.Anomalies)

	// Step 3: Root Cause Analysis (RCA) based on Anomalies
	if len(detectionResult.Anomalies) > 0 {
		sendProgress(progressChan, "RCA", "InProgress", "Running root cause analysis...")
		rcaResult, err := m.rcaEngine.Analyze(ctx, detectionResult.Anomalies)
		if err != nil {
			m.log.Warnf("RCA failed: %v", err)
		} else {
			m.log.Infof("RCA completed. Root cause: %s", rcaResult.RootCause)
			ctx = context.WithValue(ctx, "root_cause", rcaResult)
		}
	}

	sendProgress(progressChan, "Detection", "Completed", "Anomaly detection finished.")


	// Step 4: Execute Plugins (Deep Diagnosis)
	sendProgress(progressChan, "Diagnosis", "InProgress", "Running diagnosis plugins...")

	var allIssues []*models.Issue
	var mu sync.Mutex
	var wg sync.WaitGroup

	metricsData := make(map[string]interface{})

	for _, p := range plugins {
		wg.Add(1)
		go func(pl plugin.DiagnosticPlugin) {
			defer wg.Done()
			result, err := pl.Diagnose(ctx, req)
			if err != nil {
				m.log.Warnf("Plugin %s execution failed: %v", pl.Name(), err)
				return
			}
			mu.Lock()
			if result != nil {
				allIssues = append(allIssues, result.Issues...)
				// Collect metrics if available
				// Models.DiagnosisResult has Metrics now
				if result.Metrics != nil {
					for k, v := range result.Metrics {
						metricsData[k] = v
					}
				}
			}
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	sendProgress(progressChan, "Diagnosis", "Completed", "Plugins finished.")

	// Step 5: Merge RCA results into issues if not already present
	if rcaResult := ctx.Value("root_cause"); rcaResult != nil {
		if res, ok := rcaResult.(*rca.RCAResult); ok && res.Confidence > 0.5 {
			rcaIssue := &models.Issue{
				ID: uuid.New().String(),
				Source: "RCA Engine",
				Title: res.RootCause,
				Description: fmt.Sprintf("Root cause analysis identified %s with %.2f confidence.", res.RootCause, res.Confidence),
				Severity: enum.SeverityHigh, // Default to High for RCA findings
				Recommendations: convertToRecommendations(res.Recommendations),
			}
			allIssues = append(allIssues, rcaIssue)
		}
	}

	// If Anomaly Detection found things, add them as issues too
	if detectionResult != nil && len(detectionResult.Anomalies) > 0 {
		for _, anomaly := range detectionResult.Anomalies {
			anomalyIssue := &models.Issue{
				ID: uuid.New().String(),
				Source: "AnomalyDetector",
				Title: fmt.Sprintf("Anomaly: %s", anomaly.Type),
				Description: anomaly.Description,
				Severity: convertSeverity(anomaly.Severity),
				Evidence: fmt.Sprintf("Detected at %s", anomaly.StartTime),
			}
			allIssues = append(allIssues, anomalyIssue)
		}
	}

	// Step 6: Knowledge Base Rule Engine Diagnosis (NEW)
	sendProgress(progressChan, "RuleEngine", "InProgress", "Running knowledge base rules...")

	diagCtx := &knowledge.DiagnosisContext{
		MiddlewareType: req.TargetMiddleware.String(),
		Namespace:      req.Namespace,
		Metrics:        metricsData,
		Issues:         allIssues,
	}

	ruleMatches, err := m.ruleEngine.Match(diagCtx)
	var ruleRecommendations []*models.Recommendation
	if err != nil {
		m.log.Errorf("Rule matching failed: %v", err)
	} else {
		recommendations, err := m.ruleEngine.Execute(ruleMatches)
		if err != nil {
			m.log.Errorf("Rule execution failed: %v", err)
		} else {
			for _, rec := range recommendations {
				ruleRecommendations = append(ruleRecommendations, &models.Recommendation{
					ID:          uuid.New().String(),
					Description: fmt.Sprintf("%s: %s", rec.Title, rec.Action),
					CanAutoFix:  false, // Assuming rules don't define auto-fix yet
					Priority:    models.Priority(rec.Priority), // Now valid
				})
			}
			// Append rule recommendations to a generic "Rule Engine Findings" issue or distribute them
			if len(ruleRecommendations) > 0 {
				ruleIssue := &models.Issue{
					ID:              uuid.New().String(),
					Source:          "RuleEngine",
					Title:           "Knowledge Base Recommendations",
					Description:     "Recommendations generated based on knowledge base rules.",
					Severity:        enum.SeverityInfo, // Now valid
					Recommendations: ruleRecommendations,
				}
				allIssues = append(allIssues, ruleIssue)
			}
		}
	}
	sendProgress(progressChan, "RuleEngine", "Completed", fmt.Sprintf("Matched %d rules.", len(ruleMatches)))

	// Step 7: LLM Enhancement (NEW)
	if m.config.Knowledge.EnableLLMEnhancement && m.llmIntegration != nil {
		sendProgress(progressChan, "LLM", "InProgress", "Enhancing diagnosis with LLM...")
		llmRecs, err := m.llmIntegration.GenerateRecommendations(ctx, diagCtx)
		if err != nil {
			m.log.Warnf("LLM diagnosis failed: %v", err)
		} else {
			var llmModelsRecs []*models.Recommendation
			for _, rec := range llmRecs {
				llmModelsRecs = append(llmModelsRecs, &models.Recommendation{
					ID:          uuid.New().String(),
					Description: fmt.Sprintf("[AI] %s: %s", rec.Title, rec.Action),
					CanAutoFix:  false,
				})
			}
			if len(llmModelsRecs) > 0 {
				llmIssue := &models.Issue{
					ID:              uuid.New().String(),
					Source:          "LLM",
					Title:           "AI Enhanced Recommendations",
					Description:     "Recommendations generated by AI assistant.",
					Severity:        enum.SeverityInfo,
					Recommendations: llmModelsRecs,
				}
				allIssues = append(allIssues, llmIssue)
			}
		}
		sendProgress(progressChan, "LLM", "Completed", "LLM enhancement finished.")
	}

	sendProgress(progressChan, "Analysis", "Completed", fmt.Sprintf("Analysis finished, found %d issues.", len(allIssues)))

	result := &models.DiagnosisResult{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Status:    determineOverallStatus(allIssues),
		Summary:   generateSummary(allIssues),
		Issues:    allIssues,
		Metrics:   metricsData, // Include collected metrics in result
	}

	m.cache.Set(req, result)
	if err := m.persistResult(ctx, result); err != nil {
		m.log.Warnf("Failed to persist diagnosis result: %v", err)
	}
	m.log.Infof("Diagnosis completed for %s. Found %d issues. Report ID: %s", req.TargetMiddleware, len(allIssues), result.ID)
	return result, nil
}

func convertToRecommendations(recs []string) []*models.Recommendation {
	var result []*models.Recommendation
	for i, r := range recs {
		result = append(result, &models.Recommendation{
			ID: fmt.Sprintf("rca-rec-%d", i),
			Description: r,
			CanAutoFix: false,
		})
	}
	return result
}

func convertSeverity(s string) enum.SeverityLevel {
	switch s {
	case detection_models.SeverityCritical:
		return enum.SeverityCritical
	case detection_models.SeverityHigh:
		return enum.SeverityHigh
	case detection_models.SeverityMedium:
		return enum.SeverityMedium
	case detection_models.SeverityLow:
		return enum.SeverityLow
	default:
		return enum.SeverityWarning
	}
}

func buildQueryFromData(data *models.CollectedData, mwType string) string {
	// ... (helper remains if needed, but currently unused in new flow)
	return ""
}

func convertChainResultToIssues(res *parser.DiagnosisResult) []*models.Issue {
	issues := make([]*models.Issue, 0)

	severity := enum.SeverityWarning
	switch strings.ToLower(res.Severity) {
	case "critical":
		severity = enum.SeverityCritical
	case "high":
		severity = enum.SeverityHigh
	case "medium":
		severity = enum.SeverityMedium
	case "low":
		severity = enum.SeverityLow // Fixed from SeverityInfo
	}

	// Convert recommendations/next steps
	recommendations := make([]*models.Recommendation, 0)
	for i, step := range res.NextSteps {
		recommendations = append(recommendations, &models.Recommendation{
			ID:          fmt.Sprintf("rec-%d", i),
			Description: step,
			CanAutoFix:  false, // AI usually needs verification
		})
	}

	// Create a single issue from the AI result for now
	issue := &models.Issue{
		ID:              uuid.New().String(),
		Source:          "AI",
		Title:           res.RootCause,
		Description:     strings.Join(res.ContributingFactors, ", "),
		Severity:        severity,
		Evidence:        strings.Join(res.Evidence, "\n"), // Map evidence
		Recommendations: recommendations, // Map recommendations
	}
	issues = append(issues, issue)

	return issues
}

// persistResult saves a diagnosis result to a JSON file in the configured report directory.
func (m *manager) persistResult(ctx context.Context, result *models.DiagnosisResult) error {
	filePath := filepath.Join(m.reportDir, fmt.Sprintf("%s.json", result.ID))
	m.log.Debugf("Persisting diagnosis report to %s", filePath)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal diagnosis result: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write diagnosis report to file: %w", err)
	}

	return nil
}

// AnalyzeData is kept for interface compliance but might be deprecated or unused in P4 flow
func (m *manager) AnalyzeData(ctx context.Context, req *models.DiagnosisRequest, data *models.CollectedData) ([]*models.Issue, error) {
	// ...
	return nil, nil
}

func (m *manager) GenerateReport(result *models.DiagnosisResult) (string, error) {
	return fmt.Sprintf("Diagnosis Report (ID: %s)\nStatus: %s\nSummary: %s\nFound %d issues.",
		result.ID, result.Status, result.Summary, len(result.Issues)), nil
}

// GetDiagnosisResult retrieves a previously completed diagnosis result by its ID.
func (m *manager) GetDiagnosisResult(id string) (*models.DiagnosisResult, error) {
	filePath := filepath.Join(m.reportDir, fmt.Sprintf("%s.json", id))
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("diagnosis result not found for ID: %s", id)
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read diagnosis report: %w", err)
	}

	var result models.DiagnosisResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal diagnosis result: %w", err)
	}

	return &result, nil
}

// GetKnowledgeBase exposes the KnowledgeBase instance for sharing.
func (m *manager) GetKnowledgeBase() *knowledge.KnowledgeBase {
	return m.knowledgeBase
}

// --- Helper Functions ---
func sendProgress(ch chan<- interfaces.DiagnosisProgress, step, status, msg string) {
	// If channel is nil, don't send
	if ch == nil {
		return
	}
	ch <- interfaces.DiagnosisProgress{Step: step, Status: status, Message: msg}
}

func determineOverallStatus(issues []*models.Issue) enum.DiagnosisStatus {
	if len(issues) == 0 {
		return enum.StatusHealthy
	}
	for _, issue := range issues {
		if issue.Severity == enum.SeverityCritical {
			return enum.StatusCritical
		}
	}
	return enum.StatusWarning
}

func generateSummary(issues []*models.Issue) string {
	if len(issues) == 0 {
		return "System appears to be healthy. No issues found."
	}
	return fmt.Sprintf("Found %d potential issue(s). Please review the details.", len(issues))
}

// --- Simple Cache Implementation ---
type cacheItem struct {
	result    *models.DiagnosisResult
	expiresAt time.Time
}
type diagnosisCache struct {
	items map[string]*cacheItem
	ttl   time.Duration
	mu    sync.RWMutex
}
func newDiagnosisCache(ttl time.Duration) *diagnosisCache {
	return &diagnosisCache{items: make(map[string]*cacheItem), ttl: ttl}
}
func (c *diagnosisCache) Get(req *models.DiagnosisRequest) (*models.DiagnosisResult, bool) {
	key := fmt.Sprintf("%s-%s", req.TargetMiddleware, req.Instance)
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.items[key]
	if !found || time.Now().After(item.expiresAt) {
		return nil, false
	}
	return item.result, true
}
func (c *diagnosisCache) Set(req *models.DiagnosisRequest, result *models.DiagnosisResult) {
	key := fmt.Sprintf("%s-%s", req.TargetMiddleware, req.Instance)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = &cacheItem{result: result, expiresAt: time.Now().Add(c.ttl)}
}
