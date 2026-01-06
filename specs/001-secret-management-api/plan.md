# Implementation Plan: Custom Secret Management REST API with Pluggable Encryption

**Branch**: `001-secret-management-api` | **Date**: 2026-01-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-secret-management-api/spec.md`

## Summary

Implement a Kubernetes-inspired secret management REST API for the gateway-controller that encrypts secrets at rest using pluggable encryption providers. The initial implementation supports AES-GCM encryption with 256-bit keys loaded from raw binary files, while the architecture allows future extension to external providers (Vault, KMS, HSM). Secrets are stored encrypted in SQLite with provider and key version metadata, enabling seamless key rotation through a provider chain mechanism where the first provider encrypts and all providers can decrypt.

## Technical Context

**Language/Version**: Go 1.25.1  
**Primary Dependencies**: 
- Gin web framework (existing HTTP routing)
- oapi-codegen (OpenAPI code generation)
- SQLite via mattn/go-sqlite3 (existing storage)
- Koanf (existing configuration loading)
- Zap (existing structured logging)
- Go standard library crypto/aes, crypto/cipher for AES-GCM
- Go standard library crypto/rand for nonce generation

**Storage**: SQLite (existing gateway-controller database, new `secrets` table)  
**Testing**: Go testing package, testify/assert (existing test infrastructure)  
**Target Platform**: Linux/macOS servers (gateway-controller deployment environments)  
**Project Type**: REST API service (gateway-controller extension)  
**Performance Goals**: 
- Encryption operations: <100ms for 99% of requests
- Decryption operations: <150ms for 99% of requests (with provider chain)
- Support 100+ concurrent API requests

**Constraints**: 
- No new external dependencies for encryption (use Go stdlib only)
- Must integrate with existing authentication middleware (Basic Auth/JWT)
- Must follow existing gateway-controller patterns (handler → service → storage)
- Keys loaded once at startup, no runtime key updates without restart
- Secret size limited to 10KB (practical limit)

**Scale/Scope**: 
- Target: <10,000 secrets initially
- Single gateway instance (no distributed secret replication)
- 4 REST endpoints (POST, GET, PUT, DELETE)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Status**: ✅ PASS

The gateway project constitution is a template and not yet ratified. However, evaluating against standard Go project principles:

| Principle | Status | Notes |
|-----------|--------|-------|
| Go idioms & simplicity | ✅ PASS | Uses standard library crypto, follows existing gateway patterns |
| Testing requirements | ✅ PASS | Unit tests for encryption provider, integration tests for API endpoints planned |
| Documentation | ✅ PASS | OpenAPI spec for API contracts, inline code documentation, quickstart guide planned |
| Security best practices | ✅ PASS | Authenticated encryption (AES-GCM), no plaintext persistence, generic error messages |
| Integration patterns | ✅ PASS | Extends existing storage interface, reuses authentication middleware |

**No violations requiring justification.**

## Project Structure

### Documentation (this feature)

```text
specs/001-secret-management-api/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output - encryption provider patterns, key management
├── data-model.md        # Phase 1 output - Secret entity, provider configuration schema
├── quickstart.md        # Phase 1 output - Getting started guide for operators
├── contracts/           # Phase 1 output - OpenAPI specification
│   └── secrets-api.yaml
└── spec.md              # Input specification (already exists)
```

### Source Code (gateway-controller/)

```text
gateway-controller/
├── pkg/
│   ├── encryption/                    # NEW: Encryption provider abstraction
│   │   ├── provider.go               # Provider interface definition
│   │   ├── manager.go                # Provider chain manager
│   │   ├── aesgcm/                   # AES-GCM provider implementation
│   │   │   ├── provider.go
│   │   │   ├── provider_test.go
│   │   │   └── keymgmt.go           # Key loading from raw binary files
│   │   └── errors.go                 # Encryption-specific errors
│   │
│   ├── secretstore/                   # NEW: Secret storage abstraction
│   │   ├── interface.go              # SecretStorage interface
│   │   ├── sqlite.go                 # SQLite implementation
│   │   └── sqlite_test.go
│   │
│   ├── secrets/                       # NEW: Secret business logic
│   │   ├── service.go                # Secret CRUD operations with encryption
│   │   ├── service_test.go
│   │   └── models.go                 # Secret domain models
│   │
│   ├── api/
│   │   ├── handlers/
│   │   │   └── secrets.go            # NEW: HTTP handlers for /secrets endpoints
│   │   └── openapi/
│   │       └── secrets/              # NEW: Generated OpenAPI code
│   │           ├── types.gen.go
│   │           └── server.gen.go
│   │
│   ├── storage/
│   │   ├── interface.go              # Existing storage interface (no changes)
│   │   └── sqlite.go                 # Extended with secrets table schema
│   │
│   └── config/
│       └── config.go                 # Extended with encryption provider config
│
├── cmd/
│   └── controller/
│       └── main.go                    # Extended to initialize encryption providers
│
└── api/
    └── openapi/
        └── secrets.yaml               # NEW: OpenAPI spec source (hand-written)

tests/
├── integration/
│   └── secrets_test.go                # NEW: End-to-end API tests
└── fixtures/
    └── test-keys/                     # NEW: Test encryption keys
        ├── key-v1.bin
        └── key-v2.bin
