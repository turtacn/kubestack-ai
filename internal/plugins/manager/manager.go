// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law of agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

type pluginManager struct {
	log      logger.Logger
	registry interfaces.PluginRegistry
	loader   interfaces.PluginLoader
	plugins  map[string]interfaces.DiagnosticPlugin
	mu       sync.RWMutex
}

// NewManager creates a new plugin manager.
func NewManager(registry interfaces.PluginRegistry, loader interfaces.PluginLoader) interfaces.PluginManager {
	return &pluginManager{
		log:      logger.NewLogger("plugin-manager"),
		registry: registry,
		loader:   loader,
		plugins:  make(map[string]interfaces.DiagnosticPlugin),
	}
}

func (m *pluginManager) LoadPlugins() error {
	manifests, err := m.registry.ListAvailablePlugins()
	if err != nil {
		return err
	}

	for _, manifest := range manifests {
		plugin, err := m.loader.Load(manifest)
		if err != nil {
			m.log.Warnf("Failed to load plugin %s: %v", manifest.Name, err)
			continue
		}
		m.mu.Lock()
		m.plugins[plugin.Name()] = plugin
		m.mu.Unlock()
	}
	return nil
}

func (m *pluginManager) LoadPlugin(pluginName string) (interfaces.DiagnosticPlugin, error) {
	m.mu.RLock()
	p, ok := m.plugins[pluginName]
	m.mu.RUnlock()
	if ok {
		return p, nil
	}

	manifest, err := m.registry.FindPlugin(pluginName, "")
	if err != nil {
		return nil, err
	}

	p, err = m.loader.Load(manifest)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.plugins[pluginName] = p
	m.mu.Unlock()

	return p, nil
}

func (m *pluginManager) UnloadPlugin(pluginName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.plugins[pluginName]; ok {
		if err := p.Shutdown(); err != nil {
			return err
		}
		delete(m.plugins, pluginName)
	}
	return nil
}

func (m *pluginManager) GetPlugin(name string) (interfaces.DiagnosticPlugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.plugins[name]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("plugin not found: %s", name)
}

func (m *pluginManager) ListPlugins() []interfaces.DiagnosticPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []interfaces.DiagnosticPlugin
	for _, p := range m.plugins {
		list = append(list, p)
	}
	return list
}

func (m *pluginManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.plugins {
		_ = p.Shutdown()
	}
	m.plugins = make(map[string]interfaces.DiagnosticPlugin)
}

// CollectData gathers metrics and logs from the relevant plugin.
func (m *pluginManager) CollectData(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error) {
	pluginName := ""
	if req.TargetMiddleware.String() != "Unknown" {
		// Map middleware type to plugin name (e.g., "Redis" -> "redis-diagnostics")
		middlewareName := strings.ToLower(req.TargetMiddleware.String())
		pluginName = middlewareName + "-diagnostics"
	} else {
		// Fallback logic
		return nil, fmt.Errorf("could not determine plugin name from request")
	}

	p, err := m.GetPlugin(pluginName)
	if err != nil {
		p, err = m.LoadPlugin(pluginName)
		if err != nil {
			return nil, fmt.Errorf("failed to load plugin %s: %w", pluginName, err)
		}
	}

	target := req.Instance

	metrics, err := p.CollectMetrics(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}

	logOpts := &models.LogOptions{Tail: 100}
	logs, err := p.CollectLogs(ctx, target, logOpts)
	if err != nil {
		logs = &models.LogData{}
	}

	configData, err := p.CollectConfig(ctx, target)
	if err != nil {
		configData = &models.ConfigData{}
	}

	return &models.CollectedData{
		Metrics: metrics,
		Logs:    logs,
		Config:  configData,
	}, nil
}
