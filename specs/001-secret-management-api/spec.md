# Feature Specification: Custom Secret Management REST API with Pluggable Encryption

**Feature Branch**: `001-secret-management-api`  
**Created**: 5 January 2026  
**Status**: Draft  
**Input**: User description: "Implement custom secret management REST API with pluggable encryption providers (AES-GCM) for secure storage and retrieval of encrypted secrets at rest"

## Clarifications

### Session 2026-01-05

- Q: API endpoint path structure within gateway REST API? → A: /secrets (root-level path for simplicity, no versioning)
- Q: Secret request/response body structure? → A: Request: {"id": "user-provided-id", "value": "secret-data"}, Response includes same id plus timestamps
- Q: Concurrent update conflict resolution strategy? → A: Last-write-wins with database transaction isolation
- Q: Encryption key file format? → A: Raw binary 32-byte files
- Q: Provider chain decryption failure error response? → A: 500 Internal Server Error with generic error (no sensitive details exposed)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Store and Retrieve Encrypted Secrets (Priority: P1)

Platform operators need to securely store sensitive configuration data (API keys, passwords, connection strings) that the gateway services require, ensuring the data is encrypted at rest and can only be retrieved by authorized services.

**Why this priority**: This is the core value proposition - without the ability to securely store and retrieve secrets, the feature provides no value. All other features depend on this foundation.

**Independent Test**: Can be fully tested by creating a secret via POST, verifying it's encrypted in the database, retrieving it via GET with proper authentication, and confirming the plaintext value matches. Delivers immediate value as a secure secret storage solution.

**Acceptance Scenarios**:

1. **Given** a platform operator with valid authentication credentials, **When** they submit a secret with id and plaintext value via POST /secrets, **Then** the secret is encrypted using the primary encryption provider (AES-GCM), stored in the database with the provided id, ciphertext and metadata, and the same secret ID is returned with timestamps
2. **Given** an encrypted secret exists in the database, **When** an authorized user requests it via GET /secrets/{id}, **Then** the secret is decrypted using the appropriate provider from the chain, and the plaintext value is returned in the response
3. **Given** an encrypted secret in the database, **When** examining the database directly, **Then** only ciphertext, provider identifier, and key version are visible (no plaintext data)

---

### User Story 2 - Update Secrets with Automatic Key Migration (Priority: P2)

Platform operators need to update existing secrets with new values while ensuring they are re-encrypted with the current active encryption key, enabling seamless key rotation without manual migration steps.

**Why this priority**: Enables key rotation strategy and ensures secrets remain protected with the latest security standards. Critical for production environments but can be deferred after basic CRUD operations work.

**Independent Test**: Can be tested by creating a secret with key-v1, rotating to key-v2 as primary, updating the secret, and verifying it's now encrypted with key-v2 while old secrets encrypted with key-v1 remain readable. Delivers value as an automatic migration mechanism.

**Acceptance Scenarios**:

1. **Given** an existing secret encrypted with key-v1, **When** an operator updates the secret value via PUT /secrets/{id} after key-v2 becomes the primary key, **Then** the new value is encrypted with key-v2, and the database record is updated with the new ciphertext and key version
2. **Given** multiple secrets encrypted with different key versions (key-v1, key-v2), **When** any secret is retrieved via GET, **Then** the provider chain successfully decrypts using the appropriate key version regardless of which key is currently primary
3. **Given** an operator wants to trigger re-encryption without changing the value, **When** they submit a PUT request with the same plaintext value, **Then** the secret is re-encrypted with the current primary key and updated metadata reflects the new key version

---

### User Story 3 - Delete Secrets Securely (Priority: P2)

Platform operators need to permanently remove secrets from the system when they are no longer needed, ensuring complete deletion from the database to comply with security policies.

**Why this priority**: Required for complete secret lifecycle management and compliance with data retention policies. Less critical than create/read/update operations but necessary for production readiness.

