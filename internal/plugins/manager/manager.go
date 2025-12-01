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

// Package manager implements the core logic for managing the lifecycle of plugins.
package manager

import (
	"fmt"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	intplugin "github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// pluginManager is the concrete implementation of the interfaces.PluginManager.
// It orchestrates the registry and loader to find, load, and manage plugins.
type pluginManager struct {
	log      logger.Logger
	registry interfaces.PluginRegistry
	loader   interfaces.PluginLoader

	loadedPlugins map[string]interfaces.MiddlewarePlugin
	mu            sync.RWMutex
}

// NewManager creates a new instance of the plugin manager. It requires a registry
// to find available plugins and a loader to load them into memory.
//
// Parameters:
//   registry (interfaces.PluginRegistry): The registry for discovering plugin manifests.
//   loader (interfaces.PluginLoader): The loader for loading plugin code.
//
// Returns:
// NewManager creates a new instance of the plugin manager. It requires a registry
// to find available plugins and a loader to load them into memory.
//
// Parameters:
//   registry (interfaces.PluginRegistry): The registry for discovering plugin manifests.
//   loader (interfaces.PluginLoader): The loader for loading plugin code.
//
// Returns:
// NewManager creates a new instance of the plugin manager. It requires a registry
// to find available plugins and a loader to load them into memory.
//
// Parameters:
//   registry (interfaces.PluginRegistry): The registry for discovering plugin manifests.
//   loader (interfaces.PluginLoader): The loader for loading plugin code.
//
// Returns:
//   interfaces.PluginManager: A new, configured plugin manager.
func NewManager(registry interfaces.PluginRegistry, loader interfaces.PluginLoader) interfaces.PluginManager {
	return &pluginManager{
		log:           logger.NewLogger("plugin-manager"),
		registry:      registry,
		loader:        loader,
		loadedPlugins: make(map[string]interfaces.MiddlewarePlugin),
	}
}

// LoadPlugin finds a plugin in the registry, uses the loader to load its code,
// and stores the initialized instance for future use. It is thread-safe and
// caches already loaded plugins to prevent redundant work.
//
// Parameters:
//   pluginName (string): The name of the plugin to load.
//
// Returns:
//   interfaces.MiddlewarePlugin: The loaded and initialized plugin.
//   error: An error if the plugin cannot be found, loaded, or initialized.
func (m *pluginManager) LoadPlugin(pluginName string) (interfaces.MiddlewarePlugin, error) {
	m.mu.RLock()
	plugin, loaded := m.loadedPlugins[pluginName]
	m.mu.RUnlock()

	if loaded {
		m.log.Debugf("Plugin '%s' is already loaded, returning from cache.", pluginName)
		return plugin, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check in case the plugin was loaded by another goroutine
	// while this one was waiting for the write lock.
	plugin, loaded = m.loadedPlugins[pluginName]
	if loaded {
		return plugin, nil
	}

	m.log.Infof("Loading plugin: %s", pluginName)

	// 1. Find plugin manifest from the registry.
	// An empty version constraint "" implies finding the latest compatible version.
	// First, check for a static plugin factory.
	if factory, ok := intplugin.GetPluginFactory(pluginName); ok {
		plugin := factory()
		// Case 1: Plugin is already the correct type
		if mp, ok := plugin.(interfaces.MiddlewarePlugin); ok {
			m.loadedPlugins[pluginName] = mp
			m.log.Infof("Statically linked plugin '%s' loaded successfully.", pluginName)
			return mp, nil
		}
		// Case 2: Plugin is a DiagnosticPlugin, needs adapting
		if dp, ok := plugin.(intplugin.DiagnosticPlugin); ok {
			adapter := &DiagnosticPluginAdapter{p: dp}
			m.loadedPlugins[pluginName] = adapter
			m.log.Infof("Statically linked diagnostic plugin '%s' loaded successfully with adapter.", pluginName)
			return adapter, nil
		}
		// Case 3: Plugin is a legacy Plugin, needs adapting
		if lp, ok := plugin.(intplugin.Plugin); ok {
			adapter := &LegacyPluginAdapter{p: lp}
			m.loadedPlugins[pluginName] = adapter
			m.log.Infof("Statically linked legacy plugin '%s' loaded successfully with adapter.", pluginName)
			return adapter, nil
		}

		// If none of the above, then we have a problem.
		return nil, fmt.Errorf("statically linked plugin '%s' does not implement a known plugin interface", pluginName)
	}

	manifest, err := m.registry.FindPlugin(pluginName, "")
	if err != nil {
		return nil, fmt.Errorf("could not find plugin '%s' in registry: %w", pluginName, err)
	}

	// 2. Perform compatibility and dependency checks (Placeholder).
	// In a real implementation, you would check manifest.APIVersion against the application's API version
	// and recursively load dependencies defined in manifest.Dependencies.
	// You would also perform security checks, like verifying plugin signatures.
	m.log.Debugf("Found manifest for plugin '%s' version '%s'.", manifest.Name, manifest.Version)

	// 3. Load the plugin using the loader.
	newPlugin, err := m.loader.Load(manifest)
	if err != nil {
		return nil, fmt.Errorf("could not load plugin '%s' from entrypoint '%s': %w", pluginName, manifest.Entrypoint, err)
	}

	// 4. Store and monitor the loaded plugin.
	m.loadedPlugins[pluginName] = newPlugin
	m.log.Infof("Plugin '%s' version '%s' loaded successfully.", newPlugin.Name(), newPlugin.Version())

	// TODO: Implement health checks and status monitoring for the loaded plugin in a separate goroutine.
	// TODO: Implement performance monitoring and resource limiting.

	return newPlugin, nil
}

// UnloadPlugin removes a plugin from the manager's control. This operation is thread-safe.
// NOTE: For Go plugins, this does not truly unload the code from memory but makes
// the plugin unavailable through the manager and eligible for garbage collection if
// it has a `Shutdown` method to release its resources.
//
// Parameters:
//   pluginName (string): The name of the plugin to unload.
//
// Returns:
//   error: An error if the plugin is not currently loaded.
func (m *pluginManager) UnloadPlugin(pluginName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.loadedPlugins[pluginName]; !ok {
		return fmt.Errorf("plugin '%s' is not loaded", pluginName)
	}

	delete(m.loadedPlugins, pluginName)
	m.log.Infof("Plugin '%s' unloaded.", pluginName)
	return nil
}

// GetPlugin retrieves a loaded plugin from the cache by its name. This operation is thread-safe.
//
// Parameters:
//   pluginName (string): The name of the plugin to retrieve.
//
// Returns:
//   interfaces.MiddlewarePlugin: The plugin instance if found.
//   bool: True if the plugin was found, false otherwise.
func (m *pluginManager) GetPlugin(pluginName string) (interfaces.MiddlewarePlugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	plugin, ok := m.loadedPlugins[pluginName]
	return plugin, ok
}

// ListPlugins returns a slice of all currently loaded and active plugins.
// This operation is thread-safe.
//
// Returns:
//   []interfaces.MiddlewarePlugin: A slice of the loaded plugins.
func (m *pluginManager) ListPlugins() []interfaces.MiddlewarePlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]interfaces.MiddlewarePlugin, 0, len(m.loadedPlugins))
	for _, p := range m.loadedPlugins {
		plugins = append(plugins, p)
	}
	return plugins
}

//Personal.AI order the ending
