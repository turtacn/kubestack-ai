package types

import (
	"time"
)

// Alert represents an alert event
type Alert struct {
    RuleName    string            `json:"rule_name"`
    Severity    string            `json:"severity"`
    Status      string            `json:"status"` // firing/resolved
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
    Value       float64           `json:"value"`
    FiredAt     time.Time         `json:"fired_at"`
    ResolvedAt  time.Time         `json:"resolved_at,omitempty"`
}

// AlertRule represents an alerting rule
type AlertRule struct {
    Name        string            // Rule name
    Expr        string            // Expression: cpu_usage > 80
    For         time.Duration     // Duration
    Severity    string            // Severity: critical/warning/info
    Labels      map[string]string // Custom labels
    Annotations map[string]string // Description info
    Notifiers   []string          // Notifier channel names
}

// Silence represents a silence rule
type Silence struct {
    ID        string            `json:"id"`
    RuleName  string            `json:"rule_name"`  // Empty means match all
    Labels    map[string]string `json:"labels"`     // Label matchers
    StartTime time.Time         `json:"start_time"`
    EndTime   time.Time         `json:"end_time"`
    CreatedBy string            `json:"created_by"`
    Comment   string            `json:"comment"`
}
