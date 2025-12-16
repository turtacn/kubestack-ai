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

package diagnosis

import (
	"context"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/analysis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/core/report"
)

// Orchestrator coordinates the complete diagnosis pipeline with clear stage boundaries:
// 1. Context Collection - Gather data from plugins
// 2. Analysis - Process data through analyzers
// 3. Report Generation - Build structured output
//
// This design enables:
// - Clear separation of concerns
// - Parallel evolution of each stage
// - Easy testing and mocking
// - Support for multiple analyzer types (rule-based, AI, RAG)
type Orchestrator struct {
	pluginManager interfaces.PluginManager
	analyzers     []analysis.Analyzer
	logger        logger.Logger
}

// NewOrchestrator creates a new diagnosis orchestrator
func NewOrchestrator(pm interfaces.PluginManager, analyzers []analysis.Analyzer) *Orchestrator {
	return &Orchestrator{
		pluginManager: pm,
		analyzers:     analyzers,
		logger:        logger.NewLogger("diagnosis-orchestrator"),
	}
}

// RunDiagnosis executes the complete diagnosis pipeline
func (o *Orchestrator) RunDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progress chan<- interfaces.DiagnosisProgress) (*report.DiagnosisReport, error) {
	defer close(progress)

	o.logger.Infof("Starting diagnosis for %s on instance %s", req.TargetMiddleware, req.Instance)

	// Generate unique diagnosis ID
	diagnosisID := fmt.Sprintf("%s-%s-%d", req.TargetMiddleware, req.Instance, time.Now().Unix())

	// Stage 1: Context Collection
	progress <- interfaces.DiagnosisProgress{
		Step:    "Collection",
		Status:  "InProgress",
		Message: "Collecting data from plugins...",
	}

	collectedData, err := o.collectData(ctx, req)
	if err != nil {
		o.logger.Errorf("Data collection failed: %v", err)
		progress <- interfaces.DiagnosisProgress{
			Step:    "Collection",
			Status:  "Failed",
			Message: fmt.Sprintf("Collection error: %v", err),
		}
		return nil, fmt.Errorf("data collection failed: %w", err)
	}

	progress <- interfaces.DiagnosisProgress{
		Step:    "Collection",
		Status:  "Completed",
		Message: "Data collection completed successfully",
	}

	// Stage 2: Analysis
	progress <- interfaces.DiagnosisProgress{
		Step:    "Analysis",
		Status:  "InProgress",
		Message: "Analyzing collected data...",
	}

	analysisResults, err := o.analyzeData(ctx, collectedData)
	if err != nil {
		o.logger.Errorf("Analysis failed: %v", err)
		progress <- interfaces.DiagnosisProgress{
			Step:    "Analysis",
			Status:  "Failed",
			Message: fmt.Sprintf("Analysis error: %v", err),
		}
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	progress <- interfaces.DiagnosisProgress{
		Step:    "Analysis",
		Status:  "Completed",
		Message: fmt.Sprintf("Analysis completed. Found %d analyzer results", len(analysisResults)),
	}

	// Stage 3: Report Generation
	progress <- interfaces.DiagnosisProgress{
		Step:    "Reporting",
		Status:  "InProgress",
		Message: "Generating diagnosis report...",
	}

	diagnosisReport := o.buildReport(diagnosisID, req, collectedData, analysisResults)

	progress <- interfaces.DiagnosisProgress{
		Step:    "Reporting",
		Status:  "Completed",
		Message: fmt.Sprintf("Report generated with %d issues", len(diagnosisReport.Issues)),
	}

	o.logger.Infof("Diagnosis completed for %s. Report ID: %s, Issues: %d",
		req.TargetMiddleware, diagnosisReport.ID, len(diagnosisReport.Issues))

	return diagnosisReport, nil
}

// collectData gathers data from plugins (Stage 1)
func (o *Orchestrator) collectData(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error) {
	o.logger.Debugf("Collecting data for %s/%s", req.TargetMiddleware, req.Instance)

	data, err := o.pluginManager.CollectData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("plugin data collection failed: %w", err)
	}

	return data, nil
}

// analyzeData runs all analyzers on collected data (Stage 2)
func (o *Orchestrator) analyzeData(ctx context.Context, data *models.CollectedData) ([]*analysis.AnalysisResult, error) {
	results := make([]*analysis.AnalysisResult, 0, len(o.analyzers))

	for _, analyzer := range o.analyzers {
		o.logger.Debugf("Running analyzer: %s", analyzer.Name())

		result, err := analyzer.Analyze(ctx, data)
		if err != nil {
			o.logger.Warnf("Analyzer %s failed: %v", analyzer.Name(), err)
			// Continue with other analyzers even if one fails
			continue
		}

		if result != nil {
			results = append(results, result)
			o.logger.Debugf("Analyzer %s found %d issues", analyzer.Name(), len(result.Issues))
		}
	}

	return results, nil
}

// buildReport constructs the final structured report (Stage 3)
func (o *Orchestrator) buildReport(
	id string,
	req *models.DiagnosisRequest,
	data *models.CollectedData,
	analysisResults []*analysis.AnalysisResult,
) *report.DiagnosisReport {

	// Create report with target info
	target := report.DiagnosisTarget{
		Middleware: req.TargetMiddleware,
		Instance:   req.Instance,
		Namespace:  req.Namespace,
	}

	diagReport := report.NewDiagnosisReport(id, target)

	// Aggregate all issues from analysis results
	totalIssues := 0
	for _, result := range analysisResults {
		if result != nil && len(result.Issues) > 0 {
			reportIssues := report.FromModelsIssues(result.Issues)
			diagReport.AddIssues(reportIssues)
			totalIssues += len(result.Issues)
		}
	}

	// Add metrics from collected data if available
	if data.Metrics != nil && data.Metrics.Data != nil {
		diagReport.Metrics = data.Metrics.Data
	}

	// Set summary
	if totalIssues == 0 {
		diagReport.Summary = fmt.Sprintf("Diagnosis completed for %s. No issues detected.", req.TargetMiddleware)
	} else {
		diagReport.Summary = fmt.Sprintf("Diagnosis completed for %s. Found %d issue(s).", req.TargetMiddleware, totalIssues)
	}

	// Add metadata
	diagReport.Metadata["analyzer_count"] = len(o.analyzers)
	diagReport.Metadata["target_middleware"] = req.TargetMiddleware
	diagReport.Metadata["target_instance"] = req.Instance

	return diagReport
}
