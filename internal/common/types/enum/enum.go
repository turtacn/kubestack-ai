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

// MiddlewareType defines the type of middleware.
type MiddlewareType int

const (
	MySQL MiddlewareType = iota
	Redis
	Kafka
	Elasticsearch
	PostgreSQL
	MongoDB
	RabbitMQ
	MinIO
	Prometheus
	ClickHouse
)

var middlewareTypeStrings = [...]string{
	"MySQL", "Redis", "Kafka", "Elasticsearch", "PostgreSQL",
	"MongoDB", "RabbitMQ", "MinIO", "Prometheus", "ClickHouse",
}

// String returns the string representation of MiddlewareType.
func (m MiddlewareType) String() string {
	if m < MySQL || m > ClickHouse {
		return "Unknown"
	}
	return middlewareTypeStrings[m]
}

// IsValid checks if the MiddlewareType is valid.
func (m MiddlewareType) IsValid() bool {
	return m >= MySQL && m <= ClickHouse
}

// DiagnosisStatus defines the status of a diagnosis.
type DiagnosisStatus int

const (
	StatusHealthy DiagnosisStatus = iota
	StatusWarning
	StatusCritical
	StatusUnknown
)

var diagnosisStatusStrings = [...]string{"Healthy", "Warning", "Critical", "Unknown"}

// String returns the string representation of DiagnosisStatus.
func (d DiagnosisStatus) String() string {
	if d < StatusHealthy || d > StatusUnknown {
		return "Unknown"
	}
	return diagnosisStatusStrings[d]
}

// IsValid checks if the DiagnosisStatus is valid.
func (d DiagnosisStatus) IsValid() bool {
	return d >= StatusHealthy && d <= StatusUnknown
}

// SeverityLevel defines the severity level of an issue.
type SeverityLevel int

const (
	SeverityLow SeverityLevel = iota
	SeverityMedium
	SeverityHigh
	SeverityWarning
	SeverityCritical
)

var severityLevelStrings = [...]string{"Low", "Medium", "High", "Warning", "Critical"}

// String returns the string representation of SeverityLevel.
func (s SeverityLevel) String() string {
	if s < SeverityLow || s > SeverityCritical {
		return "Unknown"
	}
	return severityLevelStrings[s]
}

// IsValid checks if the SeverityLevel is valid.
func (s SeverityLevel) IsValid() bool {
	return s >= SeverityLow && s <= SeverityCritical
}

// ActionType defines the type of an execution action.
type ActionType int

const (
	ActionCollect ActionType = iota
	ActionAnalyze
	ActionFix
	ActionMonitor
)

var actionTypeStrings = [...]string{"Collect", "Analyze", "Fix", "Monitor"}

// String returns the string representation of ActionType.
func (a ActionType) String() string {
	if a < ActionCollect || a > ActionMonitor {
		return "Unknown"
	}
	return actionTypeStrings[a]
}

// IsValid checks if the ActionType is valid.
func (a ActionType) IsValid() bool {
	return a >= ActionCollect && a <= ActionMonitor
}

// PluginStatus defines the status of a plugin.
type PluginStatus int

const (
	PluginInstalled PluginStatus = iota
	PluginLoading
	PluginActive
	PluginFailed
	PluginUninstalled
)

var pluginStatusStrings = [...]string{"Installed", "Loading", "Active", "Failed", "Uninstalled"}

// String returns the string representation of PluginStatus.
func (p PluginStatus) String() string {
	if p < PluginInstalled || p > PluginUninstalled {
		return "Unknown"
	}
	return pluginStatusStrings[p]
}

// IsValid checks if the PluginStatus is valid.
func (p PluginStatus) IsValid() bool {
	return p >= PluginInstalled && p <= PluginUninstalled
}

// EnvironmentType defines the deployment environment type.
type EnvironmentType int

const (
	EnvKubernetes EnvironmentType = iota
	EnvDocker
	EnvSystemd
	EnvBare
)

var environmentTypeStrings = [...]string{"Kubernetes", "Docker", "Systemd", "Bare"}

// String returns the string representation of EnvironmentType.
func (e EnvironmentType) String() string {
	if e < EnvKubernetes || e > EnvBare {
		return "Unknown"
	}
	return environmentTypeStrings[e]
}

// IsValid checks if the EnvironmentType is valid.
func (e EnvironmentType) IsValid() bool {
	return e >= EnvKubernetes && e <= EnvBare
}

// LogLevel defines the logging level.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

var logLevelStrings = [...]string{"Debug", "Info", "Warn", "Error", "Fatal"}

// String returns the string representation of LogLevel.
func (l LogLevel) String() string {
	if l < LogLevelDebug || l > LogLevelFatal {
		return "Unknown"
	}
	return logLevelStrings[l]
}

// IsValid checks if the LogLevel is valid.
func (l LogLevel) IsValid() bool {
	return l >= LogLevelDebug && l <= LogLevelFatal
}

//Personal.AI order the ending