**Independent Test**: Can be tested by creating a secret, deleting it via DELETE /secrets/{id}, and verifying it cannot be retrieved and does not exist in the database. Delivers value as a secure cleanup mechanism.

**Acceptance Scenarios**:

1. **Given** a secret exists in the database, **When** an authorized operator issues DELETE /secrets/{id}, **Then** the secret is permanently removed from the database and subsequent GET requests return 404 Not Found
2. **Given** a non-existent secret ID, **When** an operator attempts to delete it, **Then** the system returns 404 Not Found without exposing whether the secret ever existed
3. **Given** multiple secrets in the system, **When** one secret is deleted, **Then** other secrets remain accessible and unaffected by the deletion operation

---

### User Story 4 - Configure Multiple Encryption Providers (Priority: P3)

System administrators need to configure a chain of encryption providers at startup (AES-GCM, future external providers) to support key rotation, provider migration, and disaster recovery scenarios.

**Why this priority**: Enables the pluggable architecture and future extensibility. Important for long-term flexibility but the system can operate with a single provider initially. Can be implemented after core CRUD operations are stable.

**Independent Test**: Can be tested by configuring multiple providers in the startup config, encrypting secrets with the first provider, then changing provider order and verifying decryption still works using the provider chain fallback mechanism. Delivers value as an extensibility foundation.

**Acceptance Scenarios**:

1. **Given** a configuration file with multiple encryption providers listed in order, **When** the system starts, **Then** all providers are initialized successfully and the first provider is designated as primary for encryption operations
2. **Given** secrets encrypted by different providers in the chain (e.g., some with AES-GCM v1, some with external-kms), **When** retrieving any secret, **Then** the system attempts decryption with each provider in order until successful or all providers are exhausted
3. **Given** the provider chain configuration is modified at startup, **When** a new provider is added to the beginning of the chain, **Then** new secrets are encrypted with the new primary provider while old secrets remain decryptable by providers later in the chain

---

### Edge Cases

