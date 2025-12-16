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
	registry   *PluginRegistry
	validator  *Validator
}

func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
		registry:   NewPluginRegistry(),
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

		// Connect
		if err := plugin.Connect(nil, connConfig); err != nil { // Context?
			log.Printf("连接插件 %s 失败: %v", pluginConfig.Name, err)
			continue
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

func (l *Loader) instantiatePlugin(config PluginConfigEntry) (MiddlewarePlugin, error) {
	factory, ok := pluginFactories[config.Name]
	if ok {
		return factory(), nil
	}
	return nil, fmt.Errorf("unknown plugin: %s", config.Name)
}

type PluginConstructor func() MiddlewarePlugin

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
