package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/api/websocket"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/core/report"
)

type DiagnosisHandler struct {
	engine    interfaces.DiagnosisManager
	wsHandler *websocket.Handler
}

func NewDiagnosisHandler(engine interfaces.DiagnosisManager, wsHandler *websocket.Handler) *DiagnosisHandler {
	return &DiagnosisHandler{
		engine:    engine,
		wsHandler: wsHandler,
	}
}

type TriggerRequest struct {
	Target     string            `json:"target" binding:"required"`
	Middleware string            `json:"middleware" binding:"required"` // e.g., "redis", "mysql"
	Instance   string            `json:"instance"`
	Filters    map[string]string `json:"filters,omitempty"`
}

func (h *DiagnosisHandler) TriggerDiagnosis(c *gin.Context) {
	var req TriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mwType, err := enum.ParseMiddlewareType(req.Middleware)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid middleware type"})
		return
	}

	diagReq := &models.DiagnosisRequest{
		TargetMiddleware: mwType,
		Instance:         req.Instance,
	}

	// Generate a TaskID to track this specific request
	taskID := uuid.New().String()

	// Launch diagnosis in background
	go func() {
		// Create a channel for progress
		progressChan := make(chan interfaces.DiagnosisProgress)

		// Goroutine to forward progress to WebSocket
		go func() {
			for p := range progressChan {
				h.wsHandler.Broadcast(taskID, p)
			}
		}()

		// Run Diagnosis
		// Use a background context as the request context will be cancelled when the handler returns
		ctx := context.Background()
		result, err := h.engine.RunDiagnosis(ctx, diagReq, progressChan)

		// Final message
		if err != nil {
			h.wsHandler.Broadcast(taskID, interfaces.DiagnosisProgress{Step: "Finished", Status: "Failed", Message: err.Error()})
		} else {
			// Convert to standardized report
			diagReport := report.FromDiagnosisResult(result, diagReq)
			
			h.wsHandler.Broadcast(taskID, interfaces.DiagnosisProgress{Step: "Finished", Status: "Completed", Message: "Diagnosis completed successfully. Report ID: " + result.ID})
			// Broadcast the standardized report
			h.wsHandler.Broadcast(taskID, struct {
				Type string
				Data interface{}
			}{Type: "Result", Data: diagReport})
		}

		close(progressChan)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Diagnosis started",
		"task_id": taskID,
	})
}

func (h *DiagnosisHandler) GetDiagnosisResult(c *gin.Context) {
	id := c.Param("id")

	result, err := h.engine.GetDiagnosisResult(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RunDiagnosisSync executes diagnosis synchronously and returns a standardized DiagnosisReport
func (h *DiagnosisHandler) RunDiagnosisSync(c *gin.Context) {
	var req TriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mwType, err := enum.ParseMiddlewareType(req.Middleware)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid middleware type"})
		return
	}

	diagReq := &models.DiagnosisRequest{
		TargetMiddleware: mwType,
		Instance:         req.Instance,
	}

	// Create a channel for progress (but don't report it for sync API)
	progressChan := make(chan interfaces.DiagnosisProgress)
	go func() {
		for range progressChan {
			// Consume progress silently
		}
	}()

	// Run diagnosis synchronously
	ctx := c.Request.Context()
	result, err := h.engine.RunDiagnosis(ctx, diagReq, progressChan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to standardized report
	diagReport := report.FromDiagnosisResult(result, diagReq)

	c.JSON(http.StatusOK, diagReport)
}
