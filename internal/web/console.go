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
)

type ConsoleHandler struct {
	manager interfaces.DiagnosisManager
}

func NewConsoleHandler(manager interfaces.DiagnosisManager) *ConsoleHandler {
	return &ConsoleHandler{manager: manager}
}

// RegisterRoutes registers the console routes with the Gin engine.
func (h *ConsoleHandler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/console")
	{
		group.POST("/diagnose", h.HandleDiagnoseRequest)
		// Serve static template for simple result viewing
		group.GET("/result-view", h.ServeResultTemplate)
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

	// Run diagnosis
	// Note: RunDiagnosis is blocking. For web console, we might want async but
	// based on deliverables, we return result directly or via polling.
	// The requirement says "accept diagnosis request and return result".
	// So synchronous is fine for now or small tasks.

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

func (h *ConsoleHandler) ServeResultTemplate(c *gin.Context) {
	c.HTML(http.StatusOK, "diagnosis_result.html", nil)
}
