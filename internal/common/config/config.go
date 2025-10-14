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
// It is composed of smaller, domain-specific configuration structs and is used
// to unmarshal the configuration data from Viper.
type Config struct {
	// Logger holds all logging-related configurations.
	Logger logger.Config `mapstructure:"logger"`
	// Server holds configurations for the optional HTTP server component.
	Server ServerConfig `mapstructure:"server"`
	// LLM holds configurations for connecting to Large Language Model providers.
	LLM LLMConfig `mapstructure:"llm"`
	// Plugins holds configurations related to the plugin management system.
	Plugins PluginConfig `mapstructure:"plugins"`
	// Knowledge holds configurations for the knowledge base, including vector and document stores.
	Knowledge KnowledgeStoreConfig `mapstructure:"knowledge"`
	// Report holds configurations for storing diagnosis reports.
	Report ReportConfig `mapstructure:"report"`
}

// KnowledgeStoreConfig holds configurations for the knowledge base.
type KnowledgeStoreConfig struct {
	// VectorProvider specifies the active vector store provider (e.g., "in-memory", "chroma").
	VectorProvider string `mapstructure:"vectorProvider"`
	// DocumentProvider specifies the active document store provider (e.g., "in-memory", "elasticsearch").
	DocumentProvider string `mapstructure:"documentProvider"`
	// Chroma contains the specific configuration for the ChromaDB provider.
	Chroma ChromaConfig `mapstructure:"chroma"`
	// Elasticsearch contains the specific configuration for the Elasticsearch provider.
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch"`
}

// ReportConfig holds configurations for storing diagnosis reports.
type ReportConfig struct {
	// Directory is the path where diagnosis reports are stored.
	Directory string `mapstructure:"directory"`
}

// ElasticsearchConfig holds Elasticsearch-specific configurations.
type ElasticsearchConfig struct {
	// Addresses is a list of Elasticsearch node URLs.
	Addresses []string `mapstructure:"addresses"`
	// IndexName is the name of the index to use for storing documents.
	IndexName string `mapstructure:"indexName"`
}

// ChromaConfig holds ChromaDB-specific configurations.
type ChromaConfig struct {
	// URL is the base URL for the ChromaDB instance.
	URL string `mapstructure:"url"`
	// CollectionName is the name of the collection to use for storing vectors.
	CollectionName string `mapstructure:"collectionName"`
	// Namespace is the ChromaDB namespace to use.
	Namespace string `mapstructure:"namespace"`
}

// ServerConfig holds HTTP server-related configurations, such as the listening
// port and request timeouts.
type ServerConfig struct {
	// Port is the TCP port on which the HTTP server will listen.
	Port int `mapstructure:"port"`
	// Timeout is the request timeout for the HTTP server in seconds.
	Timeout int `mapstructure:"timeout"`
}

// LLMConfig holds configurations for Large Language Model providers, including which
// provider to use and the credentials for each.
type LLMConfig struct {
	// Provider specifies the active LLM provider (e.g., "openai", "gemini").
	Provider string `mapstructure:"provider"`
	// OpenAI contains the specific configuration for the OpenAI provider.
	OpenAI OpenAIConfig `mapstructure:"openai"`
	// Gemini contains the specific configuration for the Google Gemini provider.
	Gemini GeminiConfig `mapstructure:"gemini"`
}

// OpenAIConfig holds OpenAI-specific API configurations.
type OpenAIConfig struct {
	// APIKey is the secret key for authenticating with the OpenAI API.
	APIKey string `mapstructure:"apiKey"` // Sensitive: Handle with care.
	// Model is the specific model to use (e.g., "gpt-4", "gpt-3.5-turbo").
	Model string `mapstructure:"model"`
}

// GeminiConfig holds Google Gemini-specific API configurations.
type GeminiConfig struct {
	// APIKey is the secret key for authenticating with the Google Gemini API.
	APIKey string `mapstructure:"apiKey"` // Sensitive: Handle with care.
	// Model is the specific model to use (e.g., "gemini-pro").
	Model string `mapstructure:"model"`
}

// PluginConfig holds configurations for the plugin system.
type PluginConfig struct {
	// Directory is the path where plugins are stored.
	Directory string `mapstructure:"directory"`
}

var appConfig *Config

// LoadConfig initializes Viper, loads configuration from multiple sources (file,
// environment variables), sets up defaults, and enables hot-reloading.
// It follows a layered approach: defaults < file < environment variables.
//
// Parameters:
//   configPath (string): The path to the configuration file. If empty, only defaults
//     and environment variables will be used.
//
// Returns:
//   *Config: A pointer to the loaded and unmarshaled configuration struct.
//   error: An error if loading or unmarshaling fails.
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

// GetConfig returns the singleton instance of the loaded application configuration.
// It is crucial to call LoadConfig before calling this function, otherwise it may
// return nil.
//
// Returns:
//   *Config: A pointer to the currently active application configuration.
func GetConfig() *Config {
	return appConfig
}

// Validate checks if the loaded configuration is valid by enforcing certain rules,
// such as ensuring that API keys are present if a specific provider is selected.
//
// Returns:
//   error: An error of type *errors.ConfigError if validation fails, otherwise nil.
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

	v.SetDefault("knowledge.vectorProvider", "in-memory")
	v.SetDefault("knowledge.documentProvider", "in-memory")
	v.SetDefault("knowledge.chroma.url", "http://localhost:8000")
	v.SetDefault("knowledge.chroma.collectionName", "kubestack-ai-kb")
	v.SetDefault("knowledge.chroma.namespace", "default")
	v.SetDefault("knowledge.elasticsearch.addresses", []string{"http://localhost:9200"})
	v.SetDefault("knowledge.elasticsearch.indexName", "kubestack-ai-kb")

	v.SetDefault("report.directory", constants.DefaultDataDir+"/reports")
}

//Personal.AI order the ending
