package collectors

import (
	"context"
	"time"

	"github.com/turtacn/kubestack-ai/internal/models"
)

// Collector 接口定义数据采集行为。Collector interface for data collection.
type Collector interface {
	// 获取Pod/实例状态。Get pod/instance status.
	GetInstanceStatus(ctx context.Context) (string, error)

	// 获取资源使用情况。Get resource usage.
	GetResourceUsage(ctx context.Context) (models.Metrics, error)

	// 获取日志。Get logs.
	GetLogs(ctx context.Context, since time.Duration) (models.Logs, error)

	// 获取网络信息。Get network information.
	GetNetworkInfo(ctx context.Context) (models.Metrics, error)

	// 获取事件。Get events.
	GetEvents(ctx context.Context, since time.Duration) ([]string, error)

	// 获取环境信息。Get environment information.
	GetEnvironmentInfo(ctx context.Context) (models.Config, error)
}

//Personal.AI order the ending