```

**Structure Decision**: Extends existing gateway-controller codebase with new packages for encryption (`pkg/encryption`), secret storage (`pkg/secretstore`), and secret business logic (`pkg/secrets`). Follows the established layered architecture: API handlers → Service layer → Storage layer. The encryption provider abstraction is kept separate from secret management to support future extensibility. SQLite storage is extended with a new `secrets` table while maintaining the existing `Storage` interface unchanged.

## Complexity Tracking

**No constitution violations - this section intentionally left empty.**

---

## Phase 0: Research & Design Decisions

### Research Areas

1. **AES-GCM Implementation Patterns** - How to correctly implement AES-GCM in Go with nonce management
2. **Provider Chain Architecture** - Best practices for encryption provider abstraction and fallback mechanisms  
3. **Key Loading Strategies** - Secure key file loading, validation, and in-memory management
4. **SQLite Encryption Storage** - Optimal schema design for encrypted data with metadata
5. **OpenAPI Secret API Design** - REST API conventions for secret management endpoints

### Expected Outputs (research.md)

- Decision: AES-GCM nonce generation strategy (random vs counter-based)
- Decision: Ciphertext metadata format structure (prefix vs JSON)
- Decision: Provider interface design (methods, error handling)
- Decision: Database schema for secrets table (fields, indexes, constraints)
- Best practices: Key rotation workflows and migration strategies

---

## Phase 1: Design Artifacts

### 1. Data Model (data-model.md)

**Entities:**

- **Secret** (domain model)
  - ID (string, user-provided, unique)
  - Value (string, plaintext in memory only)
  - ProviderName (string)
  - KeyVersion (string)
  - Ciphertext ([]byte)
  - CreatedAt (time.Time)
  - UpdatedAt (time.Time)

- **EncryptionProvider** (interface)
  - Name() string
  - Encrypt(plaintext []byte) (EncryptedPayload, error)
  - Decrypt(payload EncryptedPayload) ([]byte, error)
  - HealthCheck() error

- **EncryptedPayload** (struct)
  - Provider (string)
  - KeyVersion (string)
  - Ciphertext ([]byte)
  - Format metadata

- **ProviderConfig** (configuration)
  - Type (string: "aesgcm", "external-kms")
  - Keys ([]KeyConfig)
  - Priority (int, implicit from array order)

- **KeyConfig** (configuration)
  - Name (string: "key-v1", "key-v2")
  - Path (string: file path to raw binary key)

**Relationships:**
- Secret → EncryptionProvider (many-to-one via provider name)
- EncryptionProvider → KeyConfig (one-to-many)
- ProviderManager orchestrates EncryptionProvider chain

**Validation Rules:**
- Secret.ID must be non-empty and unique (database constraint)
- Secret.Value required for POST/PUT operations
- ProviderName and KeyVersion populated automatically during encryption
- Key files must be exactly 32 bytes for AES-256

**State Transitions:**
- Secret created (POST) → encrypted with primary provider → persisted
- Secret retrieved (GET) → loaded from DB → decrypted via provider chain → returned
- Secret updated (PUT) → decrypted → re-encrypted with current primary → persisted
- Secret deleted (DELETE) → hard delete from database

### 2. API Contracts (contracts/secrets-api.yaml)

**Endpoints:**

- `POST /secrets` - Create encrypted secret
  - Request: `{"id": "string", "value": "string"}`
  - Response 201: `{"id": "string", "value": "string", "created_at": "ISO8601", "updated_at": "ISO8601"}`
  - Response 409: Conflict (duplicate ID)
  - Response 400: Bad request (missing fields)
  - Response 401: Unauthorized
  - Response 500: Internal server error

- `GET /secrets/{id}` - Retrieve and decrypt secret
  - Response 200: `{"id": "string", "value": "string", "created_at": "ISO8601", "updated_at": "ISO8601"}`
  - Response 404: Not found
  - Response 401: Unauthorized
  - Response 500: Decryption failed (generic message)

- `PUT /secrets/{id}` - Update and re-encrypt secret
  - Request: `{"value": "string"}`
  - Response 200: `{"id": "string", "value": "string", "created_at": "ISO8601", "updated_at": "ISO8601"}`
  - Response 404: Not found
  - Response 400: Bad request (missing value)
  - Response 401: Unauthorized
  - Response 500: Internal server error

- `DELETE /secrets/{id}` - Delete secret permanently
  - Response 204: No content (success)
  - Response 404: Not found
  - Response 401: Unauthorized
  - Response 500: Internal server error

**Authentication**: Reuses existing gateway-controller Basic Auth or JWT middleware

### 3. Configuration Schema Extension

```yaml
# config.yaml extension
encryption:
  providers:
    - type: aesgcm
      keys:
        - name: key-v2
          path: /etc/secrets/key-v2.bin
        - name: key-v1
          path: /etc/secrets/key-v1.bin
    # Future: external providers
    # - type: vault
    #   endpoint: https://vault.example.com
    #   token: ${VAULT_TOKEN}
```

### 4. Agent Context Update

Run `.specify/scripts/bash/update-agent-context.sh copilot` to add:
- Technology: AES-GCM encryption
- Technology: Pluggable encryption provider pattern
- New packages: `pkg/encryption`, `pkg/secretstore`, `pkg/secrets`
- New API endpoints: `/secrets` (POST, GET, PUT, DELETE)

---

## Gate Re-Check (Post-Design)

**Status**: ✅ PASS

- All designs align with existing gateway-controller architecture
- No new external dependencies introduced (only Go stdlib)
- Follows established patterns (interface-based abstractions, middleware reuse)
- Test coverage plan matches existing standards (unit + integration)
- Security requirements met (no plaintext persistence, generic error messages)

---

## Next Steps

**Phase 2**: Task Decomposition (`/speckit.tasks` command)
- Break down implementation into atomic, testable tasks
- Define task dependencies and execution order
- Assign priority to each task based on spec requirements

**Note**: This plan document serves as input to the `/speckit.tasks` command. Do not proceed to implementation until tasks.md is generated and approved.

---

**Generated by**: `/speckit.plan` command  
**Last Updated**: 2026-01-05
