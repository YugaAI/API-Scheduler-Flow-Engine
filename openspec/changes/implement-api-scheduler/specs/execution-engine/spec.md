## ADDED Requirements

### Requirement: Trigger Flow Execution
The system SHALL allow authenticated users to trigger the execution of a flow by its ID. The execution SHALL be dispatched asynchronously via the worker pool.

#### Scenario: Successful manual trigger
- **WHEN** a user sends a POST request to `/api/v1/flows/{id}/execute` for an existing flow
- **THEN** the system creates an execution record with status `pending`, dispatches it to the worker pool, and returns the execution ID with status 202

#### Scenario: Trigger non-existing flow
- **WHEN** a user sends a POST request to `/api/v1/flows/{id}/execute` for a non-existing flow
- **THEN** the system returns a 404 error with message "flow not found"

#### Scenario: Worker pool at capacity
- **WHEN** the worker pool has reached max concurrent executions (10)
- **THEN** the system queues the execution via Redis and returns status 202 with the execution ID

### Requirement: Execute Steps Sequentially
The system SHALL execute flow steps in ascending `order` value. Each step SHALL run only after the previous step completes successfully.

#### Scenario: All steps succeed
- **WHEN** a flow with 3 steps is executed and all steps complete successfully
- **THEN** each step runs in order, each step's status transitions from `pending` → `running` → `completed`, and the execution status becomes `completed`

#### Scenario: Step execution captures output
- **WHEN** a step completes (success or failure)
- **THEN** the system stores the step's stdout/stderr output in the `log` field of the execution_step record

### Requirement: Handle Step Failure with Stop Policy
The system SHALL stop execution immediately when a step fails and the failure policy is `stop` (default).

#### Scenario: Step failure with stop policy
- **WHEN** step 2 of a 3-step flow fails after all retries are exhausted and failure_policy is `stop`
- **THEN** the system marks step 2 as `failed`, marks step 3 as `skipped`, sets the execution status to `failed`, and records `finished_at`

### Requirement: Handle Step Failure with Continue Policy
The system SHALL continue to the next step when a step fails and the failure policy is `continue`.

#### Scenario: Step failure with continue policy
- **WHEN** step 2 of a 3-step flow fails after all retries and failure_policy is `continue`
- **THEN** the system marks step 2 as `failed`, proceeds to execute step 3, and sets the final execution status based on overall results

### Requirement: Execution Timeout
The system SHALL enforce a maximum execution duration. If the total execution exceeds the timeout, the execution SHALL be cancelled.

#### Scenario: Execution exceeds timeout
- **WHEN** a flow execution exceeds 300 seconds (default timeout)
- **THEN** the system cancels the currently running step, marks the execution as `failed` with log "execution timeout exceeded", and records `finished_at`

### Requirement: Store Execution Results
The system SHALL persist the execution status, timing, and per-step results throughout the execution lifecycle.

#### Scenario: Execution state persistence
- **WHEN** a flow execution transitions through states (pending → running → completed)
- **THEN** the system updates the execution record in the database at each transition with accurate timestamps

#### Scenario: Step-level result persistence
- **WHEN** each step completes
- **THEN** the system immediately persists the step's status, log output, retry count, and timestamps