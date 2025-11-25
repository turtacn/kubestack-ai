package model

import (
	"time"
)

// MetricPoint represents a single metric data point
type MetricPoint struct {
    Name      string            // Metric name: cpu_usage_percent
    Value     float64           // Metric value: 85.5
    Timestamp time.Time         // Collection timestamp
    Labels    map[string]string // Labels: {instance: "node-1", job: "kubernetes"}
}
