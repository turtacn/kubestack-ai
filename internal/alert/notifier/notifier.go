package notifier

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// NotificationMessage is the data structure for sending notifications.
type NotificationMessage struct {
	Title    string
	Content  string
	Severity string // "critical", "warning", "info"
	Link     string // Optional link to dashboard/report
}

// Notifier defines the interface for sending notifications.
type Notifier interface {
	// Send sends a notification for the diagnosis result.
	Send(ctx context.Context, msg *NotificationMessage) error
	// Type returns the type of the notifier (e.g., "dingtalk", "slack").
	Type() string
}

// DiagnosisResultWrapper is a wrapper to pass diagnosis result with alert context.
type DiagnosisResultWrapper struct {
	Result *models.DiagnosisResult
	Alert  *models.AlertEvent // or CorrelatedAlert
}
