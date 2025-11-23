package plugin

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	ErrPluginAlreadyLoaded = errors.New("plugin already loaded")
	ErrPluginNotLoaded     = errors.New("plugin not loaded")
	ErrInvalidState        = errors.New("invalid plugin state")
	ErrPluginNotAvailable  = errors.New("plugin not available")
)

type PluginManager struct {
	registry      *Registry
	loadedPlugins map[string]*PluginInstance
	mu            sync.RWMutex
	logger        *zap.Logger
}

type PluginInstance struct {
	Plugin    Plugin
	Config    *PluginConfig
	State     PluginState
	LoadedAt  time.Time
	LastError error
}

type PluginState int

const (
	StateUnloaded PluginState = iota
	StateLoaded
	StateInitialized
	StateEnabled
	StateDisabled
	StateFailed
)

func NewPluginManager(registry *Registry, logger *zap.Logger) *PluginManager {
	return &PluginManager{
		registry:      registry,
		loadedPlugins: make(map[string]*PluginInstance),
		logger:        logger,
	}
}

func (m *PluginManager) LoadPlugin(name string, config *PluginConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 2. 检查是否已加载
	if _, exists := m.loadedPlugins[name]; exists {
		return ErrPluginAlreadyLoaded
	}

	// 3. 从Registry获取插件工厂
	factory, err := m.registry.GetFactory(name)
	if err != nil {
		return err
	}

	// 4. 创建插件实例
	plugin := factory.Create()

	// 5. 初始化
	if err := plugin.Initialize(config); err != nil {
		return fmt.Errorf("plugin init failed: %w", err)
	}

	// 6. 保存实例
	m.loadedPlugins[name] = &PluginInstance{
		Plugin:   plugin,
		Config:   config,
		State:    StateInitialized,
		LoadedAt: time.Now(),
	}

	m.logger.Info("plugin loaded", zap.String("name", name))
	return nil
}

func (m *PluginManager) EnablePlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance := m.loadedPlugins[name]
	if instance == nil {
		return ErrPluginNotLoaded
	}

	// 3. 检查状态
	if instance.State != StateInitialized && instance.State != StateDisabled {
		return ErrInvalidState
	}

	// 4. 执行启用逻辑（若插件有Enable方法, but the interface Plugin doesn't have Enable, so we just change state）
	// If needed we can cast to an interface that has Enable. For now just state change.

	instance.State = StateEnabled
	m.logger.Info("plugin enabled", zap.String("name", name))
	return nil
}

func (m *PluginManager) DisablePlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance := m.loadedPlugins[name]
	if instance == nil {
		return ErrPluginNotLoaded
	}

	if instance.State != StateEnabled {
		return ErrInvalidState
	}

	instance.State = StateDisabled
	m.logger.Info("plugin disabled", zap.String("name", name))
	return nil
}

func (m *PluginManager) UnloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance := m.loadedPlugins[name]
	if instance == nil {
		return ErrPluginNotLoaded
	}

	// 3. 先Disable再Shutdown
	if instance.State == StateEnabled {
		instance.State = StateDisabled
	}

	if err := instance.Plugin.Shutdown(); err != nil {
		m.logger.Error("failed to shutdown plugin", zap.String("name", name), zap.Error(err))
		// We continue to unload even if shutdown fails
	}

	delete(m.loadedPlugins, name)
	m.logger.Info("plugin unloaded", zap.String("name", name))
	return nil
}

func (m *PluginManager) GetPlugin(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance := m.loadedPlugins[name]
	if instance == nil || instance.State != StateEnabled {
		return nil, ErrPluginNotAvailable
	}
	return instance.Plugin, nil
}

type PluginInfo struct {
	Name    string
	Version string
	State   PluginState
}

func (m *PluginManager) ListPlugins() []*PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var infos []*PluginInfo
	for _, instance := range m.loadedPlugins {
		infos = append(infos, &PluginInfo{
			Name:    instance.Plugin.Name(),
			Version: instance.Plugin.Version(),
			State:   instance.State,
		})
	}
	return infos
}
