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

package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/spf13/viper"
)

// Config is the top-level configuration for the application.
type Config struct {
	KnowledgeConfigPath string             `mapstructure:"knowledge_config_path"`
	Knowledge           KnowledgeConfig    `mapstructure:"knowledge"`
	LLM                 LLMConfig          `mapstructure:"llm"`
	Server              ServerConfig       `mapstructure:"server"`
	Auth                AuthConfig         `mapstructure:"auth"`
	RBAC                RBACConfig         `mapstructure:"rbac"`
	WebSocket           WebSocketConfig    `mapstructure:"websocket"`
	Logger              logger.Config      `mapstructure:"logger"`
	Plugins             PluginConfig       `mapstructure:"plugins"`
	TaskQueue           TaskQueueConfig    `mapstructure:"task_queue"`
	Notification        NotificationConfig `mapstructure:"notification"`
}

type TaskQueueConfig struct {
	Type  string      `mapstructure:"type"`
	Redis RedisConfig `mapstructure:"redis"`
}

type RedisConfig struct {
	Addr      string `mapstructure:"addr"`
	Password  string `mapstructure:"password"`
	DB        int    `mapstructure:"db"`
	QueueName string `mapstructure:"queue_name"`
}

type NotificationConfig struct {
	Webhook WebhookConfig `mapstructure:"webhook"`
	Email   EmailConfig   `mapstructure:"email"`
}

type WebhookConfig struct {
	URL string `mapstructure:"url"`
}

type EmailConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	From      string `mapstructure:"from"`
	DefaultTo string `mapstructure:"default_to"`
}

type PluginConfig struct {
	Directory string `mapstructure:"directory"`
}

type LLMConfig struct {
	Provider string       `mapstructure:"provider"`
	OpenAI   OpenAIConfig `mapstructure:"openai"`
	Gemini   GeminiConfig `mapstructure:"gemini"`
}

type OpenAIConfig struct {
	APIKey string `mapstructure:"api_key"`
}

type GeminiConfig struct {
	APIKey string `mapstructure:"api_key"`
}

type ServerConfig struct {
	Port int        `mapstructure:"port"`
	TLS  TLSConfig  `mapstructure:"tls"`
	CORS CORSConfig `mapstructure:"cors"`
}

type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
}

type AuthConfig struct {
	JWTSecret      string        `mapstructure:"jwt_secret"`
	TokenTTL       time.Duration `mapstructure:"token_ttl"`
	RefreshEnabled bool          `mapstructure:"refresh_enabled"`
}

type RBACConfig struct {
	Roles map[string]RoleConfig `mapstructure:"roles"`
}

type RoleConfig struct {
	Permissions []string `mapstructure:"permissions"`
}

type WebSocketConfig struct {
	PingInterval   time.Duration `mapstructure:"ping_interval"`
	MaxConnections int           `mapstructure:"max_connections"`
}

// LoadConfig loads the configuration from the specified file.
func LoadConfig(cfgFile string) (*Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".kubestack-ai")
	}

	viper.SetEnvPrefix("KSA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.KnowledgeConfigPath != "" {
		viper.SetConfigFile(cfg.KnowledgeConfigPath)
		if err := viper.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("failed to merge knowledge config: %w", err)
		}
		if err := viper.Unmarshal(&cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal merged config: %w", err)
		}
	}

	// Load server config if available in configs/server/api.yaml
	// This is a bit of a hack for development, in production it should be part of the main config
	viper.AddConfigPath("configs/server")
	viper.SetConfigName("api")
	if err := viper.MergeInConfig(); err == nil {
		if err := viper.Unmarshal(&cfg); err != nil {
			// Ignore error if server config is missing
		}
	}

	// Load task queue config
	viper.AddConfigPath("configs/task")
	viper.SetConfigName("queue_config")
	if err := viper.MergeInConfig(); err == nil {
		if err := viper.Unmarshal(&cfg.TaskQueue); err != nil {
			// Ignore error
		}
	}

	// Load notification config
	viper.AddConfigPath("configs/notification")
	viper.SetConfigName("notification_config")
	if err := viper.MergeInConfig(); err == nil {
		if err := viper.Unmarshal(&cfg.Notification); err != nil {
			// Ignore error
		}
	}

	if err := cfg.Knowledge.Validate(); err != nil {
		return nil, fmt.Errorf("invalid knowledge config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
    return c.Knowledge.Validate()
}
