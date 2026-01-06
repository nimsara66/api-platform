# Tasks: Custom Secret Management REST API with Pluggable Encryption

**Feature Branch**: `001-secret-management-api`  
**Input**: Design documents from `/specs/001-secret-management-api/`  
**Prerequisites**: ✅ plan.md, ✅ spec.md, ✅ research.md, ✅ data-model.md, ✅ contracts/secrets-api.yaml

**Tests**: Test tasks are NOT included as they were not explicitly requested in the feature specification. The spec mentions testing patterns but does not require TDD approach.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [ID] [P?] [Story] Description`

- **Checkbox**: `- [ ]` (markdown task list format)
- **[ID]**: Sequential task number (T001, T002, ...)
- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story label (US1, US2, US3, US4) - only for story-specific tasks
- **File paths**: Absolute paths from gateway-controller/ root

## Path Conventions

All paths relative to: `/Users/nimsara/wso2/api-platform/gateway/gateway-controller/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, OpenAPI code generation, and basic structure

- [X] T001 Create directory structure for encryption packages in pkg/encryption/
- [X] T002 Create directory structure for secret storage in pkg/secretstore/
- [X] T003 Create directory structure for secret business logic in pkg/secrets/
- [X] T004 [P] Generate OpenAPI server code using oapi-codegen from contracts/secrets-api.yaml to pkg/api/openapi/secrets/
- [X] T005 [P] Create test fixtures directory for encryption keys in tests/fixtures/test-keys/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 Define EncryptionProvider interface in pkg/encryption/provider.go
- [X] T007 Define EncryptedPayload struct and serialization functions in pkg/encryption/provider.go
- [X] T008 [P] Define encryption-specific error types in pkg/encryption/errors.go
- [X] T009 Implement ProviderManager with provider chain logic in pkg/encryption/manager.go
- [X] T010 Define SecretStorage interface in pkg/secretstore/interface.go
- [X] T011 Create Secret domain model struct in pkg/secretstore/interface.go
- [X] T012 Extend gateway configuration struct to include EncryptionConfig in pkg/config/config.go
- [X] T013 Create SQLite secrets table schema in pkg/secretstore/sqlite.go (CREATE TABLE IF NOT EXISTS)
- [ ] T014 [P] Generate test encryption keys (key-v1.bin, key-v2.bin) in tests/fixtures/test-keys/

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Store and Retrieve Encrypted Secrets (Priority: P1) 🎯 MVP

**Goal**: Operators can store secrets encrypted at rest and retrieve them decrypted via REST API

**Independent Test**: Create a secret via POST /secrets, verify it's encrypted in the database (ciphertext only), retrieve it via GET /secrets/{id} with authentication, confirm plaintext value matches original

### Implementation for User Story 1

- [X] T015 [P] [US1] Implement KeyManager with key loading from raw binary files in pkg/encryption/aesgcm/keymgmt.go
- [X] T016 [P] [US1] Implement AESGCMProvider with Encrypt method (random nonce generation) in pkg/encryption/aesgcm/provider.go
- [X] T017 [P] [US1] Implement AESGCMProvider with Decrypt method (nonce extraction, GCM decrypt) in pkg/encryption/aesgcm/provider.go
- [X] T018 [P] [US1] Implement AESGCMProvider HealthCheck method in pkg/encryption/aesgcm/provider.go
- [X] T019 [US1] Implement SQLiteSecretStorage SaveSecret method in pkg/secretstore/sqlite.go
- [X] T020 [US1] Implement SQLiteSecretStorage GetSecret method in pkg/secretstore/sqlite.go
- [X] T021 [US1] Implement SecretService Create method with encryption logic in pkg/secrets/service.go
- [X] T022 [US1] Implement SecretService Get method with decryption logic in pkg/secrets/service.go
- [X] T023 [US1] Implement POST /secrets HTTP handler in pkg/api/handlers/secrets.go
- [X] T024 [US1] Implement GET /secrets/{id} HTTP handler in pkg/api/handlers/secrets.go
- [X] T025 [US1] Wire up encryption provider initialization in cmd/controller/main.go
- [X] T026 [US1] Register /secrets routes with Gin router and authentication middleware in cmd/controller/main.go
- [X] T027 [US1] Add request validation for POST /secrets (required id and value fields) in pkg/api/handlers/secrets.go
- [X] T028 [US1] Add error handling with correlation IDs for secret operations in pkg/api/handlers/secrets.go
- [X] T029 [US1] Add structured logging for encryption/decryption operations in pkg/secrets/service.go

