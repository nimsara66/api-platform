# Secret Management API - Quickstart Guide

**Version**: 1.0  
**Last Updated**: 2026-01-05

---

## Overview

This guide helps platform operators get started with the Gateway Secret Management API. You'll learn how to:

- Generate encryption keys
- Configure encryption providers
- Store and retrieve encrypted secrets
- Rotate encryption keys

**Prerequisites**:
- Gateway Controller installed and running
- `openssl` installed (for key generation)
- `curl` or similar HTTP client
- Basic authentication credentials or JWT token

---

## 1. Generate Encryption Keys

Secrets are encrypted using AES-256, which requires 32-byte keys. Generate keys using `openssl`:

```bash
# Create a directory for encryption keys
mkdir -p /etc/gateway/secrets
cd /etc/gateway/secrets

# Generate the first key (key-v1)
openssl rand -out key-v1.bin 32

# Set restrictive permissions (read-only, owner-only)
chmod 400 key-v1.bin

# Verify the key size (should be exactly 32 bytes)
ls -lh key-v1.bin
# Expected output: -r-------- 1 user group 32 Jan 5 10:00 key-v1.bin
```

**Important**:
- Keys must be exactly 32 bytes (256 bits)
- Use strong file permissions (0400 or 0600)
- Store keys securely and back them up separately from the database
- Never commit keys to version control

---

## 2. Configure Encryption Providers

Add encryption configuration to your gateway `config.yaml`:

```yaml
# config.yaml
server:
  host: 0.0.0.0
  port: 9000

# ... existing configuration ...

# NEW: Encryption provider configuration
encryption:
  providers:
    - type: aesgcm
      keys:
        - name: key-v1
          path: /etc/gateway/secrets/key-v1.bin
```

**Configuration Details**:
- **providers**: Array of encryption providers (order matters!)
- **type**: Provider type (`aesgcm` for AES-GCM encryption)
- **keys**: Array of encryption keys for this provider
- **name**: Friendly name for the key (used in logs and metadata)
- **path**: Absolute path to the 32-byte raw binary key file

**Provider Priority**:
- The **first provider** in the array encrypts new secrets
- **All providers** can decrypt secrets (enables key rotation)

---

## 3. Start the Gateway

Start the gateway controller with the updated configuration:

```bash
# Start gateway-controller
./gateway-controller --config /path/to/config.yaml

# Expected startup log output:
# INFO  Initializing encryption providers
# INFO  Loaded AES-GCM provider with 1 key(s)
# INFO  Provider 'aesgcm' health check: OK
# INFO  Encryption manager initialized with 1 provider(s)
# INFO  Secret management API available at /secrets
```

If you see errors about missing keys or invalid key sizes, verify:
- Key file paths are absolute and correct
- Key files are exactly 32 bytes
- Gateway process has read permissions on key files

---

## 4. Store a Secret

Create a new encrypted secret using the REST API:

```bash
# Store a database password
curl -X POST http://localhost:9000/secrets \
  -H "Content-Type: application/json" \
  -u admin:password \
  -d '{
    "id": "database-password",
    "value": "sup3rs3cr3t!"
  }'

# Expected response (201 Created):
{
  "id": "database-password",
  "value": "sup3rs3cr3t!",
  "created_at": "2026-01-05T10:30:00Z",
  "updated_at": "2026-01-05T10:30:00Z"
}
```

**What happens**:
1. Your secret value is encrypted using AES-GCM with key-v1
2. The ciphertext is stored in the SQLite database
3. Only the ciphertext is persisted (never plaintext)
4. The response includes the plaintext value for confirmation

**Error Responses**:
- **400 Bad Request**: Missing `id` or `value` field
- **401 Unauthorized**: Invalid authentication credentials
- **409 Conflict**: A secret with this ID already exists

---

## 5. Retrieve a Secret

Retrieve and decrypt a secret:

```bash
curl -X GET http://localhost:9000/secrets/database-password \
  -u admin:password

# Expected response (200 OK):
{
  "id": "database-password",
  "value": "sup3rs3cr3t!",
  "created_at": "2026-01-05T10:30:00Z",
  "updated_at": "2026-01-05T10:30:00Z"
}
```

**What happens**:
1. The encrypted secret is loaded from the database
2. The system identifies which encryption provider to use (from metadata)
3. The secret is decrypted using the appropriate key
4. The plaintext value is returned in the response

**Error Responses**:
- **401 Unauthorized**: Invalid authentication
- **404 Not Found**: Secret does not exist
- **500 Internal Server Error**: Decryption failed (corrupted data or missing key)

---

## 6. Update a Secret

Update an existing secret with a new value:

