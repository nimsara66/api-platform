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
	"encoding/json"
	"fmt"
	"regexp"

	vault "github.com/hashicorp/vault/api"
	api "github.com/wso2/api-platform/gateway/gateway-controller/pkg/api/generated"
	"gopkg.in/yaml.v3"
)

// Parser handles parsing of API configuration files and secret resolution
type Parser struct {
	vaultAddress string
	vaultToken   string
}

// NewParser creates a new configuration parser
func NewParser() *Parser {
	return &Parser{
		vaultAddress: "http://localhost:8200",
		vaultToken:   "my-vault-token",
	}
}

func (p *Parser) resolveVaultSecrets(data []byte) ([]byte, error) {
	pattern := regexp.MustCompile(`hashicorp:vault-lookup\('([^']+)',\s*'([^']+)',\s*'([^']+)'\)`)

	// Initialize Vault client
	config := vault.DefaultConfig()
	config.Address = p.vaultAddress

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}
	client.SetToken(p.vaultToken)

	// Find and replace all secret references
	resolved := pattern.ReplaceAllFunc(data, func(match []byte) []byte {
		matches := pattern.FindStringSubmatch(string(match))
		if len(matches) != 4 {
			return match
		}

		namespace := matches[1]
		path := matches[2]
		field := matches[3]

		// Read secret from Vault
		secret, err := client.Logical().Read(fmt.Sprintf("%s/%s", namespace, path))
		if err != nil {
			return match // Return original if failed to fetch
		}

		if secret == nil || secret.Data == nil {
			return match
		}

		// Try to get the field value
		if value, ok := secret.Data[field].(string); ok {
			return []byte(value)
		}

		return match
	})

	return resolved, nil
}

// ParseYAML parses YAML content into an API configuration
func (p *Parser) ParseYAML(data []byte) (*api.APIConfiguration, error) {
	// Resolve any Vault secrets first
	resolvedData, err := p.resolveVaultSecrets(data)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve secrets: %w", err)
	}

	var config api.APIConfiguration
	if err := yaml.Unmarshal(resolvedData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

// ParseJSON parses JSON content into an API configuration
func (p *Parser) ParseJSON(data []byte) (*api.APIConfiguration, error) {
	// Resolve any Vault secrets first
	resolvedData, err := p.resolveVaultSecrets(data)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve secrets: %w", err)
	}

	var config api.APIConfiguration
	if err := json.Unmarshal(resolvedData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &config, nil
}

// Parse attempts to parse data as either YAML or JSON
func (p *Parser) Parse(data []byte, contentType string) (*api.APIConfiguration, error) {
	switch contentType {
	case "application/yaml", "application/x-yaml", "text/yaml":
		return p.ParseYAML(data)
	case "application/json":
		return p.ParseJSON(data)
	default:
		// Try YAML first, then JSON
		config, err := p.ParseYAML(data)
		if err == nil {
			return config, nil
		}

		config, err = p.ParseJSON(data)
		if err == nil {
			return config, nil
		}

		return nil, fmt.Errorf("failed to parse as YAML or JSON")
	}
}
