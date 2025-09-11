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

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// MiddlewarePlugin is the core interface that every middleware-specific plugin must implement.
// It defines the standard contract for diagnostics, data collection, and automated fixing.
type MiddlewarePlugin interface {
	// --- Basic Information ---
	Name() string
	Version() string
	Description() string
	SupportedVersions() []string

	// --- Core Functions ---
	Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error)
	CollectMetrics(ctx context.Context) (*models.MetricsData, error)
	CollectLogs(ctx context.Context, opts *models.LogOptions) (*models.LogData, error)
	GetConfiguration(ctx context.Context) (*models.ConfigData, error)

	// --- Health Checks ---
	HealthCheck(ctx context.Context) (*models.HealthStatus, error)
	Ping(ctx context.Context) error

	// --- Fix Operations ---
	CanAutoFix(issue *models.Issue) bool
	ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error)
	ValidateFix(ctx context.Context, fix *models.FixAction) error
}

// PluginManager is responsible for the entire lifecycle of plugins. It orchestrates
// loading, unloading, and provides access to active plugins.
type PluginManager interface {
	LoadPlugin(pluginName string) (MiddlewarePlugin, error)
	UnloadPlugin(pluginName string) error
	GetPlugin(pluginName string) (MiddlewarePlugin, bool)
	ListPlugins() []MiddlewarePlugin
}

// PluginRegistry is responsible for discovering available plugins, whether they are
// stored locally or in a remote repository. It deals with metadata and manifests.
type PluginRegistry interface {
	FindPlugin(name string, versionConstraint string) (*models.PluginManifest, error)
	ListAvailablePlugins() ([]*models.PluginManifest, error)
	// Future methods could include AddRepository, RemoveRepository, etc.
}

// PluginLoader is responsible for the low-level technical task of loading a plugin
// from a source (e.g., a Go plugin from a .so file) and initializing its symbols.
type PluginLoader interface {
	Load(manifest *models.PluginManifest) (MiddlewarePlugin, error)
}

//Personal.AI order the ending
