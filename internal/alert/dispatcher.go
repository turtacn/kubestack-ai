package alert

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// DispatcherConfig holds configuration for the Dispatcher.
type DispatcherConfig struct {
	DedupWindow  time.Duration
	CorrelationWindow time.Duration
}

// Dispatcher routes alerts to the diagnosis engine.
type Dispatcher struct {
	engine     *diagnosis.Manager
	correlator *Correlator
	alertCache map[string]time.Time
	cacheMu    sync.RWMutex
	config     *DispatcherConfig
	feedback   *FeedbackProcessor
	logger     logger.Logger
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(engine *diagnosis.Manager, correlator *Correlator, feedback *FeedbackProcessor, config *DispatcherConfig) *Dispatcher {
	if config.DedupWindow == 0 {
		config.DedupWindow = 5 * time.Minute
	}
	d := &Dispatcher{
		engine:     engine,
		correlator: correlator,
		alertCache: make(map[string]time.Time),
		config:     config,
		feedback:   feedback,
		logger:     logger.NewLogger("alert-dispatcher"),
	}
	go d.cleanupExpiredCache()
	return d
}

// Dispatch processes an incoming alert event.
func (d *Dispatcher) Dispatch(ctx context.Context, event *models.AlertEvent) error {
	alertKey := d.generateAlertKey(event)

	// Dedup check (skip for Critical)
	if event.Severity != enum.SeverityCritical {
		if d.isDuplicate(alertKey) {
			d.logger.Infof("Duplicate alert suppressed: %s", alertKey)
			return nil
		}
	}

	d.cacheMu.Lock()
	d.alertCache[alertKey] = time.Now()
	d.cacheMu.Unlock()

	// Correlate
	shouldTrigger, correlated := d.correlator.AddAlert(event)
	if shouldTrigger {
		return d.TriggerDiagnosis(ctx, correlated)
	}

	return nil
}

func (d *Dispatcher) generateAlertKey(event *models.AlertEvent) string {
	return fmt.Sprintf("%s-%s-%s", event.Name, event.Instance, event.Fingerprint)
}

func (d *Dispatcher) isDuplicate(key string) bool {
	d.cacheMu.RLock()
	defer d.cacheMu.RUnlock()
	lastSeen, exists := d.alertCache[key]
	if !exists {
		return false
	}
	return time.Since(lastSeen) < d.config.DedupWindow
}

func (d *Dispatcher) cleanupExpiredCache() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		d.cacheMu.Lock()
		for key, ts := range d.alertCache {
			if time.Since(ts) > d.config.DedupWindow {
				delete(d.alertCache, key)
			}
		}
		d.cacheMu.Unlock()
	}
}

// TriggerDiagnosis is public so Correlator can call it (via closure or direct call).
func (d *Dispatcher) TriggerDiagnosis(ctx context.Context, alert *CorrelatedAlert) error {
	d.logger.Infof("Triggering diagnosis for correlated alert on %s", alert.Instance)

	// Create DiagnosisRequest
	req := &models.DiagnosisRequest{
		TargetMiddleware: alert.Middleware,
		Instance:         alert.Instance,
	}

	// We run diagnosis asynchronously
	go func() {
		// Use a detached context for background execution to avoid cancellation by HTTP request context
		bgCtx := context.Background()

		progressChan := make(chan interfaces.DiagnosisProgress)
		go func() {
			for p := range progressChan {
				d.logger.Debugf("Diagnosis progress for %s: %s", alert.Instance, p.Message)
			}
		}()

		// TODO: Call DiagnoseFromAlert if available, otherwise RunDiagnosis
		// We will assume DiagnoseFromAlert is added to Manager.
		// Since we haven't updated Manager yet, this will fail to compile if we call it directly.
		// For now, I will use RunDiagnosis, but I need to pass alert context somehow.
		// I will modify Manager in the next step to add DiagnoseFromAlert.

		// result, err := d.engine.DiagnoseFromAlert(bgCtx, alert, progressChan)

		// Placeholder until Manager is updated:
		result, err := d.engine.RunDiagnosis(bgCtx, req, progressChan)
		if err != nil {
			d.logger.Errorf("Diagnosis failed: %v", err)
			return
		}

		if d.feedback != nil {
			if err := d.feedback.ProcessDiagnosisResult(bgCtx, alert, result); err != nil {
				d.logger.Errorf("Failed to send feedback: %v", err)
			}
		}
	}()

	return nil
}
