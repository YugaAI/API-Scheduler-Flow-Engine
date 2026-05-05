## Why

There is no existing system for defining, scheduling, and executing dynamic workflows via API. Teams manually orchestrate deployment pipelines, task sequences, and recurring jobs — leading to inconsistency, lack of auditability, and wasted engineering time. This change introduces an **API Scheduler Flow Engine** that enables users to define reusable workflow flows (ordered sequences of steps), trigger them manually or via cron schedules, and monitor execution with detailed step-level logging and retry capabilities.

## What Changes

- **Flow Management**: Full CRUD API for defining flows with ordered steps. Each step references a registered action (e.g., `git_pull`, `build`, `deploy`, `docker_build`) with per-step JSON configuration.
- **Execution Engine**: Sequential step-by-step execution engine using a bounded worker pool (max 10 concurrent executions). Each step runs the corresponding action handler, captures output/logs, and tracks status transitions (`pending` → `running` → `completed` | `failed`).
- **Action Registry**: Pluggable action system supporting 7 built-in actions: `git_pull`, `build`, `test`, `deploy`, `run_script`, `docker_build`, `docker_push`. Each action implements a common interface for extensibility.
- **Cron Scheduler**: Automatic flow execution based on cron expressions using `robfig/cron` with `Asia/Jakarta` timezone. Supports dynamic job registration, start/stop, and schedule persistence across restarts.
- **Monitoring & Logging**: REST endpoints for querying execution status, step-level logs, and filtered execution history with pagination and sorting.
- **Retry & Failure Handling**: Configurable per-step retry with exponential delay (default: 3 retries, 5s delay). Supports `stop` and `continue` failure policies. Execution timeout of 300 seconds.
- **Security**: JWT-based authentication with role-based access control (ADMIN, USER).
- **Containerization**: Dockerized application with docker-compose for PostgreSQL and Redis dependencies.

## Capabilities

### New Capabilities
- `flow-management`: CRUD operations for flows with ordered step definitions, JSON step configuration, and input validation
- `execution-engine`: Sequential step execution orchestrator with worker pool, status tracking, and action dispatch
- `action-registry`: Pluggable action handler system with 7 built-in actions and a common execution interface
- `scheduler`: Cron-based automatic flow execution with dynamic job management, timezone support, and schedule persistence
- `monitoring`: Execution status queries, step-level log retrieval, filtered execution listing with pagination
- `retry-failure-handling`: Configurable retry logic with delay, failure policies (stop/continue), and execution timeouts
- `auth-security`: JWT authentication and RBAC for ADMIN and USER roles

### Modified Capabilities
<!-- No existing capabilities are being modified — this is a greenfield project -->

## Impact

- **New REST API**: 9+ endpoints under `/api/v1` for flows, executions, and schedules
- **Database Schema**: 5 tables — `flows`, `steps`, `executions`, `execution_steps`, `schedules` (PostgreSQL via GORM with auto-migration)
- **Infrastructure Dependencies**: PostgreSQL (primary storage), Redis (queue for async execution dispatch)
- **External Libraries**: `gin` (HTTP framework), `gorm` (ORM), `robfig/cron` (scheduler), `golang-jwt/jwt` (auth)
- **Personas Affected**: Admin (flow management, scheduling, monitoring), Developer (API-triggered execution, log viewing)
- **Deployment**: Docker container with docker-compose for local development; standalone binary for production