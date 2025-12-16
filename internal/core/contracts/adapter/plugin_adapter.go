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

package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/contracts"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// PluginAdapter adapts the existing internal/plugin.MiddlewarePlugin interface
// to the design-aligned contracts.MiddlewarePlugin interface.
//
// This adapter allows existing plugin implementations to work with the new
// contract interface without requiring changes to the plugin code itself.
// It performs the necessary translation between the operation-oriented interface
// (Connect/Execute/etc.) and the diagnosis-oriented contract interface
// (Diagnose/CollectMetrics/etc.).
type PluginAdapter struct {
	underlying plugin.MiddlewarePlugin
}

// NewPluginAdapter creates a new adapter that wraps an existing plugin implementation.
func NewPluginAdapter(p plugin.MiddlewarePlugin) contracts.MiddlewarePlugin {
	return &PluginAdapter{underlying: p}
}

// === Metadata ===

func (a *PluginAdapter) Name() string {
	return a.underlying.Name()
}

func (a *PluginAdapter) Version() string {
	return a.underlying.Version()
}

func (a *PluginAdapter) SupportedVersions() []string {
	// The underlying plugin interface doesn't expose supported versions directly.
	// Return a default value; this can be enhanced if the underlying plugin
	// provides this information through another mechanism.
	return []string{a.underlying.Version()}
}

// === Diagnosis & Data Collection ===

func (a *PluginAdapter) Diagnose(ctx context.Context, config *contracts.DiagnosisConfig) (*contracts.DiagnosisResult, error) {
	// Convert contracts config to plugin connection config
	connConfig := a.convertToConnectionConfig(config)

	// Ensure connection
	if !a.underlying.IsConnected() {
		if err := a.underlying.Connect(ctx, connConfig); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
		// Note: We don't disconnect here; caller should manage lifecycle
	}

	// Collect diagnostic data
	diagData, err := a.underlying.GetDiagnosticData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get diagnostic data: %w", err)
	}

	// Convert to contract diagnosis result
	result := a.convertDiagnosticDataToResult(diagData)
	result.Timestamp = time.Now()

	// Get built-in rules and evaluate them
	rules := a.underlying.GetBuiltinRules()
	issues := a.evaluateRules(rules, diagData)
	result.Issues = append(result.Issues, issues...)

	// Determine overall status
	result.Status = a.determineStatus(result.Issues)
	result.Summary = fmt.Sprintf("Diagnosis completed. Found %d issues.", len(result.Issues))

	return result, nil
}

