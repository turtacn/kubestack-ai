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

// Package models defines the data structures used to pass information between the core components of the application.
// These structs are designed to be serializable to formats like JSON and YAML.
package models

import (
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
)

// DiagnosisRequest contains all the parameters required to initiate a diagnosis run.
// It specifies the target middleware and its location.
type DiagnosisRequest struct {
	// TargetMiddleware is the type of middleware to be diagnosed (e.g., Redis, MySQL).
	TargetMiddleware enum.MiddlewareType `json:"targetMiddleware" yaml:"targetMiddleware"`
	// Namespace is the Kubernetes namespace where the middleware instance is located.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	// Instance is the name of the specific middleware instance to diagnose.
	Instance string `json:"instance" yaml:"instance"`
	// OutputFormat specifies the desired format of the result (e.g., "json", "text").
	// Defaults to "text" if not specified.
	OutputFormat string `json:"outputFormat,omitempty" yaml:"outputFormat,omitempty"`
}

// DiagnosisResult is the comprehensive, structured output of a completed diagnosis run.
// It serves as the primary data object for reporting findings to the user.
type DiagnosisResult struct {
	// ID is the unique identifier for this specific diagnosis run.
	ID string `json:"id" yaml:"id"`
	// Timestamp is the UTC time when the diagnosis was completed.
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
	// Status is the overall health status determined by the diagnosis (e.g., Healthy, Warning, Critical).
	Status enum.DiagnosisStatus `json:"status" yaml:"status"`
	// Summary is a high-level, human-readable summary of the diagnosis findings.
	Summary string `json:"summary" yaml:"summary"`
	// Issues is a slice of all the specific problems or anomalies found during the diagnosis.
	Issues []*Issue `json:"issues" yaml:"issues"`
	// Metrics holds important metrics collected during diagnosis.
	Metrics map[string]interface{} `json:"metrics,omitempty" yaml:"metrics,omitempty"`
}

// Issue represents a single, specific problem or anomaly identified during diagnosis.
type Issue struct {
	// ID is the unique identifier for this issue.
	ID string `json:"id" yaml:"id"`
	// Source indicates where this issue originated (e.g., "AI", "Rule", "Manual").
	Source string `json:"source" yaml:"source"`
	// Title is a concise, human-readable title for the issue.
	Title string `json:"title" yaml:"title"`
	// Severity indicates the seriousness of the issue (e.g., Critical, High, Low).
	Severity enum.SeverityLevel `json:"severity" yaml:"severity"`
	// Description provides a detailed explanation of the issue, its context, and its potential impact.
	Description string `json:"description" yaml:"description"`
	// Evidence provides concrete data that supports the finding, such as log snippets or metric values.
	Evidence string `json:"evidence" yaml:"evidence"`
	// Recommendations is a list of suggested actions to resolve the issue.
	Recommendations []*Recommendation `json:"recommendations" yaml:"recommendations"`
}

// Recommendation provides a suggested action or set of actions to resolve a specific issue.
type Recommendation struct {
	// ID is the unique identifier for this recommendation.
	ID string `json:"id" yaml:"id"`
	// Description is a human-readable explanation of the recommended action.
	Description string `json:"description" yaml:"description"`
	// Command is the specific shell command to be executed for an automated fix, if available.
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
	// CanAutoFix indicates whether this recommendation can be safely and automatically applied by the execution engine.
	CanAutoFix bool `json:"canAutoFix" yaml:"canAutoFix"`
	// Category helps classify the action type for dependency analysis (e.g., "ConfigChange", "Restart", "Validation").
	Category string `json:"category,omitempty" yaml:"category,omitempty"`
	// Priority indicates the importance of this recommendation.
	Priority Priority `json:"priority,omitempty" yaml:"priority,omitempty"`
	// RollbackCommand is the command to revert the action if it fails.
	RollbackCommand string `json:"rollback_command,omitempty"`
	// ValidationCommand is a command to verify the action was successful.
	ValidationCommand string `json:"validation_command,omitempty"`
}

// Priority defines the importance of a recommendation.
type Priority int

// MetricsData is a generic container for collected performance metrics. It uses a
// map to allow for flexibility, as different plugins may collect different metrics.
type MetricsData struct {
	// Data holds the collected metrics, where the key is the metric name and the value is the metric's value.
	Data map[string]interface{} `json:"data" yaml:"data"`
}

