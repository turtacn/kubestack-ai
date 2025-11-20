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

	"github.com/spf13/viper"
)

// Config is the top-level configuration for the application.
type Config struct {
	KnowledgeConfigPath string          `mapstructure:"knowledge_config_path"`
	Knowledge           KnowledgeConfig `mapstructure:"knowledge"`
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

	if err := cfg.Knowledge.Validate(); err != nil {
		return nil, fmt.Errorf("invalid knowledge config: %w", err)
	}

	return &cfg, nil
}
