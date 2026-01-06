# Data Model: Secret Management API

**Feature**: Custom Secret Management REST API with Pluggable Encryption  
**Date**: 2026-01-05  
**Phase**: 1 (Design Artifacts)

---

## Overview

This document defines the data model for the secret management system, including domain entities, storage schema, configuration structures, and their relationships. The model supports pluggable encryption providers with a provider chain mechanism for key rotation.

---

## 1. Domain Entities

### 1.1 Secret (Domain Model)

Represents a sensitive value managed by the system. Exists in two forms: plaintext (in-memory only) and encrypted (persisted).

**Fields**:

| Field | Type | Description | Constraints |
|-------|------|-------------|-------------|
| ID | string | User-provided unique identifier | Required, unique, non-empty |
| Value | string | Plaintext secret data | Required, max 10KB |
| Provider | string | Encryption provider identifier | Auto-populated during encryption |
| KeyVersion | string | Key version used for encryption | Auto-populated during encryption |
| Ciphertext | []byte | Encrypted secret with metadata | Binary, includes nonce and auth tag |
| CreatedAt | time.Time | Creation timestamp | Auto-populated, UTC |
| UpdatedAt | time.Time | Last modification timestamp | Auto-populated, UTC |

**Business Rules**:
- ID provided by client (not system-generated)
- Value never persisted in plaintext
- Provider and KeyVersion immutable after creation (updated only on secret update)
- Timestamps maintained by storage layer

**Lifecycle States**:
```
[Client Request] 
    ↓
[Plaintext Secret] (in-memory only)
    ↓ (encrypt via primary provider)
[Encrypted Secret] (persisted in database)
    ↓ (decrypt via provider chain)
[Plaintext Secret] (in-memory, returned to client)
```

---

### 1.2 EncryptionProvider (Interface)

Abstract interface for encryption implementations. Supports multiple provider types (AES-GCM, future external KMS).

**Methods**:

```go
type EncryptionProvider interface {
    // Name returns the provider identifier (e.g., "aesgcm", "vault")
    Name() string
    
    // Encrypt transforms plaintext into encrypted payload using the active key
    Encrypt(plaintext []byte) (*EncryptedPayload, error)
    
    // Decrypt transforms encrypted payload back to plaintext
    Decrypt(payload *EncryptedPayload) ([]byte, error)
    
    // HealthCheck validates provider initialization and key availability
    HealthCheck() error
}
```

**Implementations**:
- **AESGCMProvider**: Uses Go stdlib crypto/cipher with 256-bit keys
- **ExternalProvider**: (Future) Vault, AWS KMS, Azure Key Vault, etc.

---

### 1.3 EncryptedPayload (Struct)

Container for encrypted data with metadata. Decouples internal representation from storage format.

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| Provider | string | Provider type identifier ("aesgcm") |
| KeyVersion | string | Key name/version ("key-v2") |
| Ciphertext | []byte | Encrypted bytes (nonce \|\| ciphertext \|\| tag) |

**Serialization**:
- **Storage Format**: `enc:aesgcm:v1:key-v2:<base64-ciphertext>`
- **In-Memory Format**: Struct with parsed fields
- **Conversion**: Bidirectional marshaling/unmarshaling functions

**Example**:
```go
payload := &EncryptedPayload{
    Provider:   "aesgcm",
    KeyVersion: "key-v2",
    Ciphertext: []byte{...}, // nonce || encrypted data || auth tag
}

// Serialized form
stored := "enc:aesgcm:v1:key-v2:A7xP3q..."
```

---

### 1.4 ProviderManager (Orchestrator)

Manages the encryption provider chain. Handles encryption (first provider) and decryption (provider fallback).

**Responsibilities**:
- Initialize providers from configuration at startup
- Route encryption requests to primary provider (first in chain)
- Route decryption requests to appropriate provider based on metadata
- Provide health check aggregation across all providers

**Chain Logic**:
```
Encryption:  [Request] → Primary Provider (providers[0]) → [EncryptedPayload]

Decryption:  [EncryptedPayload] → Try Provider by Name → [Plaintext]
             If not found → Error (no provider can decrypt)
```

**Initialization**:
```go
type ProviderManager struct {
    providers []EncryptionProvider  // Ordered by configuration
    logger    *zap.Logger
}

func NewProviderManager(configs []ProviderConfig, logger *zap.Logger) (*ProviderManager, error)
```

---

## 2. Storage Schema