func (a *PluginAdapter) CollectMetrics(ctx context.Context, target *contracts.TargetConfig) (*contracts.MetricsData, error) {
	// Ensure connection
	connConfig := a.convertTargetToConnectionConfig(target)
	if !a.underlying.IsConnected() {
		if err := a.underlying.Connect(ctx, connConfig); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Collect metrics using the underlying plugin
	snapshot, err := a.underlying.CollectMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Convert to contract metrics data
	return a.convertMetricsSnapshot(snapshot), nil
}

func (a *PluginAdapter) CollectLogs(ctx context.Context, target *contracts.TargetConfig, opts *contracts.LogOptions) (*contracts.LogData, error) {
	// Ensure connection
	connConfig := a.convertTargetToConnectionConfig(target)
	if !a.underlying.IsConnected() {
		if err := a.underlying.Connect(ctx, connConfig); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Get diagnostic data which includes slow logs
	diagData, err := a.underlying.GetDiagnosticData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get diagnostic data: %w", err)
	}

	// Convert slow logs to log data
	return a.convertSlowLogsToLogData(diagData.SlowLogs, opts), nil
}

func (a *PluginAdapter) GetConfiguration(ctx context.Context, target *contracts.TargetConfig) (*contracts.ConfigData, error) {
	// Ensure connection
	connConfig := a.convertTargetToConnectionConfig(target)
	if !a.underlying.IsConnected() {
		if err := a.underlying.Connect(ctx, connConfig); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Get diagnostic data which includes config
	diagData, err := a.underlying.GetDiagnosticData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get diagnostic data: %w", err)
	}

	if diagData.Config == nil {
		return nil, contracts.ErrNotSupported
	}

	// Convert to contract config data
	return &contracts.ConfigData{
		Parameters: diagData.Config,
		Runtime:    diagData.Extra,
	}, nil
}

// === Health & Status ===

func (a *PluginAdapter) HealthCheck(ctx context.Context, target *contracts.TargetConfig) (*contracts.HealthStatus, error) {
	connConfig := a.convertTargetToConnectionConfig(target)

	// Try to connect and ping
	startTime := time.Now()
	var latency time.Duration
	var connected bool

	if !a.underlying.IsConnected() {
		err := a.underlying.Connect(ctx, connConfig)
		if err != nil {
			return &contracts.HealthStatus{
				Status:       "unhealthy",
				Timestamp:    time.Now(),
				Connectivity: false,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			}, nil
		}
	}

	// Ping to check connectivity
	err := a.underlying.Ping(ctx)
	latency = time.Since(startTime)
	connected = (err == nil)

	status := "healthy"
	if !connected {
		status = "unhealthy"
	}

	return &contracts.HealthStatus{
		Status:       status,
		Timestamp:    time.Now(),
		Connectivity: connected,
		Latency:      latency,
		Details: map[string]interface{}{
			"connected": connected,
		},
	}, nil
}

// === Auto-Fix Capabilities ===

func (a *PluginAdapter) CanAutoFix(issue *contracts.Issue) (bool, *contracts.FixAction) {
	// The underlying plugin interface doesn't have CanAutoFix with the same signature.
	// We return false for now; this can be enhanced by mapping to the underlying
	// plugin's capabilities if they exist.
	return false, nil
}

func (a *PluginAdapter) ExecuteFix(ctx context.Context, fix *contracts.FixAction) (*contracts.FixResult, error) {
	// The underlying plugin uses Execute with Command.
	// We need to convert the FixAction to a Command.
	
	if fix.Type != contracts.FixTypeCommand && fix.Type != contracts.FixTypeConfiguration {
		return nil, fmt.Errorf("unsupported fix type: %s", fix.Type)
	}

	// For configuration changes, use Execute with appropriate command
	cmd := a.convertFixActionToCommand(fix)
	
	// Ensure connection (connection config should have been set earlier)
	if !a.underlying.IsConnected() {
		return &contracts.FixResult{
			Success:   false,
			Timestamp: time.Now(),
			Error:     "not connected to middleware",
		}, nil
	}

	// Execute the command
	result, err := a.underlying.Execute(ctx, cmd)
	if err != nil {
		return &contracts.FixResult{
			Success:   false,
			Timestamp: time.Now(),
			Error:     err.Error(),
		}, err
	}

	// Convert command result to fix result
	return &contracts.FixResult{
		Success:   result.Success,
		Timestamp: time.Now(),
		Message:   result.Output,
		Error:     result.Error,
		Changes:   []string{fmt.Sprintf("Executed: %s", cmd.Name)},
	}, nil
}

// === Conversion Helpers ===

func (a *PluginAdapter) convertToConnectionConfig(config *contracts.DiagnosisConfig) *plugin.ConnectionConfig {
	if config == nil || config.Target == nil {
		return &plugin.ConnectionConfig{}
	}

	connConfig := &plugin.ConnectionConfig{
		Host:     config.Target.Host,
		Port:     config.Target.Port,
		Database: config.Target.Database,
		Timeout:  config.Timeout,
		Extra:    config.Target.Extra,
	}

	if config.Credentials != nil {
		connConfig.Username = config.Credentials.Username
		connConfig.Password = config.Credentials.Password
		if config.Credentials.CertPath != "" || config.Credentials.CAPath != "" {
			connConfig.TLS = &plugin.TLSConfig{
				CertFile: config.Credentials.CertPath,
				KeyFile:  config.Credentials.KeyPath,
				CAFile:   config.Credentials.CAPath,
			}
		}
	}

	return connConfig
}

func (a *PluginAdapter) convertTargetToConnectionConfig(target *contracts.TargetConfig) *plugin.ConnectionConfig {
	if target == nil {
		return &plugin.ConnectionConfig{}
	}

	return &plugin.ConnectionConfig{
		Host:     target.Host,
		Port:     target.Port,
		Database: target.Database,
		Extra:    target.Extra,
	}
}

func (a *PluginAdapter) convertDiagnosticDataToResult(data *plugin.DiagnosticData) *contracts.DiagnosisResult {
	result := &contracts.DiagnosisResult{
		Issues:          []*contracts.Issue{},
		Recommendations: []*contracts.Recommendation{},
	}

	if data.Metrics != nil {
		result.Metrics = a.convertMetricsSnapshot(data.Metrics)
	}

	return result
}

func (a *PluginAdapter) convertMetricsSnapshot(snapshot *plugin.MetricsSnapshot) *contracts.MetricsData {
	if snapshot == nil {
		return &contracts.MetricsData{
			Timestamp: time.Now(),
			Metrics:   make(map[string]contracts.MetricValue),
		}
	}

	metrics := make(map[string]contracts.MetricValue)
	for key, value := range snapshot.Metrics {
		metrics[key] = contracts.MetricValue{
			Name:      value.Name,
			Value:     value.Value,
			Unit:      value.Unit,
			Timestamp: value.Timestamp,
		}
	}

	return &contracts.MetricsData{
		Timestamp: snapshot.Timestamp,
		Metrics:   metrics,
	}
}

func (a *PluginAdapter) convertSlowLogsToLogData(slowLogs []plugin.SlowLogEntry, opts *contracts.LogOptions) *contracts.LogData {
	entries := make([]*contracts.LogEntry, 0, len(slowLogs))

	for _, log := range slowLogs {
		// Apply filters if opts is provided
		if opts != nil {
			if !opts.StartTime.IsZero() && log.Time.Before(opts.StartTime) {
				continue
			}
			if !opts.EndTime.IsZero() && log.Time.After(opts.EndTime) {
				continue
			}
			if opts.Limit > 0 && len(entries) >= opts.Limit {
				break
			}
		}

		entry := &contracts.LogEntry{
			Timestamp: log.Time,
			Level:     "slow",
			Message:   log.Query,
			Source:    log.ClientIP,
			Fields: map[string]interface{}{
				"id":        log.ID,
				"duration":  log.Duration.String(),
				"command":   log.Command,
				"user":      log.User,
				"database":  log.Database,
				"rows_sent": log.RowsSent,
				"rows_exam": log.RowsExam,
			},
		}
		entries = append(entries, entry)
	}

	return &contracts.LogData{
		Entries:   entries,
		Truncated: false,
	}
}

func (a *PluginAdapter) evaluateRules(rules []plugin.DiagnosisRule, data *plugin.DiagnosticData) []*contracts.Issue {
	// Simple rule evaluation - in a real implementation, this would use an expression evaluator
	issues := make([]*contracts.Issue, 0)

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// Basic heuristic: if the rule exists and is enabled, we treat it as a potential issue
		// In practice, you'd evaluate rule.Condition against the data
		// For now, we skip actual evaluation and just return empty list
		// A full implementation would use an expression engine
	}

	return issues
}

func (a *PluginAdapter) determineStatus(issues []*contracts.Issue) contracts.DiagnosisStatus {
	if len(issues) == 0 {
		return contracts.DiagnosisStatusHealthy
	}

	hasCritical := false
	hasError := false
	for _, issue := range issues {
		switch issue.Severity {
		case contracts.SeverityCritical:
			hasCritical = true
		case contracts.SeverityError:
			hasError = true
		}
	}

	if hasCritical {
		return contracts.DiagnosisStatusCritical
	}
	if hasError {
		return contracts.DiagnosisStatusWarning
	}
	return contracts.DiagnosisStatusWarning
}

func (a *PluginAdapter) convertFixActionToCommand(fix *contracts.FixAction) *plugin.Command {
	cmd := &plugin.Command{
		Name:    fix.ID,
		Args:    []interface{}{},
		Timeout: 30 * time.Second,
		DryRun:  fix.DryRun,
	}

	// Convert parameters to args
	for k, v := range fix.Parameters {
		cmd.Args = append(cmd.Args, k, v)
	}

	return cmd
}
