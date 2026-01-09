package encryption

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/reflectwalk"
	"go.uber.org/zap"
)

// SecretResolver walks through a struct and resolves SecretValue fields
type SecretResolver struct {
	providerManager *ProviderManager
	errors          []error
	currentPath     []string
}

// NewSecretResolver creates a new SecretResolver
func NewSecretResolver(m *ProviderManager) *SecretResolver {
	return &SecretResolver{
		providerManager: m,
		errors:          []error{},
		currentPath:     []string{},
	}
}

// Struct is called when entering a struct
func (s *SecretResolver) Struct(v reflect.Value) error {
	return nil
}

// StructField is called for each field in a struct
func (s *SecretResolver) StructField(field reflect.StructField, v reflect.Value) error {
	s.currentPath = append(s.currentPath, field.Name)
	defer func() {
		if len(s.currentPath) > 0 {
			s.currentPath = s.currentPath[:len(s.currentPath)-1]
		}
	}()

	// Only process fields with secret:"true" tag
	if field.Tag.Get("secret") == "true" {
		s.providerManager.logger.Debug("Found secret field",
			zap.String("field", field.Name),
			zap.String("path", strings.Join(s.currentPath, ".")),
			zap.String("kind", v.Kind().String()),
		)
		return s.processSecretField(field, v)
	}

	return nil
}

// processSecretField handles the decryption of a secret field
func (s *SecretResolver) processSecretField(field reflect.StructField, v reflect.Value) error {
	// Dereference pointers
	val := v
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil // nothing to do for nil pointer
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.String {
		s.providerManager.logger.Debug("Skipping non-string secret field",
			zap.String("field", field.Name),
			zap.String("path", strings.Join(s.currentPath, ".")),
			zap.String("kind", val.Kind().String()),
		)
		return nil
	}

	str := val.String()

	// Only process non-empty secret values
	if str == "" {
		s.providerManager.logger.Debug("Skipping empty secret field",
			zap.String("field", field.Name),
			zap.String("path", strings.Join(s.currentPath, ".")),
		)

		return nil
	}

	decrypted, err := s.resolveSecret(str)
	if err != nil {
		errMsg := fmt.Errorf("failed to resolve secret in field %s (path: %s): %w",
			field.Name, strings.Join(s.currentPath, "."), err)
		s.errors = append(s.errors, errMsg)
		s.providerManager.logger.Error("Secret resolution failed",
			zap.String("field", field.Name),
			zap.String("path", strings.Join(s.currentPath, ".")),
			zap.Error(err),
		)
		return nil // Continue walking, don't stop on error
	}

	// Set the decrypted value back
	if !val.CanSet() {
		s.providerManager.logger.Debug("Cannot set value for field",
			zap.String("field", field.Name),
			zap.String("path", strings.Join(s.currentPath, ".")),
		)
		return nil
	}

	val.SetString(decrypted)

	s.providerManager.logger.Debug("Secret resolved successfully",
		zap.String("field", field.Name),
		zap.String("path", strings.Join(s.currentPath, ".")),
	)

	return nil
}

// resolveSecret decrypts a secret value
func (s *SecretResolver) resolveSecret(secretValue string) (string, error) {
	secret, err := s.providerManager.storage.GetSecret(secretValue)
	if err != nil {
		return "", fmt.Errorf("failed to find secret value: %w", err)
	}
	payload, err := UnmarshalPayload(string(secret.Ciphertext))

	// Decrypt using provider manager
	decrypted, err := s.providerManager.Decrypt(payload)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return fmt.Sprintf("%s:%s", secretValue, decrypted), nil
}

// Slice is called when entering a slice
func (s *SecretResolver) Slice(v reflect.Value) error {
	return nil
}

// SliceElem is called for each slice element
func (s *SecretResolver) SliceElem(i int, v reflect.Value) error {
	s.currentPath = append(s.currentPath, fmt.Sprintf("[%d]", i))
	return nil
}

// Map is called when entering a map
func (s *SecretResolver) Map(m reflect.Value) error {
	return nil
}

// MapElem is called for each map element
func (s *SecretResolver) MapElem(m, k, v reflect.Value) error {
	keyStr := fmt.Sprintf("[%v]", k.Interface())
	s.currentPath = append(s.currentPath, keyStr)
	return nil
}

// Primitive is called for primitive values
func (s *SecretResolver) Primitive(v reflect.Value) error {
	return nil
}

// Exit is called when exiting a location
func (s *SecretResolver) Exit(loc reflectwalk.Location) error {
	switch loc {
	case reflectwalk.SliceElem, reflectwalk.MapValue:
		if len(s.currentPath) > 0 {
			s.currentPath = s.currentPath[:len(s.currentPath)-1]
		}
	default:
		return nil
	}
	return nil
}

// PointerEnter is called when entering a pointer
func (s *SecretResolver) PointerEnter(v bool) error {
	return nil
}

// PointerExit is called when exiting a pointer
func (s *SecretResolver) PointerExit(v bool) error {
	return nil
}
