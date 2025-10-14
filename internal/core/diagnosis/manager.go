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
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// manager is the concrete implementation of the interfaces.DiagnosisManager.
type manager struct {
	log           logger.Logger
	pluginManager interfaces.PluginManager
	analyzers     []interfaces.DiagnosisAnalyzer
	cache         *diagnosisCache
	reportDir     string
}

// NewManager creates a new instance of the diagnosis manager, which orchestrates
// the entire diagnosis process. It takes a plugin manager to load the appropriate
// middleware-specific logic and a slice of analyzers to process the collected data.
func NewManager(pm interfaces.PluginManager, analyzers []interfaces.DiagnosisAnalyzer, reportDir string) interfaces.DiagnosisManager {
	// Ensure the report directory exists
	if _, err := os.Stat(reportDir); os.IsNotExist(err) {
		os.MkdirAll(reportDir, 0755)
	}
	return &manager{
		log:           logger.NewLogger("diagnosis-manager"),
		pluginManager: pm,
		analyzers:     analyzers,
		cache:         newDiagnosisCache(10 * time.Minute), // Default 10 min cache TTL
		reportDir:     reportDir,
	}
}

// RunDiagnosis executes the full, end-to-end diagnosis workflow. It handles
// caching, plugin loading, data collection, analysis, and result summarization.
// Progress updates are sent to the provided channel throughout the process.
//
// Parameters:
//   ctx (context.Context): The context for the entire diagnosis operation.
//   req (*models.DiagnosisRequest): The request detailing what to diagnose.
//   progressChan (chan<- interfaces.DiagnosisProgress): A channel to send real-time progress updates.
//
// Returns:
//   *models.DiagnosisResult: The final result of the diagnosis, including any identified issues.
//   error: An error if a critical step in the workflow fails.
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
	issues, err := m.AnalyzeData(ctx, collectedData)
	if err != nil {
		sendProgress(progressChan, "Analysis", "Failed", err.Error())
		return nil, fmt.Errorf("failed during data analysis: %w", err)
	}
	sendProgress(progressChan, "Analysis", "Completed", fmt.Sprintf("Analysis finished, found %d issues.", len(issues)))

	result := &models.DiagnosisResult{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Status:    determineOverallStatus(issues),
		Summary:   generateSummary(issues),
		Issues:    issues,
	}

	m.cache.Set(req, result)
	if err := m.persistResult(ctx, result); err != nil {
		m.log.Warnf("Failed to persist diagnosis result: %v", err)
		// We don't return the error here, as failing to save the report
		// should not fail the entire diagnosis operation.
	}
	m.log.Infof("Diagnosis completed for %s. Found %d issues. Report ID: %s", req.TargetMiddleware, len(issues), result.ID)
	return result, nil
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
		// For simplicity, returning the first error. A real implementation might wrap all errors.
		return nil, errs[0]
	}

	return data, nil
}

// AnalyzeData runs all registered diagnosis analyzers concurrently on the collected
// data. It aggregates the issues identified by each analyzer into a single slice.
//
// Parameters:
//   ctx (context.Context): The context for the analysis operations.
//   data (*models.CollectedData): The collected data to be analyzed.
//
// Returns:
//   []*models.Issue: A slice containing all issues identified by the analyzers.
//   error: An error if the analysis process itself fails (nil in this implementation).
func (m *manager) AnalyzeData(ctx context.Context, data *models.CollectedData) ([]*models.Issue, error) {
	var allIssues []*models.Issue
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, analyzer := range m.analyzers {
		wg.Add(1)
		go func(an interfaces.DiagnosisAnalyzer) {
			defer wg.Done()
			m.log.Debugf("Running analyzer: %s", an.Name())
			var issues []*models.Issue
			var err error
			// Simplified analysis dispatch. A real implementation might have a more sophisticated way to route data to analyzers.
			if data.Metrics != nil {
				issues, err = an.AnalyzeMetrics(ctx, data.Metrics)
				if err != nil {
					m.log.Warnf("Analyzer %s failed on metrics: %v", an.Name(), err)
				} else {
					mu.Lock()
					allIssues = append(allIssues, issues...)
					mu.Unlock()
				}
			}
		}(analyzer)
	}

	wg.Wait()
	return allIssues, nil
}

// GenerateReport creates a simple, human-readable string summary of a diagnosis result.
// NOTE: This is a placeholder. A more advanced implementation would use the
// `internal/cli/ui/formatter` for rich, structured output.
//
// Parameters:
//   result (*models.DiagnosisResult): The diagnosis result to be reported.
//
// Returns:
//   string: A formatted string summarizing the report.
//   error: An error if report generation fails (nil in this implementation).
func (m *manager) GenerateReport(result *models.DiagnosisResult) (string, error) {
	// This is a placeholder for a proper report generator, which might use text/template.
	return fmt.Sprintf("Diagnosis Report (ID: %s)\nStatus: %s\nSummary: %s\nFound %d issues.",
		result.ID, result.Status, result.Summary, len(result.Issues)), nil
}

// --- Helper Functions ---
func sendProgress(ch chan<- interfaces.DiagnosisProgress, step, status, msg string) {
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

//Personal.AI order the ending
