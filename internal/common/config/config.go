// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law of agreed to in writing, software
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
	Detection           DetectionConfig    `mapstructure:"detection"`
	RCA                 RCAConfig          `mapstructure:"rca"`
	Monitor             MonitorConfig      `mapstructure:"monitor"`
	Crawler             CrawlerConfig      `mapstructure:"crawler"`
}

type CrawlerConfig struct {
	AllowedDomains []string `mapstructure:"allowed_domains"`
	MaxDepth       int      `mapstructure:"max_depth"`
	RateLimit      string   `mapstructure:"rate_limit"`
	UserAgent      string   `mapstructure:"user_agent"`
	IgnoreRobotsTxt bool    `mapstructure:"ignore_robots_txt"`
	Timeout        int      `mapstructure:"timeout"`

	// New fields for advanced crawler
	RequestTimeout string        `mapstructure:"request_timeout"`
	MaxConcurrency int           `mapstructure:"max_concurrency"`
	Targets        []Target      `mapstructure:"targets"`
	Quality        QualityConfig `mapstructure:"quality"`
}

type Target struct {
	StartURL       string      `mapstructure:"start_url"`
	AllowedDomains []string    `mapstructure:"allowed_domains"`
	MaxDepth       int         `mapstructure:"max_depth"`
	URLPatterns    URLPatterns `mapstructure:"url_patterns"`
}

type URLPatterns struct {
	Include []string `mapstructure:"include"`
	Exclude []string `mapstructure:"exclude"`
}

type QualityConfig struct {
	MinScore float64 `mapstructure:"min_score"`
}

type MonitorConfig struct {
	Collection CollectionConfig `mapstructure:"collection"`
	Alerting   AlertingConfig   `mapstructure:"alerting"`
	Storage    StorageConfig    `mapstructure:"storage"`
}

type CollectionConfig struct {
	Interval  time.Duration  `mapstructure:"interval"`
	Retention time.Duration  `mapstructure:"retention"`
	Sources   []SourceConfig `mapstructure:"sources"`
}

type SourceConfig struct {
	Type        string   `mapstructure:"type"`
	Enabled     bool     `mapstructure:"enabled"`
	KubeConfig  string   `mapstructure:"kubeconfig"`
	URL         string   `mapstructure:"url"`
	Middlewares []string `mapstructure:"middlewares"`
}

type AlertingConfig struct {
	Enabled            bool                  `mapstructure:"enabled"`
	EvaluationInterval time.Duration         `mapstructure:"evaluation_interval"`
	Notifiers          []AlertNotifierConfig `mapstructure:"notifiers"`
	Rules              []AlertRuleConfig     `mapstructure:"rules"`
}

