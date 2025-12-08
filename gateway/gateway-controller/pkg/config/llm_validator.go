/*
 * Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package config

import (
	"fmt"
	"regexp"

	api "github.com/wso2/api-platform/gateway/gateway-controller/pkg/api/generated"
)

// LLMValidator validates LLM-related configurations (provider templates, providers, proxies)
// It uses type switching to handle different LLM configuration types
type LLMValidator struct {
	// nameRegex matches valid template/provider/proxy names
	nameRegex *regexp.Regexp
}

// NewLLMValidator creates a new LLM configuration validator
func NewLLMValidator() *LLMValidator {
	return &LLMValidator{
		nameRegex: regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`),
	}
}

// Validate performs comprehensive validation on a configuration
// It uses type switching to handle different LLM configuration types:
// - LLMProviderTemplate (for /llm-providers/templates)
// - LLMProviderConfiguration (for /llm-providers)
// - LLMProxy (for /llm-proxies) - future implementation
func (v *LLMValidator) Validate(config interface{}) []ValidationError {
	// Type switch to handle different LLM configuration types
	switch cfg := config.(type) {
	case *api.LLMProviderTemplate:
		return v.validateLLMProviderTemplate(cfg)
	case api.LLMProviderTemplate:
		return v.validateLLMProviderTemplate(&cfg)
	case *api.LLMProviderConfiguration:
		return v.validateLLMProvider(cfg)
	case api.LLMProviderConfiguration:
		return v.validateLLMProvider(&cfg)
	// Future: Add cases for LLMProxy
	// case *api.LLMProxy:
	//     return v.validateLLMProxy(cfg)
	default:
		return []ValidationError{
			{
				Field:   "config",
				Message: "Unsupported configuration type for LLMValidator",
			},
		}
	}
}

// validateLLMProviderTemplate validates an LLM provider template configuration
func (v *LLMValidator) validateLLMProviderTemplate(template *api.LLMProviderTemplate) []ValidationError {
	var errors []ValidationError

	// Validate version
	if template.Version == "" {
		errors = append(errors, ValidationError{
			Field:   "version",
			Message: "Version is required",
		})
	} else if template.Version != "api-platform.wso2.com/v1" {
		errors = append(errors, ValidationError{
			Field:   "version",
			Message: "Version must be 'api-platform.wso2.com/v1'",
		})
	}

	// Validate kind
	if template.Kind == "" {
		errors = append(errors, ValidationError{
			Field:   "kind",
			Message: "Kind is required",
		})
	} else if template.Kind != "llm/provider-template" {
		errors = append(errors, ValidationError{
			Field:   "kind",
			Message: "Kind must be 'llm/provider-template'",
		})
	}

	// Validate data section
	errors = append(errors, v.validateTemplateData(&template.Data)...)

	return errors
}

// validateTemplateData validates the data section of an LLM provider template
func (v *LLMValidator) validateTemplateData(data *api.LLMProviderTemplateData) []ValidationError {
	var errors []ValidationError

	// Validate name
	if data.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "data.name",
			Message: "Template name is required",
		})
	} else if !v.nameRegex.MatchString(data.Name) {
		errors = append(errors, ValidationError{
			Field:   "data.name",
			Message: "Template name must contain only alphanumeric characters, underscores, dots, and hyphens",
		})
	}

	// Validate OpenAPI specification
	if data.Openapi == "" {
		errors = append(errors, ValidationError{
			Field:   "data.openapi",
			Message: "OpenAPI specification is required",
		})
	}

	// Validate token identifiers if present
	if data.PromptTokens != nil {
		errors = append(errors, v.validateTokenIdentifier("data.promptTokens", data.PromptTokens)...)
	}

	if data.CompletionTokens != nil {
		errors = append(errors, v.validateTokenIdentifier("data.completionTokens", data.CompletionTokens)...)
	}

	if data.TotalTokens != nil {
		errors = append(errors, v.validateTokenIdentifier("data.totalTokens", data.TotalTokens)...)
	}

	if data.RequestModel != nil {
		errors = append(errors, v.validateTokenIdentifier("data.requestModel", data.RequestModel)...)
	}

	return errors
}

// validateTokenIdentifier validates a token identifier configuration
func (v *LLMValidator) validateTokenIdentifier(fieldPrefix string, identifier *api.TokenIdentifier) []ValidationError {
	var errors []ValidationError

	if identifier.Location == "" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.location", fieldPrefix),
			Message: "Location is required",
		})
	} else if identifier.Location != "payload" && identifier.Location != "header" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.location", fieldPrefix),
			Message: "Location must be either 'payload' or 'header'",
		})
	}

	if identifier.Identifier == "" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.identifier", fieldPrefix),
			Message: "Identifier is required",
		})
	}

	return errors
}

// validateLLMProvider validates an LLM provider configuration
func (v *LLMValidator) validateLLMProvider(provider *api.LLMProviderConfiguration) []ValidationError {
	var errors []ValidationError

	// Validate version
	if provider.Version == "" {
		errors = append(errors, ValidationError{
			Field:   "version",
			Message: "Version is required",
		})
	} else if provider.Version != "api-platform.wso2.com/v1" {
		errors = append(errors, ValidationError{
			Field:   "version",
			Message: "Version must be 'api-platform.wso2.com/v1'",
		})
	}

	// Validate kind
	if provider.Kind == "" {
		errors = append(errors, ValidationError{
			Field:   "kind",
			Message: "Kind is required",
		})
	} else if provider.Kind != "llm/provider" {
		errors = append(errors, ValidationError{
			Field:   "kind",
			Message: "Kind must be 'llm/provider'",
		})
	}

	// Validate spec section
	if provider.Spec == nil {
		errors = append(errors, ValidationError{
			Field:   "spec",
			Message: "Spec (data) is required",
		})
	} else {
		errors = append(errors, v.validateProviderData(provider.Spec)...)
	}

	return errors
}

// validateProviderData validates the data section of an LLM provider
func (v *LLMValidator) validateProviderData(data *api.LLMProviderSpec) []ValidationError {
	var errors []ValidationError

	// Validate name
	if data.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "data.name",
			Message: "Provider name is required",
		})
	} else if !v.nameRegex.MatchString(data.Name) {
		errors = append(errors, ValidationError{
			Field:   "data.name",
			Message: "Provider name must contain only alphanumeric characters, underscores, dots, and hyphens",
		})
	}

	// Validate version
	if data.Version == "" {
		errors = append(errors, ValidationError{
			Field:   "data.version",
			Message: "Provider version is required",
		})
	}

	// Validate template reference
	if data.Template == "" {
		errors = append(errors, ValidationError{
			Field:   "data.template",
			Message: "Template reference is required",
		})
	}

	// Validate upstreams
	if len(data.Upstreams) == 0 {
		errors = append(errors, ValidationError{
			Field:   "data.upstreams",
			Message: "At least one upstream is required",
		})
	} else {
		for i, upstream := range data.Upstreams {
			errors = append(errors, v.validateUpstream(fmt.Sprintf("data.upstreams[%d]", i), &upstream)...)
		}
	}

	// Validate access control if present
	if data.AccessControl != nil {
		errors = append(errors, v.validateAccessControl("data.accessControl", data.AccessControl)...)
	}

	return errors
}

// validateUpstream validates an upstream configuration
func (v *LLMValidator) validateUpstream(fieldPrefix string, upstream *api.LLMUpstream) []ValidationError {
	var errors []ValidationError

	if upstream.Url == "" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.url", fieldPrefix),
			Message: "Upstream URL is required",
		})
	} else {
		// Basic URL validation
		if !regexp.MustCompile(`^https?://`).MatchString(upstream.Url) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.url", fieldPrefix),
				Message: "Upstream URL must start with http:// or https://",
			})
		}
	}

	// Validate auth if present
	if upstream.Auth != nil {
		errors = append(errors, v.validateAuth(fmt.Sprintf("%s.auth", fieldPrefix), upstream.Auth)...)
	}

	return errors
}

// validateAuth validates authentication configuration
func (v *LLMValidator) validateAuth(fieldPrefix string, auth *api.LLMAuth) []ValidationError {
	var errors []ValidationError

	if auth.Type == "" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.type", fieldPrefix),
			Message: "Auth type is required",
		})
	} else if auth.Type != "api-key" && auth.Type != "bearer" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.type", fieldPrefix),
			Message: "Auth type must be either 'api-key' or 'bearer'",
		})
	}

	if auth.Header == "" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.header", fieldPrefix),
			Message: "Auth header is required",
		})
	}

	if auth.Value == "" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.value", fieldPrefix),
			Message: "Auth value is required",
		})
	}

	return errors
}

// validateAccessControl validates access control configuration
func (v *LLMValidator) validateAccessControl(fieldPrefix string, ac *api.LLMAccessControl) []ValidationError {
	var errors []ValidationError

	if ac.Mode != nil {
		if *ac.Mode != "allow_all" && *ac.Mode != "deny_all" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.mode", fieldPrefix),
				Message: "Access control mode must be either 'allow_all' or 'deny_all'",
			})
		}
	}

	// Validate exceptions if present
	if ac.Exceptions != nil {
		for i, exception := range *ac.Exceptions {
			if exception.Path == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s.exceptions[%d].path", fieldPrefix, i),
					Message: "Exception path is required",
				})
			}

			if len(exception.Methods) == 0 {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s.exceptions[%d].methods", fieldPrefix, i),
					Message: "At least one method is required for exception",
				})
			}
		}
	}

	return errors
}

// Future: Add validation methods for other LLM entities
//
// validateLLMProxy validates an LLM proxy configuration
// func (v *LLMValidator) validateLLMProxy(proxy *api.LLMProxy) []ValidationError {
//     var errors []ValidationError
//     // Validate proxy-specific fields
//     return errors
// }
