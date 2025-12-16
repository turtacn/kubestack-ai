package diagnosis

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// Analyzer interface
type Analyzer interface {
	Name() string
	Analyze(ctx context.Context, data *plugin.DiagnosticData) ([]Issue, error)
}

// MetricsHistoryStore defines interface for retrieving historical metrics
type MetricsHistoryStore interface {
	GetHistory(metricName string, duration time.Duration) ([]MetricPoint, error)
}

// MetricPoint represents a single data point in history
type MetricPoint struct {
	Value     float64
	Timestamp time.Time
}

// ThresholdAnalyzer
type ThresholdAnalyzer struct {
	thresholds map[string]ThresholdConfig
}

type ThresholdConfig struct {
	WarningThreshold  float64
	ErrorThreshold    float64
	CriticalThreshold float64
	Unit              string
	Description       string
}

func NewThresholdAnalyzer() *ThresholdAnalyzer {
	return &ThresholdAnalyzer{
		thresholds: defaultThresholds,
	}
}

var defaultThresholds = map[string]ThresholdConfig{
	"memory_usage_ratio": {
		WarningThreshold:  0.7,
		ErrorThreshold:    0.85,
		CriticalThreshold: 0.95,
		Unit:              "ratio",
		Description:       "Memory Usage",
	},
	"connection_usage_ratio": {
		WarningThreshold:  0.7,
		ErrorThreshold:    0.85,
		CriticalThreshold: 0.95,
		Unit:              "ratio",
		Description:       "Connection Usage",
	},
}

func (a *ThresholdAnalyzer) Name() string { return "ThresholdAnalyzer" }

func (a *ThresholdAnalyzer) Analyze(ctx context.Context, data *plugin.DiagnosticData) ([]Issue, error) {
	issues := make([]Issue, 0)
	if data.Metrics == nil {
		return issues, nil
	}

	for metricName, config := range a.thresholds {
		metric, ok := data.Metrics.Metrics[metricName]
		if !ok {
			continue
		}

		var severity plugin.Severity
		var exceeded bool

		if metric.Value >= config.CriticalThreshold {
			severity = plugin.SeverityCritical
			exceeded = true
		} else if metric.Value >= config.ErrorThreshold {
			severity = plugin.SeverityError
			exceeded = true
		} else if metric.Value >= config.WarningThreshold {
			severity = plugin.SeverityWarning
			exceeded = true
		}

		if exceeded {
			issues = append(issues, Issue{
				RuleID:      "threshold-" + metricName,
				Name:        config.Description + " Threshold Exceeded",
				Severity:    severity,
				Description: fmt.Sprintf("%s current value %.2f, exceeded threshold", config.Description, metric.Value),
				Evidence: map[string]interface{}{
					"metric": metricName,
					"value":  metric.Value,
				},
				DetectedAt: time.Now(),
			})
		}
	}
	return issues, nil
}

// AnomalyAnalyzer checks for statistical anomalies
type AnomalyAnalyzer struct {
	historyStore MetricsHistoryStore
	sensitivity  float64
}

func NewAnomalyAnalyzer(store MetricsHistoryStore) *AnomalyAnalyzer {
	return &AnomalyAnalyzer{
		historyStore: store,
		sensitivity:  3.0,
	}
}

func (a *AnomalyAnalyzer) Name() string { return "AnomalyAnalyzer" }

func (a *AnomalyAnalyzer) Analyze(ctx context.Context, data *plugin.DiagnosticData) ([]Issue, error) {
	issues := make([]Issue, 0)

	if a.historyStore == nil || data.Metrics == nil {
		return issues, nil
	}

	for metricName, metric := range data.Metrics.Metrics {
		history, err := a.historyStore.GetHistory(metricName, 24*time.Hour)
		if err != nil || len(history) < 10 {
			continue
		}

		mean, stddev := calculateStats(history)
		zscore := (metric.Value - mean) / stddev

		if math.Abs(zscore) > a.sensitivity {
			issues = append(issues, Issue{
				RuleID:      "anomaly-" + metricName,
				Name:        metricName + " Anomaly",
				Severity:    plugin.SeverityWarning,
				Description: fmt.Sprintf("%s value %.2f is %.2f stddevs from mean", metricName, metric.Value, zscore),
				Evidence: map[string]interface{}{
					"metric": metricName,
					"value":  metric.Value,
					"mean":   mean,
					"zscore": zscore,
				},
				DetectedAt: time.Now(),
			})
		}
	}
	return issues, nil
}

// TrendAnalyzer checks for trends using linear regression
type TrendAnalyzer struct {
	historyStore MetricsHistoryStore
	windowSize   time.Duration
}

func NewTrendAnalyzer(store MetricsHistoryStore) *TrendAnalyzer {
	return &TrendAnalyzer{
		historyStore: store,
		windowSize:   1 * time.Hour,
	}
}

func (a *TrendAnalyzer) Name() string { return "TrendAnalyzer" }

func (a *TrendAnalyzer) Analyze(ctx context.Context, data *plugin.DiagnosticData) ([]Issue, error) {
	issues := make([]Issue, 0)

	if a.historyStore == nil {
		return issues, nil
	}

	targets := []string{"memory_usage_ratio", "cpu_usage_ratio"}
	for _, name := range targets {
		history, err := a.historyStore.GetHistory(name, a.windowSize)
		if err != nil || len(history) < 2 {
			continue
		}

		slope := calculateSlope(history)
		predicted := predictValue(history, slope, 30*time.Minute)

		if slope > 0.001 && predicted > 0.95 { // Simple heuristic
			issues = append(issues, Issue{
				RuleID:      "trend-" + name,
				Name:        name + " Increasing Trend",
				Severity:    plugin.SeverityWarning,
				Description: fmt.Sprintf("%s is increasing, predicted to reach %.2f in 30m", name, predicted),
				Evidence: map[string]interface{}{
					"slope":     slope,
					"predicted": predicted,
				},
				DetectedAt: time.Now(),
			})
		}
	}

	return issues, nil
}

// Helper math functions

func calculateStats(points []MetricPoint) (mean, stddev float64) {
	n := float64(len(points))
	if n == 0 {
		return 0, 1
	}
	var sum float64
	for _, p := range points {
		sum += p.Value
	}
	mean = sum / n
	var sumSquares float64
	for _, p := range points {
		diff := p.Value - mean
		sumSquares += diff * diff
	}
	stddev = math.Sqrt(sumSquares / n)
	if stddev == 0 {
		stddev = 1
	}
	return
}

func calculateSlope(points []MetricPoint) float64 {
	n := float64(len(points))
	var sumX, sumY, sumXY, sumX2 float64
	startTime := points[0].Timestamp.Unix()

	for _, p := range points {
		x := float64(p.Timestamp.Unix() - startTime)
		sumX += x
		sumY += p.Value
		sumXY += x * p.Value
		sumX2 += x * x
	}
	return (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
}

func predictValue(points []MetricPoint, slope float64, future time.Duration) float64 {
	if len(points) == 0 {
		return 0
	}
	last := points[len(points)-1]
	// slope is per second if x is seconds
	return last.Value + slope*float64(future.Seconds())
}
