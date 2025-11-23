package detection

import (
	"context"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/core/detection/detectors"
	"github.com/kubestack-ai/kubestack-ai/internal/core/detection/models"
)

// AnomalyDetector orchestrates multiple detectors.
type AnomalyDetector struct {
	detectors []Detector
}

// NewAnomalyDetector creates a new AnomalyDetector with sub-detectors initialized from config.
// If cfg is nil, it falls back to default hardcoded thresholds.
func NewAnomalyDetector(cfg *config.Config) *AnomalyDetector {
	var thresholds map[string]float64

	if cfg != nil && len(cfg.Detection.Thresholds) > 0 {
		// Flatten or select specific middleware thresholds
		// For simplicity, we can merge all middleware thresholds or pass the map to the detector
		// The current simple ThresholdDetector takes a flat map[string]float64
		// In a real app, we might want context-aware thresholds (e.g. separate for Redis vs MySQL)
		// For now, let's just use "default" or merge them.

		thresholds = make(map[string]float64)
		for _, mwThresh := range cfg.Detection.Thresholds {
			for k, v := range mwThresh {
				thresholds[k] = v
			}
		}
	} else {
		// Initialize with default detectors.
		thresholds = map[string]float64{
			"cpu":         90.0,
			"memory":      90.0,
			"connections": 5000.0,
		}
	}

	return &AnomalyDetector{
		detectors: []Detector{
			detectors.NewThresholdDetector(thresholds),
			detectors.NewTimeSeriesDetector(3.0),
			detectors.NewLogPatternDetector(10), // 10 errors threshold
		},
	}
}

// Detect runs all registered detectors and aggregates the results.
func (ad *AnomalyDetector) Detect(ctx context.Context, input *models.DetectionInput) (*models.DetectionResult, error) {
	var allAnomalies []models.Anomaly

	// Simply run all detectors sequentially (could be parallelized)
	for _, d := range ad.detectors {
		res, err := d.Detect(ctx, input)
		if err != nil {
			// Log error but continue with other detectors?
			// For now, let's just print/log and continue
			fmt.Printf("Detector %s failed: %v\n", d.Name(), err)
			continue
		}
		if res != nil {
			allAnomalies = append(allAnomalies, res.Anomalies...)
		}
	}

	return &models.DetectionResult{
		Anomalies:  allAnomalies,
		Confidence: 1.0, // Aggregate confidence could be calculated
		DetectedAt: time.Now(),
	}, nil
}
