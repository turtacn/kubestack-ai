package plugin

import (
	"fmt"
	"sync"
	"time"
)

// RegisteredPlugin contains a plugin and its metadata
type RegisteredPlugin struct {
	Plugin   Plugin
	State    PluginState
	Config   PluginConfig
	LoadedAt time.Time
}

// EnhancedRegistry manages enhanced plugin registration and state
type EnhancedRegistry struct {
	plugins      map[string]*RegisteredPlugin
	byType       map[PluginType][]string
	byMiddleware map[string][]string
	mu           sync.RWMutex
}

// NewEnhancedRegistry creates a new enhanced registry
func NewEnhancedRegistry() *EnhancedRegistry {
	return &EnhancedRegistry{
		plugins:      make(map[string]*RegisteredPlugin),
		byType:       make(map[PluginType][]string),
		byMiddleware: make(map[string][]string),
	}
}

// Register registers a plugin with the given configuration
func (r *EnhancedRegistry) Register(plugin Plugin, config PluginConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	info := plugin.Info()
	
	// Check if already registered
	if _, exists := r.plugins[info.ID]; exists {
		return fmt.Errorf("plugin already registered: %s", info.ID)
	}
	
	// Create registered plugin entry
	registered := &RegisteredPlugin{
		Plugin:   plugin,
		State:    StateUninitialized,
		Config:   config,
		LoadedAt: time.Now(),
	}
	
	// Store plugin
	r.plugins[info.ID] = registered
	
	// Index by type
	r.byType[info.Type] = append(r.byType[info.Type], info.ID)
	
	// Index by middleware type if applicable
	if mwPlugin, ok := plugin.(EnhancedMiddlewarePlugin); ok {
		mwType := mwPlugin.MiddlewareType()
		r.byMiddleware[mwType] = append(r.byMiddleware[mwType], info.ID)
	}
	
	return nil
}

// Unregister removes a plugin from the registry
func (r *EnhancedRegistry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	registered, exists := r.plugins[id]
	if !exists {
		return fmt.Errorf("plugin not found: %s", id)
	}
	
	// Cannot unregister running plugin
	if registered.State == StateRunning {
		return fmt.Errorf("cannot unregister running plugin: %s", id)
	}
	
	info := registered.Plugin.Info()
	
	// Remove from type index
	typeSlice := r.byType[info.Type]
	r.removeFromSlice(&typeSlice, id)
	r.byType[info.Type] = typeSlice
	
	// Remove from middleware index if applicable
	if mwPlugin, ok := registered.Plugin.(EnhancedMiddlewarePlugin); ok {
		mwType := mwPlugin.MiddlewareType()
		mwSlice := r.byMiddleware[mwType]
		r.removeFromSlice(&mwSlice, id)
		r.byMiddleware[mwType] = mwSlice
	}
	
	// Remove plugin
	delete(r.plugins, id)
	
	return nil
}

// Get retrieves a plugin by ID
func (r *EnhancedRegistry) Get(id string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	registered, exists := r.plugins[id]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", id)
	}
	
	return registered.Plugin, nil
}

// GetMiddlewarePlugin retrieves a middleware plugin by middleware type
func (r *EnhancedRegistry) GetMiddlewarePlugin(middlewareType string) (EnhancedMiddlewarePlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	ids, exists := r.byMiddleware[middlewareType]
	if !exists || len(ids) == 0 {
		return nil, fmt.Errorf("no plugin found for middleware type: %s", middlewareType)
	}
	
	// Return the first enabled plugin
	for _, id := range ids {
		registered := r.plugins[id]
		if registered.State == StateRunning || registered.State == StateInitializing {
			if mwPlugin, ok := registered.Plugin.(EnhancedMiddlewarePlugin); ok {
				return mwPlugin, nil
			}
		}
	}
	
	return nil, fmt.Errorf("no running plugin found for middleware type: %s", middlewareType)
}

// ListByType returns all plugins of a given type
func (r *EnhancedRegistry) ListByType(ptype PluginType) []EnhancedPluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	ids := r.byType[ptype]
	infos := make([]EnhancedPluginInfo, 0, len(ids))
	
	for _, id := range ids {
		if registered, exists := r.plugins[id]; exists {
			infos = append(infos, registered.Plugin.Info())
		}
	}
	
	return infos
}

// ListByMiddleware returns all plugins for a given middleware type
func (r *EnhancedRegistry) ListByMiddleware(mtype string) []EnhancedPluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	ids := r.byMiddleware[mtype]
	infos := make([]EnhancedPluginInfo, 0, len(ids))
	
	for _, id := range ids {
		if registered, exists := r.plugins[id]; exists {
			infos = append(infos, registered.Plugin.Info())
		}
	}
	
	return infos
}

// GetState returns the current state of a plugin
func (r *EnhancedRegistry) GetState(id string) PluginState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if registered, exists := r.plugins[id]; exists {
		return registered.State
	}
	
	return StateStopped
}

// SetState updates the state of a plugin
func (r *EnhancedRegistry) SetState(id string, state PluginState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	registered, exists := r.plugins[id]
	if !exists {
		return fmt.Errorf("plugin not found: %s", id)
	}
	
	registered.State = state
	return nil
}

// All returns information about all registered plugins
func (r *EnhancedRegistry) All() map[string]EnhancedPluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make(map[string]EnhancedPluginInfo, len(r.plugins))
	for id, registered := range r.plugins {
		result[id] = registered.Plugin.Info()
	}
	
	return result
}

// removeFromSlice removes an element from a slice
func (r *EnhancedRegistry) removeFromSlice(slice *[]string, value string) {
	for i, v := range *slice {
		if v == value {
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			return
		}
	}
}
