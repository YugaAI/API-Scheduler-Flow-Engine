## ADDED Requirements

### Requirement: Register Built-in Actions
The system SHALL register 7 built-in action handlers at startup: `git_pull`, `build`, `test`, `deploy`, `run_script`, `docker_build`, `docker_push`.

#### Scenario: All actions registered
- **WHEN** the application starts
- **THEN** the action registry contains all 7 built-in actions, each implementing the `Action` interface

### Requirement: Action Interface Contract
Each action SHALL implement a common interface: `Execute(ctx context.Context, config json.RawMessage) (output string, err error)`. The action reads its specific configuration from the `config` parameter.

#### Scenario: Action receives config
- **WHEN** the execution engine invokes an action
- **THEN** the action receives the step's JSON config and uses it to configure its behavior (e.g., git URL, build command, Docker image name)

#### Scenario: Action returns output
- **WHEN** an action completes successfully
- **THEN** it returns the captured output (stdout) as a string and nil error

#### Scenario: Action returns error
- **WHEN** an action fails (e.g., build command exits non-zero)
- **THEN** it returns captured output and a non-nil error with descriptive message

### Requirement: Action Validation on Flow Creation
The system SHALL validate that all step actions in a flow reference registered action names during flow creation and update.

#### Scenario: Valid action reference
- **WHEN** a flow is created with steps referencing `git_pull` and `build`
- **THEN** the system accepts the flow since both actions are registered

#### Scenario: Invalid action reference
- **WHEN** a flow is created with a step referencing `unknown_action`
- **THEN** the system rejects the flow with a 400 error: "unknown action: unknown_action"

### Requirement: Action Context Cancellation
Each action SHALL respect context cancellation for timeout enforcement. Long-running actions SHALL check context.Done() periodically.

#### Scenario: Action cancelled by timeout
- **WHEN** the execution context is cancelled during a long-running action
- **THEN** the action stops execution and returns a context cancellation error

### Requirement: run_script Action
The `run_script` action SHALL execute an arbitrary shell command specified in its config and capture stdout/stderr.

#### Scenario: Script execution
- **WHEN** the `run_script` action is invoked with config `{"command": "echo hello", "workdir": "/tmp"}`
- **THEN** the action executes the command in the specified directory and returns the output "hello"

#### Scenario: Script failure
- **WHEN** the `run_script` action executes a command that exits with non-zero status
- **THEN** the action returns stderr output and an error with the exit code
