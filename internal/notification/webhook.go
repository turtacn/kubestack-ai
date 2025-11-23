package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// WebhookNotifier sends notifications via HTTP POST request.
type WebhookNotifier struct {
	url string
}

// NewWebhookNotifier creates a new WebhookNotifier.
func NewWebhookNotifier(url string) *WebhookNotifier {
	return &WebhookNotifier{url: url}
}

// Notify sends the notification payload to the configured URL.
func (w *WebhookNotifier) Notify(payload *NotificationPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal notification payload: %w", err)
	}

	resp, err := http.Post(w.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d", resp.StatusCode)
	}

	return nil
}
