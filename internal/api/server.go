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
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge"
	"github.com/kubestack-ai/kubestack-ai/internal/notification"
	"github.com/kubestack-ai/kubestack-ai/internal/storage"
	"github.com/kubestack-ai/kubestack-ai/internal/task"
	"github.com/kubestack-ai/kubestack-ai/internal/web"
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
	taskScheduler   *task.Scheduler
	taskWorker      *task.Worker
	taskStore       storage.TaskStore

	// Knowledge Base API
	knowledgeAPI    *KnowledgeAPI
}

// NewServer creates a new API server.
// It accepts an optional KnowledgeBase. If provided, it uses it for the Knowledge API.
// This allows sharing the KB instance with the DiagnosisManager.
func NewServer(cfg *config.Config, diagnosisEngine interfaces.DiagnosisManager, kb *knowledge.KnowledgeBase) *Server {
	authService := middleware.NewAuthService(cfg.Auth)
	rbacMiddleware := middleware.NewRBACMiddleware(cfg.RBAC)
	wsHandler := websocket.NewHandler(cfg.WebSocket)

	// Initialize Task System
	var queue task.TaskQueue
	var store storage.TaskStore

	// Defaults to Redis if config present, else could fallback to memory
	if cfg.TaskQueue.Type == "redis" {
		queue = task.NewRedisQueue(
			cfg.TaskQueue.Redis.Addr,
			cfg.TaskQueue.Redis.Password,
			cfg.TaskQueue.Redis.DB,
			cfg.TaskQueue.Redis.QueueName,
		)
		store = storage.NewRedisTaskStore(
			cfg.TaskQueue.Redis.Addr,
			cfg.TaskQueue.Redis.Password,
			cfg.TaskQueue.Redis.DB,
			24*time.Hour, // TTL
		)
	} else {
		// Fallback for development
		store = storage.NewInMemoryTaskStore()
	}

	scheduler := task.NewScheduler(queue, store)

	// Composite Notifier
	notifiers := []notification.Notifier{}
	if cfg.Notification.Webhook.URL != "" {
		notifiers = append(notifiers, notification.NewWebhookNotifier(cfg.Notification.Webhook.URL))
	}
	if cfg.Notification.Email.Host != "" {
		emailConfig := notification.EmailConfig{
			Host:      cfg.Notification.Email.Host,
			Port:      cfg.Notification.Email.Port,
			Username:  cfg.Notification.Email.Username,
			Password:  cfg.Notification.Email.Password,
			From:      cfg.Notification.Email.From,
			DefaultTo: cfg.Notification.Email.DefaultTo,
		}
		notifiers = append(notifiers, notification.NewEmailNotifier(emailConfig))
	}
	compositeNotifier := notification.NewCompositeNotifier(notifiers)

	worker := task.NewWorker(queue, diagnosisEngine, store, compositeNotifier)

	// If KB is not provided, try to extract from engine or create new
	if kb == nil {
		if dm, ok := diagnosisEngine.(interface{ GetKnowledgeBase() *knowledge.KnowledgeBase }); ok {
			kb = dm.GetKnowledgeBase()
		}
		if kb == nil {
			// Fallback: create isolated KB (not ideal but safe)
			kb = knowledge.NewKnowledgeBase()
		}
	}

	loader := knowledge.NewRuleLoader(kb)

	// Ensure we load rules if creating new KB
	// This might duplicate loading if KB is shared but redundant loading is cheap (in-memory map update)
	// Actually, if shared, we assume it's already initialized.

	knowledgeAPI := NewKnowledgeAPI(kb, loader)

	s := &Server{
		router:          gin.Default(),
		config:          cfg,
		diagnosisEngine: diagnosisEngine,
		authService:     authService,
		rbacMiddleware:  rbacMiddleware,
		log:             logger.NewLogger("api-server"),
		wsHandler:       wsHandler,
		taskScheduler:   scheduler,
		taskWorker:      worker,
		taskStore:       store,
		knowledgeAPI:    knowledgeAPI,
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

	// Load templates
	s.router.LoadHTMLGlob("internal/web/templates/*")

	// Web Console Routes
	consoleHandler := web.NewConsoleHandler(s.diagnosisEngine, s.taskScheduler, s.taskStore)
	consoleHandler.RegisterRoutes(s.router)

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

	// Knowledge Base Routes (NEW)
	s.knowledgeAPI.RegisterRoutes(v1.Group("/knowledge"))

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

	// Start Task Worker
	if s.taskWorker != nil {
		s.taskWorker.Start()
		defer s.taskWorker.Stop()
	}

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
