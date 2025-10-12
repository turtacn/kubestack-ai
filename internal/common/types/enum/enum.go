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

// Package enum defines all the enumeration types used in the KubeStack-AI project.
// This provides a centralized and consistent way to manage predefined sets of values.
package enum

// MiddlewareType defines the supported types of middleware that KubeStack-AI can diagnose and manage.
type MiddlewareType int

const (
	// MySQL represents the MySQL relational database.
	MySQL MiddlewareType = iota
	// Redis represents the Redis in-memory data store.
	Redis
	// Kafka represents the Apache Kafka distributed event streaming platform.
	Kafka
	// Elasticsearch represents the Elasticsearch search and analytics engine.
	Elasticsearch
	// PostgreSQL represents the PostgreSQL object-relational database.
	PostgreSQL
	// MongoDB represents the MongoDB document-oriented database.
	MongoDB
	// RabbitMQ represents the RabbitMQ message broker.
	RabbitMQ
	// MinIO represents the MinIO high-performance object storage.
	MinIO
	// Prometheus represents the Prometheus monitoring and alerting toolkit.
	Prometheus
	// ClickHouse represents the ClickHouse column-oriented database management system.
	ClickHouse
)

var middlewareTypeStrings = [...]string{
	"MySQL", "Redis", "Kafka", "Elasticsearch", "PostgreSQL",
	"MongoDB", "RabbitMQ", "MinIO", "Prometheus", "ClickHouse",
}

// String returns the string representation of the MiddlewareType.
// It satisfies the fmt.Stringer interface.
//
// Returns:
//   string: The string name of the middleware type (e.g., "Redis"). Returns "Unknown" for invalid values.
func (m MiddlewareType) String() string {
	if m < MySQL || m > ClickHouse {
		return "Unknown"
	}
	return middlewareTypeStrings[m]
}

// IsValid checks if the MiddlewareType is a defined and supported value.
//
// Returns:
//   bool: True if the middleware type is valid, false otherwise.
func (m MiddlewareType) IsValid() bool {
	return m >= MySQL && m <= ClickHouse
}

// DiagnosisStatus defines the overall outcome of a diagnostic run.
type DiagnosisStatus int

const (
	// StatusHealthy indicates that no issues were found.
	StatusHealthy DiagnosisStatus = iota
	// StatusWarning indicates that non-critical issues or potential problems were detected.
	StatusWarning
	// StatusCritical indicates that one or more critical issues were found that require immediate attention.
	StatusCritical
	// StatusUnknown indicates that the diagnosis could not be completed or the state is indeterminate.
	StatusUnknown
)

var diagnosisStatusStrings = [...]string{"Healthy", "Warning", "Critical", "Unknown"}

// String returns the string representation of the DiagnosisStatus.
// It satisfies the fmt.Stringer interface.
//
// Returns:
//   string: The string name of the status (e.g., "Critical"). Returns "Unknown" for invalid values.
func (d DiagnosisStatus) String() string {
	if d < StatusHealthy || d > StatusUnknown {
		return "Unknown"
	}
	return diagnosisStatusStrings[d]
}

// IsValid checks if the DiagnosisStatus is a defined and supported value.
//
// Returns:
//   bool: True if the status is valid, false otherwise.
func (d DiagnosisStatus) IsValid() bool {
	return d >= StatusHealthy && d <= StatusUnknown
}

// SeverityLevel defines the severity of a detected issue, helping to prioritize responses.
type SeverityLevel int

const (
	// SeverityLow indicates a minor issue or a suggestion for best practices.
	SeverityLow SeverityLevel = iota
	// SeverityMedium indicates a moderate issue that should be addressed but is not urgent.
	SeverityMedium
	// SeverityHigh indicates a significant issue that could impact performance or stability.
	SeverityHigh
	// SeverityWarning is an alias or equivalent to High, for issues that are not critical but require attention.
	SeverityWarning
	// SeverityCritical indicates a critical issue that requires immediate attention to prevent system failure or data loss.
	SeverityCritical
)

var severityLevelStrings = [...]string{"Low", "Medium", "High", "Warning", "Critical"}

// String returns the string representation of the SeverityLevel.
// It satisfies the fmt.Stringer interface.
//
// Returns:
//   string: The string name of the severity level (e.g., "Critical"). Returns "Unknown" for invalid values.
func (s SeverityLevel) String() string {
	if s < SeverityLow || s > SeverityCritical {
		return "Unknown"
	}
	return severityLevelStrings[s]
}

// IsValid checks if the SeverityLevel is a defined and supported value.
//
// Returns:
//   bool: True if the severity level is valid, false otherwise.
func (s SeverityLevel) IsValid() bool {
	return s >= SeverityLow && s <= SeverityCritical
}

// ActionType defines the category of an action performed by the diagnosis or execution engine.
type ActionType int

