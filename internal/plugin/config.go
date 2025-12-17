package plugin

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// PluginManagerConfig contains configuration for the plugin manager
type PluginManagerConfig struct {
	PluginDirectory string                    `yaml:"directory" json:"directory"`
	AutoDiscover    bool                      `yaml:"auto_discover" json:"auto_discover"`
	Builtin         map[string]BuiltinConfig  `yaml:"builtin" json:"builtin"`
	Sandbox         SandboxConfig             `yaml:"sandbox" json:"sandbox"`
}

// BuiltinConfig contains configuration for a builtin plugin
type BuiltinConfig struct {
	Enabled  bool                   `yaml:"enabled" json:"enabled"`
	Priority int                    `yaml:"priority" json:"priority"`
	Settings map[string]interface{} `yaml:"settings" json:"settings"`
}

// SandboxConfig contains sandbox configuration
type SandboxConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	Timeout     string `yaml:"timeout" json:"timeout"`
	MemoryLimit string `yaml:"memory_limit" json:"memory_limit"`
}

// LoadPluginManagerConfig loads plugin manager configuration from a file
func LoadPluginManagerConfig(path string) (*PluginManagerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config PluginManagerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	
	return &config, nil
}

// ToPluginConfig converts BuiltinConfig to PluginConfig for registry
func (bc *BuiltinConfig) ToPluginConfig() PluginConfig {
	// Convert to basic PluginConfig compatible with registry
	return PluginConfig{
		Type:    "",  // Will be set by caller
		Connection: nil,
		Options: bc.Settings,
	}
}

// ToSandboxOptions converts SandboxConfig to SandboxOptions
func (sc *SandboxConfig) ToSandboxOptions() SandboxOptions {
	timeout := 5 * time.Minute
	if sc.Timeout != "" {
		if d, err := time.ParseDuration(sc.Timeout); err == nil {
			timeout = d
		}
	}
	
	memLimit := int64(256 * 1024 * 1024) // 256MB default
	// Parse MemoryLimit if needed (e.g., "256Mi")
	
	return SandboxOptions{
		Timeout:     timeout,
		MemoryLimit: memLimit,
		CPULimit:    1.0,
		AllowedOperations: []string{
			"diagnose", "get-metrics", "health-check",
			"get-slow-logs", "get-client-list", "get-config",
		},
	}
}

// DefaultPluginManagerConfig returns default configuration
func DefaultPluginManagerConfig() *PluginManagerConfig {
	return &PluginManagerConfig{
		PluginDirectory: "/etc/ksa/plugins",
		AutoDiscover:    true,
		Builtin: map[string]BuiltinConfig{
			"redis-diagnostics": {
				Enabled:  true,
				Priority: 100,
				Settings: map[string]interface{}{
					"default_timeout":  "30s",
					"max_slow_logs":    100,
				},
			},
			"kafka-diagnostics": {
				Enabled:  true,
				Priority: 100,
				Settings: map[string]interface{}{
					"default_timeout": "60s",
					"lag_threshold":   10000,
				},
			},
			"mysql-diagnostics": {
				Enabled:  true,
				Priority: 100,
				Settings: map[string]interface{}{
					"default_timeout":       "30s",
					"slow_query_threshold":  "1s",
				},
			},
		},
		Sandbox: SandboxConfig{
			Enabled:     true,
			Timeout:     "5m",
			MemoryLimit: "256Mi",
		},
	}
}
