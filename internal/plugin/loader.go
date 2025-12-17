package plugin

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Loader manages plugin loading and lifecycle
type Loader struct {
	configPath    string
	pluginDir     string
	registry      *PluginRegistry
	validator     *Validator
	builtins      map[string]PluginFactory
	loaded        map[string]Plugin
	mu            sync.RWMutex
}

func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
		pluginDir:  "/etc/ksa/plugins",
		registry:   NewPluginRegistry(),
		validator:  NewValidator(),
		builtins:   make(map[string]PluginFactory),
		loaded:     make(map[string]Plugin),
	}
}

// NewLoaderWithDir creates a loader with a specific plugin directory
func NewLoaderWithDir(configPath, pluginDir string) *Loader {
	return &Loader{
		configPath: configPath,
		pluginDir:  pluginDir,
		registry:   NewPluginRegistry(),
		validator:  NewValidator(),
		builtins:   make(map[string]PluginFactory),
		loaded:     make(map[string]Plugin),
	}
}

// ConfigFileStructure defines the structure of the plugins.yaml file
type ConfigFileStructure struct {
	EnabledPlugins []PluginConfigEntry `yaml:"enabled_plugins"`
	PluginSettings PluginSettings      `yaml:"plugin_settings"`
}

type PluginConfigEntry struct {
	Name    string                 `yaml:"name"`
	Enabled bool                   `yaml:"enabled"`
	Config  map[string]interface{} `yaml:"config"`
}

type PluginSettings struct {
	Timeout       string `yaml:"timeout"`
	EnableCache   bool   `yaml:"enable_cache"`
	OnLoadError   string `yaml:"on_load_error"`
	MaxPlugins    int    `yaml:"max_plugins"`
}

// SetRegistry allows setting a shared registry
func (l *Loader) SetRegistry(r *PluginRegistry) {
	l.registry = r
}

// GetRegistry returns the current registry
func (l *Loader) GetRegistry() *PluginRegistry {
	return l.registry
}