```bash
curl -X PUT http://localhost:9000/secrets/database-password \
  -H "Content-Type: application/json" \
  -u admin:password \
  -d '{
    "value": "n3w_p@ssw0rd!"
  }'

# Expected response (200 OK):
{
  "id": "database-password",
  "value": "n3w_p@ssw0rd!",
  "created_at": "2026-01-05T10:30:00Z",
  "updated_at": "2026-01-05T11:45:00Z"
}
```

**What happens**:
1. The old secret is retrieved and verified to exist
2. The new value is encrypted using the **current primary provider**
3. The database is updated with the new ciphertext
4. The `updated_at` timestamp is refreshed

**Note**: If you've rotated keys (see section 7), the update automatically re-encrypts with the new key!

---

## 7. Delete a Secret

Permanently delete a secret:

```bash
curl -X DELETE http://localhost:9000/secrets/database-password \
  -u admin:password

# Expected response (204 No Content):
# (empty response body)
```

**What happens**:
1. The secret is permanently removed from the database
2. This is a **hard delete** with no recovery mechanism
3. Subsequent GET requests return 404 Not Found

**Error Responses**:
- **401 Unauthorized**: Invalid authentication
- **404 Not Found**: Secret does not exist (or already deleted)

---

## 8. Key Rotation

Over time, you'll want to rotate encryption keys for security best practices. Here's how:

### Step 1: Generate a New Key

```bash
cd /etc/gateway/secrets

# Generate key-v2
openssl rand -out key-v2.bin 32
chmod 400 key-v2.bin
```

### Step 2: Update Configuration

Add the new key **at the beginning** of the `keys` array:

```yaml
encryption:
  providers:
    - type: aesgcm
      keys:
        - name: key-v2        # NEW: Primary key (encrypts new secrets)
          path: /etc/gateway/secrets/key-v2.bin
        - name: key-v1        # OLD: Kept for decrypting existing secrets
          path: /etc/gateway/secrets/key-v1.bin
```

**Key Order Matters**:
- First key = **primary** (used for new encryptions)
- All keys = **available for decryption** (supports reading old secrets)

### Step 3: Restart Gateway

```bash
# Restart to load the new configuration
./gateway-controller --config /path/to/config.yaml

# Expected log output:
# INFO  Loaded AES-GCM provider with 2 key(s)
# INFO  Primary key: key-v2
```

### Step 4: Verify Rotation

```bash
# Create a new secret (encrypted with key-v2)
curl -X POST http://localhost:9000/secrets \
  -u admin:password \
  -H "Content-Type: application/json" \
  -d '{"id": "new-secret", "value": "encrypted-with-v2"}'

# Retrieve an old secret (decrypted with key-v1)
curl -X GET http://localhost:9000/secrets/database-password \
  -u admin:password

# Both operations should succeed!
```

### Step 5: (Optional) Re-encrypt Old Secrets

To migrate old secrets to the new key, update them:

```bash
# Re-encrypt by updating the secret
curl -X PUT http://localhost:9000/secrets/database-password \
  -u admin:password \
  -H "Content-Type: application/json" \
  -d '{"value": "sup3rs3cr3t!"}' # Same value, new encryption

# The secret is now encrypted with key-v2
```

### Step 6: (Later) Remove Old Key

Once all secrets are re-encrypted with key-v2, you can remove key-v1:

```yaml
encryption:
  providers:
    - type: aesgcm
      keys:
        - name: key-v2
          path: /etc/gateway/secrets/key-v2.bin
        # key-v1 removed - old secrets must be re-encrypted first!
```

---

## 9. Troubleshooting

### Secret Decryption Fails (500 Error)

**Symptom**: GET request returns `{"error": "internal_error", "message": "Failed to decrypt secret"}`

**Possible Causes**:
1. **Missing Key**: The key used to encrypt the secret is no longer in the configuration
2. **Corrupted Data**: The database was manually edited or corrupted
3. **Wrong Key File**: The key file was replaced with a different key

**Solution**:
```bash
# Check server logs for detailed error information
tail -f /var/log/gateway-controller.log | grep "correlation_id"

# Look for messages like:
# ERROR Secret decryption failed correlation_id=req-abc-123 secret_id=my-secret provider=aesgcm key_version=key-v1 error="key not found"

# Solution: Add the missing key back to the configuration
```

### Secret Creation Fails (409 Conflict)

**Symptom**: POST request returns `{"error": "conflict", "message": "Secret already exists"}`

**Solution**: Choose a different ID or delete the existing secret first:
```bash
curl -X DELETE http://localhost:9000/secrets/database-password -u admin:password
```

### Encryption Provider Initialization Fails

