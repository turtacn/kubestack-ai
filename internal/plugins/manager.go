package plugins

import (
	"github.com/turtacn/kubestack-ai/internal/errors"
	"github.com/turtacn/kubestack-ai/internal/logging"
)

// PluginManager 接口定义插件管理。PluginManager interface for managing plugins.
type PluginManager interface {
	Install(name string, source string) error
	Load(name string) (Plugin, error)
	Uninstall(name string) error
}

// manager 插件管理实现。manager implementation for plugins.
type manager struct {
	plugins map[string]Plugin
}

// NewManager 创建插件管理器。NewManager creates a new plugin manager.
func NewManager() PluginManager {
	return &manager{
		plugins: make(map[string]Plugin),
	}
}

// InitManager 初始化全局管理器。InitManager initializes global manager.
var Manager PluginManager

func InitManager() {
	Manager = NewManager()
}

// Install 安装插件。Install installs a plugin.
func (m *manager) Install(name string, source string) error {
	// TODO: 下载和验证。TODO: download and verify.
	logging.Logger.Info("Installing plugin", name)
	// 示例注册。Example registration.
	if name == "mysql" {
		m.plugins[name] = &MySQLPlugin{} // 假设实现。
	}
	return nil
}

// Load 加载插件。Load loads a plugin.
func (m *manager) Load(name string) (Plugin, error) {
	p, ok := m.plugins[name]
	if !ok {
		return nil, errors.ErrPluginNotFound
	}
	return p, nil
}

// Uninstall 卸载插件。Uninstall uninstalls a plugin.
func (m *manager) Uninstall(name string) error {
	delete(m.plugins, name)
	logging.Logger.Info("Uninstalled plugin", name)
	return nil
}

//Personal.AI order the ending
