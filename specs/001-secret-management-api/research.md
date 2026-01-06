# Research: Secret Management API Implementation

**Feature**: Custom Secret Management REST API with Pluggable Encryption  
**Date**: 2026-01-05  
**Phase**: 0 (Research & Design Decisions)

---

## 1. AES-GCM Implementation in Go

### Decision: Random Nonce Generation

**Chosen Approach**: Generate cryptographically random 12-byte nonces using `crypto/rand` for each encryption operation.

**Rationale**:
- AES-GCM standard recommends 96-bit (12-byte) nonces
- Random nonce generation is safe for the expected secret volume (<10,000 secrets)
- Collision probability with 96-bit nonces is negligible (2^-96 even after billions of operations)
- Simpler implementation than counter-based nonces (no state management required)
- Each nonce is stored alongside ciphertext, enabling stateless decryption

**Alternatives Considered**:
- Counter-based nonces: Rejected due to complexity of maintaining global counter state across restarts
- Sequential nonces: Rejected due to predictability concerns
- Timestamp-based nonces: Rejected due to clock synchronization issues in distributed systems

**Implementation Pattern**:
```go
nonce := make([]byte, 12)
if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
    return nil, err
}
ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
```

**References**:
- NIST SP 800-38D: Random nonce generation acceptable for GCM
- Go crypto/cipher documentation: https://pkg.go.dev/crypto/cipher#NewGCM

---

## 2. Ciphertext Metadata Format

### Decision: Structured Prefix Format

**Chosen Approach**: `enc:aesgcm:v1:key-v2:<base64-ciphertext>`

**Format Specification**:
```
enc                    # Identifies encrypted data
aesgcm                 # Provider type identifier
v1                     # Metadata format version
key-v2                 # Key identifier/version
<base64-ciphertext>    # Base64-encoded (nonce || ciphertext || tag)
```

**Rationale**:
- Human-readable in database inspection (aids debugging)
- Self-describing format enables provider chain decryption
- Format version supports future cryptographic upgrades
- Colon-delimited structure is simple to parse
- Base64 encoding avoids binary storage issues
- Matches Kubernetes EncryptionConfiguration semantics

**Alternatives Considered**:
- JSON metadata: Rejected due to storage overhead and parsing complexity
- Binary header: Rejected due to lack of human readability
- No metadata: Rejected - cannot support provider chain or key rotation

**Parsing Strategy**:
```go
parts := strings.SplitN(stored, ":", 5)
// parts[0] = "enc" (magic prefix)
// parts[1] = provider type
// parts[2] = format version
// parts[3] = key version
// parts[4] = base64 ciphertext
```

---

## 3. Encryption Provider Interface Design

### Decision: Minimal Interface with EncryptedPayload Struct

**Interface Definition**:
```go
type EncryptionProvider interface {
    Name() string
    Encrypt(plaintext []byte) (*EncryptedPayload, error)
    Decrypt(payload *EncryptedPayload) ([]byte, error)
    HealthCheck() error
}

type EncryptedPayload struct {
    Provider   string
    KeyVersion string
    Ciphertext []byte
}
```

**Rationale**:
- Minimal interface reduces implementation burden for future providers
- `EncryptedPayload` struct decouples internal representation from storage format
- `HealthCheck()` enables startup validation without encrypting data
- `Name()` supports provider chain identification and logging
- Error handling through standard Go errors (wrapped with context)

**Alternatives Considered**:
- Including metadata serialization in interface: Rejected to keep providers focused on cryptography
- Async encryption methods: Rejected - not needed for expected throughput
- Key rotation methods in interface: Rejected - handled at configuration level

**Provider Chain Logic**:
```go
// Encryption: Use first provider
func (m *Manager) Encrypt(data []byte) (*EncryptedPayload, error) {
    return m.providers[0].Encrypt(data)
}

// Decryption: Try providers in order
func (m *Manager) Decrypt(payload *EncryptedPayload) ([]byte, error) {
    for _, provider := range m.providers {
        if provider.Name() == payload.Provider {
            return provider.Decrypt(payload)
        }
    }
    return nil, ErrNoProviderFound
}
```

**References**:
- Kubernetes encryption provider pattern: https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/

---

