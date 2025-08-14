package plugins

import "github.com/turtacn/kubestack-ai/internal/models"

// Plugin 接口定义插件行为。Plugin interface defines plugin behaviors.
type Plugin interface {
	Name() string                            // 插件名称。Plugin name.
	CollectMetrics() (models.Metrics, error) // 采集指标。Collect metrics.
	AnalyzeLogs() (models.Logs, error)       // 分析日志。Analyze logs.
	ValidateConfig() (models.Config, error)  // 验证配置。Validate config.
	// 特定诊断。Specific diagnosis.
	Diagnose() ([]models.Finding, error)
}

//Personal.AI order the ending
