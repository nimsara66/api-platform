package secrets

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wso2/api-platform/gateway/gateway-controller/pkg/encryption"
	"github.com/wso2/api-platform/gateway/gateway-controller/pkg/models"
	"github.com/wso2/api-platform/gateway/gateway-controller/pkg/storage"
	"go.uber.org/zap"
)

const (
	// MaxSecretSize is the maximum allowed size for a secret value (10KB)
	MaxSecretSize = 10 * 1024
)

// SecretService handles business logic for secret operations
type SecretService struct {
	storage         storage.Storage
	providerManager *encryption.ProviderManager
	logger          *zap.Logger
}

// NewSecretService creates a new secret service
func NewSecretService(
	storage storage.Storage,
	providerManager *encryption.ProviderManager,
	logger *zap.Logger,
) *SecretService {
	return &SecretService{
		storage:         storage,
		providerManager: providerManager,
		logger:          logger,
	}
}

// Create creates a new secret with encryption
func (s *SecretService) Create(handle, value string, correlationID string) (*models.Secret, error) {
	// Validate inputs
	if handle == "" {
		return nil, fmt.Errorf("secret handle cannot be empty")
	}
	if value == "" {
		return nil, fmt.Errorf("secret value cannot be empty")
	}
	if len(value) > MaxSecretSize {
		return nil, fmt.Errorf("secret value too large: %d bytes (max %d bytes)", len(value), MaxSecretSize)
	}

	s.logger.Info("Creating secret",
		zap.String("secret_handle", handle),
		zap.String("correlation_id", correlationID),
		zap.Int("value_size", len(value)),
	)

	// Encrypt the secret value
	payload, err := s.providerManager.Encrypt([]byte(value))
	if err != nil {
		s.logger.Error("Failed to encrypt secret",
			zap.String("secret_handle", handle),
			zap.String("correlation_id", correlationID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Serialize the encrypted payload for storage
	ciphertext := encryption.MarshalPayload(payload)

	// Create secret model
	secret := &models.Secret{
		ID:         generateUUID(),
		Handle:     handle,
		Value:      "", // Don't store plaintext
		Provider:   payload.Provider,
		KeyVersion: payload.KeyVersion,
		Ciphertext: []byte(ciphertext),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	// Persist encrypted secret
	if err := s.storage.SaveSecret(secret); err != nil {
		s.logger.Error("Failed to save secret",
			zap.String("secret_handle", handle),
			zap.String("correlation_id", correlationID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("storage failed: %w", err)
	}

	s.logger.Info("Secret created successfully",
		zap.String("secret_handle", handle),
		zap.String("correlation_id", correlationID),
		zap.String("provider", payload.Provider),
		zap.String("key_version", payload.KeyVersion),
	)

	// Return secret with plaintext value for response
	secret.Value = value
	return secret, nil
}

// Get retrieves and decrypts a secret
func (s *SecretService) Get(handle string, correlationID string) (*models.Secret, error) {
	s.logger.Info("Retrieving secret",
		zap.String("secret_handle", handle),
		zap.String("correlation_id", correlationID),
	)

	// Retrieve encrypted secret from storage
	secret, err := s.storage.GetSecret(handle)
	if err != nil {
		// Don't log details for not found errors (common case)
		if _, ok := err.(*storage.SecretNotFoundError); ok {
			s.logger.Debug("Secret not found",
				zap.String("secret_handle", handle),
				zap.String("correlation_id", correlationID),
			)
		} else {
			s.logger.Error("Failed to retrieve secret",
				zap.String("secret_handle", handle),
				zap.String("correlation_id", correlationID),
				zap.Error(err),
			)
		}
		return nil, err
	}

	// Deserialize the encrypted payload
	payload, err := encryption.UnmarshalPayload(string(secret.Ciphertext))
	if err != nil {
		s.logger.Error("Failed to unmarshal encrypted payload",
			zap.String("secret_handle", handle),
			zap.String("correlation_id", correlationID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("payload deserialization failed: %w", err)
	}

	// Decrypt the secret value
	plaintext, err := s.providerManager.Decrypt(payload)
	if err != nil {
		s.logger.Error("Failed to decrypt secret",
			zap.String("secret_handle", handle),
			zap.String("provider", payload.Provider),
			zap.String("key_version", payload.KeyVersion),
			zap.String("correlation_id", correlationID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	s.logger.Info("Secret retrieved successfully",
		zap.String("secret_handle", handle),
		zap.String("correlation_id", correlationID),
		zap.String("provider", payload.Provider),
		zap.String("key_version", payload.KeyVersion),
	)

	// Set plaintext value in secret
	secret.Value = string(plaintext)
	return secret, nil
}

// Update updates an existing secret with re-encryption using current primary key
func (s *SecretService) Update(id, value string, correlationID string) (*models.Secret, error) {
	// Validate inputs
	if value == "" {
		return nil, fmt.Errorf("secret value cannot be empty")
	}
	if len(value) > MaxSecretSize {
		return nil, fmt.Errorf("secret value too large: %d bytes (max %d bytes)", len(value), MaxSecretSize)
	}

	s.logger.Info("Updating secret",
		zap.String("secret_handle", id),
		zap.String("correlation_id", correlationID),
		zap.Int("value_size", len(value)),
	)

	// Check if secret exists
	exists, err := s.storage.SecretExists(id)
	if err != nil {
		return nil, fmt.Errorf("failed to check secret existence: %w", err)
	}
	if !exists {
		return nil, &storage.SecretNotFoundError{Handle: id}
	}

	// Encrypt with current primary key (automatic key migration)
	payload, err := s.providerManager.Encrypt([]byte(value))
	if err != nil {
		s.logger.Error("Failed to encrypt secret",
			zap.String("secret_handle", id),
			zap.String("correlation_id", correlationID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Serialize the encrypted payload
	ciphertext := encryption.MarshalPayload(payload)

	// Update secret model
	secret := &models.Secret{
		Handle:     id,
		Provider:   payload.Provider,
		KeyVersion: payload.KeyVersion,
		Ciphertext: []byte(ciphertext),
	}

	// Persist updated secret
	if err := s.storage.UpdateSecret(secret); err != nil {
		s.logger.Error("Failed to update secret",
			zap.String("secret_id", id),
			zap.String("correlation_id", correlationID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("storage update failed: %w", err)
	}

	// Retrieve updated secret with timestamps
	updatedSecret, err := s.storage.GetSecret(id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated secret: %w", err)
	}

	s.logger.Info("Secret updated successfully",
		zap.String("secret_handle", id),
		zap.String("correlation_id", correlationID),
		zap.String("provider", payload.Provider),
		zap.String("key_version", payload.KeyVersion),
	)

	// Return secret with plaintext value
	updatedSecret.Value = value
	return updatedSecret, nil
}

// Delete permanently removes a secret
func (s *SecretService) Delete(id string, correlationID string) error {
	s.logger.Info("Deleting secret",
		zap.String("secret_handle", id),
		zap.String("correlation_id", correlationID),
	)

	if err := s.storage.DeleteSecret(id); err != nil {
		// Don't log details for not found errors
		if _, ok := err.(*storage.SecretNotFoundError); ok {
			s.logger.Debug("Secret not found for deletion",
				zap.String("secret_handle", id),
				zap.String("correlation_id", correlationID),
			)
		} else {
			s.logger.Error("Failed to delete secret",
				zap.String("secret_handle", id),
				zap.String("correlation_id", correlationID),
				zap.Error(err),
			)
		}
		return err
	}

	s.logger.Info("Secret deleted successfully",
		zap.String("secret_handle", id),
		zap.String("correlation_id", correlationID),
	)

	return nil
}

// generateCorrelationID generates a unique correlation ID for request tracking
func generateCorrelationID() string {
	return "req-" + uuid.New().String()[:8]
}

// generateUUID generates a new UUID string
func generateUUID() string {
	return uuid.New().String()
}
