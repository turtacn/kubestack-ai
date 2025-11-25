package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/model"
)

// MiddlewareCollector bridges existing Plugin's MetricsCollector
type MiddlewareCollector struct {
	middlewareName string
	pluginManager  interfaces.PluginManager
}

// NewMiddlewareCollector creates a new middleware collector
func NewMiddlewareCollector(middlewareName string, pluginManager interfaces.PluginManager) *MiddlewareCollector {
	return &MiddlewareCollector{
		middlewareName: middlewareName,
		pluginManager:  pluginManager,
	}
}

func (c *MiddlewareCollector) Collect(ctx context.Context) ([]*model.MetricPoint, error) {
	// Load or Get the plugin
	p, err := c.pluginManager.LoadPlugin(c.middlewareName)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin %s: %w", c.middlewareName, err)
	}

	// Call CollectMetrics
	metricsData, err := p.CollectMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics for %s: %w", c.middlewareName, err)
	}

	// Convert metricsData to model.MetricPoint
	points := make([]*model.MetricPoint, 0)
	if metricsData == nil || metricsData.Data == nil {
		return points, nil
	}

	for name, value := range metricsData.Data {
		// Handle value conversion if needed (assuming value is float64 or convertible)
		var floatVal float64
		switch v := value.(type) {
		case float64:
			floatVal = v
		case int:
			floatVal = float64(v)
		case int64:
			floatVal = float64(v)
		default:
			continue // Skip unsupported types
		}

		points = append(points, &model.MetricPoint{
			Name:      fmt.Sprintf("%s_%s", c.middlewareName, name),
			Value:     floatVal,
			Timestamp: time.Now(),
			Labels:    map[string]string{"type": c.middlewareName},
		})
	}

	return points, nil
}

func (c *MiddlewareCollector) Name() string {
	return "middleware-" + c.middlewareName
}

func (c *MiddlewareCollector) Interval() time.Duration {
	return 30 * time.Second
}
