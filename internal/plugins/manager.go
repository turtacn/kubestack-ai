package plugins

import (
	"context"
	"sync"

	"github.com/turtacn/kubestack-ai/internal/errors"
	"github.com/turtacn/kubestack-ai/internal/logging"
)

// 定义插件构造函数类型：由具体插件实现
type PluginConstructor func() Plugin

// 全局注册表：存储插件名称到构造函数的映射
var pluginRegistry = make(map[string]PluginConstructor)

// RegisterPlugin 供具体插件调用，注册自身的构造函数
func RegisterPlugin(name string, constructor PluginConstructor) {
	pluginRegistry[name] = constructor
	logging.Logger.Infof("Plugin %s registered", name)
}

// PluginManager 接口定义插件管理。PluginManager interface for managing plugins.
type PluginManager interface {
	Install(ctx context.Context, name string, source string) error
	Load(name string) (Plugin, error)
	Uninstall(name string) error
	List() []string
	GetPluginStatus(name string) string
}

// manager 插件管理实现。manager implementation for plugins.
type manager struct {
	plugins map[string]Plugin
	status  map[string]string
	mutex   sync.RWMutex
}

// 插件状态常量。Plugin status constants.
const (
	PluginStatusInstalled   = "installed"
	PluginStatusActive      = "active"
	PluginStatusError       = "error"
	PluginStatusUninstalled = "uninstalled"
)

// NewManager 创建插件管理器。NewManager creates a new plugin manager.
func NewManager() PluginManager {
	return &manager{
		plugins: make(map[string]Plugin),
		status:  make(map[string]string),
	}
}

// InitManager 初始化全局管理器。InitManager initializes global manager.
var Manager PluginManager

func InitManager() {
	Manager = NewManager()
}

// Install 安装插件。Install installs a plugin.
// Install 方法中，通过注册表创建插件实例，替代硬编码
func (m *manager) Install(ctx context.Context, name string, source string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	logging.Logger.Infof("Installing plugin: %s from source: %s", name, source)

	if _, exists := m.plugins[name]; exists {
		logging.Logger.Warnf("Plugin %s is already installed", name)
		return nil
	}

	// 从注册表获取构造函数，替代硬编码的 switch-case
	constructor, exists := pluginRegistry[name]
	if !exists {
		logging.Logger.Errorf("Unsupported plugin: %s", name)
		return errors.ErrPluginInstallationFailed
	}

	// 调用构造函数创建插件实例
	plugin := constructor()

	// 后续初始化逻辑不变
	config := PluginConfig{"source": source}
	if err := plugin.Initialize(config); err != nil {
		logging.Logger.Errorf("Failed to initialize plugin %s: %v", name, err)
		m.status[name] = PluginStatusError
		return errors.ErrPluginInstallationFailed
	}

	if err := plugin.Validate(); err != nil {
		logging.Logger.Errorf("Plugin %s validation failed: %v", name, err)
		m.status[name] = PluginStatusError
		return errors.ErrPluginInstallationFailed
	}

	m.plugins[name] = plugin
	m.status[name] = PluginStatusInstalled
	logging.Logger.Infof("Plugin %s installed successfully", name)
	return nil
}

// Load 加载插件。Load loads a plugin.
func (m *manager) Load(name string) (Plugin, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	plugin, exists := m.plugins[name]
	if !exists {
		logging.Logger.Errorf("Plugin %s not found", name)
		return nil, errors.ErrPluginNotFound
	}

	m.status[name] = PluginStatusActive
	return plugin, nil
}

// Uninstall 卸载插件。Uninstall uninstalls a plugin.
func (m *manager) Uninstall(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		logging.Logger.Errorf("Plugin %s not found for uninstallation", name)
		return errors.ErrPluginNotFound
	}

	// 清理插件资源。Cleanup plugin resources.
	if err := plugin.Cleanup(); err != nil {
		logging.Logger.Warnf("Plugin %s cleanup failed: %v", name, err)
	}

	delete(m.plugins, name)
	m.status[name] = PluginStatusUninstalled
	logging.Logger.Infof("Plugin %s uninstalled successfully", name)
	return nil
}

// List 列出所有已安装插件。List all installed plugins.
func (m *manager) List() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	plugins := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		plugins = append(plugins, name)
	}
	return plugins
}

// GetPluginStatus 获取插件状态。Get plugin status.
func (m *manager) GetPluginStatus(name string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	status, exists := m.status[name]
	if !exists {
		return PluginStatusUninstalled
	}
	return status
}

//Personal.AI order the ending
