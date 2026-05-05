## ADDED Requirements

### Requirement: Configure Retry per Step
The system SHALL allow configuring retry count and delay for each step in a flow. Defaults: retry_count=3, retry_delay_seconds=5.

#### Scenario: Custom retry config
- **WHEN** a flow is created with a step having `retry_count: 5` and `retry_delay_seconds: 10`
- **THEN** the system stores the retry configuration and uses it during execution

#### Scenario: Default retry config
- **WHEN** a flow is created with a step that does not specify retry settings
- **THEN** the system uses defaults: `retry_count: 3`, `retry_delay_seconds: 5`

### Requirement: Automatic Step Retry
The system SHALL automatically retry a failed step up to the configured retry count, waiting the configured delay between attempts.

#### Scenario: Step succeeds on retry
- **WHEN** a step fails on the first attempt but succeeds on the second attempt
- **THEN** the system records `retry_attempts: 1` on the execution step, sets status to `completed`, and continues to the next step

#### Scenario: Step fails all retries
- **WHEN** a step fails on all retry attempts (e.g., 3 of 3)
- **THEN** the system records `retry_attempts: 3`, sets the step status to `failed`, and applies the failure policy

#### Scenario: Retry delay enforcement
- **WHEN** a step fails and is retried
- **THEN** the system waits exactly `retry_delay_seconds` before the next attempt

### Requirement: Failure Policy — Stop
The system SHALL stop execution and mark it as `failed` when a step exhausts all retries and the failure policy is `stop` (default).

#### Scenario: Stop on failure
- **WHEN** step 2 of 4 fails after all retries with failure_policy `stop`
- **THEN** steps 3 and 4 are marked as `skipped`, the execution status becomes `failed`

### Requirement: Failure Policy — Continue
The system SHALL continue to the next step when a step exhausts all retries and the failure policy is `continue`.

#### Scenario: Continue on failure
- **WHEN** step 2 of 4 fails after all retries with failure_policy `continue`
- **THEN** the system proceeds to step 3, and the final execution status reflects whether any step failed

### Requirement: Execution Timeout Enforcement
The system SHALL enforce a maximum execution timeout of 300 seconds (configurable). When exceeded, the execution SHALL be cancelled.

#### Scenario: Timeout during step execution
- **WHEN** total execution time exceeds 300 seconds
- **THEN** the system cancels the current step's context, marks the step as `failed` with log "timeout exceeded", and marks remaining steps as `skipped`

### Requirement: Error Logging for Failures
The system SHALL record detailed error information for every failed step, including error message, retry attempt number, and timestamp.

#### Scenario: Error detail capture
- **WHEN** a step fails
- **THEN** the system stores the error message in the step's `log` field, along with the retry attempt number and failure timestamp