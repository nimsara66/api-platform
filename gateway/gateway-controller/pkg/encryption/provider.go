package encryption

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// EncryptionProvider defines the interface for encryption implementations
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

// EncryptedPayload represents encrypted data with metadata
type EncryptedPayload struct {
	Provider   string // Provider type identifier (e.g., "aesgcm")
	KeyVersion string // Key name/version (e.g., "key-v2")
	Ciphertext []byte // Encrypted bytes (nonce || ciphertext || tag for AES-GCM)
}

// MarshalPayload converts EncryptedPayload to storage format
// Format: enc:provider:v1:key-version:base64-ciphertext
func MarshalPayload(payload *EncryptedPayload) string {
	encoded := base64.StdEncoding.EncodeToString(payload.Ciphertext)
	return fmt.Sprintf("enc:%s:v1:%s:%s", payload.Provider, payload.KeyVersion, encoded)
}

// UnmarshalPayload converts storage format back to EncryptedPayload
// Expects format: enc:provider:v1:key-version:base64-ciphertext
func UnmarshalPayload(stored string) (*EncryptedPayload, error) {
	parts := strings.SplitN(stored, ":", 5)
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid payload format: expected 5 parts, got %d", len(parts))
	}

	if parts[0] != "enc" {
		return nil, fmt.Errorf("invalid payload prefix: expected 'enc', got '%s'", parts[0])
	}

	if parts[2] != "v1" {
		return nil, fmt.Errorf("unsupported payload version: %s", parts[2])
	}

	ciphertext, err := base64.StdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	return &EncryptedPayload{
		Provider:   parts[1],
		KeyVersion: parts[3],
		Ciphertext: ciphertext,
	}, nil

}
