package notification

import (
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// NotificationPayload contains the information needed to send a notification.
type NotificationPayload struct {
	TaskID string                  `json:"task_id"`
	Status string                  `json:"status"` // e.g., "COMPLETED", "FAILED"
	Result *models.DiagnosisResult `json:"result,omitempty"`
	Error  error                   `json:"error,omitempty"`
	To     string                  `json:"to,omitempty"` // Recipient address (email or webhook URL if dynamic)
}

// Notifier defines the interface for sending notifications.
type Notifier interface {
	// Notify sends a notification with the given payload.
	Notify(payload *NotificationPayload) error
}
