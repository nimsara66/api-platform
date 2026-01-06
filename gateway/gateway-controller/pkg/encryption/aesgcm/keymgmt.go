package aesgcm

import (
	"fmt"
	"os"

	"github.com/wso2/api-platform/gateway/gateway-controller/pkg/encryption"
	"go.uber.org/zap"
)

const (
	// AESKeySize is the required key size for AES-256 (32 bytes)
	AESKeySize = 32
)

// Key represents a single encryption key with its version
type Key struct {
	Version string
	Data    []byte
}

// KeyManager manages loading and accessing encryption keys
type KeyManager struct {
	keys          map[string]*Key // version -> key mapping
	primaryKey    *Key            // primary key for encryption (first key in config)
	primaryVersion string
	logger        *zap.Logger
}

// NewKeyManager creates a new key manager and loads keys from files
func NewKeyManager(keyConfigs []KeyConfig, logger *zap.Logger) (*KeyManager, error) {
	if len(keyConfigs) == 0 {
		return nil, fmt.Errorf("at least one encryption key is required")
	}

	km := &KeyManager{
		keys:   make(map[string]*Key),
		logger: logger,
	}

	// Load all keys
	for i, config := range keyConfigs {
		key, err := km.loadKey(config)
		if err != nil {
			return nil, fmt.Errorf("failed to load key %s: %w", config.Version, err)
		}

		km.keys[config.Version] = key

		// First key is the primary key for encryption
		if i == 0 {
			km.primaryKey = key
			km.primaryVersion = config.Version
		}

		logger.Debug("Loaded encryption key",
			zap.String("version", config.Version),
			zap.Bool("is_primary", i == 0),
		)
	}

	logger.Info("Key manager initialized",
		zap.Int("total_keys", len(km.keys)),
		zap.String("primary_version", km.primaryVersion),
	)

	return km, nil
}

// loadKey reads a key from a file and validates its size
func (km *KeyManager) loadKey(config KeyConfig) (*Key, error) {
	// Check file permissions for security
	info, err := os.Stat(config.FilePath)
	if err != nil {
		return nil, &encryption.ErrKeyNotFound{KeyPath: config.FilePath}
	}

	// Warn if key file is world-readable (security risk)
	perm := info.Mode().Perm()
	if perm&0004 != 0 {
		km.logger.Warn("Encryption key file is world-readable - consider restricting permissions",
			zap.String("key_version", config.Version),
			zap.String("file_path", config.FilePath),
			zap.String("permissions", perm.String()),
		)
	}

	// Read key data
	data, err := os.ReadFile(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Validate key size (must be exactly 32 bytes for AES-256)
	if len(data) != AESKeySize {
		return nil, &encryption.ErrInvalidKeySize{
			Expected: AESKeySize,
			Actual:   len(data),
		}
	}

	return &Key{
		Version: config.Version,
		Data:    data,
	}, nil
}

// GetPrimaryKey returns the primary encryption key
func (km *KeyManager) GetPrimaryKey() *Key {
	return km.primaryKey
}

// GetKey returns a specific key by version
func (km *KeyManager) GetKey(version string) (*Key, error) {
	key, exists := km.keys[version]
	if !exists {
		return nil, fmt.Errorf("key version not found: %s", version)
	}
	return key, nil
}

// GetPrimaryVersion returns the primary key version
func (km *KeyManager) GetPrimaryVersion() string {
	return km.primaryVersion
}

// KeyConfig holds configuration for a single key
type KeyConfig struct {
	Version  string
	FilePath string
}
