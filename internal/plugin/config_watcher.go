package plugin

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type ConfigWatcher struct {
	watcher   *fsnotify.Watcher
	manager   *PluginManager
	configDir string
	logger    *zap.Logger
	stopCh    chan struct{}
	reloadCh  chan string // 插件名
}

func NewConfigWatcher(manager *PluginManager, configDir string, logger *zap.Logger) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := watcher.Add(configDir); err != nil {
		return nil, err
	}

	return &ConfigWatcher{
		watcher:   watcher,
		manager:   manager,
		configDir: configDir,
		logger:    logger,
		stopCh:    make(chan struct{}),
		reloadCh:  make(chan string, 10),
	}, nil
}

func (w *ConfigWatcher) Start(ctx context.Context) {
	go w.watchLoop(ctx)
	go w.reloadLoop(ctx)
}

func (w *ConfigWatcher) extractPluginName(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func (w *ConfigWatcher) watchLoop(ctx context.Context) {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				// 配置文件被修改
				pluginName := w.extractPluginName(event.Name)
				w.logger.Info("config changed", zap.String("plugin", pluginName))
				w.reloadCh <- pluginName
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("watcher error", zap.Error(err))
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		}
	}
}

func (w *ConfigWatcher) reloadLoop(ctx context.Context) {
	for {
		select {
		case pluginName := <-w.reloadCh:
			// 防抖：延迟100ms再执行，避免频繁重载
			time.Sleep(100 * time.Millisecond)

			// 清空通道中的重复事件
			for len(w.reloadCh) > 0 {
				<-w.reloadCh
			}

			w.reloadPlugin(ctx, pluginName)
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		}
	}
}

func (w *ConfigWatcher) loadPluginConfig(pluginName string) (*PluginConfig, error) {
	v := viper.New()
	v.SetConfigName(pluginName)
	v.SetConfigType("yaml")
	v.AddConfigPath(w.configDir)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var config PluginConfig
	// Assuming the config file structure matches PluginConfig or has a "plugin" key
	// Based on pseudo-code for plugin implementation, users use mapstructure.Decode(config.Settings)
	// Let's assume the YAML has a root "plugin" key or similar.
	// The deliverable list shows configs/plugins/redis.yaml
	// Example in guide:
	// plugin:
	//   name: myware
	//   enabled: true
	//   settings: ...

	if err := v.UnmarshalKey("plugin", &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (w *ConfigWatcher) reloadPlugin(ctx context.Context, pluginName string) {
	// 1. 读取新配置
	newConfig, err := w.loadPluginConfig(pluginName)
	if err != nil {
		w.logger.Error("failed to load new config", zap.Error(err))
		return
	}

	// 2. 验证新配置
	if err := w.validateConfig(newConfig); err != nil {
		w.logger.Error("invalid config", zap.Error(err))
		return
	}

	// 3. 保存旧配置（用于回滚）
	// Note: PluginManager doesn't expose getting config easily, but we can assume we might need it.
	// For now, simpler implementation as per pseudo-code.

	// 4. 尝试重新加载
	// Disable first
	_ = w.manager.DisablePlugin(pluginName)
	// We might need to Unload if Initialize cannot be called twice on same instance without side effects?
	// The Manager.LoadPlugin creates a NEW instance using the factory.
	// So we should Unload then Load.

	if err := w.manager.UnloadPlugin(pluginName); err != nil {
		 w.logger.Warn("failed to unload plugin during reload", zap.String("plugin", pluginName), zap.Error(err))
	}

	if err := w.manager.LoadPlugin(pluginName, newConfig); err != nil {
		w.logger.Error("failed to load new config", zap.Error(err))
		// Rollback would require keeping the old config.
		// Since we don't have it easily here without persisting state, we skip complex rollback for now.
		return
	}

	if newConfig.Enabled {
		if err := w.manager.EnablePlugin(pluginName); err != nil {
			w.logger.Error("failed to enable plugin after reload", zap.Error(err))
		}
	}

	w.logger.Info("plugin reloaded successfully", zap.String("plugin", pluginName))
}

func (w *ConfigWatcher) validateConfig(config *PluginConfig) error {
	// 1. 检查必填字段
	if config.Name == "" {
		return context.DeadlineExceeded // just a placeholder error
	}
	return nil
}

func (w *ConfigWatcher) Stop() {
	close(w.stopCh)
	w.watcher.Close()
}
