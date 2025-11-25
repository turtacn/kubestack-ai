package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/monitor/types"
)

type WebhookNotifier struct {
	name    string
	url     string
	timeout time.Duration
	client  *http.Client
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(name, url string, timeout time.Duration) *WebhookNotifier {
	return &WebhookNotifier{
		name:    name,
		url:     url,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (w *WebhookNotifier) Send(ctx context.Context, a *types.Alert) error {
	// Construct Webhook Payload (compatible with Alertmanager format partially)
	payload := map[string]interface{}{
		"version":  "4",
		"groupKey": a.RuleName,
		"status":   a.Status,
		"alerts": []map[string]interface{}{
			{
				"status":       a.Status,
				"labels":       a.Labels,
				"annotations":  a.Annotations,
				"startsAt":     a.FiredAt.Format(time.RFC3339),
				"endsAt":       a.ResolvedAt.Format(time.RFC3339),
				"generatorURL": fmt.Sprintf("https://kubestack-ai/alerts/%s", a.RuleName),
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned error: %d", resp.StatusCode)
	}

	return nil
}

func (w *WebhookNotifier) Name() string {
	return w.name
}
