package redis

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/turtacn/kubestack-ai/internal/errors"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/models"
	"github.com/turtacn/kubestack-ai/internal/plugins"
)

// RedisPlugin Redis插件实现。RedisPlugin implements Plugin for Redis.
type RedisPlugin struct {
	client     *redis.Client
	config     plugins.PluginConfig
	version    string
	initialized bool
}

// Name 返回名称。Name returns plugin name.
func (p *RedisPlugin) Name() string {
	return "redis"
}

// Version 返回插件版本。Version returns plugin version.
func (p *RedisPlugin) Version() string {
	return "1.0.0"
}

// SupportedMiddlewareVersions 返回支持的Redis版本。SupportedMiddlewareVersions returns supported Redis versions.
func (p *RedisPlugin) SupportedMiddlewareVersions() []string {
	return []string{"6.x", "7.x", "8.x"}
}

// Initialize 初始化插件。Initialize initializes the plugin.
func (p *RedisPlugin) Initialize(config plugins.PluginConfig) error {
	logging.Logger.Info("Initializing Redis plugin")
	
	// 存储配置。Store config.
	p.config = config
	
	// 解析配置参数。Parse config parameters.
	addr := "localhost:6379"
	if a, ok := config["address"].(string); ok {
		addr = a
	}
	
	password := ""
	if pwd, ok := config["password"].(string); ok {
		password = pwd
	}
	
	db := 0
	if d, ok := config["db"].(int); ok {
		db = d
	}
	
	// 创建Redis客户端。Create Redis client.
	p.client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	
	// 测试连接。Test connection.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := p.client.Ping(ctx).Result()
	if err != nil {
		logging.Logger.Errorf("Failed to connect to Redis: %v", err)
		return errors.ErrInvalidConfig
	}
	
	// 获取Redis版本。Get Redis version.
	info, err := p.client.Info(ctx, "server").Result()
	if err != nil {
		logging.Logger.Warnf("Failed to get Redis info: %v", err)
		p.version = "unknown"
	} else {
		for _, line := range strings.Split(info, "\r\n") {
			if strings.HasPrefix(line, "redis_version:") {
				p.version = strings.TrimPrefix(line, "redis_version:")
				break
			}
		}
	}
	
	p.initialized = true
	logging.Logger.Infof("Redis plugin initialized. Connected to Redis version: %s", p.version)
	return nil
}

// Validate 验证插件配置。Validate validates plugin configuration.
func (p *RedisPlugin) Validate() error {
	if !p.initialized {
		return errors.ErrInvalidConfig
	}
	
	ctx, cancel := context.With