**Checkpoint**: At this point, User Story 1 should be fully functional - secrets can be created and retrieved with encryption at rest

---

## Phase 4: User Story 2 - Update Secrets with Automatic Key Migration (Priority: P2)

**Goal**: Operators can update existing secrets, automatically re-encrypting them with the current primary key

**Independent Test**: Create secret with key-v1 as primary, update config to make key-v2 primary, restart gateway, update the secret, verify it's now encrypted with key-v2 while old secrets with key-v1 remain readable

### Implementation for User Story 2

- [ ] T030 [US2] Implement SQLiteSecretStorage UpdateSecret method with timestamp update in pkg/secretstore/sqlite.go
- [ ] T031 [US2] Implement SecretService Update method with decrypt → re-encrypt logic in pkg/secrets/service.go
- [ ] T032 [US2] Implement PUT /secrets/{id} HTTP handler in pkg/api/handlers/secrets.go
- [ ] T033 [US2] Add request validation for PUT /secrets/{id} (required value field) in pkg/api/handlers/secrets.go
- [ ] T034 [US2] Add provider chain fallback logic in ProviderManager Decrypt method to try all providers in pkg/encryption/manager.go
- [ ] T035 [US2] Add support for multiple keys in KeyManager with version tracking in pkg/encryption/aesgcm/keymgmt.go
- [ ] T036 [US2] Implement primary key selection logic (first key in array) in pkg/encryption/aesgcm/provider.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently - secrets can be updated with automatic key migration

---

## Phase 5: User Story 3 - Delete Secrets Securely (Priority: P2)

**Goal**: Operators can permanently remove secrets from the system with hard delete

**Independent Test**: Create a secret, delete it via DELETE /secrets/{id}, verify GET returns 404 and secret doesn't exist in database

### Implementation for User Story 3

- [ ] T037 [US3] Implement SQLiteSecretStorage DeleteSecret method with hard delete in pkg/secretstore/sqlite.go
- [ ] T038 [US3] Implement SecretService Delete method in pkg/secrets/service.go
- [ ] T039 [US3] Implement DELETE /secrets/{id} HTTP handler returning 204 on success in pkg/api/handlers/secrets.go
- [ ] T040 [US3] Add 404 error handling for non-existent secret deletion in pkg/api/handlers/secrets.go

**Checkpoint**: All CRUD operations (Create, Read, Update, Delete) are now fully functional

---

## Phase 6: User Story 4 - Configure Multiple Encryption Providers (Priority: P3)

**Goal**: System can be configured with multiple encryption providers in a chain for key rotation and future extensibility

**Independent Test**: Configure 2 providers with different key versions, encrypt secrets with first provider, swap provider order in config, restart gateway, verify old secrets still decrypt via chain fallback

### Implementation for User Story 4

- [ ] T041 [US4] Implement provider array initialization from config in ProviderManager constructor in pkg/encryption/manager.go
- [ ] T042 [US4] Add provider name matching logic in ProviderManager Decrypt to select correct provider in pkg/encryption/manager.go
- [ ] T043 [US4] Implement configuration validation for provider array (at least one provider) in pkg/config/config.go
- [ ] T044 [US4] Add health check aggregation across all providers in ProviderManager in pkg/encryption/manager.go
- [ ] T045 [US4] Add startup logging for provider chain initialization (provider names, key counts, primary provider) in cmd/controller/main.go
- [ ] T046 [US4] Add configuration examples for multiple providers in config.yaml comments

**Checkpoint**: Provider chain architecture is complete, enabling future external provider plugins (Vault, KMS, etc.)

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final improvements, security hardening, observability, and documentation

