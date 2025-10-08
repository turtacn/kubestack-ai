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

// MiddlewarePlugin defines the core interface that every middleware-specific plugin
// must implement. It provides a standard contract for discovery, diagnostics, data
// collection, and automated fixing, allowing the core engine to interact with any
// supported middleware in a consistent way.
type MiddlewarePlugin interface {
	// Name returns the official name of the middleware the plugin supports (e.g., "Redis", "PostgreSQL").
	Name() string
	// Version returns the semantic version of the plugin itself.
	Version() string
	// Description provides a brief, human-readable summary of what the plugin does.
	Description() string
	// SupportedVersions returns a slice of middleware versions that this plugin is compatible with.
	SupportedVersions() []string

	// Diagnose runs a comprehensive, high-level diagnosis for the middleware.
	Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error)
	// CollectMetrics gathers performance and operational metrics from the middleware.
	CollectMetrics(ctx context.Context) (*models.MetricsData, error)
	// CollectLogs retrieves recent log entries from the middleware.
	CollectLogs(ctx context.Context, opts *models.LogOptions) (*models.LogData, error)
	// GetConfiguration retrieves the current configuration of the middleware.
	GetConfiguration(ctx context.Context) (*models.ConfigData, error)

	// HealthCheck performs a detailed health assessment of the middleware.
	HealthCheck(ctx context.Context) (*models.HealthStatus, error)
	// Ping performs a simple, lightweight check to see if the middleware is reachable and responsive.
	Ping(ctx context.Context) error

	// CanAutoFix determines if a given issue, identified by the plugin, can be safely and automatically fixed.
	CanAutoFix(issue *models.Issue) bool
	// ExecuteFix applies a specific, automated fix for a given issue.
	ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error)
	// ValidateFix confirms that a fix was successful by re-checking the state of the original issue.
	ValidateFix(ctx context.Context, fix *models.FixAction) error
}

// PluginManager defines the contract for the component responsible for managing the
// entire lifecycle of plugins. It orchestrates loading, unloading, and provides
// access to active plugins, acting as a central point of control for the plugin system.
type PluginManager interface {
	// LoadPlugin finds a plugin by name, loads it into memory, initializes it, and
	// makes it available for use.
	//
	// Parameters:
	//   pluginName (string): The name of the plugin to load.
	//
	// Returns:
	//   MiddlewarePlugin: An interface to the loaded plugin.
	//   error: An error if the plugin cannot be found or fails to load.
	LoadPlugin(pluginName string) (MiddlewarePlugin, error)

	// UnloadPlugin safely unloads a plugin from memory, releasing its resources.
	//
	// Parameters:
	//   pluginName (string): The name of the plugin to unload.
	//
	// Returns:
	//   error: An error if the plugin cannot be unloaded.
	UnloadPlugin(pluginName string) error

	// GetPlugin retrieves a previously loaded plugin from the manager's active pool.
	//
	// Parameters:
	//   pluginName (string): The name of the plugin to retrieve.
	//
	// Returns:
	//   MiddlewarePlugin: An interface to the active plugin.
	//   bool: True if the plugin was found, false otherwise.
	GetPlugin(pluginName string) (MiddlewarePlugin, bool)

	// ListPlugins returns a list of all currently loaded and active plugins.
	//
	// Returns:
	//   []MiddlewarePlugin: A slice of active plugins.
	ListPlugins() []MiddlewarePlugin
}

// PluginRegistry defines the contract for a component that discovers available
// plugins, whether they are stored locally or in a remote repository. It deals
// with metadata and manifests rather than the plugin code itself.
type PluginRegistry interface {
	// FindPlugin searches the registry for a plugin that matches a given name and
	// version constraint.
	//
	// Parameters:
	//   name (string): The name of the plugin to find.
	//   versionConstraint (string): A semantic version constraint (e.g., ">=1.2.0").
	//
	// Returns:
	//   *models.PluginManifest: The manifest of the matching plugin.
	//   error: An error if no matching plugin is found.
	FindPlugin(name string, versionConstraint string) (*models.PluginManifest, error)

	// ListAvailablePlugins returns a list of all plugin manifests available in the registry.
	//
	// Returns:
	//   []*models.PluginManifest: A slice of all discovered plugin manifests.
	//   error: An error if the registry cannot be accessed.
	ListAvailablePlugins() ([]*models.PluginManifest, error)
}

// PluginLoader defines the contract for a component responsible for the low-level
// technical task of loading a plugin from a source (e.g., a Go plugin from a
// .so file) and initializing its symbols.
type PluginLoader interface {
	// Load takes a plugin manifest, retrieves the plugin artifact (e.g., a .so file),
	// loads it into the current process, and initializes it, returning a ready-to-use
	// MiddlewarePlugin interface.
	//
	// Parameters:
	//   manifest (*models.PluginManifest): The manifest describing the plugin to be loaded.
	//
	// Returns:
	//   MiddlewarePlugin: An interface to the newly loaded plugin.
	//   error: An error if the plugin artifact cannot be loaded or is invalid.
	Load(manifest *models.PluginManifest) (MiddlewarePlugin, error)
}

//Personal.AI order the ending
