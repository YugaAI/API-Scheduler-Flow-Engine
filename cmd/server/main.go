package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/openspec/api-scheduler-flow-engine/internal/application/service"
	"github.com/openspec/api-scheduler-flow-engine/internal/application/usecase"
	"github.com/openspec/api-scheduler-flow-engine/internal/infrastructure/action"
	"github.com/openspec/api-scheduler-flow-engine/internal/infrastructure/persistence"
	"github.com/openspec/api-scheduler-flow-engine/internal/infrastructure/persistence/postgres"
	"github.com/openspec/api-scheduler-flow-engine/internal/infrastructure/queue"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/handler"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/router"
	"github.com/openspec/api-scheduler-flow-engine/pkg/config"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found or error loading it, using environment variables")
	}

	logger.Init(os.Getenv("LOG_LEVEL"))
	defer logger.Close()

	logger.Info("Starting API Scheduler Flow Engine")

	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	redisQueue, err := queue.NewRedisQueue(cfg.RedisURL)
	if err != nil {
		logger.Error("Failed to initialize Redis queue", "error", err)
		os.Exit(1)
	}
	defer redisQueue.Close()

	db, err := postgres.NewConnection(cfg)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := persistence.RunMigrations(db); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	flowRepo := postgres.NewFlowRepository(db)
	executionRepo := postgres.NewExecutionRepository(db)
	scheduleRepo := postgres.NewScheduleRepository(db)

	actionRegistry := service.NewActionRegistry()
	actionRegistry.Register(&action.RunScriptAction{})
	actionRegistry.Register(&action.GitPullAction{})
	actionRegistry.Register(&action.BuildAction{})
	actionRegistry.Register(&action.TestAction{})
	actionRegistry.Register(&action.DeployAction{})
	actionRegistry.Register(&action.DockerBuildAction{})
	actionRegistry.Register(&action.DockerPushAction{})

	executorService := service.NewExecutorService(executionRepo, flowRepo, actionRegistry, redisQueue, cfg)

	workerPool := service.NewWorkerPool(cfg.WorkerPoolSize, executorService, redisQueue)
	workerPool.Start()

	schedulerService, err := service.NewSchedulerService(scheduleRepo, executionRepo, workerPool, cfg.Timezone)
	if err != nil {
		logger.Error("Failed to initialize scheduler", "error", err)
		os.Exit(1)
	}
	if err := schedulerService.Start(context.Background()); err != nil {
		logger.Error("Failed to start scheduler", "error", err)
		os.Exit(1)
	}

	flowUseCase := usecase.NewFlowUseCase(flowRepo, actionRegistry)
	executionUseCase := usecase.NewExecutionUseCase(executionRepo, flowRepo, workerPool)
	scheduleUseCase := usecase.NewScheduleUseCase(scheduleRepo, flowRepo, schedulerService)

	flowHandler := handler.NewFlowHandler(flowUseCase)
	executionHandler := handler.NewExecutionHandler(executionUseCase)
	scheduleHandler := handler.NewScheduleHandler(scheduleUseCase)

	r := router.SetupRouter(cfg.JWTSecret, flowHandler, executionHandler, scheduleHandler)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server di goroutine
	go func() {
		logger.Info("Starting HTTP server", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// ✅ SINGLE shutdown path — tidak ada goroutine signal handler lain
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Urutan shutdown penting:
	// 1. Stop terima request baru
	// 2. Stop scheduler (tidak dispatch job baru)
	// 3. Stop worker pool (selesaikan job yang sedang berjalan)
	// 4. Shutdown HTTP server (tunggu request aktif selesai)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Stop scheduler — tidak ada job baru masuk
	schedulerService.Stop()
	logger.Info("Scheduler stopped")

	// Step 2: Stop worker pool — tunggu job aktif selesai
	workerPool.Stop()
	logger.Info("Worker pool stopped")

	// Step 3: Graceful HTTP shutdown — tunggu request aktif
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited cleanly")
}
