# API Scheduler Flow Engine

A lightweight, Go-based microservice for executing workflows (flows) either manually via API or automatically using cron expressions. Built with **Clean Architecture** principles for maintainability and scalability.

## Tech Stack

- **Go 1.21+**
- **Gin** - HTTP web framework
- **PostgreSQL** (via GORM) - Persistent storage
- **Redis** - Job queue & retry state tracking
- **robfig/cron** - Cron-based scheduling
- **Docker & Docker Compose** - Containerization

## Architecture
```text
├── cmd/
│   └── server/                  # Application entry point
│
├── internal/
│   ├── application/
│   │   ├── service/             # Business logic services
│   │   └── usecase/             # Use case implementations
│   │
│   ├── domain/
│   │   ├── entity/              # Domain entities (Flow, Execution, Schedule, Step)
│   │   └── repository/          # Repository interfaces
│   │
│   ├── infrastructure/
│   │   ├── action/              # Built-in actions
│   │   ├── persistence/         # Database implementations (PostgreSQL)
│   │   └── queue/               # Redis queue implementation
│   │
│   └── presentation/
│       ├── dto/                 # Request/Response DTOs
│       ├── handler/             # HTTP handlers
│       ├── middleware/          # Auth, error handling, role middleware
│       └── router/              # Route definitions
│
└── pkg/
    ├── config/                  # Configuration loader
    └── logger/                  # Structured logging
```

## Features

- **Flow Management** - Create, update, delete, and list workflows
- **Manual Execution** - Trigger flows via API
- **Scheduled Execution** - Cron-based automatic flow execution
- **Worker Pool** - Concurrent job execution with configurable pool size
- **Retry Mechanism** - Automatic retry with configurable count and delay
- **Redis Queue** - Reliable job queuing with retry state tracking
- **JWT Authentication** - Secure API with role-based access control
- **Graceful Shutdown** - Clean shutdown handling for all components

## Built-in Actions

| Action | Description |
|--------|-------------|
| `run_script` | Execute shell scripts |
| `git_pull` | Pull from Git repository |
| `build` | Build application |
| `test` | Run tests |
| `deploy` | Deploy application |
| `docker_build` | Build Docker image |
| `docker_push` | Push Docker image to registry |

## Setup

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL
- Redis

### Quick Start

1. **Clone the repository**

2. **Copy environment file**
   ```bash
   cp .env.example .env
   ```

3. **Configure environment variables** (see Configuration section)

4. **Start dependencies**
   ```bash
   docker-compose up -d db redis
   ```

5. **Run the application**
   ```bash
   go run cmd/server/main.go
   ```

### Using Docker

```bash
docker-compose up -d
```

## Configuration

| Variable | Description | Default |
|---|---|---|
| `SERVER_PORT` | HTTP server port | `8080` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | PostgreSQL user | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | `postgres` |
| `DB_NAME` | Database name | `flow_engine` |
| `DB_SSL_MODE` | SSL mode | `disable` |
| `REDIS_URL` | Redis connection URL | `localhost:6379` |
| `JWT_SECRET` | JWT signing secret | _required_ |
| `WORKER_POOL_SIZE` | Number of concurrent workers | `10` |
| `TIMEZONE` | Scheduler timezone | `Asia/Jakarta` |
| `EXECUTION_TIMEOUT_SECONDS` | Max execution time | `300` |
| `DEFAULT_RETRY_COUNT` | Default retry attempts | `3` |
| `DEFAULT_RETRY_DELAY` | Delay between retries (seconds) | `5` |
| `FAILURE_POLICY` | Policy on step failure (`stop` / `continue`) | `stop` |
| `LOG_LEVEL` | Logging level | - |

---

## API Documentation

All endpoints are prefixed with `/api/v1` and require a JWT token in the header:

```http
Authorization: Bearer <token>
```

### Flows

| Method | Endpoint | Description | Role |
|---|---|---|---|
| `POST` | `/api/v1/flows` | Create a new flow | `ADMIN` |
| `GET` | `/api/v1/flows` | List all flows | - |
| `GET` | `/api/v1/flows/:id` | Get a specific flow | - |
| `PUT` | `/api/v1/flows/:id` | Update a flow | `ADMIN` |
| `DELETE` | `/api/v1/flows/:id` | Delete a flow | `ADMIN` |

---

### Executions

| Method | Endpoint | Description | Role |
|---|---|---|---|
| `POST` | `/api/v1/flows/:id/execute` | Trigger flow execution | - |
| `GET` | `/api/v1/executions` | List all executions | - |
| `GET` | `/api/v1/executions/:id` | Get execution details | - |

---

### Schedules

| Method | Endpoint | Description | Role |
|---|---|---|---|
| `POST` | `/api/v1/flows/:id/schedule` | Create a schedule | `ADMIN` |
| `PATCH` | `/api/v1/schedules/:id` | Enable / disable schedule | `ADMIN` |

---

## Postman Collection

Import the Postman collection for ready-to-use API examples:

```txt
api-scheduler-postman-collection.json
```

---

## Generate JWT Token

Use the helper script to generate JWT tokens for testing.
```
go run cmd/generate_token/main.go
```