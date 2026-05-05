## Context

This design implements the API Scheduler Flow Engine — a greenfield Go microservice that allows users to define workflows (flows) as ordered sequences of steps, execute them manually or via cron schedules, and monitor execution with detailed logging. The system follows Clean Architecture principles with four layers: presentation (REST API), application (orchestration), domain (entities & business rules), and infrastructure (database, queue, external services).

There is no existing workflow system. The project starts from scratch with a well-defined tech stack: Go 1.21+, Gin, PostgreSQL, GORM, Redis, and robfig/cron.

## Goals / Non-Goals

**Goals:**
- Enable dynamic workflow execution based on user-defined flows
- Support automated scheduling with cron expressions
- Provide detailed monitoring and logging of executions
- Implement retry and failure handling mechanisms
- Use clean architecture for maintainability

**Non-Goals:**
- Visual flow builder UI (future scope)
- Parallel/DAG execution — only sequential steps are supported in v1.0
- Webhook or notification integrations (future scope)
- Flow versioning or audit trail (future scope)
- Multi-tenancy or organization-level isolation

## Decisions

### 1. Project Structure (Clean Architecture)

```
.
├── cmd/
│   └── server/
│       └── main.go                 # Application entrypoint
├── internal/
│   ├── domain/                     # Entities & interfaces (no external deps)
│   │   ├── entity/
│   │   │   ├── flow.go
│   │   │   ├── step.go
│   │   │   ├── execution.go
│   │   │   └── execution_step.go
│   │   └── repository/
│   │       ├── flow_repository.go
│   │       ├── execution_repository.go
│   │       └── schedule_repository.go
│   ├── application/                # Use cases & business logic
│   │   ├── usecase/
│   │   │   ├── flow_usecase.go
│   │   │   ├── execution_usecase.go
│   │   │   └── schedule_usecase.go
│   │   └── service/
│   │       ├── executor_service.go # Step execution orchestrator
│   │       ├── action_registry.go  # Action handler registry
│   │       ├── scheduler_service.go
│   │       └── worker_pool.go
│   ├── infrastructure/             # External implementations
│   │   ├── persistence/
│   │   │   ├── postgres/
│   │   │   │   ├── connection.go
│   │   │   │   ├── flow_repo_impl.go
│   │   │   │   ├── execution_repo_impl.go
│   │   │   │   └── schedule_repo_impl.go
│   │   │   └── migration.go
│   │   ├── queue/
│   │   │   └── redis_queue.go
│   │   └── action/                 # Built-in action implementations
│   │       ├── git_pull.go
│   │       ├── build.go
│   │       ├── test.go
│   │       ├── deploy.go
│   │       ├── run_script.go
│   │       ├── docker_build.go
│   │       └── docker_push.go
│   └── presentation/               # HTTP handlers & middleware
│       ├── handler/
│       │   ├── flow_handler.go
│       │   ├── execution_handler.go
│       │   └── schedule_handler.go
│       ├── middleware/
│       │   ├── auth_middleware.go
│       │   └── error_handler.go
│       ├── dto/
│       │   ├── request/
│       │   └── response/
│       └── router/
│           └── router.go
├── pkg/
│   ├── config/
│   │   └── config.go
│   └── logger/
│       └── logger.go
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── go.sum
```

**Rationale**: Clean Architecture ensures the domain layer has zero external dependencies. All I/O crosses boundaries through interfaces defined in the domain layer. This makes the business logic testable in isolation and the infrastructure swappable.

### 2. Data Model

**Flows** — Define the workflow template with ordered steps.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | `gorm:"type:uuid;primary_key;default:gen_random_uuid()"` |
| name | VARCHAR(255) | Unique, required |
| description | TEXT | Optional |
| created_at | TIMESTAMP | Auto-set |
| updated_at | TIMESTAMP | Auto-set |

**Steps** — Individual steps within a flow, stored as separate rows (not embedded JSON).

| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | Auto-generated |
| flow_id | UUID (FK→flows) | CASCADE delete |
| order | INTEGER | 1-indexed, unique per flow |
| action | VARCHAR(100) | Must match registered action name |
| config | JSONB | Action-specific configuration |
| retry_count | INTEGER | Default: 3 |
| retry_delay_seconds | INTEGER | Default: 5 |

**Executions** — Instance of a flow execution.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | Auto-generated |
| flow_id | UUID (FK→flows) | SET NULL on delete |
| status | VARCHAR(20) | `pending`, `running`, `completed`, `failed`, `cancelled` |
| trigger_type | VARCHAR(20) | `manual`, `scheduled` |
| started_at | TIMESTAMP | Set when status → running |
| finished_at | TIMESTAMP | Set on completion/failure |

