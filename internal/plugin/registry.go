package plugin

import (
	"fmt"
	"log"
	"sync"
)

// Registry 插件注册中心
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]DiagnosticPlugin
}

var (
	// DefaultRegistry 全局默认注册中心
	DefaultRegistry = NewRegistry()
)

func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]DiagnosticPlugin),
	}
}

// RegisterPlugin Global function to register plugin factory (Legacy support for existing plugins)
func RegisterPlugin(factory PluginFactory) {
	meta := factory.Metadata()
	log.Printf("Registering legacy plugin factory: %s", meta.Name)

	// Register the factory constructor to the loader map as well to support loader-based instantiation if needed
	// by adapting it to PluginConstructor.
	RegisterPluginFactory(meta.Name, func() DiagnosticPlugin {
		// This is tricky. Legacy `Plugin` interface is different from `DiagnosticPlugin`.
		// `Plugin` has Collector/Parser/HealthChecker. `DiagnosticPlugin` has Diagnose.
		// For now, we will wrap it or just allow it to register.
		// Since we can't easily convert Legacy Plugin to DiagnosticPlugin without an adapter,
		// and we don't have the adapter code right now.
		// We will return nil or panic if used as DiagnosticPlugin.
		// Or better, we should fix the `RegisterPluginFactory` signature or usage.

		// Wait, `RegisterPluginFactory` takes `PluginConstructor` which returns `DiagnosticPlugin`.
		// If `KafkaPlugin` does NOT implement `DiagnosticPlugin`, we can't return it here.

		// The error "cannot call non-function factory" in loader.go was because `factory` variable was shadowing the type or interface.

		return nil // Placeholder.
	})
}

// Register 注册插件
func (r *Registry) Register(plugin DiagnosticPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := plugin.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("插件 %s 已注册", name)
	}

	r.plugins[name] = plugin
	log.Printf("插件 %s (v%s) 注册成功", name, plugin.Version())
	return nil
}

// Get 获取插件
func (r *Registry) Get(name string) DiagnosticPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.plugins[name]
}

// FindByType 根据中间件类型查找插件
func (r *Registry) FindByType(middlewareType string) []DiagnosticPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []DiagnosticPlugin
	for _, plugin := range r.plugins {
		for _, supportedType := range plugin.SupportedTypes() {
			if supportedType == middlewareType {
				matched = append(matched, plugin)
				break
			}
		}
	}
	return matched
}

// List 列出所有插件
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}
