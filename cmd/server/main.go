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
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/handler"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/router"
	"github.com/openspec/api-scheduler-flow-engine/pkg/config"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
)

func main() {
	// Attempt to load .env file
	if err := godotenv.Load(); err != nil {
		// Ignore error if file doesn't exist, as env vars might be set externally
		fmt.Println("No .env file found or error loading it, using environment variables")
	}

	// Initialize logger
	logger.Init(os.Getenv("LOG_LEVEL"))
	logger.Info("Starting API Scheduler Flow Engine")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize Database
	db, err := postgres.NewConnection(cfg)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := persistence.RunMigrations(db); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize Repositories
	flowRepo := postgres.NewFlowRepository(db)
	executionRepo := postgres.NewExecutionRepository(db)
	scheduleRepo := postgres.NewScheduleRepository(db)

	// Initialize Action Registry
	actionRegistry := service.NewActionRegistry()
	actionRegistry.Register(&action.RunScriptAction{})
	actionRegistry.Register(&action.GitPullAction{})
	actionRegistry.Register(&action.BuildAction{})
	actionRegistry.Register(&action.TestAction{})
	actionRegistry.Register(&action.DeployAction{})
	actionRegistry.Register(&action.DockerBuildAction{})
	actionRegistry.Register(&action.DockerPushAction{})

	// Initialize Executor Service
	executorService := service.NewExecutorService(executionRepo, flowRepo, actionRegistry, cfg)

	// Initialize Worker Pool
	workerPool := service.NewWorkerPool(cfg.WorkerPoolSize, executorService)
	workerPool.Start()

	// Initialize Scheduler Service
	schedulerService, err := service.NewSchedulerService(scheduleRepo, executionRepo, workerPool, cfg.Timezone)
	if err != nil {
		logger.Error("Failed to initialize scheduler", "error", err)
		os.Exit(1)
	}
	if err := schedulerService.Start(context.Background()); err != nil {
		logger.Error("Failed to start scheduler", "error", err)
		os.Exit(1)
	}

	// Initialize Use Cases
	flowUseCase := usecase.NewFlowUseCase(flowRepo, actionRegistry)
	executionUseCase := usecase.NewExecutionUseCase(executionRepo, flowRepo, workerPool)
	scheduleUseCase := usecase.NewScheduleUseCase(scheduleRepo, flowRepo, schedulerService)

	// Initialize HTTP Handlers & Router
	flowHandler := handler.NewFlowHandler(flowUseCase)
	executionHandler := handler.NewExecutionHandler(executionUseCase)
	scheduleHandler := handler.NewScheduleHandler(scheduleUseCase)

	r := router.SetupRouter(cfg.JWTSecret, flowHandler, executionHandler, scheduleHandler)

	logger.Info("Starting HTTP server", "port", cfg.ServerPort)
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Stop Scheduler, wait for WorkerPool to finish, shutdown HTTP server gracefully
	schedulerService.Stop()
	workerPool.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exiting")
}
