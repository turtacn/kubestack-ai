package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kubestack-ai/kubestack-ai/internal/api/handlers"
	"github.com/kubestack-ai/kubestack-ai/internal/api/middleware"
	"github.com/kubestack-ai/kubestack-ai/internal/api/websocket"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
)

type Server struct {
	router          *gin.Engine
	config          *config.Config
	diagnosisEngine interfaces.DiagnosisManager
	// executionMgr    interfaces.ExecutionManager // To be added when available
	authService     *middleware.AuthService
	rbacMiddleware  *middleware.RBACMiddleware
	log             logger.Logger
	wsHandler       *websocket.Handler
}

func NewServer(cfg *config.Config, diagnosisEngine interfaces.DiagnosisManager) *Server {
	authService := middleware.NewAuthService(cfg.Auth)
	rbacMiddleware := middleware.NewRBACMiddleware(cfg.RBAC)
	wsHandler := websocket.NewHandler(cfg.WebSocket)

	s := &Server{
		router:          gin.Default(),
		config:          cfg,
		diagnosisEngine: diagnosisEngine,
		authService:     authService,
		rbacMiddleware:  rbacMiddleware,
		log:             logger.NewLogger("api-server"),
		wsHandler:       wsHandler,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// CORS
	corsConfig := cors.DefaultConfig()
	if len(s.config.Server.CORS.AllowedOrigins) > 0 {
		corsConfig.AllowOrigins = s.config.Server.CORS.AllowedOrigins
	} else {
		corsConfig.AllowAllOrigins = true
	}
	corsConfig.AllowMethods = s.config.Server.CORS.AllowedMethods
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	s.router.Use(cors.New(corsConfig))

	// WebSocket
	s.router.GET("/ws/diagnosis/:id", s.wsHandler.ServeHTTP)

	// API V1
	v1 := s.router.Group("/api/v1")

	// Auth routes (public)
	authHandler := handlers.NewAuthHandler(s.authService)
	v1.POST("/auth/login", authHandler.Login)

	// Protected routes
	v1.Use(s.authService.JWTAuth())

	// Diagnosis
	diagnosisHandler := handlers.NewDiagnosisHandler(s.diagnosisEngine, s.wsHandler)
	diagnosis := v1.Group("/diagnosis")
	diagnosis.POST("", s.rbacMiddleware.CheckPermission("diagnosis:write"), diagnosisHandler.TriggerDiagnosis)
	diagnosis.GET("/:id", s.rbacMiddleware.CheckPermission("diagnosis:read"), diagnosisHandler.GetDiagnosisResult)

	// Execution (Placeholder for now)
	execution := v1.Group("/execution")
	executionHandler := handlers.NewExecutionHandler() // Placeholder
	execution.POST("/plan/:id/execute", s.rbacMiddleware.CheckPermission("execution:write"), executionHandler.ExecutePlan)
	execution.GET("/history", s.rbacMiddleware.CheckPermission("execution:read"), executionHandler.GetHistory)

    // Config
    configHandler := handlers.NewConfigHandler(s.config)
    conf := v1.Group("/config")
    conf.GET("", s.rbacMiddleware.CheckPermission("diagnosis:read"), configHandler.GetConfig)
    conf.PUT("", s.rbacMiddleware.CheckPermission("diagnosis:write"), configHandler.UpdateConfig)
}

func (s *Server) Start(ctx context.Context) error {
    // Start WebSocket hub
    go s.wsHandler.Run()

	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	go func() {
		if s.config.Server.TLS.Enabled {
			s.log.Infof("Starting HTTPS server on %s", addr)
			if err := srv.ListenAndServeTLS(s.config.Server.TLS.CertFile, s.config.Server.TLS.KeyFile); err != nil && err != http.ErrServerClosed {
				s.log.Fatalf("listen: %s\n", err)
			}
		} else {
			s.log.Infof("Starting HTTP server on %s", addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.log.Fatalf("listen: %s\n", err)
			}
		}
	}()

	<-ctx.Done()
	s.log.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	s.log.Info("Server exiting")
	return nil
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() *gin.Engine {
	return s.router
}