- What happens when an operator attempts to create a secret with an ID that already exists in the database?
- What happens when a secret is requested but all decryption providers in the chain fail to decrypt it (corrupted data, missing key version)? Returns 500 with generic error message, detailed diagnostics in server logs only.
- How does the system handle concurrent updates to the same secret (two operators updating simultaneously)?
- What occurs when the database is unavailable during a GET request (network partition, database failure)?
- How does the system respond when an operator attempts to create a secret without providing authentication credentials?
- What happens when the primary encryption provider fails during a POST operation (key file missing, KMS unreachable)?
- How does the system handle requests for secrets using malformed or invalid secret IDs?
- What occurs when attempting to delete a secret that is currently being updated by another request?
- How does the system behave when encryption keys are rotated but old keys are not available in the provider configuration (incomplete migration)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST expose secret management API at root path `/secrets` for all CRUD operations
- **FR-002**: System MUST encrypt all secret values using the first provider in the configured provider chain before persisting to the database
- **FR-003**: System MUST store encrypted secrets with metadata including provider identifier, key version, ciphertext format, and timestamps (created_at, updated_at)
- **FR-004**: System MUST never persist plaintext secret values to the database under any circumstances
- **FR-005**: System MUST support AES-GCM as the initial encryption provider with 256-bit keys loaded from raw binary 32-byte files at secure file locations during startup
- **FR-006**: System MUST attempt decryption using each provider in the configured chain sequentially until successful or all providers are exhausted
- **FR-007**: System MUST accept user-provided secret identifiers in POST /secrets requests and validate uniqueness, returning 409 Conflict if the ID already exists
- **FR-008**: System MUST require both "id" and "value" fields in POST requests, returning 400 Bad Request if either is missing
- **FR-009**: System MUST validate that the secret ID exists before allowing update or delete operations, returning 404 if not found
- **FR-010**: System MUST require authentication and authorization for all secret management operations (POST, GET, PUT, DELETE)
- **FR-011**: System MUST re-encrypt secrets with the current primary provider when updated via PUT /secrets/{id}
- **FR-012**: System MUST support hard deletion of secrets via DELETE /secrets/{id}, permanently removing all data from the database
- **FR-013**: System MUST load encryption provider configuration from a YAML file at application startup, including provider type, priority order, and key locations
- **FR-014**: System MUST support multiple key versions per provider, allowing decryption of secrets encrypted with older keys while encrypting new secrets with the active key
- **FR-015**: System MUST include provider name and key version in the ciphertext metadata format (e.g., `enc:aesgcm:v1:key-v2:<base64-data>`)
- **FR-016**: System MUST return appropriate HTTP status codes: 200 for successful GET, 201 for successful POST, 200 for successful PUT, 204 for successful DELETE, 404 for not found, 409 for conflict (duplicate ID), 401 for unauthorized, 500 for server errors
- **FR-017**: System MUST validate that "value" field is present in PUT requests, returning 400 Bad Request if missing
- **FR-018**: System MUST support concurrent read operations (multiple GET requests) without blocking or degrading performance
- **FR-019**: System MUST use database transaction isolation to handle concurrent updates with last-write-wins semantics (no explicit conflict detection or version checking)
- **FR-020**: System MUST return 500 Internal Server Error with generic "decryption failed" message when all providers fail to decrypt a secret, without exposing key versions, provider names, or specific failure reasons in the response
- **FR-021**: System MUST provide structured error messages including error type, message, and correlation ID for troubleshooting when operations fail
- **FR-020**: System MUST provide structured error messages including error type, message, and correlation ID for troubleshooting when operations fail
- **FR-020**: System MUST provide structured error messages including error type, message, and correlation ID for troubleshooting when operations fail
- **FR-021**: System MUST provide structured error messages including error type, message, and correlation ID for troubleshooting when operations fail
- **FR-022**: System MUST integrate with the existing Gateway Controller authentication middleware (Basic Auth or JWT) without requiring separate authentication mechanisms

### Key Entities *(include if feature involves data)*

- **Secret**: Represents a sensitive value stored in the system
  - Unique identifier (ID) - user-provided primary key for retrieval, must be unique across all secrets
  - Secret value - the plaintext data to be encrypted (provided by user in request)
  - Provider identifier - tracks which encryption provider was used
  - Key version - identifies the specific key used for encryption
  - Ciphertext - encrypted secret value with integrity protection
  - Created timestamp - when the secret was first created
  - Updated timestamp - when the secret was last modified
  - Metadata format - structured format for storing provider and versioning information

- **EncryptionProvider**: Represents an encryption implementation in the provider chain
  - Provider type - identifies the encryption method (aesgcm, external-kms, etc.)
  - Priority order - position in the provider chain
  - Key versions - list of available keys for this provider
  - Active key - the current key used for new encryption operations

- **EncryptionKey**: Represents a cryptographic key used by a provider
  - Key name - identifier for the key (e.g., "key-v2")
  - Key material - the actual cryptographic key data (32 bytes for AES-256)
  - File path - location where the raw binary key file is loaded from
  - Version identifier - distinguishes between key rotations

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can store secrets via API and retrieve them with correct plaintext values in under 1 second for 95% of requests
- **SC-002**: System successfully encrypts and decrypts 1000 secrets without data loss or corruption
- **SC-003**: Zero plaintext secrets are discoverable in database queries or backup files after storage operations
- **SC-004**: System correctly decrypts secrets encrypted with any provider in the configured chain without manual intervention
- **SC-005**: Secret update operations automatically migrate to the latest encryption key version 100% of the time
- **SC-006**: System handles concurrent read operations from 50 simultaneous users without errors or data corruption
- **SC-007**: All API operations return within 2 seconds under normal load conditions (database responsive, keys available)
- **SC-008**: System startup completes successfully with valid provider configuration within 10 seconds
- **SC-009**: Decryption attempts with corrupted ciphertext fail gracefully with 500 status and generic error messages without exposing key material, provider details, or specific failure reasons
- **SC-010**: Deleted secrets cannot be recovered through any API endpoint or database query
- **SC-011**: System operates without exposing encryption keys or plaintext secrets in logs or error messages
- **SC-012**: Authentication failures for secret operations return 401 status without revealing whether the secret exists

