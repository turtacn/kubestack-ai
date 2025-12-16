package alert

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/alert/notifier"
	"github.com/kubestack-ai/kubestack-ai/internal/alert/webhook"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
)

// Manager coordinates the alerting system.
type Manager struct {
	Dispatcher     *Dispatcher
	Correlator     *Correlator
	WebhookHandler *webhook.WebhookHandler
	Feedback       *FeedbackProcessor
}

// ManagerConfig holds configuration for the alert Manager.
type ManagerConfig struct {
	Dispatcher *DispatcherConfig
	Feedback   *FeedbackConfig
	Notifiers  []notifier.Notifier
}

// NewManager creates a new alert Manager.
func NewManager(diagManager *diagnosis.Manager, config *ManagerConfig) *Manager {
	if config == nil {
		config = &ManagerConfig{
			Dispatcher: &DispatcherConfig{},
			Feedback:   &FeedbackConfig{},
		}
	}

	feedback := NewFeedbackProcessor(config.Notifiers, config.Feedback)

	correlator := NewCorrelator(config.Dispatcher.CorrelationWindow, nil)

	dispatcher := NewDispatcher(diagManager, correlator, feedback, config.Dispatcher)

	// Set the callback
	correlator.onFlush = func(alert *CorrelatedAlert) {
		_ = dispatcher.TriggerDiagnosis(context.Background(), alert)
	}

	wh := webhook.NewWebhookHandler(dispatcher)

	return &Manager{
		Dispatcher:     dispatcher,
		Correlator:     correlator,
		WebhookHandler: wh,
		Feedback:       feedback,
	}
}
