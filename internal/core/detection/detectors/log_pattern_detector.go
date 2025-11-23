package detectors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/detection/models"
)

// LogPatternDetector implements detection based on log patterns (e.g., error rate).
type LogPatternDetector struct {
	errorThreshold int // Max number of errors allowed in the log set
}

// NewLogPatternDetector creates a new LogPatternDetector.
func NewLogPatternDetector(errorThreshold int) *LogPatternDetector {
	return &LogPatternDetector{
		errorThreshold: errorThreshold,
	}
}

// Name returns the name of the detector.
func (d *LogPatternDetector) Name() string {
	return "LogPatternDetector"
}

// Detect performs the detection logic.
func (d *LogPatternDetector) Detect(ctx context.Context, input *models.DetectionInput) (*models.DetectionResult, error) {
	var anomalies []models.Anomaly

	if len(input.Logs) == 0 {
		return &models.DetectionResult{DetectedAt: time.Now()}, nil
	}

	errorCount := 0
	var firstErrorTime, lastErrorTime time.Time

	for _, log := range input.Logs {
		level := strings.ToUpper(log.Level)
		if level == "ERROR" || level == "CRITICAL" || level == "FATAL" {
			errorCount++
			if firstErrorTime.IsZero() || log.Timestamp.Before(firstErrorTime) {
				firstErrorTime = log.Timestamp
			}
			if lastErrorTime.IsZero() || log.Timestamp.After(lastErrorTime) {
				lastErrorTime = log.Timestamp
			}
		}
	}

	if errorCount > d.errorThreshold {
		severity := models.SeverityMedium
		if errorCount > d.errorThreshold*5 {
			severity = models.SeverityCritical
		} else if errorCount > d.errorThreshold*2 {
			severity = models.SeverityHigh
		}

		anomalies = append(anomalies, models.Anomaly{
			Type:        models.AnomalyTypeLogPattern,
			Severity:    severity,
			Description: fmt.Sprintf("Found %d error/critical logs, exceeding threshold of %d", errorCount, d.errorThreshold),
			StartTime:   firstErrorTime,
			EndTime:     lastErrorTime,
			Metadata: map[string]string{
				"error_count": fmt.Sprintf("%d", errorCount),
			},
		})
	}

	return &models.DetectionResult{
		Anomalies:  anomalies,
		Confidence: 0.9,
		DetectedAt: time.Now(),
	}, nil
}