**Symptom**: Gateway fails to start with error: `failed to load key key-v1: open /etc/gateway/secrets/key-v1.bin: no such file or directory`

**Solutions**:
1. **Check path**: Ensure the path in `config.yaml` is absolute and correct
2. **Check permissions**: Ensure gateway process can read the key file
3. **Verify key size**: Ensure the key file is exactly 32 bytes

```bash
# Check file existence
ls -lh /etc/gateway/secrets/key-v1.bin

# Check file size (should be 32 bytes)
wc -c < /etc/gateway/secrets/key-v1.bin

# Check permissions (gateway user must have read access)
sudo -u gateway-user cat /etc/gateway/secrets/key-v1.bin
```

---

## 10. Best Practices

### Security

✅ **DO**:
- Store keys on encrypted filesystems
- Use restrictive file permissions (0400 or 0600)
- Back up keys separately from the database
- Rotate keys periodically (every 6-12 months)
- Use strong authentication for API access

❌ **DON'T**:
- Commit keys to version control
- Share keys between environments (dev/staging/prod)
- Log or expose keys in error messages
- Store keys in the same location as the database

### Operations

✅ **DO**:
- Test key rotation in non-production first
- Monitor encryption/decryption performance
- Set up alerts for failed decryptions
- Document your key rotation schedule
- Keep at least 2 keys in the configuration during rotation

❌ **DON'T**:
- Remove old keys before re-encrypting all secrets
- Restart the gateway during peak traffic
- Skip verification after key rotation

### Naming Conventions

✅ **DO**:
- Use descriptive secret IDs: `prod-db-password`, `api-key-stripe`
- Version key names: `key-v1`, `key-v2`, `key-2026-01`
- Namespace secrets: `<env>.<service>.<resource>`

❌ **DON'T**:
- Use generic IDs: `secret1`, `password`, `key`
- Include sensitive information in IDs: `password-sup3rs3cr3t`

---

## 11. Integration Examples

### Using Secrets in Application Code

```go
// Go example: Retrieve database credentials
func getDBPassword(client *http.Client) (string, error) {
    req, _ := http.NewRequest("GET", "http://localhost:9000/secrets/database-password", nil)
    req.SetBasicAuth("admin", "password")
    
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return "", fmt.Errorf("failed to retrieve secret: %s", resp.Status)
    }
    
    var result struct {
        ID    string `json:"id"`
        Value string `json:"value"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    return result.Value, nil
}
```

```python
# Python example: Retrieve API key
import requests
from requests.auth import HTTPBasicAuth

def get_api_key():
    response = requests.get(
        'http://localhost:9000/secrets/api-key-stripe',
        auth=HTTPBasicAuth('admin', 'password')
    )
    response.raise_for_status()
    return response.json()['value']

# Use in application
api_key = get_api_key()
stripe.api_key = api_key
```

### Automated Secret Rotation Script

```bash
#!/bin/bash
# rotate-secrets.sh - Automate secret re-encryption after key rotation

GATEWAY_URL="http://localhost:9000"
AUTH="admin:password"

# List of secrets to rotate (customize for your environment)
SECRETS=(
    "database-password"
    "api-key-stripe"
    "redis-password"
)

echo "Starting secret rotation..."

for secret_id in "${SECRETS[@]}"; do
    echo "Re-encrypting: $secret_id"
    
    # Get current value
    value=$(curl -s -X GET "$GATEWAY_URL/secrets/$secret_id" -u "$AUTH" | jq -r '.value')
    
    if [ "$value" != "null" ]; then
        # Update with same value (re-encrypts with new key)
        curl -s -X PUT "$GATEWAY_URL/secrets/$secret_id" \
            -u "$AUTH" \
            -H "Content-Type: application/json" \
            -d "{\"value\": \"$value\"}"
        
        echo "  ✓ Re-encrypted with new key"
    else
        echo "  ✗ Failed to retrieve secret"
    fi
done

echo "Rotation complete!"
```

---

## 12. Next Steps

Now that you've completed the quickstart:

- **Read the full specification**: [spec.md](./spec.md)
- **Explore the API contract**: [contracts/secrets-api.yaml](./contracts/secrets-api.yaml)
- **Review the data model**: [data-model.md](./data-model.md)
- **Set up monitoring**: Track encryption/decryption metrics
- **Plan key rotation**: Establish a regular rotation schedule

**Questions or Issues?**
- Check server logs: `/var/log/gateway-controller.log`
- Review troubleshooting section (Section 9)
- Contact: WSO2 API Platform Team

---

**Document Version**: 1.0  
**Last Updated**: 2026-01-05  
**Related Documents**: [spec.md](./spec.md), [plan.md](./plan.md), [data-model.md](./data-model.md)
