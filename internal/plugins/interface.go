package plugins

import (
	"context"

	"github.com/turtacn/kubestack-ai/internal/models"
)

// Plugin 接口定义插件行为。Plugin interface defines plugin behaviors.
type Plugin interface {
	// 基础信息
	Name() string
	Version() string
	SupportedMiddlewareVersions() []string

	// 生命周期管理
	Initialize(config PluginConfig) error
	Validate() error
	Cleanup() error

	// 核心功能
	Diagnose(ctx context.Context, target DiagnosticTarget) (*models.DiagnosisResult, error)
	Analyze(ctx context.Context, metrics models.Metrics) (*AnalysisResult, error)
	Repair(ctx context.Context, issue models.Finding) (*RepairResult, error)

	// 数据收集
	CollectMetrics(ctx context.Context) (*models.Metrics, error)
	CollectLogs(ctx context.Context) (models.Logs, error)
	CollectConfig(ctx context.Context) (*models.Config, error)
}

// PluginConfig 插件配置。Plugin configuration.
type PluginConfig map[string]interface{}

// DiagnosticTarget 诊断目标。Diagnostic target specification.
type DiagnosticTarget struct {
	Namespace      string
	ResourceName   string
	Labels         map[string]string
	SpecificChecks []string
}

// AnalysisResult 分析结果。Analysis result structure.
type AnalysisResult struct {
	HealthScore     float64
	IssuesFound     int
	Recommendations []models.Recommendation
}

// RepairResult 修复结果。Repair result structure.
type RepairResult struct {
	Success           bool
	Message           string
	AffectedResources []string
	DurationMs        int64
}

//Personal.AI order the ending
