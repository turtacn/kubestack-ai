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

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// Plugin provides a skeletal implementation of the `interfaces.MiddlewarePlugin`
// interface. It is designed to be embedded in concrete plugin implementations
// (e.g., for Redis, MySQL). This provides default "not implemented" behavior for
// most methods, allowing plugin developers to only implement the methods relevant
// to their specific middleware, reducing boilerplate code.
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
// contextual logger for it. This method should be called by the constructor of
// the embedding plugin.
//
// Parameters:
//   name (string): The name of the plugin.
//   version (string): The version of the plugin.
//   description (string): A short description of the plugin.
func (p *Plugin) Init(name, version, description string) {
	p.PluginName = name
	p.PluginVersion = version
	p.PluginDesc = description
	p.Log = logger.NewLogger(fmt.Sprintf("plugin-%s", name))
}

// --- Basic Information ---

// Name returns the name of the plugin.
func (p *Plugin) Name() string { return p.PluginName }

// Version returns the version of the plugin.
func (p *Plugin) Version() string { return p.PluginVersion }

// Description returns the description of the plugin.
func (p *Plugin) Description() string { return p.PluginDesc }

// SupportedVersions provides a default implementation that returns an empty list.
// Concrete plugins should override this method to specify which versions of the
// target middleware they support.
func (p *Plugin) SupportedVersions() []string {
	p.Log.Warn("SupportedVersions() is not implemented, returning empty list.")
	return []string{}
}

// --- Core Functions (Must be overridden by embedding plugins) ---

// Diagnose provides a default implementation that returns a "not implemented" error.
// This method MUST be overridden by concrete plugins to provide actual diagnostic logic.
func (p *Plugin) Diagnose(_ context.Context, _ *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	return nil, fmt.Errorf("method 'Diagnose' not implemented for plugin %s", p.Name())
}

// CollectMetrics provides a default implementation that returns a "not implemented" error.
// This method MUST be overridden by concrete plugins to collect relevant metrics.
func (p *Plugin) CollectMetrics(_ context.Context) (*models.MetricsData, error) {
	return nil, fmt.Errorf("method 'CollectMetrics' not implemented for plugin %s", p.Name())
}

// CollectLogs provides a default implementation that returns a "not implemented" error.
// This method MUST be overridden by concrete plugins to collect relevant logs.
func (p *Plugin) CollectLogs(_ context.Context, _ *models.LogOptions) (*models.LogData, error) {
	return nil, fmt.Errorf("method 'CollectLogs' not implemented for plugin %s", p.Name())
}

// GetConfiguration provides a default implementation that returns a "not implemented" error.
// This method MUST be overridden by concrete plugins to retrieve middleware configuration.
func (p *Plugin) GetConfiguration(_ context.Context) (*models.ConfigData, error) {
	return nil, fmt.Errorf("method 'GetConfiguration' not implemented for plugin %s", p.Name())
}

// --- Health Checks ---

// Ping provides a generic, minimal health check that simply indicates the plugin is loaded.
// It can be used to verify that the plugin manager can communicate with the plugin.
func (p *Plugin) Ping(_ context.Context) error {
	p.Log.Debug("Ping received.")
	return nil
}

// HealthCheck provides a default implementation that indicates the base plugin is active.
// Concrete plugins should override this to perform a more detailed check of the
// target middleware's health (e.g., checking cluster status, connection pools).
func (p *Plugin) HealthCheck(_ context.Context) (*models.HealthStatus, error) {
	p.Log.Debug("Performing base health check.")
	return &models.HealthStatus{IsHealthy: true, Message: "Base plugin is active and loaded."}, nil
}

// --- Fix Operations (Must be overridden by embedding plugins) ---

// CanAutoFix provides a default implementation that always returns false for safety.
// Concrete plugins should override this to indicate if a specific, identified
// issue is safe to be fixed automatically by the execution engine.
func (p *Plugin) CanAutoFix(_ *models.Issue) bool {
	return false
}

// ExecuteFix provides a default implementation that returns a "not implemented" error.
// This method MUST be overridden by concrete plugins to provide logic for applying automated fixes.
func (p *Plugin) ExecuteFix(_ context.Context, _ *models.FixAction) (*models.FixResult, error) {
	return nil, fmt.Errorf("method 'ExecuteFix' not implemented for plugin %s", p.Name())
}

// ValidateFix provides a default implementation that returns a "not implemented" error.
// This method MUST be overridden by concrete plugins to provide logic for validating
// that a fix has successfully resolved an issue.
func (p *Plugin) ValidateFix(_ context.Context, _ *models.FixAction) error {
	return fmt.Errorf("method 'ValidateFix' not implemented for plugin %s", p.Name())
}

// --- Lifecycle Hooks ---

// Shutdown provides a hook for graceful resource cleanup (e.g., closing database
// connections). This method is not part of the MiddlewarePlugin interface but can be
// called by the plugin manager before unloading to ensure a clean shutdown.
func (p *Plugin) Shutdown() error {
	p.Log.Info("Shutting down base plugin.")
	// Placeholder for cleanup logic.
	return nil
}

//Personal.AI order the ending
