package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// PluginFactory is a function that creates a new plugin instance
type PluginFactory func(config *PluginConfig) (MiddlewarePlugin, error)

// PluginConfig contains configuration for creating a plugin
type PluginConfig struct {
	Type       MiddlewareType
	Connection *ConnectionConfig
	Options    map[string]interface{}
}

// PluginRegistry manages plugin registration and creation
type PluginRegistry struct {
	plugins   map[MiddlewareType]MiddlewarePlugin
	factories map[MiddlewareType]PluginFactory
	mu        sync.RWMutex
	log       logger.Logger
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins:   make(map[MiddlewareType]MiddlewarePlugin),
		factories: make(map[MiddlewareType]PluginFactory),
		log:       logger.NewLogger("PluginRegistry"),
	}
}

// RegisterFactory registers a factory function for a middleware type
func (r *PluginRegistry) RegisterFactory(mwType MiddlewareType, factory PluginFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[mwType]; exists {
		return fmt.Errorf("factory for %s already registered", mwType)
	}

	r.factories[mwType] = factory
	r.log.Info("plugin factory registered", "type", mwType)
	return nil
}

// CreatePlugin creates and connects a plugin
func (r *PluginRegistry) CreatePlugin(ctx context.Context, config *PluginConfig) (MiddlewarePlugin, error) {
	r.mu.RLock()
	factory, ok := r.factories[config.Type]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no factory registered for type: %s", config.Type)
	}

	plugin, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}

	// Connect
	if err := plugin.Connect(ctx, config.Connection); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Register instance
	r.mu.Lock()
	r.plugins[config.Type] = plugin
	r.mu.Unlock()

	return plugin, nil
}

// GetPlugin retrieves an existing plugin instance
func (r *PluginRegistry) GetPlugin(mwType MiddlewareType) (MiddlewarePlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.plugins[mwType]
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", mwType)
	}

	return plugin, nil
}

// PluginInfo contains basic info about a registered plugin
type PluginInfo struct {
	Type      MiddlewareType
	Name      string
	Version   string
	Connected bool
}

// ListPlugins lists all registered plugin instances
func (r *PluginRegistry) ListPlugins() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(r.plugins))
	for mwType, plugin := range r.plugins {
		infos = append(infos, PluginInfo{
			Type:      mwType,
			Name:      plugin.Name(),
			Version:   plugin.Version(),
			Connected: plugin.IsConnected(),
		})
	}
	return infos
}

// Close disconnects all plugins
func (r *PluginRegistry) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for mwType, plugin := range r.plugins {
		if err := plugin.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s disconnect failed: %w", mwType, err))
		}
	}

	// Clear plugins
	r.plugins = make(map[MiddlewareType]MiddlewarePlugin)

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}
