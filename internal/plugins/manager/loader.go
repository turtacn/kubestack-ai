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

package manager

import (
	"context"
	"fmt"
	"plugin"
	"strings"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	intplugin "github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// goPluginLoader is a concrete implementation of PluginLoader for standard Go plugins,
// which are compiled as shared object (.so) files.
type goPluginLoader struct {
	log logger.Logger
}

// NewLoader creates a new plugin loader for standard Go plugins. In the future,
// this could be extended to take a configuration to decide which type of loader
// to create (e.g., for scripts, containers, or other plugin formats).
//
// Returns:
//   interfaces.PluginLoader: A new instance of a Go plugin loader.
func NewLoader() interfaces.PluginLoader {
	return &goPluginLoader{
		log: logger.NewLogger("plugin-loader"),
	}
}

// Load implements the PluginLoader interface for standard Go plugins. It opens the
// shared object (.so) file specified in the manifest's entrypoint, looks up the
// conventional `New` factory function, and executes it to create an instance of
// the plugin.
//
// Parameters:
//   manifest (*models.PluginManifest): The manifest describing the plugin to load.
//
// Returns:
//   interfaces.MiddlewarePlugin: An initialized, ready-to-use plugin instance.
//   error: An error if the file cannot be opened, the 'New' symbol is missing or
//          has the wrong signature, or the factory function itself returns an error.
func (l *goPluginLoader) Load(manifest *models.PluginManifest) (interfaces.MiddlewarePlugin, error) {
	l.log.Infof("Attempting to load Go plugin from entrypoint: %s", manifest.Entrypoint)

	// Check for static plugins
	if strings.HasPrefix(manifest.Entrypoint, "static:") {
		name := strings.TrimPrefix(manifest.Entrypoint, "static:")
		factory, ok := intplugin.GetPluginFactory(name)
		if !ok {
			return nil, fmt.Errorf("static plugin factory not found for: %s", name)
		}

		// Instantiate
		dp := factory()

		// Convert internal/plugin.DiagnosticPlugin to interfaces.MiddlewarePlugin
		// This is the tricky part. The interfaces might differ.
		// If built-in plugins (like Kafka) implement the old interface (via wrapper or directly),
		// we need to cast it.
		// Actually, pkg/plugins/kafka implements `plugin.Plugin` (legacy interface in `internal/plugin`).
		// My changes to `internal/plugin/interface.go` kept `Plugin` interface but added `DiagnosticPlugin`.
		// `KafkaPlugin` implements `Plugin` (legacy).
		// `PluginFactory` returns `Plugin` (legacy) in my latest edit? No, I changed `PluginFactory` to return `Plugin` in interface.go, but `PluginConstructor` in loader.go returns `DiagnosticPlugin`.
		// Wait, `internal/plugin/loader.go` defines `PluginConstructor` returning `DiagnosticPlugin`.
		// `RegisterPluginFactory` takes `PluginConstructor`.
		// But `KafkaPluginFactory.Create` returns `plugin.Plugin`.
		// So `KafkaPlugin` implements `plugin.Plugin`.
		// Does `plugin.Plugin` (legacy) satisfy `interfaces.MiddlewarePlugin`?
		// Let's compare.
		// `interfaces.MiddlewarePlugin` (core/interfaces) has:
		// Name, Version, Description, SupportedVersions, Diagnose, CollectMetrics, CollectLogs, GetConfiguration, HealthCheck, Ping, CanAutoFix, ExecuteFix, ValidateFix.

		// `plugin.Plugin` (internal/plugin) has:
		// Name, Version, Description, SupportedMiddlewareVersions, Initialize, Shutdown, Collector, Parser, HealthChecker.

		// THEY ARE DIFFERENT.
		// The built-in plugins (pkg/plugins/...) implement `internal/plugin.Plugin`.
		// The `PluginManager` (internal/plugins/manager) expects `interfaces.MiddlewarePlugin`.
		// So we need an ADAPTER here.

		if legacyPlugin, ok := dp.(intplugin.Plugin); ok {
			return &LegacyPluginAdapter{p: legacyPlugin}, nil
		}

		// If it implements MiddlewarePlugin directly
		if mp, ok := dp.(interfaces.MiddlewarePlugin); ok {
			return mp, nil
		}

		return nil, fmt.Errorf("plugin %s does not implement MiddlewarePlugin interface", name)
	}

	// Security checks would be performed here in a production system, for example:
	// 1. Signature Verification: Check if the .so file is signed by a trusted key to ensure its integrity.
	// 2. Sandbox Setup: Prepare a sandboxed environment if untrusted plugins are allowed (this is very complex).
	// if err := l.verifySignature(manifest.Entrypoint); err != nil { ... }

	p, err := plugin.Open(manifest.Entrypoint)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin file '%s': %w", manifest.Entrypoint, err)
	}

	// By convention, we expect each plugin to export a symbol named "New".
	// This symbol must be a factory function that creates an instance of the plugin.
	newFuncSymbol, err := p.Lookup("New")
	if err != nil {
		return nil, fmt.Errorf("failed to find required symbol 'New' in plugin '%s': %w", manifest.Name, err)
	}

	// Type-assert the symbol to the expected factory function signature.
	// The signature is: func() (interfaces.MiddlewarePlugin, error)
	newFunc, ok := newFuncSymbol.(func() (interfaces.MiddlewarePlugin, error))
	if !ok {
		return nil, fmt.Errorf("symbol 'New' in plugin '%s' has an incorrect function signature", manifest.Name)
	}

	// Call the factory function to create the plugin instance.
	pluginInstance, err := newFunc()
	if err != nil {
		return nil, fmt.Errorf("the factory function 'New' for plugin '%s' returned an error on creation: %w", manifest.Name, err)
	}

	// As a final sanity check, ensure the loaded plugin's name matches the manifest.
	if pluginInstance.Name() != manifest.Name {
		l.log.Warnf("Loaded plugin identified itself as '%s', but manifest name is '%s'.", pluginInstance.Name(), manifest.Name)
	}

	l.log.Infof("Successfully loaded and instantiated plugin: %s", manifest.Name)
	return pluginInstance, nil
}

