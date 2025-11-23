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
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// manager is the concrete implementation of the interfaces.DiagnosisManager.
type manager struct {
	log            logger.Logger
	pluginRegistry *plugin.Registry
	// pluginManager  interfaces.PluginManager // Legacy support if needed, but we are switching to registry
	analyzers      []interfaces.DiagnosisAnalyzer
	diagnosisChain *chain.DiagnosisChain
	cache          *diagnosisCache
	reportDir      string
}

// NewManager creates a new instance of the diagnosis manager.
func NewManager(
	registry *plugin.Registry,
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
		pluginRegistry: registry,
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
	sendProgress(progressChan, "Initialization", "InProgress", "Finding plugins...")

	// Step 1: 根据中间件类型查找插件
	plugins := m.pluginRegistry.FindByType(req.TargetMiddleware.String())
	if len(plugins) == 0 {
		return nil, fmt.Errorf("未找到支持 %s 的插件", req.TargetMiddleware)
	}
	sendProgress(progressChan, "Initialization", "Completed", fmt.Sprintf("Found %d plugins.", len(plugins)))

	// Step 2: 执行所有匹配的插件
	sendProgress(progressChan, "Diagnosis", "InProgress", "Running diagnosis plugins...")

	var allIssues []*models.Issue
	var mu sync.Mutex
	var wg sync.WaitGroup

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
			}
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	sendProgress(progressChan, "Diagnosis", "Completed", "Plugins finished.")

	// Step 3: Analysis (Optional: Reuse analyzers if needed, but Plugin does most work now)
	// We can still run AI analysis if we have collected data, but the new plugin interface
	// doesn't return raw data (metrics/logs) explicitly in the same way.
	// It returns a DiagnosisResult directly.
	// If we want to keep AI chain, we might need plugins to attach raw evidence to issues.

	// AI Analysis using DiagnosisChain (if applicable and if we have data to feed it)
	// Since the new plugin interface encapsulates data collection, we might skip centralized AI analysis
	// OR we assume the plugin puts enough info in the Issue description/evidence for a 2nd pass.
	// For now, let's assume plugins provide the primary diagnosis.

	// Merge issues
	// (Already done in loop)

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
