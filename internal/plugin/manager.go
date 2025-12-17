package plugin

import (
	"context"
	"fmt"
	"sync"
)

// Manager is a centralized plugin management system
type Manager struct {
	loader    *Loader
	registry  *EnhancedRegistry
	lifecycle *LifecycleManager
	sandbox   *Sandbox
	mu        sync.RWMutex
}

// NewManager creates a new plugin manager
func NewManager() *Manager {
	loader := NewLoader("")
	registry := NewEnhancedRegistry()
	lifecycle := NewLifecycleManager(registry, loader)
	sandbox := DefaultSandbox()
	
	return &Manager{
		loader:    loader,
		registry:  registry,
		lifecycle: lifecycle,
		sandbox:   sandbox,
	}
}

// RegisterBuiltinPlugin registers a builtin plugin factory
func (m *Manager) RegisterBuiltinPlugin(id string, factory PluginFactory) {
	m.loader.RegisterBuiltin(id, factory)
}

// LoadPlugin loads a plugin by ID
func (m *Manager) LoadPlugin(ctx context.Context, id string, config PluginConfig) error {
	// Load the plugin
	plugin, err := m.loader.LoadBuiltin(ctx, id, config)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}
	
	// Register it
	if err := m.registry.Register(plugin, config); err != nil {
		return fmt.Errorf("failed to register plugin: %w", err)
	}
	
	// Initialize and start it
	if err := m.lifecycle.InitPlugin(ctx, id, config); err != nil {
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}
	
	if err := m.lifecycle.StartPlugin(ctx, id); err != nil {
		return fmt.Errorf("failed to start plugin: %w", err)
	}
	
	return nil
}

// GetPlugin retrieves a plugin by ID
func (m *Manager) GetPlugin(id string) (Plugin, error) {
	return m.registry.Get(id)
}

// GetMiddlewarePlugin retrieves a middleware plugin by type
func (m *Manager) GetMiddlewarePlugin(middlewareType string) (EnhancedMiddlewarePlugin, error) {
	return m.registry.GetMiddlewarePlugin(middlewareType)
}

// ListPlugins returns all registered plugins
func (m *Manager) ListPlugins() map[string]EnhancedPluginInfo {
	return m.registry.All()
}

// GetRegistry returns the registry
func (m *Manager) GetRegistry() *EnhancedRegistry {
	return m.registry
}

// GetSandbox returns the sandbox
func (m *Manager) GetSandbox() *Sandbox {
	return m.sandbox
}

// Shutdown shuts down all plugins
func (m *Manager) Shutdown(ctx context.Context) error {
	return m.lifecycle.StopAll(ctx)
}

// GlobalManager is a singleton instance
var (
	globalManager     *Manager
	globalManagerOnce sync.Once
)

// GetGlobalManager returns the global plugin manager
func GetGlobalManager() *Manager {
	globalManagerOnce.Do(func() {
		globalManager = NewManager()
	})
	return globalManager
}
