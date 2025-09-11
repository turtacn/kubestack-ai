// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package config provides a centralized configuration management system for KubeStack-AI.
// It uses Viper to support loading from YAML files, environment variables, and command-line flags,
// and also supports features like hot-reloading.
package config

import (
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/constants"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/errors"
	"github.com/spf13/viper"
)

// Config is the root configuration structure for the application.
// It is composed of smaller, domain-specific configuration structs.
type Config struct {
	Logger  logger.Config `mapstructure:"logger"`
	Server  ServerConfig  `mapstructure:"server"`
	LLM     LLMConfig     `mapstructure:"llm"`
	Plugins PluginConfig  `mapstructure:"plugins"`
}

// ServerConfig holds HTTP server-related configurations.
type ServerConfig struct {
	Port    int `mapstructure:"port"`
	Timeout int `mapstructure:"timeout"`
}

// LLMConfig holds configurations for Large Language Model providers.
type LLMConfig struct {
	Provider string       `mapstructure:"provider"`
	OpenAI   OpenAIConfig `mapstructure:"openai"`
	Gemini   GeminiConfig `mapstructure:"gemini"`
}

// OpenAIConfig holds OpenAI-specific API configurations.
type OpenAIConfig struct {
	APIKey string `mapstructure:"apiKey"` // Sensitive: Handle with care.
	Model  string `mapstructure:"model"`
}

// GeminiConfig holds Google Gemini-specific API configurations.
type GeminiConfig struct {
	APIKey string `mapstructure:"apiKey"` // Sensitive: Handle with care.
	Model  string `mapstructure:"model"`
}

// PluginConfig holds configurations for the plugin system.
type PluginConfig struct {
	Directory string `mapstructure:"directory"`
}

var appConfig *Config

// LoadConfig initializes Viper, loads configuration from multiple sources,
// and sets up hot-reloading.
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()
	setDefaults(v)

	if configPath != "" {
		v.SetConfigFile(configPath)
		v.SetConfigType("yaml")
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, errors.WrapConfigError(err, errors.ConfigLoadFailedCode, "failed to read config file", "Ensure the file exists and is readable.")
			}
		}
	}

	v.SetEnvPrefix("KSA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.WrapConfigError(err, errors.ConfigLoadFailedCode, "failed to unmarshal config", "Check the configuration structure.")
	}

	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		log := logger.GetLogger()
		log.Infof("Configuration file changed: %s", e.Name)
		if err := v.Unmarshal(&cfg); err != nil {
			log.Errorf("Failed to reload config: %v", err)
		} else {
			appConfig = &cfg
			logger.InitGlobalLogger(&cfg.Logger) // Re-initialize logger with new settings
			log.Info("Configuration reloaded successfully.")
		}
	})

	appConfig = &cfg
	return &cfg, nil
}

// GetConfig returns the loaded application configuration.
func GetConfig() *Config {
	return appConfig
}

// Validate checks if the loaded configuration is valid.
func (c *Config) Validate() error {
	if c.LLM.Provider == "openai" && c.LLM.OpenAI.APIKey == "" {
		return errors.NewConfigError(errors.ConfigValidationFailedCode, "OpenAI API key is missing", "Set the KSA_LLM_OPENAI_APIKEY environment variable or llm.openai.apiKey in the config file.")
	}
	if c.LLM.Provider == "gemini" && c.LLM.Gemini.APIKey == "" {
		return errors.NewConfigError(errors.ConfigValidationFailedCode, "Gemini API key is missing", "Set the KSA_LLM_GEMINI_APIKEY environment variable or llm.gemini.apiKey in the config file.")
	}
	// Add more validation rules here
	return nil
}

// setDefaults defines the default values for configuration keys.
func setDefaults(v *viper.Viper) {
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "text")
	v.SetDefault("logger.output", "console")
	v.SetDefault("logger.file", "/var/log/kubestack-ai/ksa.log")
	v.SetDefault("logger.maxSize", 100)
	v.SetDefault("logger.maxBackups", 3)
	v.SetDefault("logger.maxAge", 7)
	v.SetDefault("logger.compress", true)

	v.SetDefault("server.port", 8080)
	v.SetDefault("server.timeout", 30)

	v.SetDefault("llm.provider", "openai")
	v.SetDefault("llm.openai.model", "gpt-4")
	v.SetDefault("llm.gemini.model", "gemini-pro")

	v.SetDefault("plugins.directory", constants.DefaultPluginDir)
}

//Personal.AI order the ending
