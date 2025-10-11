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

	"github.com/gin-gonic/gin"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
)

// Server is the main struct for the API server.
type Server struct {
	router       *gin.Engine
	orchestrator interfaces.Orchestrator
	log          logger.Logger
	cfg          *config.ServerConfig
}

// NewServer creates a new API server instance.
func NewServer(orchestrator interfaces.Orchestrator, cfg *config.ServerConfig) *Server {
	s := &Server{
		router:       gin.Default(),
		orchestrator: orchestrator,
		log:          logger.NewLogger("api-server"),
		cfg:          cfg,
	}
	s.setupRoutes()
	return s
}

// setupRoutes defines all the API routes and their handlers.
func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")
	{
		// Health check endpoint
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Diagnosis routes
		api.POST("/diagnose", s.handleDiagnose)
		api.GET("/diagnose/results/:jobId", s.handleGetDiagnosisResult)

		// Natural Language Query routes
		api.POST("/ask", s.handleAsk)

		// Add other routes here in subsequent steps
	}
}

// Start runs the HTTP server on the configured address.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.log.Infof("Starting API server on %s", addr)
	return s.router.Run(addr)
}