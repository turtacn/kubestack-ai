package detectors

import (
	"context"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/detection/models"
)

// ThresholdDetector implements detection based on static thresholds.
type ThresholdDetector struct {
	thresholds map[string]float64
}

// NewThresholdDetector creates a new ThresholdDetector with the given thresholds.
// Supported keys: "cpu", "memory", "connections".
func NewThresholdDetector(thresholds map[string]float64) *ThresholdDetector {
	return &ThresholdDetector{
		thresholds: thresholds,
	}
}

// Name returns the name of the detector.
func (d *ThresholdDetector) Name() string {
	return "ThresholdDetector"
}

// Detect performs the detection logic.
func (d *ThresholdDetector) Detect(ctx context.Context, input *models.DetectionInput) (*models.DetectionResult, error) {
	var anomalies []models.Anomaly

	if input.Metrics == nil {
		return &models.DetectionResult{DetectedAt: time.Now()}, nil
	}

	// Check CPU
	if threshold, ok := d.thresholds["cpu"]; ok {
		if input.Metrics.CPUUsage > threshold {
			anomalies = append(anomalies, models.Anomaly{
				Type:        models.AnomalyTypeHighCPU,
				Severity:    d.calculateSeverity(input.Metrics.CPUUsage, threshold),
				Description: fmt.Sprintf("CPU usage %.2f%% exceeds threshold %.2f%%", input.Metrics.CPUUsage, threshold),
				StartTime:   time.Now(),
				EndTime:     time.Now(),
			})
		}
	}

	// Check Memory
	if threshold, ok := d.thresholds["memory"]; ok {
		if input.Metrics.MemoryUsage > threshold {
			anomalies = append(anomalies, models.Anomaly{
				Type:        models.AnomalyTypeHighMemory,
				Severity:    d.calculateSeverity(input.Metrics.MemoryUsage, threshold),
				Description: fmt.Sprintf("Memory usage %.2f%% exceeds threshold %.2f%%", input.Metrics.MemoryUsage, threshold),
				StartTime:   time.Now(),
				EndTime:     time.Now(),
			})
		}
	}

	// Check Connections
	if threshold, ok := d.thresholds["connections"]; ok {
		if float64(input.Metrics.ConnectionCount) > threshold {
			anomalies = append(anomalies, models.Anomaly{
				Type:        models.AnomalyTypeHighConnections,
				Severity:    models.SeverityHigh, // Generally high connections are high severity
				Description: fmt.Sprintf("Connection count %d exceeds threshold %d", input.Metrics.ConnectionCount, int(threshold)),
				StartTime:   time.Now(),
				EndTime:     time.Now(),
			})
		}
	}

	return &models.DetectionResult{
		Anomalies:  anomalies,
		Confidence: 1.0, // Static thresholds have high confidence in what they detected (it's binary)
		DetectedAt: time.Now(),
	}, nil
}

func (d *ThresholdDetector) calculateSeverity(value, threshold float64) string {
	ratio := value / threshold
	if ratio >= 1.5 {
		return models.SeverityCritical
	} else if ratio >= 1.2 {
		return models.SeverityHigh
	} else if ratio >= 1.0 {
		return models.SeverityMedium
	}
	return models.SeverityLow
}