- [ ] T047 [P] Add metrics tracking for encryption/decryption operations (duration, success/failure counts) in pkg/secrets/service.go
- [ ] T048 [P] Add detailed error logging for decryption failures with provider/key context in pkg/encryption/manager.go
- [ ] T049 [P] Implement constant-time comparison for authentication tokens (if not already in middleware) in authentication middleware
- [ ] T050 [P] Add database index on secrets.updated_at for audit queries in pkg/secretstore/sqlite.go
- [ ] T051 Add validation for secret ID format (alphanumeric + hyphens/underscores, max 255 chars) in pkg/secrets/service.go
- [ ] T052 Add validation for secret value size limit (max 10KB) in pkg/secrets/service.go
- [ ] T053 Add correlation ID generation and propagation through request context in pkg/api/handlers/secrets.go
- [ ] T054 [P] Ensure encryption keys never appear in logs or error messages (audit logging code) in pkg/encryption/ and pkg/secrets/
- [ ] T055 [P] Add key file permission check at startup (warn if keys are world-readable) in pkg/encryption/aesgcm/keymgmt.go
- [ ] T056 [P] Document configuration options in gateway-controller README.md
- [ ] T057 [P] Add example config snippets to quickstart.md examples
- [ ] T058 Implement 500 error response with generic message for all decryption failures in pkg/api/handlers/secrets.go
- [ ] T059 Add HTTP status code constants for all error responses (400, 401, 404, 409, 500) in pkg/api/handlers/secrets.go
- [ ] T060 Verify transaction isolation for concurrent updates in SQLiteSecretStorage in pkg/secretstore/sqlite.go

---

## Dependencies & Execution Order

### Critical Path (Must Complete in Order)

1. **Phase 1** (Setup) → **Phase 2** (Foundation) → **Phase 3** (US1) ✅ Delivers MVP
2. Phase 3 (US1) → Phase 4 (US2) ✅ Adds update capability
3. Phase 3 (US1) → Phase 5 (US3) ✅ Adds delete capability
4. Phase 2 (Foundation) → Phase 6 (US4) ✅ Adds provider chain

### Parallel Execution Opportunities

**Within Phase 2 (Foundational)**:
- T006-T008 can run in parallel (different files)
- T011-T014 can run in parallel (different files)

**Within Phase 3 (US1)**:
- T015-T018 (AES-GCM provider implementation) can run in parallel
- T023-T024 (HTTP handlers) can run after T021-T022 (service layer)

**Within Phase 7 (Polish)**:
- T047-T048 (observability) can run in parallel
- T049-T055 (security hardening) can run in parallel
- T056-T057 (documentation) can run in parallel

### User Story Dependencies

```
US1 (Store & Retrieve)     [P1 - MVP]
  ├─→ US2 (Update)         [P2 - Depends on US1 CRUD foundation]
  └─→ US3 (Delete)         [P2 - Depends on US1 CRUD foundation]

Foundation
  └─→ US4 (Multi-Provider) [P3 - Independent of CRUD, depends on foundation]
```

**MVP Delivery**: Complete Phase 1 → Phase 2 → Phase 3 (US1) = 29 tasks

**Full Feature Delivery**: All phases = 60 tasks

---

## Implementation Strategy

### Incremental Delivery Approach

**Sprint 1 (MVP)**: Phases 1-3
- **Deliverable**: Basic secret storage and retrieval with AES-GCM encryption
- **Tasks**: T001-T029 (29 tasks)
- **Value**: Operators can store and retrieve encrypted secrets
- **Test**: POST secret → verify encrypted in DB → GET secret → verify plaintext matches

**Sprint 2 (Key Rotation)**: Phase 4
- **Deliverable**: Secret updates with automatic key migration
- **Tasks**: T030-T036 (7 tasks)
- **Value**: Operators can rotate encryption keys without data loss
- **Test**: Create with key-v1 → rotate to key-v2 → update → verify re-encrypted

**Sprint 3 (Complete CRUD)**: Phase 5
- **Deliverable**: Secret deletion capability
- **Tasks**: T037-T040 (4 tasks)
- **Value**: Complete secret lifecycle management
- **Test**: Create → delete → verify 404

**Sprint 4 (Extensibility)**: Phase 6
- **Deliverable**: Provider chain architecture
- **Tasks**: T041-T046 (6 tasks)
- **Value**: Foundation for future external providers (Vault, KMS)
- **Test**: Configure multi-provider chain → verify fallback decryption

