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

// Plugin is a skeletal implementation of the interfaces.MiddlewarePlugin.
// It is designed to be embedded in concrete plugin implementations (e.g., Redis, MySQL).
// This provides default behavior for many methods, allowing plugin developers to only
// override the methods relevant to their specific middleware.
type Plugin struct {
	Log         logger.Logger
	PluginName  string
	PluginVersion string
	PluginDesc  string
}

// Init initializes the base plugin with essential information and a contextual logger.
func (p *Plugin) Init(name, version, description string) {
	p.PluginName = name
	p.PluginVersion = version
	p.PluginDesc = description
	p.Log = logger.NewLogger(fmt.Sprintf("plugin-%s", name))
}

// --- Basic Information ---

func (p *Plugin) Name() string    { return p.PluginName }
func (p *Plugin) Version() string { return p.PluginVersion }
func (p *Plugin) Description() string { return p.PluginDesc }

// SupportedVersions should be overridden by the concrete plugin to specify which
// versions of the middleware it supports.
func (p *Plugin) SupportedVersions() []string {
	p.Log.Warn("SupportedVersions() is not implemented, returning empty list.")
	return []string{}
}

// --- Core Functions (Must be overridden by embedding plugins) ---

func (p *Plugin) Diagnose(_ context.Context, _ *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	return nil, fmt.Errorf("method 'Diagnose' not implemented for plugin %s", p.Name())
}

func (p *Plugin) CollectMetrics(_ context.Context) (*models.MetricsData, error) {
	return nil, fmt.Errorf("method 'CollectMetrics' not implemented for plugin %s", p.Name())
}

func (p *Plugin) CollectLogs(_ context.Context, _ *models.LogOptions) (*models.LogData, error) {
	return nil, fmt.Errorf("method 'CollectLogs' not implemented for plugin %s", p.Name())
}

func (p *Plugin) GetConfiguration(_ context.Context) (*models.ConfigData, error) {
	return nil, fmt.Errorf("method 'GetConfiguration' not implemented for plugin %s", p.Name())
}

// --- Health Checks ---

// Ping provides a generic, basic health check that simply indicates the plugin is loaded.
func (p *Plugin) Ping(_ context.Context) error {
	p.Log.Debug("Ping received.")
	return nil
}

// HealthCheck should be overridden by concrete plugins to perform a more detailed check
// of the target middleware's health.
func (p *Plugin) HealthCheck(_ context.Context) (*models.HealthStatus, error) {
	p.Log.Debug("Performing base health check.")
	return &models.HealthStatus{IsHealthy: true, Message: "Base plugin is active and loaded."}, nil
}

// --- Fix Operations (Must be overridden by embedding plugins) ---

// CanAutoFix should be overridden to indicate if a specific issue is auto-fixable.
func (p *Plugin) CanAutoFix(_ *models.Issue) bool {
	return false // Default to no auto-fix for safety.
}

func (p *Plugin) ExecuteFix(_ context.Context, _ *models.FixAction) (*models.FixResult, error) {
	return nil, fmt.Errorf("method 'ExecuteFix' not implemented for plugin %s", p.Name())
}

func (p *Plugin) ValidateFix(_ context.Context, _ *models.FixAction) error {
	return fmt.Errorf("method 'ValidateFix' not implemented for plugin %s", p.Name())
}

// --- Lifecycle Hooks ---

// Shutdown provides a hook for graceful resource cleanup (e.g., closing database connections).
// This method is not part of the interface but can be called by the plugin manager before unloading.
func (p *Plugin) Shutdown() error {
	p.Log.Info("Shutting down base plugin.")
	// Placeholder for cleanup logic.
	return nil
}

//Personal.AI order the ending
