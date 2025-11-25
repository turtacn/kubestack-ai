package plugin

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// DiagnosticPlugin 诊断插件接口 (P4)
type DiagnosticPlugin interface {
	// 插件名称
	Name() string

	// 支持的中间件类型
	SupportedTypes() []string

	// 插件版本
	Version() string

	// 初始化插件
	Init(config map[string]interface{}) error

	// 执行诊断
	Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error)

	// 关闭插件，释放资源
	Shutdown() error
}

// 插件元数据
type PluginMetadata struct {
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	APIVersion     string   `json:"api_version"` // Added APIVersion
	SupportedTypes []string `json:"supported_types"`
	Author         string   `json:"author"`
	Description    string   `json:"description"`
}

// PluginFactory 插件工厂接口
type PluginFactory interface {
	Create() Plugin
	Metadata() *PluginMetadata
}

// Plugin 插件核心接口 (Legacy, keeping for compatibility if needed, but DiagnosticPlugin is the main focus for P4)
type Plugin interface {
	// 元数据
	Name() string
	Version() string
	Description() string
	SupportedMiddlewareVersions() []string // 如 ["6.x", "7.x"] for Redis

	// 生命周期
	Initialize(config *PluginConfig) error
	Shutdown() error

	// 能力接口
	Collector() DataCollector
	Parser() MetricParser
	HealthChecker() HealthChecker
}

// DataCollector 数据收集接口
type DataCollector interface {
	// 收集原始数据（日志、命令输出、API响应等）
	Collect(ctx context.Context, target *Target) (*CollectedData, error)

	// 支持的数据源类型
	SupportedDataSources() []DataSourceType
}

// MetricParser 指标解析接口
type MetricParser interface {
	// 将原始数据解析为结构化指标
	Parse(ctx context.Context, data *CollectedData) (*ParsedMetrics, error)

	// 返回支持的指标列表
	AvailableMetrics() []MetricDefinition
}

// HealthChecker 健康检查接口
type HealthChecker interface {
	// 执行健康检查
	Check(ctx context.Context, target *Target) (*HealthStatus, error)

	// 健康检查项列表
	CheckItems() []HealthCheckItem
}

// PluginConfig 插件配置
type PluginConfig struct {
	Name     string
	Enabled  bool
	Settings map[string]interface{}
}

// DataSourceType 数据源类型
type DataSourceType string

const (
	DataSourceCommand DataSourceType = "command"
	DataSourceLog     DataSourceType = "log"
	DataSourceAPI     DataSourceType = "api"
)

// Standardized Data Structures

type CollectedData struct {
	PluginName string
	Target     *Target
	Timestamp  time.Time
	RawData    map[string]interface{} // 原始数据：key=数据源类型, value=内容
	Metadata   map[string]string
}

type ParsedMetrics struct {
	PluginName string
	Timestamp  time.Time
	Metrics    map[string]*MetricValue // key=指标名, value=指标值+元数据
}

type MetricValue struct {
	Name      string
	Value     interface{}
	Unit      string
	Labels    map[string]string
	Threshold *Threshold // 可选：阈值定义
}

type Threshold struct {
	Warning  float64
	Critical float64
}

type MetricDefinition struct {
	Name        string
	Unit        string
	Description string
}

type HealthStatus struct {
	PluginName string
	Overall    HealthLevel // Healthy, Degraded, Unhealthy
	Items      []*HealthCheckResult
	Timestamp  time.Time
	Summary    string
}

type HealthCheckResult struct {
	Name    string
	Status  HealthLevel
	Message string
	Details map[string]interface{}
}

type HealthCheckItem struct {
	Name        string
	Description string
}

type HealthLevel int

const (
	HealthyLevel   HealthLevel = iota
	DegradedLevel
	UnhealthyLevel
)

func (h HealthLevel) String() string {
	switch h {
	case HealthyLevel:
		return "Healthy"
	case DegradedLevel:
		return "Degraded"
	case UnhealthyLevel:
		return "Unhealthy"
	default:
		return "Unknown"
	}
}

// Target 诊断目标
type Target struct {
	Type        string // redis, kafka, mysql, elasticsearch
	Address     string // 连接地址
	Credentials *Credentials
	Options     map[string]string
}

type Credentials struct {
	Username  string
	Password  string
	TLSConfig *tls.Config
}
