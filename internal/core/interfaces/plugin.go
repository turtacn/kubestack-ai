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

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// MiddlewarePlugin is alias to DiagnosticPlugin to maintain backward compatibility during refactor.
// Ideally we should move to single DiagnosticPlugin interface.
type MiddlewarePlugin = DiagnosticPlugin

// DiagnosticPlugin defines the comprehensive interface that every middleware-specific plugin
// must implement. It provides a standard contract for discovery, diagnostics, data
// collection, and automated fixing.
type DiagnosticPlugin interface {
	// Metadata
	Name() string
	Version() string
	Description() string
	SupportedTypes() []enum.MiddlewareType
	SupportedVersions() []string

	// Lifecycle
	Init(config *config.PluginConfig) error
	Shutdown() error

	// Diagnostics & Data Collection
	Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error)
	CollectMetrics(ctx context.Context, target string) (*models.MetricsData, error)
	CollectLogs(ctx context.Context, target string, opts *models.LogOptions) (*models.LogData, error)
	CollectConfig(ctx context.Context, target string) (*models.ConfigData, error)

	// Health
	HealthCheck(ctx context.Context, target string) (*models.HealthStatus, error)
	Ping(ctx context.Context, target string) error

	// Remediation
	CanAutoFix(issue *models.Issue) (bool, *models.FixAction)
	ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error)
	ValidateFix(ctx context.Context, issue *models.Issue, result *models.FixResult) (bool, string, error)
}

// PluginManager defines the contract for the component responsible for managing the
// entire lifecycle of plugins.
type PluginManager interface {
	// LoadPlugins discovers and loads plugins from the configured directory.
	LoadPlugins() error
	// GetPlugin retrieves a plugin instance by its name.
	GetPlugin(name string) (DiagnosticPlugin, error)
	// ListPlugins returns a list of all loaded plugins.
	ListPlugins() []DiagnosticPlugin
	// CollectData invokes relevant plugins to gather data for a diagnosis request.
	CollectData(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error)
	// Shutdown gracefully stops all loaded plugins.
	Shutdown()

	// Methods for managing individual plugins
	LoadPlugin(pluginName string) (DiagnosticPlugin, error)
	UnloadPlugin(pluginName string) error
}

// PluginRegistry defines the contract for a component that discovers available plugins.
type PluginRegistry interface {
	FindPlugin(name string, versionConstraint string) (*models.PluginManifest, error)
	ListAvailablePlugins() ([]*models.PluginManifest, error)
}

// PluginLoader defines the contract for a component responsible for loading a plugin.
type PluginLoader interface {
	Load(manifest *models.PluginManifest) (DiagnosticPlugin, error)
}