**Sprint 5 (Production Ready)**: Phase 7
- **Deliverable**: Security hardening, observability, documentation
- **Tasks**: T047-T060 (14 tasks)
- **Value**: Production-ready with metrics, logging, security validation
- **Test**: Load testing, security audit, operator documentation review

---

## Validation Checklist

Before marking feature complete, verify:

- [ ] ✅ US1: Can create secret via POST and retrieve via GET with encryption at rest
- [ ] ✅ US2: Can update secret with automatic re-encryption to current primary key
- [ ] ✅ US3: Can delete secret with hard delete (404 after deletion)
- [ ] ✅ US4: Can configure multiple providers with chain fallback decryption
- [ ] ✅ All HTTP status codes correct (201, 200, 204, 400, 401, 404, 409, 500)
- [ ] ✅ Authentication required for all endpoints (Basic Auth or JWT)
- [ ] ✅ No plaintext secrets in database (only ciphertext with metadata)
- [ ] ✅ Decryption failures return generic 500 error (no key version leakage)
- [ ] ✅ Correlation IDs in all error responses
- [ ] ✅ Structured logging with zap for all operations
- [ ] ✅ Configuration validation at startup (key files exist, correct size)
- [ ] ✅ Key rotation workflow documented and tested
- [ ] ✅ Provider chain fallback works (old secrets readable after key rotation)
- [ ] ✅ Concurrent reads supported without blocking
- [ ] ✅ Last-write-wins for concurrent updates (transaction isolation)

---

## File Checklist

**New Files to Create** (organized by package):

### pkg/encryption/
- [ ] `provider.go` - EncryptionProvider interface, EncryptedPayload struct
- [ ] `manager.go` - ProviderManager with chain logic
- [ ] `errors.go` - Encryption-specific error types

### pkg/encryption/aesgcm/
- [ ] `provider.go` - AESGCMProvider implementation
- [ ] `provider_test.go` - Unit tests for AES-GCM encryption/decryption
- [ ] `keymgmt.go` - KeyManager for loading keys from files

### pkg/secretstore/
- [ ] `interface.go` - SecretStorage interface
- [ ] `sqlite.go` - SQLiteSecretStorage implementation
- [ ] `sqlite_test.go` - Unit tests for storage operations

### pkg/secrets/
- [ ] `service.go` - SecretService with CRUD + encryption logic
- [ ] `service_test.go` - Unit tests for business logic
- [ ] `models.go` - Secret domain model

### pkg/api/handlers/
- [ ] `secrets.go` - HTTP handlers for /secrets endpoints

### pkg/api/openapi/secrets/ (generated)
- [ ] `types.gen.go` - Generated types from OpenAPI spec
- [ ] `server.gen.go` - Generated server interface from OpenAPI spec

### tests/fixtures/
- [ ] `test-keys/key-v1.bin` - Test encryption key (32 bytes)
- [ ] `test-keys/key-v2.bin` - Test encryption key for rotation tests (32 bytes)

### tests/integration/
- [ ] `secrets_test.go` - End-to-end API tests (optional)

**Files to Modify**:
- `pkg/config/config.go` - Add EncryptionConfig struct
- `cmd/controller/main.go` - Initialize providers, register routes
- `config.yaml` - Add encryption provider configuration example

---

## Task Execution Tips

1. **Start with Setup (Phase 1)**: Create directory structure first
2. **Complete Foundation (Phase 2)**: Define all interfaces before implementation
3. **Implement US1 (Phase 3)**: This is the MVP - focus here first
4. **Verify Independently**: Each user story should work standalone
5. **Use [P] Tasks**: Run parallel tasks to speed up development
6. **Test After Each Phase**: Don't accumulate technical debt
7. **Follow File Paths**: All paths are specified in task descriptions
8. **Check Dependencies**: Review dependency graph before starting

---

**Total Tasks**: 60  
**MVP Tasks**: 29 (Phase 1-3)  
**Estimated Complexity**: Medium (extends existing codebase with well-defined patterns)

**Generated**: 2026-01-05  
**Feature Branch**: `001-secret-management-api`  
**Status**: Ready for implementation ✅
