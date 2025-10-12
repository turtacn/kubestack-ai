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

package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// handleDiagnose is the handler for the POST /api/v1/diagnose endpoint.
// It starts an asynchronous diagnosis job.
func (s *Server) handleDiagnose(c *gin.Context) {
	var req models.DiagnosisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Since diagnosis is a long-running task, we run it in a goroutine
	// and immediately return a job ID to the client.
	jobID := "diag-job-" + uuid.New().String()

	go func() {
		// We use a background context because the HTTP request will have already finished.
		// In a real system, you'd want more sophisticated job management.
		progressChan := make(chan interfaces.DiagnosisProgress)
		go func() {
			for p := range progressChan {
				s.log.Infof("[Job %s] %s: %s", jobID, p.Step, p.Message)
			}
		}()

		if _, err := s.orchestrator.ExecuteDiagnosis(c, &req, progressChan); err != nil {
			s.log.Errorf("Diagnosis job %s failed: %v", jobID, err)
			// In a real system, we'd update the job status in a database.
			return
		}
		// In a real system, we'd update the job status and store the result.
		s.log.Infof("Diagnosis job %s completed successfully.", jobID)
		// For this implementation, we are relying on the fact that the diagnosis manager
		// already persists the report to a file.
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"jobId":     jobID,
		"status":    "Pending",
		"createdAt": time.Now().UTC(),
	})
}

// handleGetDiagnosisResult is the handler for the GET /api/v1/diagnose/results/{jobId} endpoint.
func (s *Server) handleGetDiagnosisResult(c *gin.Context) {
	jobID := c.Param("jobId")

	result, err := s.orchestrator.GetDiagnosis(c, jobID)
	if err != nil {
		// This could be a 404 Not Found or a 500 Internal Server Error,
		// depending on the error type. For now, we'll just return 404.
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// handleAsk is the handler for the POST /api/v1/ask endpoint.
// It streams the AI's response using Server-Sent Events (SSE).
func (s *Server) handleAsk(c *gin.Context) {
	var req struct {
		Question string `json:"question"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the streaming channel from the orchestrator.
	responseChan, err := s.orchestrator.ProcessNaturalLanguageStream(c, req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start stream"})
		return
	}

	// Set the headers for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	// Stream the response
	for chunk := range responseChan {
		if chunk.Err != nil {
			// Can't write a JSON error as the headers are already sent.
			// Log the error and close the connection.
			s.log.Errorf("Error from AI stream: %v", chunk.Err)
			return
		}
		// Format as an SSE message
		fmt.Fprintf(c.Writer, "data: %s\n\n", chunk.Content)
		c.Writer.Flush()
	}
}

// handleListPlugins is the handler for the GET /api/v1/plugins endpoint.
func (s *Server) handleListPlugins(c *gin.Context) {
	// In a real application, we would get the plugin manager from the orchestrator.
	// For now, we'll just return a placeholder.
	c.JSON(http.StatusOK, gin.H{
		"plugins": []string{"redis", "mysql", "kafka", "elasticsearch", "postgresql"},
	})
}