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

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	intplugin "github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// goPluginLoader is a concrete implementation of PluginLoader for standard Go plugins,
// which are compiled as shared object (.so) files.
type goPluginLoader struct {
	log logger.Logger
}

// NewLoader creates a new plugin loader for standard Go plugins.
func NewLoader() interfaces.PluginLoader {
	return &goPluginLoader{
		log: logger.NewLogger("plugin-loader"),
	}
}

// Load implements the PluginLoader interface for standard Go plugins.
func (l *goPluginLoader) Load(manifest *models.PluginManifest) (interfaces.DiagnosticPlugin, error) {
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

		if legacyPlugin, ok := dp.(intplugin.Plugin); ok {
			return &LegacyPluginAdapter{p: legacyPlugin}, nil
		}

		// If it implements DiagnosticPlugin directly
		if mp, ok := dp.(interfaces.DiagnosticPlugin); ok {
			return mp, nil
		}

		return nil, fmt.Errorf("plugin %s does not implement DiagnosticPlugin interface", name)
	}

	p, err := plugin.Open(manifest.Entrypoint)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin file '%s': %w", manifest.Entrypoint, err)
	}

	newFuncSymbol, err := p.Lookup("New")
	if err != nil {
		return nil, fmt.Errorf("failed to find required symbol 'New' in plugin '%s': %w", manifest.Name, err)
	}

	newFunc, ok := newFuncSymbol.(func() (interfaces.DiagnosticPlugin, error))
	if !ok {
		return nil, fmt.Errorf("symbol 'New' in plugin '%s' has an incorrect function signature", manifest.Name)
	}

	pluginInstance, err := newFunc()
	if err != nil {
		return nil, fmt.Errorf("the factory function 'New' for plugin '%s' returned an error on creation: %w", manifest.Name, err)
	}

	if pluginInstance.Name() != manifest.Name {
		l.log.Warnf("Loaded plugin identified itself as '%s', but manifest name is '%s'.", pluginInstance.Name(), manifest.Name)
	}

	l.log.Infof("Successfully loaded and instantiated plugin: %s", manifest.Name)
	return pluginInstance, nil
}

// LegacyPluginAdapter adapts internal/plugin.Plugin to interfaces.DiagnosticPlugin
type LegacyPluginAdapter struct {
	p intplugin.Plugin
}

func (a *LegacyPluginAdapter) Name() string { return a.p.Name() }
func (a *LegacyPluginAdapter) Version() string { return a.p.Version() }
func (a *LegacyPluginAdapter) Description() string { return "" }
func (a *LegacyPluginAdapter) SupportedVersions() []string { return []string{} }
func (a *LegacyPluginAdapter) SupportedTypes() []enum.MiddlewareType {
	// Map string name to enum if possible
	t, _ := enum.ParseMiddlewareType(a.p.Name())
	if t == -1 {
		// Try from SupportedTypes string array if available
		for _, st := range a.p.SupportedTypes() {
			if t, err := enum.ParseMiddlewareType(st); err == nil {
				return []enum.MiddlewareType{t}
			}
		}
		return nil
	}
	return []enum.MiddlewareType{t}
}

func (a *LegacyPluginAdapter) Init(config *config.PluginConfig) error {
	// Map config.PluginConfig to map[string]interface{}
	// This is a rough mapping.
	m := make(map[string]interface{})
	if config != nil {
		m["directory"] = config.Directory
	}
	return a.p.Init(m)
}

func (a *LegacyPluginAdapter) Shutdown() error {
	return a.p.Shutdown()
}

// Diagnose forwards to the plugin's Diagnose method
func (a *LegacyPluginAdapter) Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error) {
	return a.p.Diagnose(ctx, req)
}

func (a *LegacyPluginAdapter) CollectMetrics(ctx context.Context, target string) (*models.MetricsData, error) {
	// If the plugin supports MiddlewarePlugin interface, use it
	if mp, ok := a.p.(intplugin.MiddlewarePlugin); ok {
		snap, err := mp.CollectMetrics(ctx)
		if err != nil {
			return nil, err
		}
		out := &models.MetricsData{Data: make(map[string]interface{})}
		for k, v := range snap.Metrics {
			out.Data[k] = v.Value
		}
		return out, nil
	}
	// Fallback for DiagnosticPlugin (Small) which doesn't support metrics collection directly
	return &models.MetricsData{Data: make(map[string]interface{})}, nil
}

func (a *LegacyPluginAdapter) CollectLogs(ctx context.Context, target string, opts *models.LogOptions) (*models.LogData, error) {
	return &models.LogData{}, nil
}
func (a *LegacyPluginAdapter) CollectConfig(ctx context.Context, target string) (*models.ConfigData, error) {
	return &models.ConfigData{}, nil
}
func (a *LegacyPluginAdapter) HealthCheck(ctx context.Context, target string) (*models.HealthStatus, error) {
	return &models.HealthStatus{IsHealthy: true}, nil
}
func (a *LegacyPluginAdapter) Ping(ctx context.Context, target string) error { return nil }

// CanAutoFix updated to return (bool, *models.FixAction)
func (a *LegacyPluginAdapter) CanAutoFix(issue *models.Issue) (bool, *models.FixAction) {
	return false, nil
}

// ExecuteFix
func (a *LegacyPluginAdapter) ExecuteFix(ctx context.Context, fix *models.FixAction) (*models.FixResult, error) {
	return nil, fmt.Errorf("not implemented")
}

// ValidateFix updated signature
func (a *LegacyPluginAdapter) ValidateFix(ctx context.Context, issue *models.Issue, result *models.FixResult) (bool, string, error) {
	return false, "", fmt.Errorf("not implemented")
}
