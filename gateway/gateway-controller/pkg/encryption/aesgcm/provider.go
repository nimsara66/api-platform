package aesgcm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/wso2/api-platform/gateway/gateway-controller/pkg/encryption"
	"go.uber.org/zap"
)

const (
	// NonceSize is the size of the nonce for AES-GCM (12 bytes is standard)
	NonceSize = 12
)

// AESGCMProvider implements encryption using AES-GCM
type AESGCMProvider struct {
	name       string
	keyManager *KeyManager
	logger     *zap.Logger
}

// NewAESGCMProvider creates a new AES-GCM encryption provider
func NewAESGCMProvider(keyConfigs []KeyConfig, logger *zap.Logger) (*AESGCMProvider, error) {
	keyManager, err := NewKeyManager(keyConfigs, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize key manager: %w", err)
	}

	provider := &AESGCMProvider{
		name:       "aesgcm",
		keyManager: keyManager,
		logger:     logger,
	}

	logger.Info("AES-GCM provider initialized",
		zap.String("provider", provider.name),
		zap.String("primary_key_version", keyManager.GetPrimaryVersion()),
	)

	return provider, nil
}

// Name returns the provider identifier
func (p *AESGCMProvider) Name() string {
	return p.name
}

// Encrypt encrypts plaintext using AES-GCM with a random nonce
func (p *AESGCMProvider) Encrypt(plaintext []byte) (*encryption.EncryptedPayload, error) {
	// Get primary key for encryption
	key := p.keyManager.GetPrimaryKey()

	// Create AES cipher
	block, err := aes.NewCipher(key.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce (12 bytes)
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	// GCM appends the auth tag to the ciphertext automatically
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	p.logger.Debug("Encrypted data with AES-GCM",
		zap.String("key_version", key.Version),
		zap.Int("plaintext_size", len(plaintext)),
		zap.Int("ciphertext_size", len(ciphertext)),
	)

	return &encryption.EncryptedPayload{
		Provider:   p.name,
		KeyVersion: key.Version,
		Ciphertext: ciphertext, // nonce || encrypted data || auth tag
	}, nil
}

// Decrypt decrypts ciphertext using AES-GCM
func (p *AESGCMProvider) Decrypt(payload *encryption.EncryptedPayload) ([]byte, error) {
	// Get the key used for encryption
	key, err := p.keyManager.GetKey(payload.KeyVersion)
	if err != nil {
		return nil, fmt.Errorf("key not found for version %s: %w", payload.KeyVersion, err)
	}

	// Validate ciphertext length (must be at least nonce size + tag size)
	if len(payload.Ciphertext) < NonceSize {
		return nil, fmt.Errorf("ciphertext too short: %d bytes", len(payload.Ciphertext))
	}

	// Create AES cipher
	block, err := aes.NewCipher(key.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce from the beginning of ciphertext
	nonce := payload.Ciphertext[:NonceSize]
	ciphertext := payload.Ciphertext[NonceSize:]

	// Decrypt and verify authentication tag
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (authentication error): %w", err)
	}

	p.logger.Debug("Decrypted data with AES-GCM",
		zap.String("key_version", key.Version),
		zap.Int("ciphertext_size", len(payload.Ciphertext)),
		zap.Int("plaintext_size", len(plaintext)),
	)

	return plaintext, nil
}

// HealthCheck validates that the provider is properly initialized
func (p *AESGCMProvider) HealthCheck() error {
	// Verify we have a primary key
	primaryKey := p.keyManager.GetPrimaryKey()
	if primaryKey == nil {
		return fmt.Errorf("no primary key available")
	}

	// Verify key size
	if len(primaryKey.Data) != AESKeySize {
		return &encryption.ErrInvalidKeySize{
			Expected: AESKeySize,
			Actual:   len(primaryKey.Data),
		}
	}

	// Test encryption/decryption round-trip
	testData := []byte("health-check-test-data")
	encrypted, err := p.Encrypt(testData)
	if err != nil {
		return fmt.Errorf("health check encryption failed: %w", err)
	}

	decrypted, err := p.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("health check decryption failed: %w", err)
	}

	if string(decrypted) != string(testData) {
		return fmt.Errorf("health check round-trip failed: data mismatch")
	}

	p.logger.Debug("AES-GCM provider health check passed")
	return nil
}
