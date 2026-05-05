## 1. Project Setup & Foundation

- [x] 1.1 Initialize Go module (`go mod init`) and create the clean architecture directory structure: `cmd/server/`, `internal/{domain,application,infrastructure,presentation}/`, `pkg/{config,logger}/`
- [x] 1.2 Create `pkg/config/config.go` — load config from environment variables (DB host, port, user, password, db name, Redis URL, JWT secret, server port, worker pool size, timezone)
- [x] 1.3 Create `pkg/logger/logger.go` — structured JSON logger (use `log/slog` or `zerolog`) with log level configuration
- [x] 1.4 Add Go dependencies: `gin`, `gorm`, `gorm/driver/postgres`, `robfig/cron/v3`, `go-redis/redis/v9`, `golang-jwt/jwt/v5`, `google/uuid`
- [x] 1.5 Create `cmd/server/main.go` — application entrypoint that wires all dependencies (config → DB → repositories → services → handlers → router → server start)
- [x] 1.6 Create `docker-compose.yml` with PostgreSQL 14 and Redis 7 services
- [x] 1.7 Create `Dockerfile` (multi-stage build: Go build → distroless/static runtime)

## 2. Domain Layer (Entities & Interfaces)

- [x] 2.1 Create `internal/domain/entity/flow.go` — Flow struct with UUID, Name, Description, CreatedAt, UpdatedAt, and Steps relationship
- [x] 2.2 Create `internal/domain/entity/step.go` — Step struct with UUID, FlowID, Order, Action, Config (JSONB), RetryCount, RetryDelaySeconds
- [x] 2.3 Create `internal/domain/entity/execution.go` — Execution struct with UUID, FlowID, Status, TriggerType, StartedAt, FinishedAt
- [x] 2.4 Create `internal/domain/entity/execution_step.go` — ExecutionStep struct with UUID, ExecutionID, StepOrder, Action, Status, Log, RetryAttempts, StartedAt, FinishedAt
- [x] 2.5 Create `internal/domain/entity/schedule.go` — Schedule struct with UUID, FlowID, CronExpression, Enabled, LastRunAt, CreatedAt
- [x] 2.6 Create `internal/domain/repository/flow_repository.go` — FlowRepository interface (Create, FindByID, FindAll, Update, Delete with pagination params)
- [x] 2.7 Create `internal/domain/repository/execution_repository.go` — ExecutionRepository interface (Create, FindByID, FindAll, Update, UpdateStep with filter/pagination params)
- [x] 2.8 Create `internal/domain/repository/schedule_repository.go` — ScheduleRepository interface (Create, FindByFlowID, FindAllEnabled, Update, Delete)

## 3. Infrastructure — Database

- [x] 3.1 Create `internal/infrastructure/persistence/postgres/connection.go` — GORM connection setup with connection pool config
- [x] 3.2 Create `internal/infrastructure/persistence/migration.go` — Auto-migrate all entities, enable `uuid-ossp` extension
- [x] 3.3 Create `internal/infrastructure/persistence/postgres/flow_repo_impl.go` — implement FlowRepository with GORM (preload Steps, paginated queries)
- [x] 3.4 Create `internal/infrastructure/persistence/postgres/execution_repo_impl.go` — implement ExecutionRepository with GORM (preload ExecutionSteps, filtered queries)
- [x] 3.5 Create `internal/infrastructure/persistence/postgres/schedule_repo_impl.go` — implement ScheduleRepository with GORM

## 4. Infrastructure — Action Registry

- [x] 4.1 Define `Action` interface in `internal/application/service/action_registry.go`: `Name() string`, `Execute(ctx, config) (string, error)`
- [x] 4.2 Create `ActionRegistry` struct with `Register(action)`, `Get(name) (Action, error)`, `Validate(name) bool` methods
- [x] 4.3 Implement `internal/infrastructure/action/run_script.go` — executes shell commands from config, captures stdout/stderr, respects context cancellation
- [x] 4.4 Implement `internal/infrastructure/action/git_pull.go` — runs `git pull` in specified directory
- [x] 4.5 Implement `internal/infrastructure/action/build.go` — runs configurable build command
- [x] 4.6 Implement `internal/infrastructure/action/test.go` — runs configurable test command
- [x] 4.7 Implement `internal/infrastructure/action/deploy.go` — runs deployment command/script
- [x] 4.8 Implement `internal/infrastructure/action/docker_build.go` — runs `docker build` with config (image name, tag, dockerfile path)
- [x] 4.9 Implement `internal/infrastructure/action/docker_push.go` — runs `docker push` with config (image name, tag, registry)

