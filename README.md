# API Scheduler Flow Engine

The API Scheduler Flow Engine is a lightweight, Go-based microservice for executing workflows (flows) either manually via API or automatically using cron expressions. 

It is built with **Clean Architecture** principles and uses:
- **Go 1.21+**
- **Gin** for the HTTP API
- **PostgreSQL** (via GORM) for persistent storage
- **robfig/cron** for scheduling
- **Docker & Docker Compose** for easy deployment

## Setup

1. Copy `.env.example` to `.env` and adjust the variables if needed.
2. Start the database and redis using docker-compose:
   ```bash
   docker-compose up -d db redis
   ```
3. Run the application:
   ```bash
   go run cmd/server/main.go
   ```

## API Documentation

All endpoints are prefixed with `/api/v1` and require a JWT token in the `Authorization: Bearer <token>` header.

- **POST /api/v1/flows**: Create a new flow (Requires `ADMIN` role)
- **GET /api/v1/flows**: List all flows
- **GET /api/v1/flows/:id**: Get a specific flow
- **PUT /api/v1/flows/:id**: Update a flow
- **DELETE /api/v1/flows/:id**: Delete a flow

- **POST /api/v1/flows/:id/execute**: Trigger a flow execution manually
- **GET /api/v1/executions**: List all executions
- **GET /api/v1/executions/:id**: Get a specific execution

- **POST /api/v1/flows/:id/schedule**: Create a schedule for a flow
- **PATCH /api/v1/schedules/:id**: Enable or disable a schedule