### 2.1 Database Table: `secrets`

**SQLite Schema**:

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

**Column Descriptions**:

| Column | Type | Description | Constraints |
|--------|------|-------------|-------------|
| id | TEXT | Secret identifier (user-provided) | PRIMARY KEY, NOT NULL |
| provider | TEXT | Provider type for query optimization | NOT NULL |
| key_version | TEXT | Key version for query optimization | NOT NULL |
| ciphertext | BLOB | Full encrypted payload with metadata | NOT NULL |
| created_at | TIMESTAMP | Creation time (UTC) | NOT NULL, DEFAULT CURRENT_TIMESTAMP |
| updated_at | TIMESTAMP | Last update time (UTC) | NOT NULL, DEFAULT CURRENT_TIMESTAMP |

**Indexes**:
- Primary key on `id` (unique constraint, fast lookups)
- Index on `updated_at` (supports audit queries, cleanup operations)

**Design Rationale**:
- `provider` and `key_version` duplicated from `ciphertext` for efficient filtering
- No `value` column (enforces no-plaintext-persistence requirement)
- Timestamps at database level ensure consistency

**Storage Size**:
- Base64-encoded metadata: ~50 bytes
- AES-GCM overhead: 28 bytes (12-byte nonce + 16-byte auth tag)
- For 1KB secret: ~1.4KB total storage per secret

---

### 2.2 Storage Interface Extension

The existing `pkg/storage/interface.go` is NOT modified. A new `pkg/secretstore/interface.go` is created for secret-specific operations.

**SecretStorage Interface**:

```go
package secretstore

type SecretStorage interface {
    // SaveSecret persists a new encrypted secret
    SaveSecret(secret *models.Secret) error
    
    // GetSecret retrieves a secret by ID
    GetSecret(id string) (*models.Secret, error)
    
    // UpdateSecret updates an existing secret
    UpdateSecret(secret *models.Secret) error
    
    // DeleteSecret removes a secret by ID
    DeleteSecret(id string) error
    
    // ListSecrets returns all secret IDs (for admin/debug)
    ListSecrets() ([]string, error)
}
```

**Implementation**: `SQLiteSecretStorage` in `pkg/secretstore/sqlite.go`

---

## 3. Configuration Model

### 3.1 EncryptionConfig (YAML Configuration)

**Structure**:

```yaml
encryption:
  providers:
    - type: aesgcm
      keys:
        - name: key-v2
          path: /etc/secrets/key-v2.bin
        - name: key-v1
          path: /etc/secrets/key-v1.bin
```

**Go Struct**:

```go
type EncryptionConfig struct {
    Providers []ProviderConfig `yaml:"providers"`
}

type ProviderConfig struct {
    Type string       `yaml:"type"`  // "aesgcm", "vault", etc.
    Keys []KeyConfig  `yaml:"keys"`
}

type KeyConfig struct {
    Name string `yaml:"name"`  // "key-v1", "key-v2"
    Path string `yaml:"path"`  // "/etc/secrets/key-v1.bin"
}
```

**Validation Rules**:
- At least one provider required
- At least one key required per provider
- Key paths must be absolute
- Key files must exist and be readable at startup
- Key files must be exactly 32 bytes for AES-256

**Provider Priority**:
- Array order determines priority (first = primary for encryption)
- All providers remain available for decryption

---

### 3.2 KeyManager (Runtime Key Storage)

**Purpose**: Loads and caches encryption keys in memory for fast access.

**Structure**:

```go
type KeyManager struct {
    keys map[string][]byte  // keyName → 32-byte key material
    mu   sync.RWMutex       // Protects keys map for concurrent access
}

func NewKeyManager(configs []KeyConfig) (*KeyManager, error)
func (km *KeyManager) GetKey(name string) ([]byte, error)
```

**Lifecycle**:
1. **Startup**: Load all keys from configured file paths
2. **Runtime**: Serve key material from in-memory cache
3. **Shutdown**: Keys cleared from memory (no persistence)

**Security**:
- Keys never logged
- Keys never serialized
- Keys never exposed via API
- File permissions: 0400 or 0600 (read-only, owner-only)

---

## 4. Entity Relationships

