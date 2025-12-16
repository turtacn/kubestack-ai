package models

import (
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
)

// AlertEvent represents a normalized alert from any source (Prometheus, Grafana, etc.).
type AlertEvent struct {
	Fingerprint  string            `json:"fingerprint"`
	Name         string            `json:"name"`
	Status       string            `json:"status"` // firing, resolved
	Severity     enum.SeverityLevel `json:"severity"`
	Instance     string            `json:"instance"`
	Summary      string            `json:"summary"`
	Description  string            `json:"description"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}