## 5. Application Layer — Use Cases

- [x] 5.1 Create `internal/application/usecase/flow_usecase.go` — CreateFlow (validate actions via registry), GetFlow, ListFlows, UpdateFlow, DeleteFlow
- [x] 5.2 Create `internal/application/usecase/execution_usecase.go` — TriggerExecution (create record, dispatch to worker pool), GetExecution, ListExecutions
- [x] 5.3 Create `internal/application/usecase/schedule_usecase.go` — CreateSchedule (validate cron), EnableSchedule, DisableSchedule, register/unregister with cron runner

## 6. Application Layer — Execution Engine

- [x] 6.1 Create `internal/application/service/worker_pool.go` — bounded goroutine pool (configurable max workers, default 10) with job queue channel
- [x] 6.2 Create `internal/application/service/executor_service.go` — sequential step executor: iterate steps in order, invoke action, handle retry with delay, apply failure policy (stop/continue), enforce timeout via context.WithTimeout
- [x] 6.3 Integrate executor with worker pool: execution dispatch sends job to pool, pool worker picks up and runs executor

## 7. Application Layer — Scheduler Service

- [x] 7.1 Create `internal/application/service/scheduler_service.go` — wrap robfig/cron with methods: Start, Stop, AddJob(scheduleID, cronExpr, flowID), RemoveJob(scheduleID)
- [x] 7.2 Implement schedule reload on startup: load all enabled schedules from DB, register each with cron runner
- [x] 7.3 Implement cron job callback: on trigger, create execution with trigger_type="scheduled", dispatch to worker pool, update last_run_at

## 8. Presentation Layer — DTOs

- [x] 8.1 Create request DTOs: `CreateFlowRequest`, `UpdateFlowRequest`, `CreateScheduleRequest`, `UpdateScheduleRequest` in `internal/presentation/dto/request/`
- [x] 8.2 Create response DTOs: `FlowResponse`, `ExecutionResponse`, `ExecutionStepResponse`, `ScheduleResponse`, `PaginatedResponse`, `ErrorResponse` in `internal/presentation/dto/response/`

## 9. Presentation Layer — Middleware

- [x] 9.1 Create `internal/presentation/middleware/auth_middleware.go` — JWT validation, extract claims (sub, role), set in Gin context
- [x] 9.2 Create `internal/presentation/middleware/role_middleware.go` — check user role against required role for the endpoint (ADMIN vs ALL)
- [x] 9.3 Create `internal/presentation/middleware/error_handler.go` — centralized error recovery middleware, maps domain errors to HTTP status codes

## 10. Presentation Layer — Handlers

- [x] 10.1 Create `internal/presentation/handler/flow_handler.go` — CreateFlow, GetFlow, ListFlows, UpdateFlow, DeleteFlow handlers with request validation and response mapping
- [x] 10.2 Create `internal/presentation/handler/execution_handler.go` — TriggerExecution, GetExecution, ListExecutions handlers with filter/pagination parsing
- [x] 10.3 Create `internal/presentation/handler/schedule_handler.go` — CreateSchedule, UpdateSchedule (enable/disable) handlers

## 11. Presentation Layer — Router

- [x] 11.1 Create `internal/presentation/router/router.go` — register all routes under `/api/v1` with auth and role middleware applied per route group

## 12. Infrastructure — Redis Queue (Optional Enhancement)

- [x] 12.1 Create `internal/infrastructure/queue/redis_queue.go` — Redis-based job queue for execution dispatch overflow when worker pool is at capacity

## 13. Testing

- [x] 13.1 Write unit tests for domain entities (validation logic, status transitions)
- [x] 13.2 Write unit tests for flow_usecase (mock repository and action registry)
- [x] 13.3 Write unit tests for execution_usecase (mock repository and worker pool)
- [x] 13.4 Write unit tests for executor_service (mock actions, test retry logic, failure policies, timeout)
- [x] 13.5 Write unit tests for scheduler_service (mock cron runner, test job registration)
- [x] 13.6 Write unit tests for auth_middleware (valid/invalid/missing token scenarios)
- [x] 13.7 Write integration tests for flow CRUD API endpoints
- [x] 13.8 Write integration tests for execution trigger and status retrieval
- [x] 13.9 Write integration tests for schedule creation and cron triggering

## 14. Documentation & Finalization

- [x] 14.1 Create `README.md` with project overview, setup instructions, API documentation, and environment variable reference
- [x] 14.2 Add `.env.example` with all required environment variables
- [x] 14.3 Verify docker-compose up starts PostgreSQL, Redis, and the application successfully