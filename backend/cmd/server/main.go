package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/auth"
	"stage-rigging-cue-interlock/backend/internal/config"
	"stage-rigging-cue-interlock/backend/internal/database"
	"stage-rigging-cue-interlock/backend/internal/handler"
	"stage-rigging-cue-interlock/backend/internal/middleware"
	"stage-rigging-cue-interlock/backend/internal/repository"
	"stage-rigging-cue-interlock/backend/internal/router"
	"stage-rigging-cue-interlock/backend/internal/service"
	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration rejected", "error", err)
		os.Exit(1)
	}
	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)
	db, err := database.Open(cfg)
	if err != nil {
		logger.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("database handle failed", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	auditRepository := audit.NewRepository(db)
	authRepository := auth.NewRepository(db)
	deviceRepository := repository.NewRiggingDeviceRepository(db, auditRepository)
	cueRepository := repository.NewCueDefinitionRepository(db, auditRepository)
	ruleRepository := repository.NewInterlockRuleRepository(db, auditRepository)
	runRepository := repository.NewRehearsalRunRepository(db, auditRepository)

	authService := auth.NewService(authRepository, cfg.JWTSecret, cfg.JWTTTL)
	deviceService := service.NewRiggingDeviceService(deviceRepository, ruleRepository)
	cueService := service.NewCueDefinitionService(cueRepository, deviceRepository)
	ruleService := service.NewInterlockRuleService(ruleRepository, deviceRepository)
	runService := service.NewRehearsalRunService(runRepository, cueRepository, deviceRepository, ruleRepository, cfg.TimelineStepMS, cfg.MaxCuesPerRun)

	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Recovery(logger))
	engine.Use(middleware.AccessLog(logger))
	engine.Use(cors(cfg.CORSOrigins))
	engine.Use(middleware.NewLocalRateLimiter(cfg.RateLimitPerMinute).Handler())
	engine.GET("/healthz", func(c *gin.Context) { util.OK(c, gin.H{"status": "ok", "mode": "offline-rehearsal"}) })
	engine.GET("/readyz", func(c *gin.Context) {
		if err := database.Ping(db); err != nil {
			util.Fail(c, util.Internal(err))
			return
		}
		util.OK(c, gin.H{"status": "ready", "database": cfg.DBDriver, "timeline_step_ms": cfg.TimelineStepMS})
	})

	api := engine.Group("/api/v1")
	api.POST("/auth/login", auth.NewHandler(authService).Login)
	protected := api.Group("")
	protected.Use(middleware.Auth(authService))
	write := middleware.RBAC(auth.RoleProgrammer, auth.RoleAdmin)
	review := middleware.RBAC(auth.RoleSafetyReviewer, auth.RoleAdmin)
	router.RegisterRiggingDeviceRoutes(protected, handler.NewRiggingDeviceHandler(deviceService), write)
	router.RegisterCueDefinitionRoutes(protected, handler.NewCueDefinitionHandler(cueService), write, review)
	router.RegisterInterlockRuleRoutes(protected, handler.NewInterlockRuleHandler(ruleService), write, review)
	router.RegisterRehearsalRunRoutes(protected, handler.NewRehearsalRunHandler(runService), write, review)
	protected.GET("/audit-events", review, audit.NewHandler(auditRepository).List)

	server := &http.Server{Addr: ":" + cfg.Port, Handler: engine, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("server listening", "port", cfg.Port, "database", cfg.DBDriver, "mode", "offline-rehearsal")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), cfg.GracefulShutdownPeriod)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return
	}
	logger.Info("server stopped")
}

func cors(origins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		allowed[origin] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func newLogger(level string) *slog.Logger {
	minimum := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		minimum = slog.LevelDebug
	case "warn":
		minimum = slog.LevelWarn
	case "error":
		minimum = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: minimum}))
}
