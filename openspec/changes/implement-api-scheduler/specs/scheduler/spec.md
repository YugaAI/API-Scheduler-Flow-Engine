## ADDED Requirements

### Requirement: Create Schedule
The system SHALL allow ADMIN users to create a cron schedule for a flow using a standard cron expression.

#### Scenario: Successful schedule creation
- **WHEN** an ADMIN sends a POST request to `/api/v1/flows/{id}/schedule` with a valid cron expression
- **THEN** the system creates the schedule, registers it with the cron runner, and returns the schedule details with status 201

#### Scenario: Invalid cron expression
- **WHEN** an ADMIN sends a POST request with an invalid cron expression
- **THEN** the system returns a 400 error with message "invalid cron expression"

#### Scenario: Flow already has a schedule
- **WHEN** an ADMIN creates a schedule for a flow that already has one
- **THEN** the system updates the existing schedule's cron expression and re-registers the cron job

### Requirement: Automatic Execution by Schedule
The system SHALL automatically trigger flow executions when a scheduled cron time arrives. The timezone SHALL be `Asia/Jakarta`.

#### Scenario: Cron trigger fires
- **WHEN** the cron schedule's next execution time arrives
- **THEN** the system creates a new execution with `trigger_type: "scheduled"`, dispatches it to the worker pool, and updates `last_run_at` on the schedule

#### Scenario: Disabled schedule does not fire
- **WHEN** the cron time arrives for a disabled schedule
- **THEN** the system does NOT trigger an execution

### Requirement: Enable and Disable Schedule
The system SHALL allow ADMIN users to enable or disable a schedule without deleting it.

#### Scenario: Disable an active schedule
- **WHEN** an ADMIN sends a PATCH request to disable a schedule
- **THEN** the system sets `enabled: false`, removes the cron job from the runner, and returns the updated schedule

#### Scenario: Enable a disabled schedule
- **WHEN** an ADMIN sends a PATCH request to enable a schedule
- **THEN** the system sets `enabled: true`, registers the cron job with the runner, and returns the updated schedule

### Requirement: Schedule Persistence Across Restarts
The system SHALL reload all enabled schedules from the database on application startup and register them with the cron runner.

#### Scenario: Application restart
- **WHEN** the application restarts
- **THEN** all enabled schedules are loaded from the database and registered with the cron runner, resuming automatic execution

### Requirement: Dynamic Job Registration
The system SHALL support adding, removing, and updating cron jobs at runtime without restarting the application.

#### Scenario: Runtime job update
- **WHEN** a schedule's cron expression is updated via the API
- **THEN** the system removes the old cron job and registers the new one immediately without affecting other schedules