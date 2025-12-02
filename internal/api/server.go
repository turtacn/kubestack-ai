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
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/alert"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/alert/channels"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/collector"
	"github.com/kubestack-ai/kubestack-ai/internal/monitor/storage"
	"github.com/kubestack-ai/kubestack-ai/internal/notification"
	storage_pkg "github.com/kubestack-ai/kubestack-ai/internal/storage"
	"github.com/kubestack-ai/kubestack-ai/internal/task"
	"github.com/kubestack-ai/kubestack-ai/internal/web"
)

type Server struct {
	router          *gin.Engine
	config          *config.Config
	diagnosisEngine interfaces.DiagnosisManager
	pluginManager   interfaces.PluginManager // Needed for middleware collectors
	// executionMgr    interfaces.ExecutionManager // To be added when available
	authService    *middleware.AuthService
	rbacMiddleware *middleware.RBACMiddleware
	log            logger.Logger
	wsHandler      *websocket.Handler
	taskScheduler  *task.Scheduler
	taskWorker     *task.Worker
	taskStore      storage_pkg.TaskStore

	// Knowledge Base API
	knowledgeAPI *KnowledgeAPI

	// Monitoring
	monitorHandler     *handlers.MonitorHandler
	collectorScheduler *collector.CollectorScheduler
	alertEvaluator     *alert.AlertEvaluator
	silenceManager     *alert.SilenceManager
	alertStore         storage.AlertStore
	silenceStore       storage.SilenceStore
	timeseriesStore    storage.TimeseriesStore
}

