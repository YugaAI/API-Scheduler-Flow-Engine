## ADDED Requirements

### Requirement: JWT Authentication
The system SHALL require a valid JWT token in the `Authorization: Bearer <token>` header for all API endpoints.

#### Scenario: Valid token
- **WHEN** a request includes a valid JWT token
- **THEN** the system extracts user ID and role from claims and proceeds with the request

#### Scenario: Missing token
- **WHEN** a request does not include an Authorization header
- **THEN** the system returns a 401 error with message "authentication required"

#### Scenario: Invalid or expired token
- **WHEN** a request includes an invalid or expired JWT token
- **THEN** the system returns a 401 error with message "invalid or expired token"

### Requirement: Role-Based Access Control
The system SHALL enforce role-based access: ADMIN has full access, USER has read-only access to flows and can trigger/view executions.

#### Scenario: ADMIN creates a flow
- **WHEN** an authenticated ADMIN sends a POST request to `/api/v1/flows`
- **THEN** the system allows the operation

#### Scenario: USER attempts to create a flow
- **WHEN** an authenticated USER sends a POST request to `/api/v1/flows`
- **THEN** the system returns a 403 error with message "insufficient permissions"

#### Scenario: USER triggers an execution
- **WHEN** an authenticated USER sends a POST request to `/api/v1/flows/{id}/execute`
- **THEN** the system allows the operation

#### Scenario: USER lists flows
- **WHEN** an authenticated USER sends a GET request to `/api/v1/flows`
- **THEN** the system allows the operation and returns the flow list

#### Scenario: USER attempts to delete a flow
- **WHEN** an authenticated USER sends a DELETE request to `/api/v1/flows/{id}`
- **THEN** the system returns a 403 error with message "insufficient permissions"

### Requirement: JWT Claims Structure
The JWT token SHALL contain the following claims: `sub` (user ID), `role` (ADMIN or USER), `exp` (expiration), `iat` (issued at).

#### Scenario: Token claims extraction
- **WHEN** the auth middleware processes a valid token
- **THEN** it extracts `sub` and `role` from claims and sets them in the Gin context for downstream handlers