## Scope *(optional)*

### In Scope

- REST API endpoints for CRUD operations on secrets (POST, GET, PUT, DELETE)
- AES-GCM encryption provider implementation with 256-bit keys
- Provider chain architecture supporting multiple encryption backends
- Secure key management for loading and storing encryption keys at startup
- Integration with existing Gateway Controller authentication and authorization
- Metadata format for tracking provider and key version information
- Database schema for storing encrypted secrets with all required metadata
- Error handling and validation for all API operations
- Support for multiple key versions per provider to enable rotation

### Out of Scope

- External encryption provider implementations (Vault, KMS, HSM) - architecture supports but not implemented in initial version
- Automatic key rotation mechanisms - keys must be rotated manually via configuration updates
- Secret versioning or history tracking - only current version is retained
- Secret expiration or time-to-live (TTL) functionality
- Secret sharing or access control lists (ACLs) - authorization handled at API level
- Secrets replication across multiple gateway instances - single database model
- Soft delete or recovery mechanisms - only hard delete supported
- Audit logging of secret access operations - handled by existing gateway logging
- Secret migration or bulk operations - one secret per API call
- User interface or portal integration - REST API only

## Assumptions *(optional)*

- Gateway Controller already has robust authentication and authorization middleware (Basic Auth and JWT) that can be reused
- SQLite database is already configured and operational for storing gateway configurations and can be extended for secrets
- Encryption keys are managed externally as raw binary 32-byte files and securely provided to the gateway at startup via file system paths specified in configuration
- Platform operators have the necessary permissions to access secret management endpoints through existing RBAC mechanisms
- Database transactions ensure ACID properties for secret storage operations without additional implementation
- Network communication between API clients and gateway is protected by TLS (HTTPS)
- Storage backend has sufficient performance characteristics for expected secret access patterns (< 10,000 secrets initially)
- System administrators are responsible for backing up encryption keys separately from the database
- Key rotation is an infrequent operation (weeks to months) managed through configuration file updates and service restarts
- Secrets are relatively small in size (< 10KB) and do not require streaming or chunked upload/download
- The existing Go project structure and coding standards are followed for consistency with gateway-controller codebase
- Unit tests and integration tests follow the existing patterns in the gateway repository

## Dependencies *(optional)*

- Existing Gateway Controller authentication middleware (Basic Auth and JWT validation)
- SQLite storage interface already defined in `pkg/storage/interface.go`
- Go standard library `crypto/aes` and `crypto/cipher` for AES-GCM implementation
- Go standard library `crypto/rand` for generating secure nonces
- Existing configuration loading mechanism using Koanf for reading provider configuration at startup
- Existing API generation tooling (oapi-codegen) for creating REST API server stubs from OpenAPI specification
- Gin web framework already in use for HTTP routing and middleware
- Zap structured logging framework for operation tracing and error logging
- Existing database transaction patterns in the storage layer for consistency

## Non-Functional Requirements *(optional)*

### Performance

- Secret encryption operations complete within 100ms for 99% of requests
- Secret decryption operations complete within 150ms for 99% of requests (accounting for provider chain traversal)
- System supports minimum 100 concurrent API requests without degradation
- Database queries for secret retrieval use indexed lookups (O(1) or O(log n) complexity)

### Security

