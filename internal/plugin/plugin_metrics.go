package plugin

import (
	"time"

	"go.uber.org/zap"
)

// PluginMetrics manages runtime metrics for plugins.
// Note: Prometheus integration is omitted for now to avoid extra dependencies in this phase,
// but the structure is ready for it.
type PluginMetrics struct {
	logger *zap.Logger
}

func NewPluginMetrics(logger *zap.Logger) *PluginMetrics {
	return &PluginMetrics{
		logger: logger,
	}
}

// RecordCall records a plugin method call.
func (m *PluginMetrics) RecordCall(pluginName, method string, duration time.Duration, err error) {
	// Log slow calls
	if duration > time.Second {
		m.logger.Warn("slow plugin call",
			zap.String("plugin", pluginName),
			zap.String("method", method),
			zap.Duration("duration", duration),
		)
	}
	// Here we would increment Prometheus counters
}

// TrackActiveRequest tracks active requests.
func (m *PluginMetrics) TrackActiveRequest(pluginName string) func() {
	// Here we would increment active request gauge
	return func() {
		// Here we would decrement active request gauge
	}
}