// LogData is a container for collected log information.
type LogData struct {
	// Entries is a slice of individual log lines or structured log entries.
	Entries []string `json:"entries" yaml:"entries"`
}

// ConfigData holds configuration key-value pairs retrieved from a middleware instance.
type ConfigData struct {
	// Data holds the configuration parameters, where the key is the parameter name and the value is its setting.
	Data map[string]string `json:"data" yaml:"data"`
}

// AIAnalysisResult contains the structured output from an analysis step that uses
// a Large Language Model (LLM).
type AIAnalysisResult struct {
	// Summary is the high-level summary of the AI's findings.
	Summary string `json:"summary" yaml:"summary"`
	// Reasoning provides the chain-of-thought or rationale behind the AI's conclusions.
	Reasoning string `json:"reasoning,omitempty" yaml:"reasoning,omitempty"`
	// Issues is a slice of issues identified by the AI.
	Issues []*Issue `json:"issues" yaml:"issues"`
}

// ComponentDiagnosisResult represents the result of a plugin's diagnosis for a specific component.
// It is used to aggregate results from multiple plugins.
type ComponentDiagnosisResult struct {
	Component string                 `json:"component"`
	Status    string                 `json:"status"`
	Issues    []*Issue               `json:"issues"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

// --- Supporting Structs for Interfaces ---

// CollectedData is an umbrella container for all the different types of data
// gathered by a plugin during the data collection phase. It is then passed to the
// analysis engine.
type CollectedData struct {
	// Metrics holds the collected performance metrics.
	Metrics *MetricsData
	// Logs holds the collected log entries.
	Logs *LogData
	// Config holds the collected configuration data.
	Config *ConfigData
}

// SystemCorrelationData is a generic container for passing multiple, varied data
// sources to an analyzer for cross-system correlation.
type SystemCorrelationData struct {
	// DataSources maps a data source name (e.g., "metrics", "logs") to the collected data.
	DataSources map[string]interface{} `json:"dataSources" yaml:"dataSources"`
}

// LogOptions provides parameters to control the log collection process, such as
// defining the time range or number of lines to retrieve.
type LogOptions struct {
	// Since specifies the relative duration from the present to start collecting logs (e.g., "5m", "1h").
	Since time.Duration `json:"since,omitempty" yaml:"since,omitempty"`
	// Tail specifies the number of recent log lines to retrieve from the end of the log.
	Tail int `json:"tail,omitempty" yaml:"tail,omitempty"`
	// Follow indicates whether to stream logs in real-time (not typically used in standard diagnosis).
	Follow bool `json:"follow,omitempty" yaml:"follow,omitempty"`
}

// HealthStatus represents the simple health status of a component, typically
// returned from a ping or a basic health check.
type HealthStatus struct {
	// IsHealthy is true if the component is healthy, false otherwise.
	IsHealthy bool `json:"isHealthy" yaml:"isHealthy"`
	// Message provides additional details about the health status, especially in case of failure.
	Message string `json:"message" yaml:"message"`
}

// FixAction represents a specific, concrete fix to be executed. It is often
// derived from a Recommendation but is a more structured object for the execution engine.
type FixAction struct {
	// ID is the unique identifier for this fix action.
	ID string `json:"id" yaml:"id"`
	// Description is a human-readable explanation of what the fix does.
	Description string `json:"description" yaml:"description"`
	// Command is the shell command to be executed.
	Command string `json:"command" yaml:"command"`
	// Parameters provides any additional parameters needed to execute the fix.
	Parameters map[string]string `json:"parameters" yaml:"parameters"`
	// Category helps classify the action type for dependency analysis (e.g., "ConfigChange", "Restart", "Validation").
	Category string `json:"category,omitempty" yaml:"category,omitempty"`
	// RollbackCommand is the command to revert the action if it fails.
	RollbackCommand string `json:"rollback_command,omitempty"`
	// ValidationCommand is a command to verify the action was successful.
	ValidationCommand string `json:"validation_command,omitempty"`
}

// FixResult represents the outcome of a single executed fix action.
type FixResult struct {
	// Success is true if the fix was applied successfully, false otherwise.
	Success bool `json:"success" yaml:"success"`
	// Message provides details about the outcome, such as stdout or an error message.
	Message string `json:"message" yaml:"message"`
}
