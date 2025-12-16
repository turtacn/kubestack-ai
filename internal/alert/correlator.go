package alert

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// CorrelatedAlert represents a grouped set of alerts.
type CorrelatedAlert struct {
	Instance   string
	Middleware enum.MiddlewareType
	Alerts     []*models.AlertEvent
	Severity   enum.SeverityLevel
	Summary    string
}

// AlertBucket holds alerts for a specific instance within a time window.
type AlertBucket struct {
	Instance    string
	Alerts      []*models.AlertEvent
	FirstSeen   time.Time
	LastUpdated time.Time
	Middleware  enum.MiddlewareType
}

// Correlator aggregates alerts.
type Correlator struct {
	windowSize  time.Duration
	buckets     map[string]*AlertBucket // instance -> bucket
	bucketMu    sync.RWMutex
	onFlush     func(*CorrelatedAlert)
}

// NewCorrelator creates a new Correlator.
func NewCorrelator(windowSize time.Duration, onFlush func(*CorrelatedAlert)) *Correlator {
	c := &Correlator{
		windowSize: windowSize,
		buckets:    make(map[string]*AlertBucket),
		onFlush:    onFlush,
	}
	go c.flushLoop()
	return c
}

// AddAlert adds an alert to the correlator.
func (c *Correlator) AddAlert(event *models.AlertEvent) (bool, *CorrelatedAlert) {
	c.bucketMu.Lock()
	defer c.bucketMu.Unlock()

	key := event.Instance
	bucket, exists := c.buckets[key]
	if !exists {
		mwType := c.inferMiddlewareType(event)
		bucket = &AlertBucket{
			Instance:    event.Instance,
			Alerts:      []*models.AlertEvent{},
			FirstSeen:   time.Now(),
			Middleware:  mwType,
		}
		c.buckets[key] = bucket
	}

	bucket.Alerts = append(bucket.Alerts, event)
	bucket.LastUpdated = time.Now()

	// If critical, trigger immediately but keep bucket for context?
	// Or just return immediately.
	if event.Severity == enum.SeverityCritical {
		// Flush immediately for this alert? Or return it as correlated single.
		// To keep simple: treat as correlated immediately.
		correlated := c.buildCorrelatedAlert(bucket)
		// We might want to remove bucket or keep it?
		// If we trigger now, we should probably clear the bucket to avoid double trigger later?
		// Or maybe we just return it and let the bucket accumulate more?
		// Let's return true, and NOT clear bucket, but maybe mark as handled?
		// For simplicity, critical alerts bypass aggregation window wait, but include currently aggregated context.
		// And we clear bucket to restart aggregation.
		delete(c.buckets, key)
		return true, correlated
	}

	return false, nil
}

func (c *Correlator) flushLoop() {
	ticker := time.NewTicker(time.Second * 10)
	for range ticker.C {
		c.flushExpiredBuckets()
	}
}

func (c *Correlator) flushExpiredBuckets() {
	c.bucketMu.Lock()
	defer c.bucketMu.Unlock()

	now := time.Now()
	for key, bucket := range c.buckets {
		if now.Sub(bucket.LastUpdated) > c.windowSize {
			correlated := c.buildCorrelatedAlert(bucket)
			if c.onFlush != nil {
				c.onFlush(correlated)
			}
			delete(c.buckets, key)
		}
	}
}

func (c *Correlator) buildCorrelatedAlert(bucket *AlertBucket) *CorrelatedAlert {
	maxSeverity := enum.SeverityInfo
	var summaries []string

	// Determine max severity and collect summaries
	for _, a := range bucket.Alerts {
		if a.Severity < maxSeverity { // Assuming lower int value is higher severity? No, Check enum.
			// Enum: Critical=2, Warning=1, Healthy=0. Wait.
			// SeverityLevel: Low=0, Medium=1, High=2, Warning=3, Critical=4, Info=5.
			// So higher int is usually higher severity except Info.
			// Let's check logic. Critical is 4. Info is 5.
			// We need a helper to compare severity.
		}
		// Let's just take the one that is "most severe".
		if isMoreSevere(a.Severity, maxSeverity) {
			maxSeverity = a.Severity
		}
		summaries = append(summaries, a.Summary)
	}

	return &CorrelatedAlert{
		Instance:   bucket.Instance,
		Middleware: bucket.Middleware,
		Alerts:     bucket.Alerts,
		Severity:   maxSeverity,
		Summary:    fmt.Sprintf("%d alerts on %s: %s", len(bucket.Alerts), bucket.Instance, strings.Join(uniqueStrings(summaries), "; ")),
	}
}

func (c *Correlator) inferMiddlewareType(event *models.AlertEvent) enum.MiddlewareType {
	// Try to guess from labels
	if val, ok := event.Labels["middleware_type"]; ok {
		mw, err := enum.ParseMiddlewareType(val)
		if err == nil {
			return mw
		}
	}

	// Try to guess from alert name or job
	name := strings.ToLower(event.Name)
	job := strings.ToLower(event.Labels["job"])

	if strings.Contains(name, "redis") || strings.Contains(job, "redis") {
		return enum.Redis
	}
	if strings.Contains(name, "mysql") || strings.Contains(job, "mysql") {
		return enum.MySQL
	}
	if strings.Contains(name, "kafka") || strings.Contains(job, "kafka") {
		return enum.Kafka
	}

	return enum.MiddlewareType(-1) // Unknown
}

func isMoreSevere(a, b enum.SeverityLevel) bool {
	// Custom logic based on enum values
	// Low=0, Medium=1, High=2, Warning=3, Critical=4, Info=5
	// Priority: Critical > High > Warning > Medium > Low > Info

	// Map to rank
	rank := func(s enum.SeverityLevel) int {
		switch s {
		case enum.SeverityCritical: return 10
		case enum.SeverityHigh: return 8
		case enum.SeverityWarning: return 7
		case enum.SeverityMedium: return 5
		case enum.SeverityLow: return 3
		case enum.SeverityInfo: return 1
		default: return 0
		}
	}
	return rank(a) > rank(b)
}

func uniqueStrings(input []string) []string {
	u := make(map[string]bool)
	var res []string
	for _, val := range input {
		if !u[val] {
			u[val] = true
			res = append(res, val)
		}
	}
	return res
}
