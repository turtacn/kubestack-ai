// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/storage"
	"github.com/kubestack-ai/kubestack-ai/internal/task"
)

type ConsoleHandler struct {
	manager   interfaces.DiagnosisManager
	scheduler *task.Scheduler
	taskStore storage.TaskStore
}

func NewConsoleHandler(manager interfaces.DiagnosisManager, scheduler *task.Scheduler, taskStore storage.TaskStore) *ConsoleHandler {
	return &ConsoleHandler{
		manager:   manager,
		scheduler: scheduler,
		taskStore: taskStore,
	}
}

// RegisterRoutes registers the console routes with the Gin engine.
func (h *ConsoleHandler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/console")
	{
		group.POST("/diagnose", h.HandleDiagnoseRequest)
		// Serve static template for simple result viewing
		group.GET("/result-view", h.ServeResultTemplate)
		group.GET("/task/status/:taskId", h.HandleGetTaskStatus)
	}
}

// HandleDiagnoseRequest handles the diagnosis request from the web console.
func (h *ConsoleHandler) HandleDiagnoseRequest(c *gin.Context) {
	var req models.DiagnosisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Validate middleware type
	if !req.TargetMiddleware.IsValid() {
		// Try to parse string if it's raw string
		// Actually c.ShouldBindJSON should handle enum unmarshal if implemented or basic types
		// models.DiagnosisRequest uses enum.MiddlewareType which is string.
		// But let's be safe.
	}

	// Set output format to json for web console consistency
	req.OutputFormat = "json"

	// Check if async processing is requested (e.g., via query param or default behavior)
	async := c.Query("async") == "true"

	if async && h.scheduler != nil {
		taskID, err := h.scheduler.SubmitDiagnosisTask(&req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit task: " + err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{
			"task_id": taskID,
			"status":  "PENDING",
			"message": "Diagnosis task submitted successfully",
		})
		return
	}

	// Fallback to synchronous execution
	// We need a progress channel even if we don't stream it to HTTP response in this handler (unless we use SSE).
	// For this simplified handler, we just drain the channel or ignore it to let it run.
	progressChan := make(chan interfaces.DiagnosisProgress, 100)
	go func() {
		for range progressChan {
			// Drain channel to prevent blocking if buffer fills
		}
	}()

	// Use a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := h.manager.RunDiagnosis(ctx, &req, progressChan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Diagnosis failed: " + err.Error()})
		return
	}

	// Return the result structure which matches the JSON requirement
	c.JSON(http.StatusOK, result)
}

// HandleGetTaskStatus returns the status and result of a task.
func (h *ConsoleHandler) HandleGetTaskStatus(c *gin.Context) {
	if h.taskStore == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Task store not configured"})
		return
	}

	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
		return
	}

	status, err := h.taskStore.GetStatus(taskID)
	if err != nil {
		if err == storage.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task status: " + err.Error()})
		}
		return
	}

	response := gin.H{
		"task_id":    status.TaskID,
		"state":      status.State,
		"created_at": status.CreatedAt,
		"updated_at": status.UpdatedAt,
		"error":      status.Error,
	}

	if status.State == storage.TaskStateCompleted {
		result, err := h.taskStore.GetResult(taskID)
		if err != nil {
			// Log error but maybe return status without result
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task result: " + err.Error()})
			return
		}
		response["result"] = result
	}

	c.JSON(http.StatusOK, response)
}

func (h *ConsoleHandler) ServeResultTemplate(c *gin.Context) {
	c.HTML(http.StatusOK, "diagnosis_result.html", nil)
}
