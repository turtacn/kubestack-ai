package plugin

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Loader 插件加载器
type Loader struct {
	configPath string
	registry   *Registry
	validator  *Validator
}

func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
		registry:   NewRegistry(),
		validator:  NewValidator(),
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
func (l *Loader) SetRegistry(r *Registry) {
	l.registry = r
}

// GetRegistry returns the current registry
func (l *Loader) GetRegistry() *Registry {
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

		// Step 3: 初始化插件
		if err := plugin.Init(pluginConfig.Config); err != nil {
			log.Printf("初始化插件 %s 失败: %v", pluginConfig.Name, err)
			continue
		}

		// Step 4: 注册插件
		if err := l.registry.Register(plugin); err != nil {
			log.Printf("注册插件 %s 失败: %v", pluginConfig.Name, err)
		}
	}

	return nil
}

// instantiatePlugin 实例化插件（通过反射或插件包）
// In a real generic loader, we might use Go's 'plugin' package to load .so files.
// However, since we are implementing built-in plugins for now, we will use a factory map
// or rely on manual registration of available types.
// Ideally, the Loader should know about available plugin types.
// For the purpose of this task, I'll expose a way to register "Constructors" or just
// use a hardcoded switch for the known built-ins, as we are adding them to the codebase.
func (l *Loader) instantiatePlugin(config PluginConfigEntry) (DiagnosticPlugin, error) {
	// Simple factory for built-in plugins.
	// To avoid circular dependencies (Loader -> Plugins -> Loader),
	// we usually register factories. But here we might need to assume
	// the plugins are in a separate package `plugins/...`.
	// We can't import `plugins/redis` here if `plugins/redis` imports `internal/plugin` (interface).
	// That is fine. `internal/plugin` is low level.

	// However, `internal/plugin` (this package) cannot import `plugins/redis`.
	// So we need a mechanism to register constructors.

	factory, ok := pluginFactories[config.Name]
	if ok {
		return factory(), nil
	}

	return nil, fmt.Errorf("未知插件类型: %s (Did you forget to register it?)", config.Name)
}

// PluginConstructor is a function that creates a new instance of a plugin
type PluginConstructor func() DiagnosticPlugin

var pluginFactories = make(map[string]PluginConstructor)

// RegisterPluginFactory registers a factory function for a plugin name.
// This should be called by the plugin packages in their init() functions or by main.
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
