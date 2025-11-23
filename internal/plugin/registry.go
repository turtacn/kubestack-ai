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

func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]DiagnosticPlugin),
	}
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