// LegacyPluginAdapter adapts internal/plugin.Plugin to interfaces.MiddlewarePlugin
type LegacyPluginAdapter struct {
	p intplugin.Plugin
}

func (a *LegacyPluginAdapter) Name() string { return a.p.Name() }
func (a *LegacyPluginAdapter) Version() string { return a.p.Version() }
func (a *LegacyPluginAdapter) Description() string { return a.p.Description() }
func (a *LegacyPluginAdapter) SupportedVersions() []string { return a.p.SupportedMiddlewareVersions() }

func (a *LegacyPluginAdapter) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	// Legacy plugins don't implement Diagnose directly usually (or do they?).
	// If they implement DiagnosticPlugin (new), they do.
	// But here we caught `intplugin.Plugin`.
	// We can try to use Collector/Parser to simulate a diagnosis or return not implemented.
	// Actually, P4 says `DiagnosticPlugin` is the new one.
	// But `LegacyPluginAdapter` is adapting the OLD `Plugin` interface.
	return nil, fmt.Errorf("legacy diagnosis not fully implemented in adapter")
}

func (a *LegacyPluginAdapter) CollectMetrics(ctx context.Context) (*models.MetricsData, error) {
	// Construct a minimal target. For legacy plugins that rely on internal config (initialized via Init),
	// this might be sufficient if they don't use target fields for connection.
	target := &intplugin.Target{
		Type:    a.Name(),
		Address: "legacy-configured", // Placeholder
	}

	data, err := a.p.Collector().Collect(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("legacy collector failed: %w", err)
	}

	metrics, err := a.p.Parser().Parse(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("legacy parser failed: %w", err)
	}

	// Convert internal/plugin.ParsedMetrics to models.MetricsData
	out := &models.MetricsData{
		Data: make(map[string]interface{}),
	}

	if metrics != nil && metrics.Metrics != nil {
		for k, v := range metrics.Metrics {
			if v != nil {
				out.Data[k] = v.Value
			}
		}
	}

	return out, nil
}

func (a *LegacyPluginAdapter) CollectLogs(ctx context.Context, opts *models.LogOptions) (*models.LogData, error) {
	return nil, fmt.Errorf("not implemented")
}
func (a *LegacyPluginAdapter) GetConfiguration(ctx context.Context) (*models.ConfigData, error) {
	return nil, fmt.Errorf("not implemented")
}
func (a *LegacyPluginAdapter) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	// target := ...
	// status, err := a.p.HealthChecker().Check(ctx, target)
	return nil, fmt.Errorf("health check requires target info")
}
func (a *LegacyPluginAdapter) Ping(ctx context.Context) error { return nil }
func (a *LegacyPluginAdapter) CanAutoFix(issue *models.Issue) bool { return false }
func (a *LegacyPluginAdapter) ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (a *LegacyPluginAdapter) ValidateFix(ctx context.Context, fix *models.FixAction) error {
	return fmt.Errorf("not implemented")
}

// TODO: Implement loaders for other plugin formats if the system needs to support them.
// For example, a loader for plugins written as Python scripts:
//
// type scriptPluginLoader struct { ... }
// func (l *scriptPluginLoader) Load(...) (interfaces.MiddlewarePlugin, error) {
//   // Logic to execute a script and communicate with it, perhaps over stdin/stdout or a unix socket.
//   ...
// }
//
// Or for plugins distributed as containers:
//
// type containerPluginLoader struct { ... }
// func (l *containerPluginLoader) Load(...) (interfaces.MiddlewarePlugin, error) {
//   // Logic to start a container and interact with it via an exposed API.
//   ...
// }

//Personal.AI order the ending