// NewServer creates a new API server.
func NewServer(cfg *config.Config, diagnosisEngine interfaces.DiagnosisManager, kb *knowledge.KnowledgeBase, pluginManager interfaces.PluginManager) *Server {
	authService := middleware.NewAuthService(cfg.Auth)
	rbacMiddleware := middleware.NewRBACMiddleware(cfg.RBAC)
	wsHandler := websocket.NewHandler(cfg.WebSocket)
	log := logger.NewLogger("api-server")

	// Initialize Task System
	var queue task.TaskQueue
	var store storage_pkg.TaskStore

	if cfg.TaskQueue.Type == "redis" {
		queue = task.NewRedisQueue(
			cfg.TaskQueue.Redis.Addr,
			cfg.TaskQueue.Redis.Password,
			cfg.TaskQueue.Redis.DB,
			cfg.TaskQueue.Redis.QueueName,
		)
		store = storage_pkg.NewRedisTaskStore(
			cfg.TaskQueue.Redis.Addr,
			cfg.TaskQueue.Redis.Password,
			cfg.TaskQueue.Redis.DB,
			24*time.Hour,
		)
	} else {
		store = storage_pkg.NewInMemoryTaskStore()
	}

	scheduler := task.NewScheduler(queue, store)

	// Composite Notifier for Tasks
	compositeNotifier := notification.NewCompositeNotifier(cfg.Notification)

	worker := task.NewWorker(queue, diagnosisEngine, store, compositeNotifier, cfg.Notification)

	// KB Init
	if kb == nil {
		if dm, ok := diagnosisEngine.(interface {
			GetKnowledgeBase() *knowledge.KnowledgeBase
		}); ok {
			kb = dm.GetKnowledgeBase()
		}
		if kb == nil {
			kb = knowledge.NewKnowledgeBase()
		}
	}
	loader := knowledge.NewRuleLoader(kb)
	knowledgeAPI := NewKnowledgeAPI(kb, loader)

	// --- Monitoring Subsystem Init ---
	tsStore, err := storage.NewSQLiteTimeseriesStore(cfg.Monitor.Storage.Path)
	if err != nil {
		log.Warnf("Failed to init timeseries store, monitoring disabled: %v", err)
	}

	alertStore, err := storage.NewSQLiteAlertStore(cfg.Monitor.Storage.Path)
	if err != nil {
		log.Warnf("Failed to init alert store: %v", err)
	}

	silenceStore, err := storage.NewSQLiteSilenceStore(cfg.Monitor.Storage.Path)
	if err != nil {
		log.Warnf("Failed to init silence store: %v", err)
	}

	var colScheduler *collector.CollectorScheduler
	var alEvaluator *alert.AlertEvaluator
	var monHandler *handlers.MonitorHandler
	var silenceMgr *alert.SilenceManager

	if tsStore != nil && alertStore != nil {
		colScheduler = collector.NewCollectorScheduler(tsStore, log)

		// Register Collectors
		for _, src := range cfg.Monitor.Collection.Sources {
			if !src.Enabled {
				continue
			}
			if src.Type == "kubernetes" {
				kc, err := collector.NewK8sCollector(src.KubeConfig, "")
				if err == nil {
					colScheduler.Register(kc)
				} else {
					log.Errorf("Failed to create k8s collector: %v", err)
				}
			}
			if src.Type == "middleware" {
				for _, mw := range src.Middlewares {
					// Create middleware collector
					// Note: We pass pluginManager to collector, allowing it to load/use the plugin.
					mc := collector.NewMiddlewareCollector(mw, pluginManager)
					colScheduler.Register(mc)
				}
			}
		}

		// Alerting
		ruleEngine := alert.NewRuleEngine(log)
		if err := ruleEngine.LoadRules(cfg.Monitor.Alerting.Rules); err != nil {
			log.Errorf("Failed to load alert rules: %v", err)
		}

		notifier := alert.NewNotifier(log)
		for _, n := range cfg.Monitor.Alerting.Notifiers {
			if !n.Enabled {
				continue
			}
			if n.Type == "webhook" {
				notifier.Register(channels.NewWebhookNotifier(n.Name, n.URL, n.Timeout))
			} else if n.Type == "email" {
				notifier.Register(channels.NewEmailNotifier(n.Name, channels.SMTPConfig{
					Host: n.SMTP.Host, Port: n.SMTP.Port, Username: n.SMTP.Username, Password: n.SMTP.Password, From: n.SMTP.From, To: n.To,
				}))
			}
		}

		silenceMgr = alert.NewSilenceManager(silenceStore)
		alEvaluator = alert.NewAlertEvaluator(ruleEngine, tsStore, alertStore, notifier, silenceMgr, log)
		monHandler = handlers.NewMonitorHandler(colScheduler, tsStore, alertStore, ruleEngine, silenceMgr)
	}
	// -----------------------------

	s := &Server{
		router:             gin.Default(),
		config:             cfg,
		diagnosisEngine:    diagnosisEngine,
		pluginManager:      pluginManager,
		authService:        authService,
		rbacMiddleware:     rbacMiddleware,
		log:                log,
		wsHandler:          wsHandler,
		taskScheduler:      scheduler,
		taskWorker:         worker,
		taskStore:          store,
		knowledgeAPI:       knowledgeAPI,
		monitorHandler:     monHandler,
		collectorScheduler: colScheduler,
		alertEvaluator:     alEvaluator,
		silenceManager:     silenceMgr,
		timeseriesStore:    tsStore,
		alertStore:         alertStore,
		silenceStore:       silenceStore,
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

	// Serve Static files for UI
	s.router.Static("/static", "./internal/web/static")

	// Serve Dashboard template
	s.router.LoadHTMLGlob("internal/web/templates/*")
	s.router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", nil)
	})

	// WebSocket route - ensure it handles the 'id' parameter query correctly
	s.router.GET("/api/v1/ws/diagnose", s.wsHandler.ServeHTTP)

	// Web Console Routes
	consoleHandler := web.NewConsoleHandler(s.diagnosisEngine, s.taskScheduler, s.taskStore)
	consoleHandler.RegisterRoutes(s.router)

	// API V1
	v1 := s.router.Group("/api/v1")

	// Auth routes (public)
	authHandler := handlers.NewAuthHandler(s.authService)
	v1.POST("/auth/login", authHandler.Login)

	// Protected routes (optionally disabled for this Phase demo/dev)
	// v1.Use(s.authService.JWTAuth())

	// Diagnosis Trigger (mapped to /api/v1/diagnose to match stream.js)
	// IMPORTANT: stream.js calls /api/v1/diagnose, so we map it there.
	diagnosisHandler := handlers.NewDiagnosisHandler(s.diagnosisEngine, s.wsHandler)
	v1.POST("/diagnose", diagnosisHandler.TriggerDiagnosis)

	// Original path support
	diagnosis := v1.Group("/diagnosis")
	// Protected if needed: diagnosis.Use(s.rbacMiddleware.CheckPermission("diagnosis:write"))
	diagnosis.POST("", diagnosisHandler.TriggerDiagnosis)
	diagnosis.GET("/:id", diagnosisHandler.GetDiagnosisResult)

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

	// Monitor Routes
	if s.monitorHandler != nil {
		// mon := v1.Group("/monitor") // Unused for now
		// Requirement says: GET /api/v1/metrics
		v1.GET("/metrics", s.rbacMiddleware.CheckPermission("monitor:read"), s.monitorHandler.GetMetrics)

		alerts := v1.Group("/alerts")
		alerts.GET("/history", s.rbacMiddleware.CheckPermission("monitor:read"), s.monitorHandler.GetAlertHistory)
		alerts.POST("/silence", s.rbacMiddleware.CheckPermission("monitor:write"), s.monitorHandler.CreateSilence)
	}
}

func (s *Server) Start(ctx context.Context) error {
	// Note: wsHandler.Run() is already called in NewHandler, so we don't call it here to avoid double run.

	// Start Task Worker
	if s.taskWorker != nil {
		s.taskWorker.Start()
		defer s.taskWorker.Stop()
	}

	// Start Monitoring
	if s.collectorScheduler != nil {
		go s.collectorScheduler.Start(ctx)
		defer s.collectorScheduler.Stop()
	}
	if s.alertEvaluator != nil {
		go s.alertEvaluator.Start(ctx, s.config.Monitor.Alerting.EvaluationInterval)
	}
	if s.silenceManager != nil {
		go s.silenceManager.GC(ctx)
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

	if s.timeseriesStore != nil {
		s.timeseriesStore.Close()
	}
	if s.alertStore != nil {
		s.alertStore.Close()
	}
	if s.silenceStore != nil {
		s.silenceStore.Close()
	}

	s.log.Info("Server exiting")
	return nil
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() *gin.Engine {
	return s.router
}
