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

// DiagnosisProgress represents a single progress update from a diagnosis run.
// It's used to provide real-time feedback to the user via the CLI or API.
type DiagnosisProgress struct {
	Step    string // e.g., "Collecting Metrics", "Analyzing Logs"
	Status  string // e.g., "InProgress", "Completed", "Failed"
	Message string // Detailed message about the current step
}

// DiagnosisManager orchestrates the entire diagnosis workflow. It's responsible for
// coordinating data collection, analysis, and report generation.
type DiagnosisManager interface {
	// RunDiagnosis executes a complete diagnosis based on the request.
	// It's a long-running operation that streams progress updates to the provided channel.
	RunDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progressChan chan<- DiagnosisProgress) (*models.DiagnosisResult, error)

	// AnalyzeData invokes registered analyzers to process collected data and identify issues.
	AnalyzeData(ctx context.Context, collectedData *models.CollectedData) ([]*models.Issue, error)

	// GenerateReport takes the final diagnosis result and formats it into a human-readable report.
	GenerateReport(result *models.DiagnosisResult) (string, error)
}

// DiagnosisAnalyzer defines the contract for a component that can analyze a specific
// type of diagnostic data. This allows for a pluggable architecture where new analysis
// capabilities can be easily added.
type DiagnosisAnalyzer interface {
	// Name returns the unique name of the analyzer.
	Name() string

	// AnalyzeMetrics analyzes performance metrics to find anomalies and bottlenecks.
	AnalyzeMetrics(ctx context.Context, data *models.MetricsData) ([]*models.Issue, error)

	// AnalyzeLogs scans log data for errors, warnings, and other significant patterns.
	AnalyzeLogs(ctx context.Context, data *models.LogData) ([]*models.Issue, error)

	// CorrelateSystems performs cross-system analysis to find correlations between different data sources.
	CorrelateSystems(ctx context.Context, data *models.SystemCorrelationData) ([]*models.Issue, error)
}

//Personal.AI order the ending