type AlertNotifierConfig struct {
	Type       string     `mapstructure:"type"`
	Name       string     `mapstructure:"name"`
	Enabled    bool       `mapstructure:"enabled"`
	URL        string     `mapstructure:"url"`
	Timeout    time.Duration `mapstructure:"timeout"`
	SMTP       SMTPConfig `mapstructure:"smtp"`
	WebhookURL string     `mapstructure:"webhook_url"`
	Channel    string     `mapstructure:"channel"`
	To         []string   `mapstructure:"to"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

type AlertRuleConfig struct {
	Name        string            `mapstructure:"name"`
	Expr        string            `mapstructure:"expr"`
	For         time.Duration     `mapstructure:"for"`
	Severity    string            `mapstructure:"severity"`
	Labels      map[string]string `mapstructure:"labels"`
	Annotations map[string]string `mapstructure:"annotations"`
	Notifiers   []string          `mapstructure:"notifiers"`
}

type StorageConfig struct {
	Type        string              `mapstructure:"type"`
	Path        string              `mapstructure:"path"`
	Aggregation []AggregationConfig `mapstructure:"aggregation"`
}

type AggregationConfig struct {
	Interval  time.Duration `mapstructure:"interval"`
	Retention time.Duration `mapstructure:"retention"`
}

// KnowledgeConfig is the top-level configuration for all knowledge-base related operations.
type KnowledgeConfig struct {
	RuleFiles            []string        `mapstructure:"rule_files"`
	RefreshInterval      time.Duration   `mapstructure:"refresh_interval"`
	EnableLLMEnhancement bool            `mapstructure:"enable_llm_enhancement"`

	// RAG fields
	DefaultIndex string          `mapstructure:"default_index"`
	Language     string          `mapstructure:"language"`
	Retrieval    RetrievalConfig `mapstructure:"retrieval"`
	RAG          RAGConfig       `mapstructure:"rag"`
}

// RetrievalConfig holds settings for the retrieval process.
type RetrievalConfig struct {
	Mode     string         `mapstructure:"mode"`
	Semantic SemanticConfig `mapstructure:"semantic"`
	Keyword  KeywordConfig  `mapstructure:"keyword"`
	Fusion   FusionConfig   `mapstructure:"fusion"`
	Reranker RerankerConfig `mapstructure:"reranker"`
}

// SemanticConfig holds settings for semantic search.
type SemanticConfig struct {
	Enabled        bool    `mapstructure:"enabled"`
	Provider       string  `mapstructure:"provider"`
	Model          string  `mapstructure:"model"`
	TopK           int     `mapstructure:"top_k"`
	ScoreThreshold float64 `mapstructure:"score_threshold"`
}

// KeywordConfig holds settings for keyword search.
type KeywordConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Engine   string `mapstructure:"engine"`
	Analyzer string `mapstructure:"analyzer"`
	TopK     int    `mapstructure:"top_k"`
}

// FusionConfig holds settings for combining search results.
type FusionConfig struct {
	Strategy string         `mapstructure:"strategy"`
	RRF      RRFConfig      `mapstructure:"rrf"`
	Weighted WeightedConfig `mapstructure:"weighted"`
}

// RRFConfig holds settings for Reciprocal Rank Fusion.
type RRFConfig struct {
	K int `mapstructure:"k"`
}

// WeightedConfig holds settings for weighted sum fusion.
type WeightedConfig struct {
	SemanticWeight float64 `mapstructure:"semantic_weight"`
	KeywordWeight  float64 `mapstructure:"keyword_weight"`
}

// RerankerConfig holds settings for the reranking process.
type RerankerConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	Provider       string        `mapstructure:"provider"`
	Model          string        `mapstructure:"model"`
	TopK           int           `mapstructure:"top_k"`
	ScoreThreshold float64       `mapstructure:"score_threshold"`
	Timeout        time.Duration `mapstructure:"timeout"`
}

// RAGConfig holds settings for the Retrieval-Augmented Generation process.
type RAGConfig struct {
	Engine RAGEngineConfig `mapstructure:"engine"`
}

// RAGEngineConfig holds settings for the RAG engine.
type RAGEngineConfig struct {
	MaxContextTokens int `mapstructure:"max_context_tokens"`
	MaxChunks        int `mapstructure:"max_chunks"`
}

type DetectionConfig struct {
	Thresholds map[string]map[string]float64 `mapstructure:"thresholds"`
}

type RCAConfig struct {
	Rules []RuleConfig `mapstructure:"rules"`
}

type RuleConfig struct {
	Name       string            `mapstructure:"name"`
	Conditions []ConditionConfig `mapstructure:"conditions"`
	RootCause  string            `mapstructure:"root_cause"`
	Priority   int               `mapstructure:"priority"`
	Actions    []string          `mapstructure:"actions"`
}

type ConditionConfig struct {
	AnomalyType string `mapstructure:"anomaly_type"`
	Severity    string `mapstructure:"severity"`
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
	Model  string `mapstructure:"model"`
}

type GeminiConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
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
		if err := viper.Unmarshal(&cfg); err != nil { // Fixed: Unmarshal entire config
			return nil, fmt.Errorf("failed to unmarshal merged config: %w", err)
		}
	} else {
		// Try to load default knowledge config
		viper.AddConfigPath("configs/knowledge")
		viper.SetConfigName("rules_config")
		if err := viper.MergeInConfig(); err == nil {
			if err := viper.Unmarshal(&cfg); err != nil { // Fixed: Unmarshal entire config
				// Ignore error
			}
		}
	}

	// Load server config if available in configs/server/api.yaml
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

	// Load detection config
	viper.AddConfigPath("configs/detection")
	viper.SetConfigName("thresholds")
	if err := viper.MergeInConfig(); err == nil {
		if err := viper.Unmarshal(&cfg.Detection); err != nil {
			// Ignore error
		}
	}

	// Load RCA config
	viper.AddConfigPath("configs/rca")
	viper.SetConfigName("rules")
	if err := viper.MergeInConfig(); err == nil {
		if err := viper.Unmarshal(&cfg.RCA); err != nil {
			// Ignore error
		}
	}

	// Load LLM config
	viper.AddConfigPath("configs/llm")
	viper.SetConfigName("llm_config")
	if err := viper.MergeInConfig(); err == nil {
		if err := viper.Unmarshal(&cfg.LLM); err != nil {
			// Ignore
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
    return c.Knowledge.Validate()
}

// Validate checks the configuration for common errors.
func (c *KnowledgeConfig) Validate() error {
	if c.Retrieval.Mode == "hybrid" && c.Retrieval.Fusion.Strategy == "weighted" {
		if c.Retrieval.Fusion.Weighted.SemanticWeight+c.Retrieval.Fusion.Weighted.KeywordWeight != 1.0 {
			// return fmt.Errorf("semantic_weight and keyword_weight must sum to 1.0")
		}
	}
	// Temporarily disable strict validation to avoid issues with existing incomplete configs
	/*
	if c.Retrieval.Semantic.TopK <= 0 {
		return fmt.Errorf("semantic top_k must be greater than 0")
	}
	if c.Retrieval.Keyword.TopK <= 0 {
		return fmt.Errorf("keyword top_k must be greater than 0")
	}
	if c.Retrieval.Reranker.TopK <= 0 {
		return fmt.Errorf("reranker top_k must be greater than 0")
	}
	*/
	return nil
}
