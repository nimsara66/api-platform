package encryption

import "fmt"

// ErrProviderNotFound indicates no provider can decrypt the payload
type ErrProviderNotFound struct {
	ProviderName string
}

func (e *ErrProviderNotFound) Error() string {
	return fmt.Sprintf("no encryption provider found for: %s", e.ProviderName)
}

// ErrEncryptionFailed indicates encryption operation failed
type ErrEncryptionFailed struct {
	ProviderName string
	Cause        error
}

func (e *ErrEncryptionFailed) Error() string {
	return fmt.Sprintf("encryption failed for provider %s: %v", e.ProviderName, e.Cause)
}

func (e *ErrEncryptionFailed) Unwrap() error {
	return e.Cause
}

// ErrDecryptionFailed indicates decryption operation failed
type ErrDecryptionFailed struct {
	ProviderName string
	Cause        error
}

func (e *ErrDecryptionFailed) Error() string {
	return fmt.Sprintf("decryption failed for provider %s: %v", e.ProviderName, e.Cause)
}

func (e *ErrDecryptionFailed) Unwrap() error {
	return e.Cause
}

// ErrInvalidKeySize indicates encryption key has wrong size
type ErrInvalidKeySize struct {
	Expected int
	Actual   int
}

func (e *ErrInvalidKeySize) Error() string {
	return fmt.Sprintf("invalid key size: expected %d bytes, got %d bytes", e.Expected, e.Actual)
}

// ErrKeyNotFound indicates encryption key file not found
type ErrKeyNotFound struct {
	KeyPath string
}

func (e *ErrKeyNotFound) Error() string {
	return fmt.Sprintf("encryption key not found: %s", e.KeyPath)
}