## 4. SQLite Database Schema for Secrets

### Decision: Dedicated `secrets` Table with Metadata Columns

**Schema Definition**:
```sql
CREATE TABLE IF NOT EXISTS secrets (
    id TEXT PRIMARY KEY NOT NULL,
    provider TEXT NOT NULL,
    key_version TEXT NOT NULL,
    ciphertext BLOB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_secrets_updated_at ON secrets(updated_at);
```

**Rationale**:
- `id` as TEXT PRIMARY KEY enforces uniqueness and supports user-provided IDs
- Separate `provider` and `key_version` columns enable efficient key rotation queries
- `ciphertext` as BLOB stores raw encrypted bytes (including metadata prefix)
- Timestamps support audit requirements and cleanup operations
- Index on `updated_at` enables efficient "recently updated" queries
- No foreign keys to existing tables - secrets are independent entities

**Storage Approach**:
- Full metadata string stored in `ciphertext` column (self-contained)
- `provider` and `key_version` duplicated for query optimization
- No plaintext value column (enforces security requirement FR-003)

**Migration Strategy**:
```go
// In sqlite.go initialization
func (s *SQLiteStorage) initSecretsTable() error {
    _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS secrets (...)`)
    return err
}
```

**Alternatives Considered**:
- Separate metadata table: Rejected due to join overhead
- JSON column for metadata: Rejected - not all SQLite builds support JSON
- Reusing existing `configs` table: Rejected - different entity lifecycle

---

## 5. Key Management and Loading

### Decision: Startup Key Loading with In-Memory Cache

**Approach**:
```go
type KeyManager struct {
    keys map[string][]byte  // keyName -> 32-byte key material
}

func LoadKeysFromConfig(cfg ProviderConfig) (*KeyManager, error) {
    km := &KeyManager{keys: make(map[string][]byte)}
    for _, keyConfig := range cfg.Keys {
        data, err := os.ReadFile(keyConfig.Path)
        if err != nil {
            return nil, fmt.Errorf("failed to load key %s: %w", keyConfig.Name, err)
        }
        if len(data) != 32 {
            return nil, fmt.Errorf("key %s invalid size: got %d bytes, expected 32", keyConfig.Name, len(data))
        }
        km.keys[keyConfig.Name] = data
    }
    return km, nil
}
```

**Rationale**:
- Keys loaded once at startup minimizes file I/O overhead
- In-memory cache provides fast key lookups during encryption/decryption
- Validation ensures all keys are correct size (32 bytes for AES-256)
- Startup failure if any key is missing/invalid prevents runtime errors
- No key persistence in application - keys remain in filesystem

**Key File Security**:
- Keys stored as raw binary files (no encoding overhead)
- File permissions should be 0400 or 0600 (read-only, owner-only)
- Keys never logged or included in error messages
- Generated using: `openssl rand -out key.bin 32`

**Key Rotation Workflow**:
1. Generate new key file (e.g., `key-v3.bin`)
2. Add to configuration YAML at top of `keys` array (becomes primary)
3. Keep old keys in array for backward compatibility
4. Restart gateway-controller to load new configuration
5. New secrets encrypted with key-v3, old secrets remain readable
6. Optional: Trigger re-encryption by updating secrets via PUT

**Alternatives Considered**:
- Environment variable keys: Rejected due to key size and encoding complexity
- KMS integration for key loading: Deferred to future external provider implementation
- Hot-reload of keys: Rejected due to complexity and restart requirement

---

## 6. OpenAPI Specification Design

### Decision: Simple Request/Response Schema

**API Schema**:
```yaml
components:
  schemas:
    SecretCreateRequest:
      type: object
      required: [id, value]
      properties:
        id:
          type: string
          description: Unique secret identifier (user-provided)
        value:
          type: string
          description: Secret value to encrypt and store
    
    SecretUpdateRequest:
      type: object
      required: [value]
      properties:
        value:
          type: string
          description: New secret value to encrypt and store
    
    SecretResponse:
      type: object
      properties:
        id:
          type: string
        value:
          type: string
          description: Decrypted secret value
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    
    ErrorResponse:
      type: object
      properties:
        error:
          type: string
        message:
          type: string
        correlation_id:
          type: string
