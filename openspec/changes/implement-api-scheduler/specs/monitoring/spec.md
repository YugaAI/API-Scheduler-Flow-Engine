## ADDED Requirements

### Requirement: Get Execution Status
The system SHALL allow authenticated users to retrieve the status of an execution by its ID, including all step details.

#### Scenario: Get running execution
- **WHEN** a user sends a GET request to `/api/v1/executions/{id}` for a running execution
- **THEN** the system returns the execution with status `running`, trigger_type, timestamps, and all execution steps with their current statuses and logs

#### Scenario: Get completed execution
- **WHEN** a user sends a GET request to `/api/v1/executions/{id}` for a completed execution
- **THEN** the system returns the execution with status `completed`, `started_at`, `finished_at`, and all step results

#### Scenario: Non-existing execution
- **WHEN** a user sends a GET request to `/api/v1/executions/{id}` for a non-existing UUID
- **THEN** the system returns a 404 error with message "execution not found"

### Requirement: List Executions
The system SHALL allow authenticated users to list executions with filtering and pagination.

#### Scenario: List with filters
- **WHEN** a user sends a GET request to `/api/v1/executions?flow_id=<uuid>&status=failed`
- **THEN** the system returns only executions matching the specified flow_id and status, paginated

#### Scenario: List with default pagination
- **WHEN** a user sends a GET request to `/api/v1/executions` without pagination params
- **THEN** the system returns the first page (page=1, page_size=20) ordered by `started_at` descending

#### Scenario: List with sorting
- **WHEN** a user sends a GET request to `/api/v1/executions?sort=started_at&order=asc`
- **THEN** the system returns executions sorted by the specified field and direction

### Requirement: Structured Logging
The system SHALL use structured JSON logging for all application events, including execution lifecycle events.

#### Scenario: Execution lifecycle logging
- **WHEN** an execution transitions state (pending → running → completed)
- **THEN** the system emits a structured log entry with fields: `execution_id`, `flow_id`, `status`, `timestamp`, `trigger_type`

#### Scenario: Step execution logging
- **WHEN** a step starts or completes execution
- **THEN** the system emits a structured log entry with fields: `execution_id`, `step_order`, `action`, `status`, `duration_ms`

#### Scenario: Error logging
- **WHEN** an error occurs during execution
- **THEN** the system logs the error with fields: `execution_id`, `step_order`, `error`, `retry_attempt`, `stack_trace`