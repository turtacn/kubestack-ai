package notification

import (
	"context"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// Notifier defines the interface for sending notifications.
type Notifier interface {
	// Notify sends a notification for the diagnosis result.
	Notify(ctx context.Context, result *models.DiagnosisResult) error
}
