package alert

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/model"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/storage"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/types"
)

// AlertEvaluator evaluates alert rules
type AlertEvaluator struct {
	ruleEngine *RuleEngine
	store      storage.TimeseriesStore
	alertStore storage.AlertStore
	notifier   *Notifier
	silence    *SilenceManager
	log        logger.Logger

	// State tracking (rule name -> first firing time)
	firingState map[string]time.Time
	mu          sync.RWMutex
}

// NewAlertEvaluator creates a new evaluator
func NewAlertEvaluator(engine *RuleEngine, store storage.TimeseriesStore, alertStore storage.AlertStore, notifier *Notifier, silence *SilenceManager, log logger.Logger) *AlertEvaluator {
	return &AlertEvaluator{
		ruleEngine:  engine,
		store:       store,
		alertStore:  alertStore,
		notifier:    notifier,
		silence:     silence,
		log:         log,
		firingState: make(map[string]time.Time),
	}
}

// Start starts the evaluation loop
func (e *AlertEvaluator) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.evaluate(ctx)
		}
	}
}

func (e *AlertEvaluator) evaluate(ctx context.Context) {
	rules := e.ruleEngine.GetRules()

	for _, rule := range rules {
		// 1. Query timeseries data
		query := e.buildQuery(rule.Expr)
		points, err := e.store.Query(ctx, query)
		if err != nil {
			e.log.Errorf("Query metrics failed [%s]: %v", rule.Name, err)
			continue
		}

		// 2. Check condition
		triggered, value := e.checkCondition(rule, points)

		if triggered {
			// 3. Check duration (For)
			if e.shouldFire(rule) {
				// 4. Check silence
				if e.silence.IsSilenced(rule.Name, rule.Labels) {
					e.log.Debugf("Rule [%s] is silenced", rule.Name)
					continue
				}

				// 5. Send Alert
				alert := &types.Alert{
					RuleName:    rule.Name,
					Severity:    rule.Severity,
					Status:      "firing",
					Labels:      rule.Labels,
					Annotations: rule.Annotations,
					FiredAt:     time.Now(),
					Value:       value,
				}

				if err := e.notifier.Send(ctx, alert, rule.Notifiers); err != nil {
					e.log.Errorf("Failed to send alert [%s]: %v", rule.Name, err)
				}

				// 6. Save history
				if err := e.alertStore.Save(ctx, alert); err != nil {
					e.log.Errorf("Failed to save alert: %v", err)
				}
			}
		} else {
			// Rule resolved
			if e.isFiring(rule.Name) {
				e.clearFiringState(rule.Name)

				// Send recovery notification
				recoveryAlert := &types.Alert{
					RuleName:   rule.Name,
					Status:     "resolved",
					ResolvedAt: time.Now(),
					Labels:      rule.Labels,
					Annotations: rule.Annotations,
				}
				_ = e.notifier.Send(ctx, recoveryAlert, rule.Notifiers)

				// Save resolved state
				_ = e.alertStore.Save(ctx, recoveryAlert)
			}
		}
	}
}

func (e *AlertEvaluator) shouldFire(rule *types.AlertRule) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	firstFiredAt, exists := e.firingState[rule.Name]

	if !exists {
		e.firingState[rule.Name] = now
		return rule.For == 0 // Fire immediately if For is 0
	}

	if now.Sub(firstFiredAt) >= rule.For {
		return true
	}

	return false
}

func (e *AlertEvaluator) buildQuery(expr string) *storage.Query {
	// Parse expression to extract metric name
	// Example: "cpu_usage > 80" -> Query{Metric: "cpu_usage", Range: "5m"}
	parts := strings.Fields(expr)
	metricName := parts[0]

	return &storage.Query{
		Metric: metricName,
		Start:  time.Now().Add(-5 * time.Minute), // Query recent 5m
		End:    time.Now(),
	}
}

func (e *AlertEvaluator) checkCondition(rule *types.AlertRule, points []*model.MetricPoint) (bool, float64) {
	// Need to fix import cycle if using collector.MetricPoint, but we moved it to model package.
	// points type is []*model.MetricPoint

	if len(points) == 0 {
		return false, 0
	}

	// Simplified: check latest value
	latestValue := points[len(points)-1].Value

	parts := strings.Fields(rule.Expr)
	if len(parts) != 3 {
		return false, 0
	}

	operator := parts[1]
	threshold, _ := strconv.ParseFloat(parts[2], 64)

	var triggered bool
	switch operator {
	case ">":
		triggered = latestValue > threshold
	case ">=":
		triggered = latestValue >= threshold
	case "<":
		triggered = latestValue < threshold
	case "<=":
		triggered = latestValue <= threshold
	case "==":
		triggered = latestValue == threshold
	case "!=":
		triggered = latestValue != threshold
	default:
		triggered = false
	}

	return triggered, latestValue
}

func (e *AlertEvaluator) isFiring(ruleName string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, exists := e.firingState[ruleName]
	return exists
}

func (e *AlertEvaluator) clearFiringState(ruleName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.firingState, ruleName)
}
