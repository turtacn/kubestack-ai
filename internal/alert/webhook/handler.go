package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// DispatcherInterface avoids circular dependency
type DispatcherInterface interface {
	Dispatch(ctx context.Context, event *models.AlertEvent) error
}

// WebhookHandler handles incoming webhooks.
type WebhookHandler struct {
	dispatcher DispatcherInterface
	logger     logger.Logger
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(d DispatcherInterface) *WebhookHandler {
	return &WebhookHandler{
		dispatcher: d,
		logger:     logger.NewLogger("webhook-handler"),
	}
}

// AlertmanagerPayload matches Prometheus Alertmanager webhook format.
type AlertmanagerPayload struct {
	Status string `json:"status"`
	Alerts []struct {
		Status       string            `json:"status"`
		Labels       map[string]string `json:"labels"`
		Annotations  map[string]string `json:"annotations"`
		StartsAt     time.Time         `json:"startsAt"`
		EndsAt       time.Time         `json:"endsAt"`
		GeneratorURL string            `json:"generatorURL"`
		Fingerprint  string            `json:"fingerprint"`
	} `json:"alerts"`
}

// HandleAlertmanagerWebhook handles Alertmanager webhooks.
func (h *WebhookHandler) HandleAlertmanagerWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Errorf("Failed to read body: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload AlertmanagerPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Errorf("Failed to decode Alertmanager payload: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	for _, alertData := range payload.Alerts {
		event := &models.AlertEvent{
			Fingerprint:  alertData.Fingerprint,
			Name:         alertData.Labels["alertname"],
			Status:       alertData.Status,
			Severity:     parseSeverity(alertData.Labels["severity"]),
			Instance:     alertData.Labels["instance"],
			Summary:      alertData.Annotations["summary"],
			Description:  alertData.Annotations["description"],
			Labels:       alertData.Labels,
			Annotations:  alertData.Annotations,
			StartsAt:     alertData.StartsAt,
			EndsAt:       alertData.EndsAt,
			GeneratorURL: alertData.GeneratorURL,
		}

		if err := h.dispatcher.Dispatch(r.Context(), event); err != nil {
			h.logger.Errorf("Failed to dispatch alert: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// GrafanaPayload matches Grafana alert webhook format.
type GrafanaPayload struct {
	Title       string `json:"title"`
	RuleName    string `json:"ruleName"`
	State       string `json:"state"` // alerting, ok
	EvalMatches []struct {
		Value  float64           `json:"value"`
		Metric string            `json:"metric"`
		Tags   map[string]string `json:"tags"`
	} `json:"evalMatches"`
	Message string `json:"message"`
}

// HandleGrafanaWebhook handles Grafana webhooks.
func (h *WebhookHandler) HandleGrafanaWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload GrafanaPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if payload.State != "alerting" {
		w.WriteHeader(http.StatusOK)
		return
	}

	for _, match := range payload.EvalMatches {
		event := &models.AlertEvent{
			Name:        payload.RuleName,
			Status:      "firing",
			Severity:    enum.SeverityWarning, // Default for Grafana?
			Instance:    match.Tags["instance"],
			Summary:     payload.Title,
			Description: payload.Message,
			Labels:      match.Tags,
			StartsAt:    time.Now(),
		}

		if event.Instance == "" {
			// fallback if instance not in tags
			event.Instance = "unknown"
		}

		if err := h.dispatcher.Dispatch(r.Context(), event); err != nil {
			h.logger.Errorf("Failed to dispatch Grafana alert: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func parseSeverity(s string) enum.SeverityLevel {
	switch strings.ToLower(s) {
	case "critical":
		return enum.SeverityCritical
	case "warning":
		return enum.SeverityWarning
	case "info":
		return enum.SeverityInfo
	default:
		return enum.SeverityMedium // Default
	}
}
