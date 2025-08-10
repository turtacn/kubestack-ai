package collectors

import "github.com/turtacn/kubestack-ai/internal/models"

// Collector 接口定义数据采集行为。Collector interface for data collection.
type Collector interface {
	GetPodStatus() (string, error)             // 获取Pod状态。Get pod status.
	GetResourceUsage() (models.Metrics, error) // 获取资源使用。Get resource usage.
	GetLogs() (models.Logs, error)             // 获取日志。Get logs.
	// 更多通用检查。More general checks.
}

//Personal.AI order the ending