const (
	// ActionCollect represents a data collection step (e.g., fetching metrics or logs).
	ActionCollect ActionType = iota
	// ActionAnalyze represents a data analysis step (e.g., running rules against collected data).
	ActionAnalyze
	// ActionFix represents a step that attempts to apply a fix to an issue.
	ActionFix
	// ActionMonitor represents a step that observes the system over a period of time.
	ActionMonitor
)

var actionTypeStrings = [...]string{"Collect", "Analyze", "Fix", "Monitor"}

// String returns the string representation of the ActionType.
// It satisfies the fmt.Stringer interface.
//
// Returns:
//   string: The string name of the action type (e.g., "Fix"). Returns "Unknown" for invalid values.
func (a ActionType) String() string {
	if a < ActionCollect || a > ActionMonitor {
		return "Unknown"
	}
	return actionTypeStrings[a]
}

// IsValid checks if the ActionType is a defined and supported value.
//
// Returns:
//   bool: True if the action type is valid, false otherwise.
func (a ActionType) IsValid() bool {
	return a >= ActionCollect && a <= ActionMonitor
}

// PluginStatus defines the lifecycle status of a dynamically loaded plugin.
type PluginStatus int

const (
	// PluginInstalled indicates the plugin files are present on disk but not yet loaded.
	PluginInstalled PluginStatus = iota
	// PluginLoading indicates the plugin is in the process of being loaded and initialized.
	PluginLoading
	// PluginActive indicates the plugin has been successfully loaded and is ready to use.
	PluginActive
	// PluginFailed indicates that an error occurred while loading or initializing the plugin.
	PluginFailed
	// PluginUninstalled indicates the plugin has been removed from the system.
	PluginUninstalled
)

var pluginStatusStrings = [...]string{"Installed", "Loading", "Active", "Failed", "Uninstalled"}

// String returns the string representation of the PluginStatus.
// It satisfies the fmt.Stringer interface.
//
// Returns:
//   string: The string name of the plugin status (e.g., "Active"). Returns "Unknown" for invalid values.
func (p PluginStatus) String() string {
	if p < PluginInstalled || p > PluginUninstalled {
		return "Unknown"
	}
	return pluginStatusStrings[p]
}

// IsValid checks if the PluginStatus is a defined and supported value.
//
// Returns:
//   bool: True if the plugin status is valid, false otherwise.
func (p PluginStatus) IsValid() bool {
	return p >= PluginInstalled && p <= PluginUninstalled
}

// EnvironmentType defines the type of environment where the target middleware is running.
type EnvironmentType int

const (
	// EnvKubernetes represents a Kubernetes cluster environment.
	EnvKubernetes EnvironmentType = iota
	// EnvDocker represents a containerized environment managed by Docker.
	EnvDocker
	// EnvSystemd represents a service running on a Linux system managed by systemd.
	EnvSystemd
	// EnvBare represents a service running directly on a bare-metal or virtual machine without a known management layer.
	EnvBare
)

var environmentTypeStrings = [...]string{"Kubernetes", "Docker", "Systemd", "Bare"}

// String returns the string representation of the EnvironmentType.
// It satisfies the fmt.Stringer interface.
//
// Returns:
//   string: The string name of the environment type (e.g., "Kubernetes"). Returns "Unknown" for invalid values.
func (e EnvironmentType) String() string {
	if e < EnvKubernetes || e > EnvBare {
		return "Unknown"
	}
	return environmentTypeStrings[e]
}

// IsValid checks if the EnvironmentType is a defined and supported value.
//
// Returns:
//   bool: True if the environment type is valid, false otherwise.
func (e EnvironmentType) IsValid() bool {
	return e >= EnvKubernetes && e <= EnvBare
}

// LogLevel defines the verbosity of logging output.
type LogLevel int

const (
	// LogLevelDebug enables detailed, verbose logging, useful for development and debugging.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo provides general informational messages about application state.
	LogLevelInfo
	// LogLevelWarn indicates potential issues or situations that might require attention.
	LogLevelWarn
	// LogLevelError signals that a recoverable error has occurred.
	LogLevelError
	// LogLevelFatal indicates a non-recoverable error that forces the application to exit.
	LogLevelFatal
)

var logLevelStrings = [...]string{"Debug", "Info", "Warn", "Error", "Fatal"}

// String returns the string representation of the LogLevel.
// It satisfies the fmt.Stringer interface.
//
// Returns:
//   string: The string name of the log level (e.g., "Debug"). Returns "Unknown" for invalid values.
func (l LogLevel) String() string {
	if l < LogLevelDebug || l > LogLevelFatal {
		return "Unknown"
	}
	return logLevelStrings[l]
}

// IsValid checks if the LogLevel is a defined and supported value.
//
// Returns:
//   bool: True if the log level is valid, false otherwise.
func (l LogLevel) IsValid() bool {
	return l >= LogLevelDebug && l <= LogLevelFatal
}

//Personal.AI order the ending
