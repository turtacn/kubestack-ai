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

package interfaces

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// DiagnosisProgress represents a single, structured progress update from a diagnosis run.
// It is designed to be sent over a channel to provide real-time feedback to the
// user via a CLI or API.
type DiagnosisProgress struct {
	// Step is the high-level stage of the diagnosis workflow (e.g., "Data Collection", "Analysis").
	Step string
	// Status indicates the current state of the step (e.g., "InProgress", "Completed", "Failed").
	Status string
	// Message provides a detailed, human-readable description of the current action or its outcome.
	Message string
}

// DiagnosisManager defines the contract for the component that orchestrates the
// entire diagnosis workflow. It is responsible for coordinating data collection,
// analysis, and report generation.
type DiagnosisManager interface {
	// RunDiagnosis executes a complete, end-to-end diagnosis based on a given request.
	// This is typically a long-running operation that involves multiple stages. It provides
	// real-time feedback by sending DiagnosisProgress updates to the provided channel.
	//
	// Parameters:
	//   ctx (context.Context): The context for the entire diagnosis operation.
	//   req (*models.DiagnosisRequest): The request detailing what to diagnose.
	//   progressChan (chan<- DiagnosisProgress): A channel for sending progress updates.
	//
	// Returns:
	//   *models.DiagnosisResult: The final result of the diagnosis, including any identified issues.
	//   error: An error if a critical step in the workflow fails.
	RunDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progressChan chan<- DiagnosisProgress) (*models.DiagnosisResult, error)

	// AnalyzeData invokes all registered analyzers to process the collected data and
	// identify potential issues.
	//
	// Parameters:
	//   ctx (context.Context): The context for the analysis operation.
	//   collectedData (*models.CollectedData): The aggregated data collected from the plugin.
	//
	// Returns:
	//   []*models.Issue: A slice containing all issues identified by the analyzers.
	//   error: An error if the analysis process itself fails.
	AnalyzeData(ctx context.Context, collectedData *models.CollectedData) ([]*models.Issue, error)

	// GenerateReport takes a final diagnosis result and formats it into a human-readable report.
	//
	// Parameters:
	//   result (*models.DiagnosisResult): The result to be formatted.
	//
	// Returns:
	//   string: The formatted report.
	//   error: An error if report generation fails.
	GenerateReport(result *models.DiagnosisResult) (string, error)
}

// DiagnosisAnalyzer defines the contract for a pluggable analysis component. It is
// responsible for inspecting a specific type of diagnostic data (or correlating
// multiple types) to identify potential issues. This interface-based approach
// allows for a flexible and extensible diagnosis engine, where different analysis
// strategies (e.g., rule-based, ML-based) can be used interchangeably.
type DiagnosisAnalyzer interface {
	// Name returns the unique, human-readable name of the analyzer (e.g.,
	// "RuleBasedAnalyzer", "AIAnomalyDetector").
	Name() string

	// AnalyzeMetrics inspects performance metric data to find issues like threshold
	// breaches, performance bottlenecks, or anomalies.
	//
	// Parameters:
	//   ctx (context.Context): The context for the analysis operation.
	//   data (*models.MetricsData): The collected metric data to be analyzed.
	//
	// Returns:
	//   []*models.Issue: A slice of issues identified from the metrics.
	//   error: An error if the analysis process fails.
	AnalyzeMetrics(ctx context.Context, data *models.MetricsData) ([]*models.Issue, error)

	// AnalyzeLogs scans log data to find significant entries, such as errors,
	// warnings, or specific known problem patterns.
	//
	// Parameters:
	//   ctx (context.Context): The context for the analysis operation.
	//   data (*models.LogData): The collected log data to be analyzed.
	//
	// Returns:
	//   []*models.Issue: A slice of issues identified from the logs.
	//   error: An error if the analysis process fails.
	AnalyzeLogs(ctx context.Context, data *models.LogData) ([]*models.Issue, error)

	// CorrelateSystems performs advanced, cross-domain analysis to find causal
	// relationships between different data sources. For example, it might correlate
	// a spike in CPU metrics with a specific error message in the logs to pinpoint a root cause.
	//
	// Parameters:
	//   ctx (context.Context): The context for the analysis operation.
	//   data (*models.SystemCorrelationData): A struct containing multiple data types for correlation.
	//
	// Returns:
	//   []*models.Issue: A slice of issues identified through correlation.
	//   error: An error if the correlation process fails.
	CorrelateSystems(ctx context.Context, data *models.SystemCorrelationData) ([]*models.Issue, error)
}

//Personal.AI order the ending