```

**Rationale**:
- Minimal request schema reduces client complexity
- Separate create/update schemas reflect different required fields
- Response includes timestamps for audit purposes
- Error schema supports structured error handling with correlation IDs
- No metadata field in v1 (can be added in future without breaking changes)

**Code Generation**:
- Use `oapi-codegen` (existing gateway-controller tool)
- Generate types and server interface
- Manual implementation of handler logic

**Alternatives Considered**:
- Exposing provider/key version in response: Rejected for security (information leakage)
- Batch operations: Deferred to future enhancements
- Query/filter endpoints: Deferred (not in scope)

---

## 7. Error Handling Strategy

### Decision: Security-First Generic Errors

**Error Response Patterns**:

| Scenario | HTTP Status | Response | Server Logs |
|----------|-------------|----------|-------------|
| Secret not found | 404 | `{"error": "not_found", "message": "Secret not found"}` | Secret ID logged |
| Duplicate ID | 409 | `{"error": "conflict", "message": "Secret already exists"}` | Secret ID logged |
| Decryption failed | 500 | `{"error": "internal_error", "message": "Failed to decrypt secret"}` | Provider, key version, error details logged |
| Missing auth | 401 | `{"error": "unauthorized", "message": "Authentication required"}` | Request details logged |
| Invalid request | 400 | `{"error": "bad_request", "message": "Missing required field: value"}` | Validation errors logged |

**Rationale**:
- Generic 500 errors for decryption failures prevent information leakage
- Detailed diagnostics in server logs enable troubleshooting without exposing to clients
- Correlation IDs link client errors to server logs
- Consistent error structure across all endpoints

**Implementation**:
```go
func (h *SecretsHandler) handleDecryptionError(c *gin.Context, err error) {
    correlationID := uuid.New().String()
    h.logger.Error("Secret decryption failed",
        zap.String("correlation_id", correlationID),
        zap.String("secret_id", secretID),
        zap.String("provider", payload.Provider),
        zap.String("key_version", payload.KeyVersion),
        zap.Error(err),
    )
    c.JSON(500, ErrorResponse{
        Error: "internal_error",
        Message: "Failed to decrypt secret",
        CorrelationID: correlationID,
    })
}
```

---

## 8. Testing Strategy

### Decision: Multi-Layer Test Coverage

**Test Layers**:

1. **Unit Tests** (pkg/encryption/aesgcm/)
   - Test encryption produces valid ciphertext
   - Test decryption recovers original plaintext
   - Test nonce uniqueness across multiple encryptions
   - Test key version handling
   - Test invalid key size detection
   - Test corrupted ciphertext detection

2. **Unit Tests** (pkg/secretstore/)
   - Test CRUD operations on secrets table
   - Test unique constraint on secret ID
   - Test transaction semantics
   - Mock database for fast tests

3. **Integration Tests** (tests/integration/)
   - Test full API flow: POST → GET → PUT → DELETE
   - Test authentication integration
   - Test key rotation scenario (encrypt with key-v1, rotate, still decrypt)
   - Test provider chain fallback
   - Test concurrent requests
   - Use real SQLite in-memory database

4. **Contract Tests**
   - Validate OpenAPI spec against implementation
   - Test request/response schema compliance

**Test Data**:
- Fixture keys in `tests/fixtures/test-keys/`
- Sample secrets with known plaintext/ciphertext pairs
- Invalid inputs for negative testing

**Coverage Target**: >80% for new packages

---

## Summary

All research areas resolved with concrete technical decisions:

✅ **AES-GCM**: Random nonce generation, Go stdlib implementation  
✅ **Metadata Format**: Structured prefix `enc:provider:version:key:<base64>`  
✅ **Provider Interface**: Minimal design with EncryptedPayload struct  
✅ **Database Schema**: Dedicated `secrets` table with metadata columns  
✅ **Key Management**: Startup loading from raw binary files  
✅ **API Design**: Simple request/response, security-first error handling  
✅ **Testing**: Multi-layer coverage (unit, integration, contract)

**Ready for Phase 1**: Data model formalization, API contract generation, and quickstart documentation.

---

**Next Phase**: Phase 1 (Design Artifacts) - Generate data-model.md, contracts/secrets-api.yaml, and quickstart.md
