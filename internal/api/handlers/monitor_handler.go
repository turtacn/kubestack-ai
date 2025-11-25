package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/alert"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/collector"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/storage"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/types"
)

type MonitorHandler struct {
	collector  *collector.CollectorScheduler
	store      storage.TimeseriesStore
	alertStore storage.AlertStore
	ruleEngine *alert.RuleEngine
	silence    *alert.SilenceManager
}

func NewMonitorHandler(
	collector *collector.CollectorScheduler,
	store storage.TimeseriesStore,
	alertStore storage.AlertStore,
	ruleEngine *alert.RuleEngine,
	silence *alert.SilenceManager,
) *MonitorHandler {
	return &MonitorHandler{
		collector:  collector,
		store:      store,
		alertStore: alertStore,
		ruleEngine: ruleEngine,
		silence:    silence,
	}
}

// GetMetrics queries metrics
func (h *MonitorHandler) GetMetrics(c *gin.Context) {
	// GET /api/v1/metrics?type=redis&instance=redis-0&range=1h

	metricType := c.Query("type")
	instance := c.Query("instance")
	rangeStr := c.DefaultQuery("range", "1h")

	duration, err := time.ParseDuration(rangeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid range parameter"})
		return
	}

	query := &storage.Query{
		Metric: fmt.Sprintf("%s_*", metricType),
		Labels: map[string]string{},
		Start:  time.Now().Add(-duration),
		End:    time.Now(),
	}

	if instance != "" {
		query.Labels["instance"] = instance
	}

	points, err := h.store.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": points,
		"count":   len(points),
	})
}

// GetAlertHistory queries alert history
func (h *MonitorHandler) GetAlertHistory(c *gin.Context) {
	// GET /api/v1/alerts/history?severity=critical&limit=100

	severity := c.Query("severity")
	limit := c.DefaultQuery("limit", "100")
	status := c.Query("status")

	alerts, err := h.alertStore.Query(c.Request.Context(), &storage.AlertQuery{
		Severity: severity,
		Limit:    limit,
		Status:   status,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"total":  len(alerts),
	})
}

// CreateSilence creates a silence rule
func (h *MonitorHandler) CreateSilence(c *gin.Context) {
	// POST /api/v1/alerts/silence

	var req struct {
		RuleName string            `json:"rule_name"`
		Labels   map[string]string `json:"labels"`
		Duration string            `json:"duration"`
		Comment  string            `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid duration"})
		return
	}

	silence := &types.Silence{
		RuleName:  req.RuleName,
		Labels:    req.Labels,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(duration),
		Comment:   req.Comment,
	}

	if err := h.silence.Add(c.Request.Context(), silence); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"silence_id": silence.ID,
		"expires_at": silence.EndTime,
	})
}