```
┌─────────────────────────┐
│   Configuration         │
│   (encryption.yaml)     │
└───────────┬─────────────┘
            │ loads at startup
            ▼
┌─────────────────────────┐
│   ProviderManager       │
│   - providers: []       │
└───────────┬─────────────┘
            │ contains
            ▼
┌─────────────────────────┐       ┌─────────────────────────┐
│  EncryptionProvider     │       │   KeyManager            │
│  (interface)            │◄──────│   - keys: map[string][] │
└───────────┬─────────────┘ uses  └─────────────────────────┘
            │
            │ implements
            ▼
┌─────────────────────────┐
│   AESGCMProvider        │
│   - keyManager          │
└───────────┬─────────────┘
            │
            │ produces/consumes
            ▼
┌─────────────────────────┐
│   EncryptedPayload      │
│   - Provider: string    │
│   - KeyVersion: string  │
│   - Ciphertext: []byte  │
└───────────┬─────────────┘
            │
            │ persisted as
            ▼
┌─────────────────────────┐
│   Secret (Storage)      │
│   - id                  │
│   - provider            │
│   - key_version         │
│   - ciphertext          │
│   - created_at          │
│   - updated_at          │
└─────────────────────────┘
```

**Relationships**:
- ProviderManager → EncryptionProvider (1:N, composition)
- EncryptionProvider → KeyManager (N:1, dependency)
- EncryptionProvider → EncryptedPayload (produces/consumes)
- Secret → EncryptedPayload (contains serialized form)

---

## 5. Data Flow

### 5.1 Secret Creation (POST /secrets)

```
[Client Request: {id, value}]
    ↓
[Handler validates request]
    ↓
[Service: secretService.Create(id, value)]
    ↓
[ProviderManager.Encrypt(value)]
    ↓
[AESGCMProvider.Encrypt(value)]
    ├─ Load active key from KeyManager
    ├─ Generate random 12-byte nonce
    ├─ AES-GCM encrypt: ciphertext = AES-256-GCM(key, nonce, value)
    └─ Return EncryptedPayload{Provider:"aesgcm", KeyVersion:"key-v2", Ciphertext:...}
    ↓
[EncryptedPayload.Serialize() → "enc:aesgcm:v1:key-v2:<base64>"]
    ↓
[SecretStorage.SaveSecret(Secret{ID, Provider, KeyVersion, Ciphertext, Timestamps})]
    ↓
[SQLite INSERT INTO secrets]
    ↓
[Handler returns 201: {id, value, created_at, updated_at}]
```

### 5.2 Secret Retrieval (GET /secrets/{id})

```
[Client Request: GET /secrets/my-secret]
    ↓
[Handler extracts ID from path]
    ↓
[Service: secretService.Get(id)]
    ↓
[SecretStorage.GetSecret(id)]
    ↓
[SQLite SELECT * FROM secrets WHERE id = ?]
    ↓
[Secret loaded with ciphertext BLOB]
    ↓
[EncryptedPayload.Parse(ciphertext) → {Provider, KeyVersion, Ciphertext}]
    ↓
[ProviderManager.Decrypt(payload)]
    ├─ Find provider by name (payload.Provider == "aesgcm")
    └─ AESGCMProvider.Decrypt(payload)
        ├─ Load key from KeyManager (payload.KeyVersion == "key-v2")
        ├─ Extract nonce from ciphertext (first 12 bytes)
        ├─ AES-GCM decrypt: plaintext = AES-256-GCM-Decrypt(key, nonce, ciphertext)
        └─ Verify authentication tag (integrity check)
    ↓
[Service returns plaintext value]
    ↓
[Handler returns 200: {id, value, created_at, updated_at}]
```

### 5.3 Secret Update (PUT /secrets/{id})

```
[Client Request: PUT /secrets/my-secret {value}]
    ↓
[Handler validates request]
    ↓
[Service: secretService.Update(id, newValue)]
    ↓
[SecretStorage.GetSecret(id) - verify exists]
    ↓
[ProviderManager.Encrypt(newValue) - use current primary provider]
    ↓
[SecretStorage.UpdateSecret(Secret{ID, NewCiphertext, UpdatedAt})]
    ↓
[SQLite UPDATE secrets SET ciphertext=?, updated_at=? WHERE id=?]
    ↓
[Handler returns 200: {id, value, created_at, updated_at}]
```

**Key Rotation Effect**: If primary provider changed from key-v1 to key-v2, the update automatically re-encrypts with key-v2. Old secrets encrypted with key-v1 remain readable via provider chain.

---

## 6. Validation Rules

### 6.1 Secret Validation

