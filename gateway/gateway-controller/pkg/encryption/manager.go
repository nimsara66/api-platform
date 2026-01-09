package encryption

import (
	"fmt"

	"github.com/mitchellh/reflectwalk"
	api "github.com/wso2/api-platform/gateway/gateway-controller/pkg/api/generated"
	"github.com/wso2/api-platform/gateway/gateway-controller/pkg/storage"
	"go.uber.org/zap"
)

// ProviderManager orchestrates the encryption provider chain
type ProviderManager struct {
	providers []EncryptionProvider
	storage   storage.Storage
	logger    *zap.Logger
}

// NewProviderManager creates a new provider manager with the given providers
func NewProviderManager(providers []EncryptionProvider, storage storage.Storage, logger *zap.Logger) (*ProviderManager, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("at least one encryption provider is required")
	}

	// Validate all providers
	for _, provider := range providers {
		if err := provider.HealthCheck(); err != nil {
			return nil, fmt.Errorf("provider %s failed health check: %w", provider.Name(), err)
		}
	}

	logger.Info("Initialized encryption provider chain",
		zap.Int("provider_count", len(providers)),
		zap.String("primary_provider", providers[0].Name()),
	)

	return &ProviderManager{
		providers: providers,
		storage:   storage,
		logger:    logger,
	}, nil
}

// Encrypt encrypts plaintext using the primary provider (first in chain)
func (m *ProviderManager) Encrypt(plaintext []byte) (*EncryptedPayload, error) {
	primaryProvider := m.providers[0]

	m.logger.Debug("Encrypting with primary provider",
		zap.String("provider", primaryProvider.Name()),
		zap.Int("plaintext_size", len(plaintext)),
	)

	payload, err := primaryProvider.Encrypt(plaintext)
	if err != nil {
		m.logger.Error("Encryption failed",
			zap.String("provider", primaryProvider.Name()),
			zap.Error(err),
		)
		return nil, &ErrEncryptionFailed{
			ProviderName: primaryProvider.Name(),
			Cause:        err,
		}
	}

	m.logger.Debug("Encryption successful",
		zap.String("provider", payload.Provider),
		zap.String("key_version", payload.KeyVersion),
	)

	return payload, nil
}

// Decrypt decrypts the payload using the provider chain
// It tries to match the provider by name from the payload metadata
func (m *ProviderManager) Decrypt(payload *EncryptedPayload) ([]byte, error) {
	m.logger.Debug("Decrypting payload",
		zap.String("provider", payload.Provider),
		zap.String("key_version", payload.KeyVersion),
	)

	// Find the provider that can decrypt this payload
	for _, provider := range m.providers {
		if provider.Name() == payload.Provider {
			plaintext, err := provider.Decrypt(payload)
			if err != nil {
				m.logger.Error("Decryption failed",
					zap.String("provider", provider.Name()),
					zap.String("key_version", payload.KeyVersion),
					zap.Error(err),
				)
				return nil, &ErrDecryptionFailed{
					ProviderName: provider.Name(),
					Cause:        err,
				}
			}

			m.logger.Debug("Decryption successful",
				zap.String("provider", provider.Name()),
				zap.Int("plaintext_size", len(plaintext)),
			)

			return plaintext, nil
		}
	}

	// No provider found that can decrypt this payload
	m.logger.Error("No provider found for decryption",
		zap.String("requested_provider", payload.Provider),
		zap.String("key_version", payload.KeyVersion),
	)

	return nil, &ErrProviderNotFound{
		ProviderName: payload.Provider,
	}
}

// HealthCheck validates all providers in the chain
func (m *ProviderManager) HealthCheck() error {
	for _, provider := range m.providers {
		if err := provider.HealthCheck(); err != nil {
			return fmt.Errorf("provider %s health check failed: %w", provider.Name(), err)
		}
	}
	return nil
}

// GetPrimaryProvider returns the primary encryption provider (first in chain)
func (m *ProviderManager) GetPrimaryProvider() EncryptionProvider {
	return m.providers[0]
}

// GetProviders returns all configured providers
func (m *ProviderManager) GetProviders() []EncryptionProvider {
	return m.providers
}

func (m *ProviderManager) ResolveSecrets(cfg *api.LLMProviderConfiguration) error {
	resolver := NewSecretResolver(m)

	err := reflectwalk.Walk(cfg, resolver)
	if err != nil {
		return fmt.Errorf("failed to walk config: %w", err)
	}

	// Log secret resolution errors but do NOT fail
	if len(resolver.errors) > 0 {
		for _, err := range resolver.errors {
			m.logger.Error("Secret resolution error", zap.Error(err))
		}

		m.logger.Warn("Completed secret resolution with errors",
			zap.Int("error_count", len(resolver.errors)),
		)
	} else {
		m.logger.Debug("Secret resolution completed successfully")
	}

	return nil
}
