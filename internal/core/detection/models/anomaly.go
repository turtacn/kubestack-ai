package models

import (
	"time"
)

// AnomalyType constants
const (
	AnomalyTypeHighCPU       = "HighCPU"
	AnomalyTypeHighMemory    = "HighMemory"
	AnomalyTypeHighConnections = "HighConnections"
	AnomalyTypeTrafficSpike  = "TrafficSpike"
	AnomalyTypeTrafficDrop   = "TrafficDrop"
	AnomalyTypeSlowQuery     = "SlowQuery"
	AnomalyTypeLogPattern    = "LogPattern"
)

// Severity levels
const (
	SeverityLow      = "LOW"
	SeverityMedium   = "MEDIUM"
	SeverityHigh     = "HIGH"
	SeverityCritical = "CRITICAL"
)

// Anomaly represents a detected anomaly.
type Anomaly struct {
	Type        string            `json:"type"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// DetectionResult holds the output of a detection run.
type DetectionResult struct {
	Anomalies  []Anomaly `json:"anomalies"`
	Confidence float64   `json:"confidence"`
	DetectedAt time.Time `json:"detected_at"`
}

// DetectionInput holds the data needed for detection.
// It is expected to be populated by data collectors.
type DetectionInput struct {
	Metrics     *Metrics          `json:"metrics,omitempty"`
	Logs        []LogEntry        `json:"logs,omitempty"`
	TimeSeries  []DataPoint       `json:"time_series,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
}

// Metrics is a generic struct for common metrics.
type Metrics struct {
	CPUUsage        float64 `json:"cpu_usage"`
	MemoryUsage     float64 `json:"memory_usage"`
	ConnectionCount int     `json:"connection_count"`
	// Add other common metrics as needed
}

// LogEntry represents a single log line or structured log object.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// DataPoint represents a point in a time series.
type DataPoint struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}
