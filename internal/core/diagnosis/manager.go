// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
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
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/chain"
	"github.com/kubestack-ai/kubestack-ai/internal/llm/parser"
)

// manager is the concrete implementation of the interfaces.DiagnosisManager.
type manager struct {
	log            logger.Logger
	pluginManager  interfaces.PluginManager
	analyzers      []interfaces.DiagnosisAnalyzer
	diagnosisChain *chain.DiagnosisChain
	cache          *diagnosisCache
	reportDir      string
}

// NewManager creates a new instance of the diagnosis manager.
func NewManager(
	pm interfaces.PluginManager,
	analyzers []interfaces.DiagnosisAnalyzer,
	diagnosisChain *chain.DiagnosisChain,
	reportDir string,
) interfaces.DiagnosisManager {
	// Ensure the report directory exists
	if reportDir == "" {
		reportDir = "reports"
	}
	if _, err := os.Stat(reportDir); os.IsNotExist(err) {
		os.MkdirAll(reportDir, 0755)
	}
	return &manager{
		log:            logger.NewLogger("diagnosis-manager"),
		pluginManager:  pm,
		analyzers:      analyzers,
		diagnosisChain: diagnosisChain,
		cache:          newDiagnosisCache(10 * time.Minute),
		reportDir:      reportDir,
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
	sendProgress(progressChan, "Initialization", "InProgress", "Loading plugin...")

	plugin, err := m.pluginManager.LoadPlugin(req.TargetMiddleware.String())
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}
	sendProgress(progressChan, "Initialization", "Completed", "Plugin loaded successfully.")

	sendProgress(progressChan, "Data Collection", "InProgress", "Collecting metrics, logs, and config...")
	collectedData, err := m.collectData(ctx, plugin)
	if err != nil {
		sendProgress(progressChan, "Data Collection", "Failed", err.Error())
		return nil, fmt.Errorf("failed to collect data: %w", err)
	}
	sendProgress(progressChan, "Data Collection", "Completed", "Data collection finished.")

	sendProgress(progressChan, "Analysis", "InProgress", "Analyzing collected data...")

	// Rule-based analysis
	ruleIssues, err := m.AnalyzeData(ctx, req, collectedData)
	if err != nil {
		m.log.Warnf("Rule-based analysis failed: %v", err)
	}

	// AI Analysis using DiagnosisChain
	var aiIssues []*models.Issue
	if m.diagnosisChain != nil {
		sendProgress(progressChan, "AI Analysis", "InProgress", "Running AI diagnosis chain...")

		// Build query from collected data (simplified for now)
		query := buildQueryFromData(collectedData, req.TargetMiddleware.String())
		chainResult, err := m.diagnosisChain.Execute(ctx, query)
		if err != nil {
			m.log.Warnf("AI analysis failed, falling back to rule-based only: %v", err)
			sendProgress(progressChan, "AI Analysis", "Failed", "AI analysis failed, continuing with rule-based results.")
		} else {
			aiIssues = convertChainResultToIssues(chainResult)
			sendProgress(progressChan, "AI Analysis", "Completed", "AI analysis finished.")
		}
	}

	// Merge issues
	allIssues := append(ruleIssues, aiIssues...)

	sendProgress(progressChan, "Analysis", "Completed", fmt.Sprintf("Analysis finished, found %d issues.", len(allIssues)))

	result := &models.DiagnosisResult{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Status:    determineOverallStatus(allIssues),
		Summary:   generateSummary(allIssues),
		Issues:    allIssues,
	}

	m.cache.Set(req, result)
	if err := m.persistResult(ctx, result); err != nil {
		m.log.Warnf("Failed to persist diagnosis result: %v", err)
	}
	m.log.Infof("Diagnosis completed for %s. Found %d issues. Report ID: %s", req.TargetMiddleware, len(allIssues), result.ID)
	return result, nil
}

