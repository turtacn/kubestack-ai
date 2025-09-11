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

// Package models defines the data structures used to pass information between the core components of the application.
// These structs are designed to be serializable to formats like JSON and YAML.
package models

import (
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
)

// DiagnosisRequest contains all parameters for initiating a diagnosis run.
type DiagnosisRequest struct {
	TargetMiddleware enum.MiddlewareType `json:"targetMiddleware" yaml:"targetMiddleware"`
	Namespace        string              `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Instance         string              `json:"instance" yaml:"instance"`
	// Additional options like specific checks to run can be added here.
}

// DiagnosisResult is the comprehensive output of a completed diagnosis run.
type DiagnosisResult struct {
	ID        string                `json:"id" yaml:"id"`
	Timestamp time.Time             `json:"timestamp" yaml:"timestamp"`
	Status    enum.DiagnosisStatus  `json:"status" yaml:"status"`
	Summary   string                `json:"summary" yaml:"summary"`
	Issues    []*Issue              `json:"issues" yaml:"issues"`
}

// Issue represents a single, specific problem or anomaly found during diagnosis.
type Issue struct {
	ID              string              `json:"id" yaml:"id"`
	Title           string              `json:"title" yaml:"title"`
	Severity        enum.SeverityLevel  `json:"severity" yaml:"severity"`
	Description     string              `json:"description" yaml:"description"`
	Evidence        string              `json:"evidence" yaml:"evidence"` // e.g., log snippets, metric values.
	Recommendations []*Recommendation   `json:"recommendations" yaml:"recommendations"`
}

// Recommendation provides a suggested action or a set of actions to resolve a specific issue.
type Recommendation struct {
	ID          string `json:"id" yaml:"id"`
	Description string `json:"description" yaml:"description"`
	Command     string `json:"command,omitempty" yaml:"command,omitempty"`
	CanAutoFix  bool   `json:"canAutoFix" yaml:"canAutoFix"`
}

// MetricsData holds collected performance metrics, typically as a map of metric names to their values or time-series data.
type MetricsData struct {
	Data map[string]interface{} `json:"data" yaml:"data"`
}

// LogData holds collected log information.
type LogData struct {
	Entries []string `json:"entries" yaml:"entries"`
}

// ConfigData holds configuration information retrieved from a middleware instance.
type ConfigData struct {
	Data map[string]string `json:"data" yaml:"data"`
}

// AIAnalysisResult contains the structured output from the LLM analysis step.
type AIAnalysisResult struct {
	Summary   string   `json:"summary" yaml:"summary"`
	Reasoning string   `json:"reasoning,omitempty" yaml:"reasoning,omitempty"` // The chain-of-thought from the AI.
	Issues    []*Issue `json:"issues" yaml:"issues"`
}

// --- Supporting Structs for Interfaces ---

// CollectedData is a container for all data collected during the diagnosis phase, ready for analysis.
type CollectedData struct {
	Metrics *MetricsData
	Logs    *LogData
	Config  *ConfigData
	// Other data types can be added here.
}

// SystemCorrelationData is a placeholder for data used in cross-system analysis.
type SystemCorrelationData struct {
	DataSources map[string]interface{} `json:"dataSources" yaml:"dataSources"`
}

// LogOptions specifies parameters for log collection.
type LogOptions struct {
	Since  time.Duration `json:"since,omitempty" yaml:"since,omitempty"`
	Tail   int           `json:"tail,omitempty" yaml:"tail,omitempty"`
	Follow bool          `json:"follow,omitempty" yaml:"follow,omitempty"`
}

// HealthStatus represents the health of a component, typically from a ping or health check.
type HealthStatus struct {
	IsHealthy bool   `json:"isHealthy" yaml:"isHealthy"`
	Message   string `json:"message" yaml:"message"`
}

// FixAction represents a specific fix to be executed, often derived from a Recommendation.
type FixAction struct {
	ID          string            `json:"id" yaml:"id"`
	Description string            `json:"description" yaml:"description"`
	Command     string            `json:"command" yaml:"command"`
	Parameters  map[string]string `json:"parameters" yaml:"parameters"`
}

// FixResult represents the outcome of a single executed fix action.
type FixResult struct {
	Success bool   `json:"success" yaml:"success"`
	Message string `json:"message" yaml:"message"`
}

//Personal.AI order the ending
