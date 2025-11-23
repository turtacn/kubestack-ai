package plugin

import (
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// PluginLoader 负责扫描配置目录并加载插件
type PluginLoader struct {
	manager   *PluginManager
	configDir string
	logger    *zap.Logger
}

func NewPluginLoader(manager *PluginManager, configDir string, logger *zap.Logger) *PluginLoader {
	return &PluginLoader{
		manager:   manager,
		configDir: configDir,
		logger:    logger,
	}
}

func (l *PluginLoader) LoadAll() error {
	files, err := filepath.Glob(filepath.Join(l.configDir, "*.yaml"))
	if err != nil {
		return err
	}

	for _, file := range files {
		pluginName := l.extractPluginName(file)
		l.logger.Info("found plugin config", zap.String("plugin", pluginName))

		config, err := l.loadPluginConfig(pluginName)
		if err != nil {
			l.logger.Error("failed to load plugin config", zap.String("plugin", pluginName), zap.Error(err))
			continue
		}

		if err := l.manager.LoadPlugin(pluginName, config); err != nil {
			l.logger.Error("failed to load plugin", zap.String("plugin", pluginName), zap.Error(err))
			continue
		}

		if config.Enabled {
			if err := l.manager.EnablePlugin(pluginName); err != nil {
				l.logger.Error("failed to enable plugin", zap.String("plugin", pluginName), zap.Error(err))
			}
		}
	}

	return nil
}

func (l *PluginLoader) extractPluginName(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func (l *PluginLoader) loadPluginConfig(pluginName string) (*PluginConfig, error) {
	v := viper.New()
	v.SetConfigName(pluginName)
	v.SetConfigType("yaml")
	v.AddConfigPath(l.configDir)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var config PluginConfig
	if err := v.UnmarshalKey("plugin", &config); err != nil {
		return nil, err
	}
	return &config, nil
}
