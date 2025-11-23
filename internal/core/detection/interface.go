package detection

import (
	"context"
	"github.com/kubestack-ai/kubestack-ai/internal/core/detection/models"
)

// Detector is the interface that all anomaly detectors must implement.
type Detector interface {
	// Detect performs anomaly detection on the given input.
	Detect(ctx context.Context, input *models.DetectionInput) (*models.DetectionResult, error)

	// Name returns the name of the detector.
	Name() string
}