- AES-GCM provides authenticated encryption with integrity protection (detects tampering)
- Random nonces generated for each encryption operation to prevent replay attacks
- Encryption keys loaded only at startup and kept in memory, never logged or exposed via API
- Constant-time comparison for authentication tokens to prevent timing attacks
- Encrypted secrets include metadata format version to support future cryptographic upgrades
- Database backups contain only ciphertext, keys must be backed up separately via secure channels

### Reliability

- Database transactions ensure all-or-nothing semantics for secret storage operations
- Provider chain fallback ensures secrets remain accessible during key rotation periods
- System gracefully handles provider initialization failures with clear error messages
- Corrupted ciphertext detected via AES-GCM authentication tag before returning errors
- Concurrent updates use last-write-wins strategy with database transaction isolation (no optimistic locking overhead)

### Maintainability

- Encryption provider interface abstracts implementation details for easy extension
- Provider implementations are isolated in separate packages for independent testing
- Configuration schema is versioned to support backward compatibility with older configs
- Code follows existing gateway-controller patterns for consistency (handler → service → storage)

### Observability

- All secret operations logged with correlation IDs for request tracing
- Metrics tracked for encryption/decryption performance and error rates
- Provider initialization status visible in startup logs
- Detailed error messages include operation context without exposing sensitive data
- Decryption failures logged with full diagnostic details (provider names, key versions, failure reasons) in server logs, but only generic errors returned to API clients

## Technical Constraints *(optional)*

- Must use Go 1.25.1+ to match existing gateway-controller language version
- Must integrate with SQLite as the primary storage backend (existing gateway constraint)
- Must follow existing API design patterns using oapi-codegen for OpenAPI-driven development
- Must use Gin framework for HTTP routing to maintain consistency with gateway-controller
- Must not introduce new external dependencies for core encryption (use Go standard library for AES-GCM)
- Must support the existing Gateway Controller configuration format (YAML via Koanf)
- Must integrate with existing middleware stack without breaking current authentication flows
- Must follow the established project structure (cmd/, pkg/, api/, handlers/, storage/)
- Provider configuration loaded once at startup (no runtime configuration changes without restart)
- Secret data size limited by SQLite BLOB maximum (approximately 1GB, but practical limit 10KB recommended)

## Open Questions *(optional)*

*This section intentionally left empty - all critical design decisions have been addressed with informed defaults based on the architecture document and existing gateway patterns.*

## Future Enhancements *(optional)*

- External encryption provider support (HashiCorp Vault, AWS KMS, Azure Key Vault, HSM)
- Automatic key rotation with scheduled re-encryption of existing secrets
- Secret versioning to track changes and support rollback
- Time-to-live (TTL) support for automatic secret expiration
- Audit logging of all secret access operations with detailed timestamps and user tracking
- Secret tags and metadata for categorization and search
- Bulk operations for migrating multiple secrets between providers
- Soft delete with recovery window before permanent deletion
- Integration with Kubernetes external secrets operator
- Secret replication across multiple gateway instances
- Access control lists (ACLs) for fine-grained secret permissions
- Secret usage analytics and unused secret detection
- Integration with certificate management for TLS secrets
- Webhooks for secret lifecycle events (created, updated, deleted, rotated)
- CLI tool for secret management operations outside the REST API

## References *(optional)*

- Kubernetes EncryptionConfiguration: https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/
- AES-GCM NIST Special Publication 800-38D: https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38d.pdf
- Go crypto/cipher GCM documentation: https://pkg.go.dev/crypto/cipher#NewGCM
- Gateway Controller Architecture: `/Users/nimsara/wso2/api-platform/gateway/spec/architecture/architecture.md`
- Storage Interface: `/Users/nimsara/wso2/api-platform/gateway/gateway-controller/pkg/storage/interface.go`
- API Handler Patterns: `/Users/nimsara/wso2/api-platform/gateway/gateway-controller/pkg/api/handlers/handlers.go`
- Authentication Middleware: `/Users/nimsara/wso2/api-platform/gateway/docs/ai-gateway/ai-gateway-rest-api/authentication.md`
