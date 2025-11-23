package detection_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/kubestack-ai/kubestack-ai/internal/core/detection/detectors"
	"github.com/kubestack-ai/kubestack-ai/internal/core/detection/models"
)

func TestThresholdDetector(t *testing.T) {
	thresholds := map[string]float64{"cpu": 90.0}
	detector := detectors.NewThresholdDetector(thresholds)
	metrics := &models.Metrics{CPUUsage: 95.0}
	input := &models.DetectionInput{Metrics: metrics}

	result, err := detector.Detect(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result.Anomalies)
	assert.Equal(t, 1, len(result.Anomalies))
	assert.Equal(t, models.AnomalyTypeHighCPU, result.Anomalies[0].Type)
	assert.Equal(t, models.SeverityMedium, result.Anomalies[0].Severity)
}

func TestTimeSeriesDetector(t *testing.T) {
	detector := detectors.NewTimeSeriesDetector(2.0)

	// Use more points to establish a stable baseline
	baseTime := time.Now()
	timeseries := []models.DataPoint{
		{Time: baseTime.Add(-5 * time.Hour), Value: 1000},
		{Time: baseTime.Add(-4 * time.Hour), Value: 990},
		{Time: baseTime.Add(-3 * time.Hour), Value: 1010},
		{Time: baseTime.Add(-2 * time.Hour), Value: 1005},
		{Time: baseTime.Add(-1 * time.Hour), Value: 995},
		{Time: baseTime, Value: 200}, // Anomaly: Significant drop
	}

	input := &models.DetectionInput{TimeSeries: timeseries}

	result, err := detector.Detect(context.Background(), input)

	assert.NoError(t, err)
	assert.NotEmpty(t, result.Anomalies)
	assert.Equal(t, models.AnomalyTypeTrafficDrop, result.Anomalies[0].Type)
}

func TestLogPatternDetector(t *testing.T) {
	detector := detectors.NewLogPatternDetector(2)
	logs := []models.LogEntry{
		{Timestamp: time.Now(), Level: "ERROR", Message: "Err 1"},
		{Timestamp: time.Now(), Level: "ERROR", Message: "Err 2"},
		{Timestamp: time.Now(), Level: "ERROR", Message: "Err 3"},
	}
	input := &models.DetectionInput{Logs: logs}

	result, err := detector.Detect(context.Background(), input)

	assert.NoError(t, err)
	assert.NotEmpty(t, result.Anomalies)
	assert.Equal(t, models.AnomalyTypeLogPattern, result.Anomalies[0].Type)
}
