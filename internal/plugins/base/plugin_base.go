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

// Package base provides base implementations for plugins and collectors to reduce boilerplate.
package base

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// Plugin provides a skeletal implementation of the `interfaces.MiddlewarePlugin`
// interface. It is designed to be embedded in concrete plugin implementations.
type Plugin struct {
	// Log is a contextualized logger specific to the plugin instance.
	Log logger.Logger
	// PluginName is the name of the plugin.
	PluginName string
	// PluginVersion is the semantic version of the plugin.
	PluginVersion string
	// PluginDesc is a short description of the plugin.
	PluginDesc string
}

// Init initializes the base plugin with essential information and creates a
// contextual logger for it.
func (p *Plugin) Init(name, version, description string) {
	p.PluginName = name
	p.PluginVersion = version
	p.PluginDesc = description
	p.Log = logger.NewLogger(fmt.Sprintf("plugin-%s", name))
}

// Init from config - compatibility with interface
func (p *Plugin) InitWithConfig(config *config.PluginConfig) error {
	// In a real scenario, we might load more config here.
	return nil
}

// --- Basic Information ---

// Name returns the name of the plugin.
func (p *Plugin) Name() string { return p.PluginName }

// Version returns the version of the plugin.
func (p *Plugin) Version() string { return p.PluginVersion }

// Description returns the description of the plugin.
func (p *Plugin) Description() string { return p.PluginDesc }

// SupportedVersions provides a default implementation that returns an empty list.
func (p *Plugin) SupportedVersions() []string {
	p.Log.Warn("SupportedVersions() is not implemented, returning empty list.")
	return []string{}
}

// SupportedTypes provides a default implementation that returns an empty list.
func (p *Plugin) SupportedTypes() []enum.MiddlewareType {
	p.Log.Warn("SupportedTypes() is not implemented, returning empty list.")
	return []enum.MiddlewareType{}
}

// --- Core Functions (Must be overridden by embedding plugins) ---

// Diagnose provides a default implementation that returns a "not implemented" error.
func (p *Plugin) Diagnose(_ context.Context, _ *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	return nil, fmt.Errorf("method 'Diagnose' not implemented for plugin %s", p.Name())
}

// CollectMetrics provides a default implementation that returns a "not implemented" error.
func (p *Plugin) CollectMetrics(_ context.Context, target string) (*models.MetricsData, error) {
	return nil, fmt.Errorf("method 'CollectMetrics' not implemented for plugin %s", p.Name())
}

// CollectLogs provides a default implementation that returns a "not implemented" error.
func (p *Plugin) CollectLogs(_ context.Context, target string, _ *models.LogOptions) (*models.LogData, error) {
	return nil, fmt.Errorf("method 'CollectLogs' not implemented for plugin %s", p.Name())
}

// CollectConfig provides a default implementation that returns a "not implemented" error.
func (p *Plugin) CollectConfig(_ context.Context, target string) (*models.ConfigData, error) {
	return nil, fmt.Errorf("method 'CollectConfig' not implemented for plugin %s", p.Name())
}

// --- Health Checks ---

// Ping provides a generic, minimal health check that simply indicates the plugin is loaded.
func (p *Plugin) Ping(_ context.Context, target string) error {
	p.Log.Debug("Ping received.")
	return nil
}

// HealthCheck provides a default implementation that indicates the base plugin is active.
func (p *Plugin) HealthCheck(_ context.Context, target string) (*models.HealthStatus, error) {
	p.Log.Debug("Performing base health check.")
	return &models.HealthStatus{IsHealthy: true, Message: "Base plugin is active and loaded."}, nil
}

// --- Fix Operations (Must be overridden by embedding plugins) ---

// CanAutoFix provides a default implementation that always returns false for safety.
func (p *Plugin) CanAutoFix(_ *models.Issue) (bool, *models.FixAction) {
	return false, nil
}

// ExecuteFix provides a default implementation that returns a "not implemented" error.
func (p *Plugin) ExecuteFix(_ context.Context, _ *models.FixAction) (*models.FixResult, error) {
	return nil, fmt.Errorf("method 'ExecuteFix' not implemented for plugin %s", p.Name())
}

// ValidateFix provides a default implementation that returns a "not implemented" error.
func (p *Plugin) ValidateFix(_ context.Context, _ *models.Issue, _ *models.FixResult) (bool, string, error) {
	return false, "", fmt.Errorf("method 'ValidateFix' not implemented for plugin %s", p.Name())
}

// --- Lifecycle Hooks ---

// Shutdown provides a hook for graceful resource cleanup.
func (p *Plugin) Shutdown() error {
	p.Log.Info("Shutting down base plugin.")
	// Placeholder for cleanup logic.
	return nil
}
