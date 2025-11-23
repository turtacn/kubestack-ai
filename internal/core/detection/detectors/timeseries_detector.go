package detectors

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/detection/models"
)

// TimeSeriesDetector implements detection based on statistical analysis of time series data.
type TimeSeriesDetector struct {
	threshold float64 // Z-score threshold
}

// NewTimeSeriesDetector creates a new TimeSeriesDetector.
// threshold is typically around 2.0 or 3.0 for Z-score.
func NewTimeSeriesDetector(threshold float64) *TimeSeriesDetector {
	return &TimeSeriesDetector{
		threshold: threshold,
	}
}

// Name returns the name of the detector.
func (d *TimeSeriesDetector) Name() string {
	return "TimeSeriesDetector"
}

// Detect performs the detection logic.
func (d *TimeSeriesDetector) Detect(ctx context.Context, input *models.DetectionInput) (*models.DetectionResult, error) {
	var anomalies []models.Anomaly

	if len(input.TimeSeries) < 2 {
		return &models.DetectionResult{DetectedAt: time.Now()}, nil
	}

	mean, stdDev := d.calculateStats(input.TimeSeries)

	// Avoid division by zero
	if stdDev == 0 {
		return &models.DetectionResult{DetectedAt: time.Now()}, nil
	}

	for _, point := range input.TimeSeries {
		zScore := math.Abs((point.Value - mean) / stdDev)

		if zScore > d.threshold {
			anomalyType := d.classifyAnomaly(point.Value, mean)
			anomalies = append(anomalies, models.Anomaly{
				Type:        anomalyType,
				Severity:    d.calculateSeverity(zScore),
				Description: fmt.Sprintf("Value %.2f deviates from mean %.2f (Z-score=%.2f) at %s", point.Value, mean, zScore, point.Time.Format(time.RFC3339)),
				StartTime:   point.Time,
				EndTime:     point.Time,
				Metadata: map[string]string{
					"z_score": fmt.Sprintf("%.2f", zScore),
					"mean":    fmt.Sprintf("%.2f", mean),
				},
			})
		}
	}

	return &models.DetectionResult{
		Anomalies:  anomalies,
		Confidence: 0.85,
		DetectedAt: time.Now(),
	}, nil
}

func (d *TimeSeriesDetector) calculateStats(series []models.DataPoint) (float64, float64) {
	var sum, sumSq float64
	for _, point := range series {
		sum += point.Value
		sumSq += point.Value * point.Value
	}
	n := float64(len(series))
	mean := sum / n
	variance := (sumSq / n) - (mean * mean)
	// Variance can be slightly negative due to floating point errors
	if variance < 0 {
		variance = 0
	}
	stdDev := math.Sqrt(variance)
	return mean, stdDev
}

func (d *TimeSeriesDetector) classifyAnomaly(value, mean float64) string {
	if value > mean {
		return models.AnomalyTypeTrafficSpike
	}
	return models.AnomalyTypeTrafficDrop
}

func (d *TimeSeriesDetector) calculateSeverity(zScore float64) string {
	if zScore > 5.0 {
		return models.SeverityCritical
	} else if zScore > 4.0 {
		return models.SeverityHigh
	} else if zScore > 3.0 {
		return models.SeverityMedium
	}
	return models.SeverityLow
}