**Create (POST)**:
- ID: Non-empty string, max 255 characters, alphanumeric + hyphens/underscores
- Value: Non-empty string, max 10KB (10,240 bytes)
- Duplicate ID: Return 409 Conflict

**Update (PUT)**:
- Value: Non-empty string, max 10KB
- Secret must exist: Return 404 if not found

**Retrieve (GET)**:
- Secret must exist: Return 404 if not found
- Decryption must succeed: Return 500 if all providers fail

**Delete (DELETE)**:
- Secret must exist: Return 404 if not found (or 204 regardless, for security)

---

### 6.2 Configuration Validation

**Startup Validation**:
```go
func ValidateEncryptionConfig(cfg *EncryptionConfig) error {
    if len(cfg.Providers) == 0 {
        return errors.New("at least one encryption provider required")
    }
    
    for _, provider := range cfg.Providers {
        if len(provider.Keys) == 0 {
            return fmt.Errorf("provider %s requires at least one key", provider.Type)
        }
        
        for _, key := range provider.Keys {
            if !filepath.IsAbs(key.Path) {
                return fmt.Errorf("key path must be absolute: %s", key.Path)
            }
            
            data, err := os.ReadFile(key.Path)
            if err != nil {
                return fmt.Errorf("cannot read key %s: %w", key.Name, err)
            }
            
            if len(data) != 32 {
                return fmt.Errorf("key %s invalid size: %d bytes (expected 32)", key.Name, len(data))
            }
        }
    }
    
    return nil
}
```

---

## 7. State Transitions

### 7.1 Secret Lifecycle

```
                    POST /secrets
                         ↓
                   ┌──────────┐
                   │ Encrypted │ ←─┐
                   │  Secret   │   │
                   └──────────┘   │
                         │         │
              ┌──────────┼──────────┐
              │          │          │
         GET /secrets  PUT /secrets  DELETE /secrets
              │          │          │
              ↓          ↓          ↓
        [Decrypt]   [Re-encrypt] [Remove]
         [Return]    [Update DB]  [204 No Content]
```

**State**: Secrets have no explicit state field - existence in database is the only state.

---

### 7.2 Key Rotation Workflow

```
Initial State:
  Config: [key-v1]
  Secrets: All encrypted with key-v1

Step 1: Add new key
  Config: [key-v2, key-v1]  ← key-v2 is now primary
  Secrets: Still encrypted with key-v1 (readable via chain)

Step 2: Restart gateway
  New secrets encrypted with key-v2
  Old secrets still readable (key-v1 in chain)

Step 3: (Optional) Trigger re-encryption
  PUT /secrets/{id} for each secret
  Secrets migrated to key-v2

Step 4: (Later) Remove old key
  Config: [key-v2]
  Assumption: All secrets re-encrypted or old ones acceptable to lose
```

---

## 8. Error Conditions

### 8.1 Domain Errors

| Error | Condition | HTTP Status |
|-------|-----------|-------------|
| `ErrSecretNotFound` | Secret ID does not exist | 404 |
| `ErrSecretExists` | Duplicate secret ID on create | 409 |
| `ErrInvalidSecretID` | Empty or invalid ID format | 400 |
| `ErrInvalidSecretValue` | Empty or oversized value | 400 |
| `ErrDecryptionFailed` | No provider can decrypt | 500 |
| `ErrProviderNotFound` | Payload references unknown provider | 500 |
| `ErrKeyNotFound` | Payload references unknown key version | 500 |

### 8.2 Error Handling Strategy

**Client-Facing**:
- Generic error messages (no sensitive details)
- Correlation ID for troubleshooting
- Standard HTTP status codes

**Server-Side Logging**:
- Detailed error context (provider, key version, stack trace)
- Correlation ID links client error to logs
- Structured logging with zap

---

## Summary

The data model provides a clear separation of concerns:

✅ **Domain Layer**: Secret, EncryptionProvider, ProviderManager  
✅ **Storage Layer**: SQLite schema, SecretStorage interface  
✅ **Configuration Layer**: EncryptionConfig, KeyManager  
✅ **Data Flow**: Well-defined paths for CRUD operations with encryption/decryption

**Key Design Principles**:
- No plaintext persistence
- Provider abstraction for extensibility
- Metadata enables provider chain and key rotation
- Validation at all layers (request, domain, storage)

**Ready for**: API contract definition (OpenAPI spec) and implementation.

---

**Next**: Generate `contracts/secrets-api.yaml` (OpenAPI specification)
