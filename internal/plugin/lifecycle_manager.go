package plugin

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// LifecycleManager manages plugin lifecycle operations
type LifecycleManager struct {
	registry   *EnhancedRegistry
	loader     *Loader
	hooks      []PluginHooks
	shutdownCh chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(registry *EnhancedRegistry, loader *Loader) *LifecycleManager {
	return &LifecycleManager{
		registry:   registry,
		loader:     loader,
		hooks:      make([]PluginHooks, 0),
		shutdownCh: make(chan struct{}),
	}
}

// AddHooks adds lifecycle hooks
func (m *LifecycleManager) AddHooks(hooks PluginHooks) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks = append(m.hooks, hooks)
}

// InitPlugin initializes a plugin
func (m *LifecycleManager) InitPlugin(ctx context.Context, id string, config PluginConfig) error {
	plugin, err := m.registry.Get(id)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}
	
	if err := m.registry.SetState(id, StateUninitialized); err != nil {
		return err
	}
	
	if err := plugin.Init(ctx, config); err != nil {
		m.registry.SetState(id, StateError)
		m.triggerOnError(plugin, err)
		return fmt.Errorf("init failed: %w", err)
	}
	
	if err := m.registry.SetState(id, StateInitializing); err != nil {
		return err
	}
	
	m.triggerOnLoad(plugin)
	return nil
}

// StartPlugin starts a plugin
func (m *LifecycleManager) StartPlugin(ctx context.Context, id string) error {
	plugin, err := m.registry.Get(id)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}
	
	state := m.registry.GetState(id)
	if state != StateInitializing {
		return fmt.Errorf("plugin must be initialized before starting: current state=%v", state)
	}
	
	if err := plugin.Start(ctx); err != nil {
		m.registry.SetState(id, StateError)
		m.triggerOnError(plugin, err)
		return fmt.Errorf("start failed: %w", err)
	}
	
	if err := m.registry.SetState(id, StateRunning); err != nil {
		return err
	}
	
	// Start health check goroutine
	m.wg.Add(1)
	go m.healthCheckLoop(ctx, id, 30*time.Second)
	
	return nil
}

// StopPlugin stops a plugin
func (m *LifecycleManager) StopPlugin(ctx context.Context, id string) error {
	plugin, err := m.registry.Get(id)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}
	
	state := m.registry.GetState(id)
	if state != StateRunning {
		return fmt.Errorf("plugin is not running: current state=%v", state)
	}
	
	if err := m.registry.SetState(id, StateStopped); err != nil {
		return err
	}
	
	if err := plugin.Stop(ctx); err != nil {
		m.triggerOnError(plugin, err)
		return fmt.Errorf("stop failed: %w", err)
	}
	
	m.triggerOnUnload(plugin)
	return nil
}

// ReloadPlugin reloads a plugin with new configuration
func (m *LifecycleManager) ReloadPlugin(ctx context.Context, id string, newConfig PluginConfig) error {
	// Stop the plugin
	if err := m.StopPlugin(ctx, id); err != nil {
		return fmt.Errorf("failed to stop plugin: %w", err)
	}
	
	// Reinitialize with new config
	if err := m.InitPlugin(ctx, id, newConfig); err != nil {
		return fmt.Errorf("failed to reinitialize plugin: %w", err)
	}
	
	// Restart
	if err := m.StartPlugin(ctx, id); err != nil {
		return fmt.Errorf("failed to start plugin: %w", err)
	}
	
	return nil
}

// StartAll starts all initialized plugins
func (m *LifecycleManager) StartAll(ctx context.Context) error {
	allPlugins := m.registry.All()
	
	for id := range allPlugins {
		state := m.registry.GetState(id)
		if state == StateInitializing {
			if err := m.StartPlugin(ctx, id); err != nil {
				log.Printf("Failed to start plugin %s: %v", id, err)
			}
		}
	}
	
	return nil
}

// StopAll stops all running plugins
func (m *LifecycleManager) StopAll(ctx context.Context) error {
	allPlugins := m.registry.All()
	
	// Build a list to stop in reverse order (simple approach)
	toStop := make([]string, 0)
	for id := range allPlugins {
		state := m.registry.GetState(id)
		if state == StateRunning {
			toStop = append(toStop, id)
		}
	}
	
	// Stop in reverse order
	for i := len(toStop) - 1; i >= 0; i-- {
		id := toStop[i]
		if err := m.StopPlugin(ctx, id); err != nil {
			log.Printf("Failed to stop plugin %s: %v", id, err)
		}
	}
	
	// Signal shutdown and wait for goroutines
	close(m.shutdownCh)
	m.wg.Wait()
	
	return nil
}

// healthCheckLoop periodically checks plugin health
func (m *LifecycleManager) healthCheckLoop(ctx context.Context, id string, interval time.Duration) {
	defer m.wg.Done()
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.shutdownCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			plugin, err := m.registry.Get(id)
			if err != nil {
				log.Printf("Plugin %s not found in registry", id)
				return
			}
			
			if err := plugin.HealthCheck(ctx); err != nil {
				log.Printf("Health check failed for plugin %s: %v", id, err)
				m.registry.SetState(id, StateError)
				m.triggerOnError(plugin, err)
			}
		}
	}
}

// triggerOnLoad triggers OnLoad hooks
func (m *LifecycleManager) triggerOnLoad(plugin Plugin) {
	m.mu.Lock()
	hooks := m.hooks
	m.mu.Unlock()
	
	for _, h := range hooks {
		if err := h.OnLoad(plugin); err != nil {
			log.Printf("OnLoad hook failed: %v", err)
		}
	}
}

// triggerOnUnload triggers OnUnload hooks
func (m *LifecycleManager) triggerOnUnload(plugin Plugin) {
	m.mu.Lock()
	hooks := m.hooks
	m.mu.Unlock()
	
	for _, h := range hooks {
		if err := h.OnUnload(plugin); err != nil {
			log.Printf("OnUnload hook failed: %v", err)
		}
	}
}

// triggerOnError triggers OnError hooks
func (m *LifecycleManager) triggerOnError(plugin Plugin, err error) {
	m.mu.Lock()
	hooks := m.hooks
	m.mu.Unlock()
	
	for _, h := range hooks {
		h.OnError(plugin, err)
	}
}