**ExecutionSteps** — Per-step execution result.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | Auto-generated |
| execution_id | UUID (FK→executions) | CASCADE delete |
| step_order | INTEGER | Matches step.order |
| action | VARCHAR(100) | Snapshot of action name |
| status | VARCHAR(20) | `pending`, `running`, `completed`, `failed`, `skipped` |
| log | TEXT | Captured stdout/stderr |
| retry_attempts | INTEGER | How many retries were used |
| started_at | TIMESTAMP | |
| finished_at | TIMESTAMP | |

**Schedules** — Cron schedule linking to a flow.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | Auto-generated |
| flow_id | UUID (FK→flows) | CASCADE delete |
| cron_expression | VARCHAR(100) | Standard cron format |
| enabled | BOOLEAN | Default: true |
| last_run_at | TIMESTAMP | Nullable |
| created_at | TIMESTAMP | |

**Rationale**: Steps are stored as separate rows (not embedded JSON) for query flexibility, referential integrity, and easier retry tracking per step. JSONB is used only for action-specific config that varies by action type.

### 3. Action Registry Pattern

Actions implement a common interface:

```go
type Action interface {
    Name() string
    Execute(ctx context.Context, config json.RawMessage) (output string, err error)
}
```

The `ActionRegistry` maps action names to implementations and validates during flow creation that all referenced actions exist. Built-in actions: `git_pull`, `build`, `test`, `deploy`, `run_script`, `docker_build`, `docker_push`.

**Rationale**: Interface-based registry allows adding new action types without modifying the execution engine. Each action is a self-contained unit with its own config schema.

### 4. Execution Flow

```
POST /api/v1/flows/{id}/execute
    → ExecutionUseCase.Execute(flowID)
        → Create Execution record (status: pending)
        → Dispatch to WorkerPool
            → WorkerPool picks up job
                → Set Execution status: running
                → For each Step (ordered):
                    → Set ExecutionStep status: running
                    → ActionRegistry.Get(step.action).Execute(ctx, step.config)
                    → On success: status: completed, save log
                    → On failure:
                        → Retry up to step.retry_count with delay
                        → If all retries fail:
                            → If failure_policy == "stop": mark execution failed, stop
                            → If failure_policy == "continue": mark step failed, continue
                → Set Execution status: completed (or failed)
```

**Rationale**: Worker pool provides bounded concurrency (max 10 goroutines). Execution dispatch is async — the API returns immediately with execution ID. Context propagation allows timeout enforcement (300s default).

### 5. Authentication & Authorization

- JWT tokens via `Authorization: Bearer <token>` header
- Middleware validates token and extracts claims (user ID, role)
- ADMIN: full access to all endpoints
- USER: read-only access to flows, can trigger executions, view own executions

**Rationale**: JWT is stateless and well-suited for API-first services. Role-based access keeps the auth model simple for v1.0.

### 6. API Design

All endpoints under `/api/v1`:

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /flows | ADMIN | Create flow with steps |
| GET | /flows | ALL | List flows (paginated) |
| GET | /flows/{id} | ALL | Get flow detail |
| PUT | /flows/{id} | ADMIN | Update flow |
| DELETE | /flows/{id} | ADMIN | Delete flow |
| POST | /flows/{id}/execute | ALL | Trigger execution |
| GET | /executions | ALL | List executions (filtered) |
| GET | /executions/{id} | ALL | Get execution status + steps |
| POST | /flows/{id}/schedule | ADMIN | Create/update schedule |

Pagination: `?page=1&page_size=20` (default page_size=20, max=100).
Filtering: `?flow_id=<uuid>&status=<status>` on execution list.

## Risks / Trade-offs

- **[Sequential-only execution]** → Limits throughput for flows with independent steps. Mitigation: Designed for DAG extension in future scope; current interface supports refactoring to parallel.
- **[Worker pool saturation]** → 10 concurrent executions may bottleneck under high load. Mitigation: Configurable via env var; Redis queue absorbs burst traffic.
- **[Action execution security]** → `run_script` action executes arbitrary commands. Mitigation: ADMIN-only flow creation; consider sandboxing in future.
- **[Database contention]** → Frequent status updates during execution. Mitigation: Batch log writes; use PostgreSQL advisory locks for execution ownership.
- **[Cron reliability]** → In-memory cron scheduler loses jobs on crash. Mitigation: Schedules persisted in DB; reload on startup; `last_run_at` prevents duplicate triggers.

## Migration Plan

- **Deployment**: New microservice — no impact on existing systems
- **Database**: Auto-migration via GORM on startup (create tables if not exist)
- **Infrastructure**: Requires PostgreSQL 14+ and Redis 7+ (provided via docker-compose)
- **Rollback**: Stop the service, drop the database schema. No downstream dependencies.

## Open Questions

- None — all technical decisions are resolved based on `.openspec.yaml` specification.