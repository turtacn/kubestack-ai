package plugin

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrPluginNameConflict     = errors.New("plugin name conflict")
	ErrIncompatibleAPIVersion = errors.New("incompatible API version")
	ErrPluginNotFound         = errors.New("plugin not found")
)

type ErrPluginConflict struct {
	Conflicting string
}

func (e ErrPluginConflict) Error() string {
	return fmt.Sprintf("plugin conflict with: %s", e.Conflicting)
}

type Registry struct {
	factories map[string]PluginFactory // key=插件名
	mu        sync.RWMutex
}

type PluginFactory interface {
	Create() Plugin
	Metadata() *PluginMetadata
}

type PluginMetadata struct {
	Name                 string
	Version              string
	APIVersion           string // KSA API版本要求，如"v1"
	SupportedMiddlewares []string
	Description          string
	Author               string
	Conflicts            []string // 与哪些插件冲突
}

func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]PluginFactory)}
}

func (r *Registry) Register(factory PluginFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata := factory.Metadata()

	// 3. 检查名称冲突
	if _, exists := r.factories[metadata.Name]; exists {
		return ErrPluginNameConflict
	}

	// 4. 检查版本兼容性
	if !r.isAPICompatible(metadata.APIVersion) {
		return ErrIncompatibleAPIVersion
	}

	// 5. 检查与已注册插件的冲突
	for _, conflict := range metadata.Conflicts {
		if _, exists := r.factories[conflict]; exists {
			return ErrPluginConflict{Conflicting: conflict}
		}
	}

	r.factories[metadata.Name] = factory
	return nil
}

func (r *Registry) GetFactory(name string) (PluginFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory := r.factories[name]
	if factory == nil {
		return nil, ErrPluginNotFound
	}
	return factory, nil
}

// isAPICompatible checks if the plugin's required API version is compatible with the current system.
// For now, we assume "v1" is the current version and we are compatible with "v1" and "v1.x".
func (r *Registry) isAPICompatible(requiredVersion string) bool {
	// Simple check for now. In a real system, we might use semver.
	return requiredVersion == "v1" || requiredVersion == "v1.0"
}

// Global registry instance
var GlobalRegistry = NewRegistry()

// RegisterPlugin is a helper to register to the global registry
func RegisterPlugin(factory PluginFactory) error {
	return GlobalRegistry.Register(factory)
}
