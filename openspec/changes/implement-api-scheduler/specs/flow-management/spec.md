## ADDED Requirements

### Requirement: Create Flow
The system SHALL allow authenticated users with ADMIN role to create a new flow with a name, optional description, and an ordered list of steps.

#### Scenario: Successful flow creation
- **WHEN** an ADMIN sends a POST request to `/api/v1/flows` with a valid name and steps array
- **THEN** the system creates the flow with its steps and returns the flow details with status 201

#### Scenario: Invalid steps — empty array
- **WHEN** an ADMIN sends a POST request to `/api/v1/flows` with an empty steps array
- **THEN** the system returns a 400 error with message "at least one step is required"

#### Scenario: Invalid steps — unknown action
- **WHEN** an ADMIN sends a POST request with a step referencing an unregistered action name
- **THEN** the system returns a 400 error with message "unknown action: <name>"

#### Scenario: Invalid steps — duplicate order
- **WHEN** an ADMIN sends a POST request with steps having duplicate order values
- **THEN** the system returns a 400 error with message "step orders must be unique"

#### Scenario: Missing required fields
- **WHEN** an ADMIN sends a POST request without a name
- **THEN** the system returns a 400 error with validation details

### Requirement: Get Flow by ID
The system SHALL allow authenticated users to retrieve a flow by its UUID, including all associated steps.

#### Scenario: Existing flow
- **WHEN** a user sends a GET request to `/api/v1/flows/{id}` for an existing flow
- **THEN** the system returns the flow details including name, description, steps (ordered), and timestamps

#### Scenario: Non-existing flow
- **WHEN** a user sends a GET request to `/api/v1/flows/{id}` for a non-existing UUID
- **THEN** the system returns a 404 error with message "flow not found"

#### Scenario: Invalid UUID format
- **WHEN** a user sends a GET request with an invalid UUID format
- **THEN** the system returns a 400 error with message "invalid flow ID format"

### Requirement: Update Flow
The system SHALL allow ADMIN users to update an existing flow's name, description, and steps. Updating steps replaces the entire step list.

#### Scenario: Successful update
- **WHEN** an ADMIN sends a PUT request to `/api/v1/flows/{id}` with updated name and steps
- **THEN** the system replaces the flow's steps, updates the name, sets `updated_at`, and returns the updated flow

#### Scenario: Update non-existing flow
- **WHEN** an ADMIN sends a PUT request to `/api/v1/flows/{id}` for a non-existing flow
- **THEN** the system returns a 404 error

#### Scenario: Update flow with active schedule
- **WHEN** an ADMIN updates a flow that has an active schedule
- **THEN** the system updates the flow and the next scheduled execution uses the updated steps

### Requirement: Delete Flow
The system SHALL allow ADMIN users to delete a flow by ID. Deleting a flow SHALL cascade-delete all associated steps and schedules.

#### Scenario: Successful deletion
- **WHEN** an ADMIN sends a DELETE request to `/api/v1/flows/{id}` for an existing flow
- **THEN** the system deletes the flow, its steps, and associated schedules, and returns status 204

#### Scenario: Delete flow with running execution
- **WHEN** an ADMIN deletes a flow while an execution is running
- **THEN** the system deletes the flow but the running execution continues to completion (execution.flow_id set to NULL)

### Requirement: List Flows
The system SHALL allow authenticated users to list all flows with pagination.

#### Scenario: List with default pagination
- **WHEN** a user sends a GET request to `/api/v1/flows` without pagination params
- **THEN** the system returns the first page (page=1, page_size=20) of flows ordered by `created_at` descending

#### Scenario: List with custom pagination
- **WHEN** a user sends a GET request to `/api/v1/flows?page=2&page_size=10`
- **THEN** the system returns the second page with 10 items and includes total count in response metadata

#### Scenario: Empty list
- **WHEN** there are no flows in the system
- **THEN** the system returns an empty array with total count 0