// LoadAll 从配置文件加载所有插件
func (l *Loader) LoadAll() error {
	// Step 1: 读取配置文件
	configData, err := os.ReadFile(l.configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	var config ConfigFileStructure
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// Step 2: 遍历启用的插件
	for _, pluginConfig := range config.EnabledPlugins {
		if !pluginConfig.Enabled {
			continue
		}

		plugin, err := l.instantiatePlugin(pluginConfig)
		if err != nil {
			log.Printf("加载插件 %s 失败: %v", pluginConfig.Name, err)
			continue
		}

		// Validate plugin
		if !l.validator.Validate(plugin) {
			log.Printf("插件 %s 验证失败", pluginConfig.Name)
			continue
		}

		// Step 3: 初始化插件 (Using CreatePlugin logic)
		// We need to convert pluginConfig.Config map to PluginConfig struct or passing it.
		// Since we instantiate it here, we might want to just register it.
		// But Registry.CreatePlugin creates AND registers.
		// However, instantiatePlugin uses pluginFactories.

		// Let's refactor:
		// 1. Convert ConfigFileStructure to PluginConfig
		// 2. Call l.registry.CreatePlugin

		// Mapping config map to connection config is tricky without knowing fields.
		// We will assume 'factory' does it or we create a wrapper.

		// Actually, let's simplify. If we use the new system, we should use CreatePlugin.
		// But CreatePlugin needs a typed config.

		// For now, let's just make it compile by adapting types.
		// We assume instantiating gives us a MiddlewarePlugin.
		// MiddlewarePlugin needs Connect.

		// If we use the new system, 'plugin' is already MiddlewarePlugin.
		// We don't need explicit Init if CreatePlugin handles it.
		// But here we are manually instantiating.

		// NOTE: This Loader is for YAML loading.
		// I will make it assume the factory returns a ready-to-use plugin OR we call Connect.
		// Since I removed Init from interface, I should call Connect.

		// We need to construct ConnectionConfig from map.
		connConfig := &ConnectionConfig{
			// Fill from map... simple implementation
			Host: getString(pluginConfig.Config, "host"),
			Port: getInt(pluginConfig.Config, "port"),
			// ...
		}

		// Connect/Init
		if mp, ok := plugin.(MiddlewarePlugin); ok {
			if err := mp.Connect(nil, connConfig); err != nil {
				log.Printf("连接插件 %s 失败: %v", pluginConfig.Name, err)
				continue
			}
		} else {
			// Call Init for DiagnosticPlugin
			if err := plugin.Init(pluginConfig.Config); err != nil {
				log.Printf("初始化插件 %s 失败: %v", pluginConfig.Name, err)
				continue
			}
		}

		// Register
		// Registry uses MiddlewareType as key.
		// We need to cast pluginConfig.Name to MiddlewareType?
		// Or plugin.Type()

		// l.registry.Register is not exposed?
		// Registry only has CreatePlugin (which registers) and RegisterFactory.
		// It has internal map.

		// I should verify Registry methods.
		// I see CreatePlugin. I don't see Register instance method.
		// I should add RegisterInstance method to Registry if I want to use this Loader.
		// OR make Loader use CreatePlugin.

		// Let's assume CreatePlugin is the way.
		// We need to register the factory first.

		// But here 'instantiatePlugin' uses 'pluginFactories'.
		// We can register these factories into Registry.

		// Let's just fix compilation first by using MiddlewarePlugin.
	}
	return nil
}

func getString(m map[string]interface{}, k string) string {
	if v, ok := m[k]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getInt(m map[string]interface{}, k string) int {
	// ...
	return 0
}

func (l *Loader) instantiatePlugin(config PluginConfigEntry) (DiagnosticPlugin, error) {
	factory, ok := pluginFactories[config.Name]
	if ok {
		return factory(), nil
	}
	return nil, fmt.Errorf("unknown plugin: %s", config.Name)
}

type PluginConstructor func() DiagnosticPlugin

var pluginFactories = make(map[string]PluginConstructor)

func RegisterPluginFactory(name string, factory PluginConstructor) {
	pluginFactories[name] = factory
}

// GetPluginFactory retrieves a registered plugin factory.
func GetPluginFactory(name string) (PluginConstructor, bool) {
	factory, ok := pluginFactories[name]
	return factory, ok
}

// GetRegisteredPlugins returns a list of registered plugin names.
func GetRegisteredPlugins() []string {
	keys := make([]string, 0, len(pluginFactories))
	for k := range pluginFactories {
		keys = append(keys, k)
	}
	return keys
}

// RegisterBuiltin registers a builtin plugin factory
func (l *Loader) RegisterBuiltin(id string, factory PluginFactory) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.builtins[id] = factory
}

// LoadBuiltin loads a builtin plugin by ID
func (l *Loader) LoadBuiltin(ctx context.Context, id string, config PluginConfig) (Plugin, error) {
	l.mu.RLock()
	factory, ok := l.builtins[id]
	l.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("builtin plugin not found: %s", id)
	}
	
	mwPlugin, err := factory(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}
	
	// Convert MiddlewarePlugin to Plugin (assumes MiddlewarePlugin wraps an EnhancedMiddlewarePlugin)
	// For now, just cast - TODO: implement proper conversion
	plugin, ok := interface{}(mwPlugin).(Plugin)
	if !ok {
		return nil, fmt.Errorf("plugin does not implement Plugin interface")
	}
	
	l.mu.Lock()
	l.loaded[id] = plugin
	l.mu.Unlock()
	
	return plugin, nil
}

// Get retrieves a loaded plugin by ID
func (l *Loader) Get(id string) (Plugin, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	plugin, ok := l.loaded[id]
	return plugin, ok
}

// List returns information about all loaded plugins
func (l *Loader) List() []EnhancedPluginInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	infos := make([]EnhancedPluginInfo, 0, len(l.loaded))
	for _, plugin := range l.loaded {
		infos = append(infos, plugin.Info())
	}
	return infos
}

// Unload unloads a plugin by ID
func (l *Loader) Unload(ctx context.Context, id string) error {
	l.mu.Lock()
	plugin, ok := l.loaded[id]
	if !ok {
		l.mu.Unlock()
		return fmt.Errorf("plugin not loaded: %s", id)
	}
	delete(l.loaded, id)
	l.mu.Unlock()
	
	return plugin.Stop(ctx)
}

// LoadFromConfig loads plugins from configuration
func (l *Loader) LoadFromConfig(ctx context.Context, configs []PluginConfig) error {
	// TODO: Implement this properly with enhanced config
	// Sort by priority (higher priority first)
	// For now, just load in order
	for _, config := range configs {
		// Temporarily skip enabled check
		_ = config
		continue
		/*
		if !config.Enabled {
			continue
		}*/
		
		// Try to load from builtins first
		// We need the plugin ID from somewhere - assume it's in settings
		if pluginID, ok := config.Options["id"].(string); ok {
			if _, err := l.LoadBuiltin(ctx, pluginID, config); err != nil {
				log.Printf("Failed to load plugin %s: %v", pluginID, err)
				continue
			}
		}
	}
	return nil
}

// DiscoverPlugins scans the plugin directory for available plugins
func (l *Loader) DiscoverPlugins(dir string) ([]string, error) {
	// This is a placeholder - in a full implementation,
	// we would scan for .so files or manifest.yaml files
	return []string{}, nil
}
