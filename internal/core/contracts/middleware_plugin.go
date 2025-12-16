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

package contracts

import (
	"context"
	"errors"
	"time"
)

// ErrNotSupported indicates a plugin does not support a specific capability.
var ErrNotSupported = errors.New("capability not supported by this plugin")

// MiddlewarePlugin defines the design-aligned contract interface for all middleware plugins.
// This interface represents the canonical contract from the architecture design documentation,
// providing a stable API surface for diagnosis orchestration, data collection, and auto-fixing.
//
// Implementation Note: Existing plugins (internal/plugin.MiddlewarePlugin) use a different
// interface focused on operations (Connect/Execute/etc). The adapter layer in
// internal/core/contracts/adapter bridges between the two interfaces without requiring
// existing plugin implementations to change.
type MiddlewarePlugin interface {
	// === Metadata ===

	// Name returns the plugin name (e.g., "Redis Plugin", "MySQL Plugin")
	Name() string

	// Version returns the plugin version (e.g., "1.0.0")
	Version() string

	// SupportedVersions returns the list of middleware versions this plugin supports
	// (e.g., ["6.0", "7.0"] for Redis)
	SupportedVersions() []string

	// === Diagnosis & Data Collection ===

	// Diagnose performs a comprehensive diagnosis and returns structured results
	// containing identified issues, metrics snapshot, and recommendations.
	// The config parameter provides connection and context information.
	Diagnose(ctx context.Context, config *DiagnosisConfig) (*DiagnosisResult, error)

	// CollectMetrics collects current metrics snapshot from the middleware instance.
	// Returns structured metrics data including performance counters, resource usage, etc.
	CollectMetrics(ctx context.Context, target *TargetConfig) (*MetricsData, error)

	// CollectLogs retrieves logs from the middleware instance based on provided options.
	// Supports filtering by time range, severity, and other criteria.
	CollectLogs(ctx context.Context, target *TargetConfig, opts *LogOptions) (*LogData, error)

	// GetConfiguration retrieves the current configuration of the middleware instance.
	// Returns structured configuration data including runtime settings and parameters.
	GetConfiguration(ctx context.Context, target *TargetConfig) (*ConfigData, error)

	// === Health & Status ===

	// HealthCheck performs a comprehensive health check and returns detailed status.
	// Includes connectivity, resource availability, and operational state.
	HealthCheck(ctx context.Context, target *TargetConfig) (*HealthStatus, error)

	// === Auto-Fix Capabilities ===

	// CanAutoFix determines if the plugin can automatically fix the given issue.
	// Returns true and a FixAction if auto-fix is supported, false otherwise.
	CanAutoFix(issue *Issue) (bool, *FixAction)

	// ExecuteFix executes the specified fix action on the middleware instance.
	// Returns the result of the fix operation including success status and details.
	ExecuteFix(ctx context.Context, fix *FixAction) (*FixResult, error)
}

// === Configuration Types ===

// DiagnosisConfig provides context and configuration for a diagnosis operation.
type DiagnosisConfig struct {
	Target      *TargetConfig
	Options     map[string]interface{}
	Timeout     time.Duration
	Credentials *Credentials
}

// TargetConfig identifies the middleware instance to operate on.
type TargetConfig struct {
	Host     string
	Port     int
	Database string // For databases
	Extra    map[string]string
}

// Credentials contains authentication information.
type Credentials struct {
	Username string
	Password string
	Token    string
	CertPath string
	KeyPath  string
	CAPath   string
}

// === Data Types ===

// DiagnosisResult contains the complete results of a diagnosis operation.
type DiagnosisResult struct {
	Timestamp     time.Time
	Issues        []*Issue
	Metrics       *MetricsData
	Summary       string
	Status        DiagnosisStatus
	Recommendations []*Recommendation
}

// DiagnosisStatus represents the overall diagnosis status.
type DiagnosisStatus string

const (
	DiagnosisStatusHealthy  DiagnosisStatus = "healthy"
	DiagnosisStatusWarning  DiagnosisStatus = "warning"
	DiagnosisStatusCritical DiagnosisStatus = "critical"
	DiagnosisStatusUnknown  DiagnosisStatus = "unknown"
)

// Issue represents a detected problem or anomaly.
type Issue struct {
	ID          string
	Title       string
	Description string
	Severity    Severity
	Source      string
	Timestamp   time.Time
	Metrics     map[string]interface{}
	Context     map[string]interface{}
}

// Severity defines issue severity levels.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Recommendation provides actionable suggestions for addressing issues.
type Recommendation struct {
	Description string
	Priority    int
	Category    string
	Fix         *FixAction
}

// MetricsData contains collected metrics from the middleware.
type MetricsData struct {
	Timestamp time.Time
	Metrics   map[string]MetricValue
	Labels    map[string]string
}

// MetricValue represents a single metric measurement.
type MetricValue struct {
	Name      string
	Value     float64
	Unit      string
	Timestamp time.Time
}

// LogOptions specifies criteria for log collection.
type LogOptions struct {
	StartTime time.Time
	EndTime   time.Time
	Level     string
	Limit     int
	Pattern   string
}

// LogData contains collected log entries.
type LogData struct {
	Entries   []*LogEntry
	Truncated bool
	NextToken string
}

// LogEntry represents a single log entry.
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Source    string
	Fields    map[string]interface{}
}

// ConfigData contains middleware configuration data.
type ConfigData struct {
	Parameters map[string]interface{}
	Runtime    map[string]interface{}
	Static     map[string]interface{}
}

// HealthStatus represents the health status of a middleware instance.
type HealthStatus struct {
	Status       string
	Timestamp    time.Time
	Connectivity bool
	Latency      time.Duration
	Details      map[string]interface{}
}

// === Fix Types ===

// FixAction describes an automated fix operation.
type FixAction struct {
	ID          string
	Type        FixType
	Description string
	Parameters  map[string]interface{}
	RiskLevel   RiskLevel
	Reversible  bool
	DryRun      bool
}

// FixType categorizes the type of fix action.
type FixType string

const (
	FixTypeConfiguration FixType = "configuration"
	FixTypeCommand       FixType = "command"
	FixTypeScript        FixType = "script"
	FixTypeRestart       FixType = "restart"
)

// RiskLevel indicates the risk associated with a fix action.
type RiskLevel int

const (
	RiskLevelLow      RiskLevel = 1
	RiskLevelMedium   RiskLevel = 2
	RiskLevelHigh     RiskLevel = 3
	RiskLevelCritical RiskLevel = 4
)

// FixResult contains the outcome of a fix operation.
type FixResult struct {
	Success   bool
	Timestamp time.Time
	Message   string
	Error     string
	Changes   []string
	Rollback  *RollbackInfo
}

// RollbackInfo contains information needed to rollback a fix.
type RollbackInfo struct {
	Available   bool
	Action      *FixAction
	Description string
}