func buildQueryFromData(data *models.CollectedData, mwType string) string {
	// Simple query builder. In a real system, this should extract anomalies.
	// For now, we construct a generic query about the middleware and any obvious errors.
	query := fmt.Sprintf("Diagnose %s issues.", mwType)

	if data.Metrics != nil {
		// Example: Add metrics summary if possible, or just mention we have metrics.
		// For now, just appending a string.
		query += " Analyzing metrics."
	}

	if data.Logs != nil && len(data.Logs.Entries) > 0 {
		// Append first few log lines as context or indicators
		count := 3
		if len(data.Logs.Entries) < count {
			count = len(data.Logs.Entries)
		}
		query += " Recent logs: "
		for i := 0; i < count; i++ {
			query += data.Logs.Entries[i] + "; "
		}
	}
	return query
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

func (m *manager) collectData(ctx context.Context, plugin interfaces.MiddlewarePlugin) (*models.CollectedData, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	data := &models.CollectedData{}

	collect := func(collectFunc func(context.Context) (interface{}, error), targetSetter func(interface{})) {
		defer wg.Done()
		res, err := collectFunc(ctx)
		if err != nil {
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
			return
		}
		mu.Lock()
		targetSetter(res)
		mu.Unlock()
	}

	wg.Add(3)
	go collect(func(c context.Context) (interface{}, error) { return plugin.CollectMetrics(c) }, func(r interface{}) { data.Metrics = r.(*models.MetricsData) })
	go collect(func(c context.Context) (interface{}, error) { return plugin.CollectLogs(c, &models.LogOptions{Tail: 1000}) }, func(r interface{}) { data.Logs = r.(*models.LogData) })
	go collect(func(c context.Context) (interface{}, error) { return plugin.GetConfiguration(c) }, func(r interface{}) { data.Config = r.(*models.ConfigData) })

	wg.Wait()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	return data, nil
}

// AnalyzeData runs all registered diagnosis analyzers concurrently.
func (m *manager) AnalyzeData(ctx context.Context, req *models.DiagnosisRequest, data *models.CollectedData) ([]*models.Issue, error) {
	var allIssues []*models.Issue
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	addIssues := func(issues []*models.Issue) {
		if len(issues) > 0 {
			mu.Lock()
			allIssues = append(allIssues, issues...)
			mu.Unlock()
		}
	}

	addErr := func(err error) {
		if err != nil {
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}
	}

	for _, analyzer := range m.analyzers {
		wg.Add(3)

		// Analyze Metrics
		go func(an interfaces.DiagnosisAnalyzer) {
			defer wg.Done()
			if data.Metrics != nil {
				m.log.Debugf("Running %s.AnalyzeMetrics", an.Name())
				issues, err := an.AnalyzeMetrics(ctx, data.Metrics)
				addErr(err)
				addIssues(issues)
			}
		}(analyzer)

		// Analyze Logs
		go func(an interfaces.DiagnosisAnalyzer) {
			defer wg.Done()
			if data.Logs != nil {
				m.log.Debugf("Running %s.AnalyzeLogs", an.Name())
				issues, err := an.AnalyzeLogs(ctx, data.Logs)
				addErr(err)
				addIssues(issues)
			}
		}(analyzer)

		// Correlate Systems
		go func(an interfaces.DiagnosisAnalyzer) {
			defer wg.Done()
			correlationData := &models.SystemCorrelationData{
				DataSources: map[string]interface{}{
					"middlewareName": req.TargetMiddleware.String(),
					"instanceName":   req.Instance,
					"timestamp":      time.Now(),
					"metrics":        data.Metrics,
					"logs":           data.Logs,
					"config":         data.Config,
				},
			}
			m.log.Debugf("Running %s.CorrelateSystems", an.Name())
			issues, err := an.CorrelateSystems(ctx, correlationData)
			addErr(err)
			addIssues(issues)
		}(analyzer)
	}

	wg.Wait()

	if len(errs) > 0 {
		for _, err := range errs {
			m.log.Warnf("An error occurred during analysis: %v", err)
		}
	}

	return allIssues, nil
}

func (m *manager) GenerateReport(result *models.DiagnosisResult) (string, error) {
	return fmt.Sprintf("Diagnosis Report (ID: %s)\nStatus: %s\nSummary: %s\nFound %d issues.",
		result.ID, result.Status, result.Summary, len(result.Issues)), nil
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
