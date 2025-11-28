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

package storage

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	api "github.com/wso2/api-platform/gateway/gateway-controller/pkg/api/generated"
	"github.com/wso2/api-platform/gateway/gateway-controller/pkg/models"
	"go.uber.org/zap"
)

//go:embed gateway-controller-db.sql
var schemaSQL string

// SQLiteStorage implements the Storage interface using SQLite
type SQLiteStorage struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string, logger *zap.Logger) (*SQLiteStorage, error) {
	// Build connection string with SQLite pragmas for optimal performance
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=2000&_foreign_keys=ON", dbPath)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// CRITICAL: Prevents "database is locked" errors with concurrent access
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	storage := &SQLiteStorage{
		db:     db,
		logger: logger,
	}

	// Initialize schema if needed
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	logger.Info("SQLite storage initialized",
		zap.String("database_path", dbPath),
		zap.String("journal_mode", "WAL"))

	return storage, nil
}

// initSchema creates the database schema if it doesn't exist
func (s *SQLiteStorage) initSchema() error {
	// Check schema version
	var version int
	err := s.db.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		return fmt.Errorf("failed to query schema version: %w", err)
	}

	if version == 0 {
		s.logger.Info("Initializing database schema (version 1)")

		// Execute schema creation SQL
		if _, err := s.db.Exec(schemaSQL); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}

		s.logger.Info("Database schema initialized successfully")
	} else {
		// Migrations
		if version == 1 {
			// Add policy_definitions table (idempotent due to IF NOT EXISTS in embedded schema)
			if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS policy_definitions (
				name TEXT NOT NULL,
				version TEXT NOT NULL,
				provider TEXT NOT NULL,
				description TEXT,
				flows_request_require_header INTEGER,
				flows_request_require_body INTEGER,
				flows_response_require_header INTEGER,
				flows_response_require_body INTEGER,
				parameters_schema TEXT,
				PRIMARY KEY (name, version)
			);`); err != nil {
				return fmt.Errorf("failed to migrate schema to version 2: %w", err)
			}
			if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_policy_provider ON policy_definitions(provider);`); err != nil {
				return fmt.Errorf("failed to create policy_definitions index: %w", err)
			}
			if _, err := s.db.Exec("PRAGMA user_version = 2"); err != nil {
				return fmt.Errorf("failed to set schema version to 2: %w", err)
			}
			s.logger.Info("Schema migrated to version 2 (policy_definitions)")
			version = 2
		}

		if version == 2 {
			// Add llm_provider_templates table
			if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS llm_provider_templates (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL UNIQUE,
				configuration TEXT NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);`); err != nil {
				return fmt.Errorf("failed to migrate schema to version 3: %w", err)
			}
			if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_template_name ON llm_provider_templates(name);`); err != nil {
				return fmt.Errorf("failed to create llm_provider_templates index: %w", err)
			}
			if _, err := s.db.Exec("PRAGMA user_version = 3"); err != nil {
				return fmt.Errorf("failed to set schema version to 3: %w", err)
			}
			s.logger.Info("Schema migrated to version 3 (llm_provider_templates)")
			version = 3
		}

		if version == 3 {
			// Add original_configuration column to api_configs table
			if _, err := s.db.Exec(`ALTER TABLE api_configs ADD COLUMN original_configuration TEXT;`); err != nil {
				return fmt.Errorf("failed to migrate schema to version 4: %w", err)
			}
			if _, err := s.db.Exec("PRAGMA user_version = 4"); err != nil {
				return fmt.Errorf("failed to set schema version to 4: %w", err)
			}
			s.logger.Info("Schema migrated to version 4 (original_configuration)")
			version = 4
		}

		s.logger.Info("Database schema already exists", zap.Int("version", version))
	}

	return nil
}

// SaveConfig persists a new API configuration
func (s *SQLiteStorage) SaveConfig(cfg *models.StoredAPIConfig) error {
	// Serialize configuration to JSON
	configJSON, err := json.Marshal(cfg.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Serialize original configuration if present
	var originalConfigJSON *string
	if cfg.OriginalConfiguration != nil {
		originalJSON, err := json.Marshal(cfg.OriginalConfiguration)
		if err != nil {
			return fmt.Errorf("failed to marshal original configuration: %w", err)
		}
		originalStr := string(originalJSON)
		originalConfigJSON = &originalStr
	}

	// Extract fields for indexed columns
	name := cfg.GetAPIName()
	version := cfg.GetAPIVersion()
	context := cfg.Configuration.Data.Context
	kind := string(cfg.Configuration.Kind)

	query := `
		INSERT INTO api_configs (
			id, name, version, context, kind, configuration,
			status, created_at, updated_at, deployed_version, original_configuration
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err = s.db.Exec(query,
		cfg.ID,
		name,
		version,
		context,
		kind,
		string(configJSON),
		cfg.Status,
		now,
		now,
		cfg.DeployedVersion,
		originalConfigJSON,
	)

	if err != nil {
		// Check for unique constraint violation
		if isUniqueConstraintError(err) {
			return fmt.Errorf("%w: configuration with name '%s' and version '%s' already exists", ErrConflict, name, version)
		}
		return fmt.Errorf("failed to insert configuration: %w", err)
	}

	s.logger.Info("Configuration saved",
		zap.String("id", cfg.ID),
		zap.String("name", name),
		zap.String("version", version))

	return nil
}

// UpdateConfig updates an existing API configuration
func (s *SQLiteStorage) UpdateConfig(cfg *models.StoredAPIConfig) error {
	// Check if configuration exists
	_, err := s.GetConfig(cfg.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("cannot update non-existent configuration: %w", err)
		}
		return err
	}

	// Serialize configuration to JSON
	configJSON, err := json.Marshal(cfg.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Serialize original configuration if present
	var originalConfigJSON *string
	if cfg.OriginalConfiguration != nil {
		originalJSON, err := json.Marshal(cfg.OriginalConfiguration)
		if err != nil {
			return fmt.Errorf("failed to marshal original configuration: %w", err)
		}
		originalStr := string(originalJSON)
		originalConfigJSON = &originalStr
	}

	// Extract fields for indexed columns
	name := cfg.GetAPIName()
	version := cfg.GetAPIVersion()
	context := cfg.Configuration.Data.Context
	kind := string(cfg.Configuration.Kind)

	query := `
		UPDATE api_configs
		SET name = ?, version = ?, context = ?, kind = ?,
		    configuration = ?, status = ?, updated_at = ?,
		    deployed_version = ?, original_configuration = ?
		WHERE id = ?
	`

	result, err := s.db.Exec(query,
		name,
		version,
		context,
		kind,
		string(configJSON),
		cfg.Status,
		time.Now(),
		cfg.DeployedVersion,
		originalConfigJSON,
		cfg.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("%w: id=%s", ErrNotFound, cfg.ID)
	}

	s.logger.Info("Configuration updated",
		zap.String("id", cfg.ID),
		zap.String("name", name),
		zap.String("version", version))

	return nil
}

// DeleteConfig removes an API configuration by ID
func (s *SQLiteStorage) DeleteConfig(id string) error {
	query := `DELETE FROM api_configs WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete configuration: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("%w: id=%s", ErrNotFound, id)
	}

	s.logger.Info("Configuration deleted", zap.String("id", id))

	return nil
}

// GetConfig retrieves an API configuration by ID
func (s *SQLiteStorage) GetConfig(id string) (*models.StoredAPIConfig, error) {
	query := `
		SELECT id, configuration, status, created_at, updated_at,
		       deployed_at, deployed_version, original_configuration
		FROM api_configs
		WHERE id = ?
	`

	var cfg models.StoredAPIConfig
	var configJSON string
	var deployedAt sql.NullTime
	var originalConfigJSON sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&cfg.ID,
		&configJSON,
		&cfg.Status,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
		&deployedAt,
		&cfg.DeployedVersion,
		&originalConfigJSON,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id=%s", ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to query configuration: %w", err)
	}

	// Parse deployed_at (nullable field)
	if deployedAt.Valid {
		cfg.DeployedAt = &deployedAt.Time
	}

	// Deserialize JSON configuration
	if err := json.Unmarshal([]byte(configJSON), &cfg.Configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Deserialize original configuration if present
	if originalConfigJSON.Valid && originalConfigJSON.String != "" {
		var originalConfig map[string]interface{}
		if err := json.Unmarshal([]byte(originalConfigJSON.String), &originalConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal original configuration: %w", err)
		}
		cfg.OriginalConfiguration = originalConfig
	}

	return &cfg, nil
}

// GetConfigByNameVersion retrieves an API configuration by name and version
func (s *SQLiteStorage) GetConfigByNameVersion(name, version string) (*models.StoredAPIConfig, error) {
	query := `
		SELECT id, configuration, status, created_at, updated_at,
		       deployed_at, deployed_version, original_configuration
		FROM api_configs
		WHERE name = ? AND version = ?
	`

	var cfg models.StoredAPIConfig
	var configJSON string
	var deployedAt sql.NullTime
	var originalConfigJSON sql.NullString

	err := s.db.QueryRow(query, name, version).Scan(
		&cfg.ID,
		&configJSON,
		&cfg.Status,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
		&deployedAt,
		&cfg.DeployedVersion,
		&originalConfigJSON,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: name=%s, version=%s", ErrNotFound, name, version)
		}
		return nil, fmt.Errorf("failed to query configuration: %w", err)
	}

	// Parse deployed_at (nullable field)
	if deployedAt.Valid {
		cfg.DeployedAt = &deployedAt.Time
	}

	// Deserialize JSON configuration
	if err := json.Unmarshal([]byte(configJSON), &cfg.Configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Deserialize original configuration if present
	if originalConfigJSON.Valid && originalConfigJSON.String != "" {
		var originalConfig map[string]interface{}
		if err := json.Unmarshal([]byte(originalConfigJSON.String), &originalConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal original configuration: %w", err)
		}
		cfg.OriginalConfiguration = originalConfig
	}

	return &cfg, nil
}

// GetAllConfigs retrieves all API configurations
func (s *SQLiteStorage) GetAllConfigs() ([]*models.StoredAPIConfig, error) {
	query := `
		SELECT id, configuration, status, created_at, updated_at,
		       deployed_at, deployed_version, original_configuration
		FROM api_configs
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query configurations: %w", err)
	}
	defer rows.Close()

	var configs []*models.StoredAPIConfig

	for rows.Next() {
		var cfg models.StoredAPIConfig
		var configJSON string
		var deployedAt sql.NullTime
		var originalConfigJSON sql.NullString

		err := rows.Scan(
			&cfg.ID,
			&configJSON,
			&cfg.Status,
			&cfg.CreatedAt,
			&cfg.UpdatedAt,
			&deployedAt,
			&cfg.DeployedVersion,
			&originalConfigJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Parse deployed_at (nullable field)
		if deployedAt.Valid {
			cfg.DeployedAt = &deployedAt.Time
		}

		// Deserialize JSON configuration
		if err := json.Unmarshal([]byte(configJSON), &cfg.Configuration); err != nil {
			return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
		}

		// Deserialize original configuration if present
		if originalConfigJSON.Valid && originalConfigJSON.String != "" {
			var originalConfig map[string]interface{}
			if err := json.Unmarshal([]byte(originalConfigJSON.String), &originalConfig); err != nil {
				return nil, fmt.Errorf("failed to unmarshal original configuration: %w", err)
			}
			cfg.OriginalConfiguration = originalConfig
		}

		configs = append(configs, &cfg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return configs, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	s.logger.Info("Closing SQLite storage")
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}

// ReplacePolicyDefinitions atomically replaces all policy definitions
func (s *SQLiteStorage) ReplacePolicyDefinitions(defs []api.PolicyDefinition) (err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Clear existing
	if _, err = tx.Exec("DELETE FROM policy_definitions"); err != nil {
		return fmt.Errorf("failed to clear existing policy definitions: %w", err)
	}

	var insertStmt *sql.Stmt
	insertStmt, err = tx.Prepare(`INSERT INTO policy_definitions (
		name, version, provider, description,
		flows_request_require_header, flows_request_require_body,
		flows_response_require_header, flows_response_require_body,
		parameters_schema
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer insertStmt.Close()

	for _, d := range defs {
		// Serialize parameters schema
		paramsJSON := "{}"
		if d.ParametersSchema != nil {
			var b []byte
			b, err = json.Marshal(d.ParametersSchema)
			if err != nil {
				return fmt.Errorf("failed to marshal parametersSchema for policy %s:%s: %w", d.Name, d.Version, err)
			}
			paramsJSON = string(b)
		}
		var reqHeader, reqBody, respHeader, respBody int
		if d.Flows.Request != nil {
			if d.Flows.Request.RequireHeader != nil && *d.Flows.Request.RequireHeader {
				reqHeader = 1
			}
			if d.Flows.Request.RequireBody != nil && *d.Flows.Request.RequireBody {
				reqBody = 1
			}
		}
		if d.Flows.Response != nil {
			if d.Flows.Response.RequireHeader != nil && *d.Flows.Response.RequireHeader {
				respHeader = 1
			}
			if d.Flows.Response.RequireBody != nil && *d.Flows.Response.RequireBody {
				respBody = 1
			}
		}

		// Convert *string to concrete string for DB driver
		var desc string
		if d.Description != nil {
			desc = *d.Description
		}

		if _, err = insertStmt.Exec(
			d.Name,
			d.Version,
			d.Provider,
			desc,
			reqHeader,
			reqBody,
			respHeader,
			respBody,
			paramsJSON,
		); err != nil {
			return fmt.Errorf("failed to insert policy definition %s:%s: %w", d.Name, d.Version, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit policy definitions replace: %w", err)
	}

	s.logger.Info("Policy definitions replaced", zap.Int("count", len(defs)))
	return nil
}

// GetAllPolicyDefinitions retrieves all policies
func (s *SQLiteStorage) GetAllPolicyDefinitions() ([]api.PolicyDefinition, error) {
	query := `SELECT name, version, provider, description,
		flows_request_require_header, flows_request_require_body,
		flows_response_require_header, flows_response_require_body,
		parameters_schema FROM policy_definitions ORDER BY name, version`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query policy definitions: %w", err)
	}
	defer rows.Close()

	defs := make([]api.PolicyDefinition, 0)
	for rows.Next() {
		var name, version, provider string
		var description sql.NullString
		var reqHeader, reqBody, respHeader, respBody sql.NullInt64
		var paramsJSON sql.NullString
		if err := rows.Scan(&name, &version, &provider, &description, &reqHeader, &reqBody, &respHeader, &respBody, &paramsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan policy definition: %w", err)
		}
		def := api.PolicyDefinition{Name: name, Version: version, Provider: provider}
		if description.Valid {
			def.Description = &description.String
		}
		// Flows
		if reqHeader.Valid || reqBody.Valid || respHeader.Valid || respBody.Valid {
			// Initialize flow structs only if any value present
			if reqHeader.Valid || reqBody.Valid {
				def.Flows.Request = &api.PolicyFlowRequirements{}
			}
			if respHeader.Valid || respBody.Valid {
				def.Flows.Response = &api.PolicyFlowRequirements{}
			}
			if reqHeader.Valid {
				b := reqHeader.Int64 == 1
				def.Flows.Request.RequireHeader = &b
			}
			if reqBody.Valid {
				b := reqBody.Int64 == 1
				def.Flows.Request.RequireBody = &b
			}
			if respHeader.Valid {
				b := respHeader.Int64 == 1
				def.Flows.Response.RequireHeader = &b
			}
			if respBody.Valid {
				b := respBody.Int64 == 1
				def.Flows.Response.RequireBody = &b
			}
		}
		if paramsJSON.Valid && paramsJSON.String != "" {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(paramsJSON.String), &m); err == nil {
				def.ParametersSchema = &m
			}
		}
		defs = append(defs, def)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policy definitions rows: %w", err)
	}
	return defs, nil
}

// LoadFromDatabase loads all configurations from database into the in-memory cache
func LoadFromDatabase(storage Storage, cache *ConfigStore) error {
	// Get all configurations from persistent storage
	configs, err := storage.GetAllConfigs()
	if err != nil {
		return fmt.Errorf("failed to load configurations from database: %w", err)
	}

	// Load into in-memory cache
	for _, cfg := range configs {
		if err := cache.Add(cfg); err != nil {
			return fmt.Errorf("failed to load config %s into cache: %w", cfg.ID, err)
		}
	}

	return nil
}

// LoadLLMProviderTemplatesFromDatabase loads all LLM Provider templates from database into in-memory store
func LoadLLMProviderTemplatesFromDatabase(storage Storage, cache *ConfigStore) error {
	// Get all llm provider template configurations from persistent storage
	templates, err := storage.GetAllLLMProviderTemplates()
	if err != nil {
		return fmt.Errorf("failed to load templates from database: %w", err)
	}

	for _, template := range templates {
		if err := cache.AddTemplate(template); err != nil {
			return fmt.Errorf("failed to load llm provider template %s into cache: %w", template.GetName(), err)
		}
	}

	return nil
}

// isUniqueConstraintError checks if the error is a UNIQUE constraint violation
func isUniqueConstraintError(err error) bool {
	// SQLite error code 19 is CONSTRAINT error
	// Error message contains "UNIQUE constraint failed"
	return err != nil && (err.Error() == "UNIQUE constraint failed: api_configs.name, api_configs.version" ||
		err.Error() == "UNIQUE constraint failed: api_configs.id")
}

// SaveLLMProviderTemplate persists a new LLM provider template
func (s *SQLiteStorage) SaveLLMProviderTemplate(template *models.StoredLLMProviderTemplate) error {
	// Serialize configuration to JSON
	configJSON, err := json.Marshal(template.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal template configuration: %w", err)
	}

	name := template.GetName()

	query := `
		INSERT INTO llm_provider_templates (
			id, name, configuration, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err = s.db.Exec(query,
		template.ID,
		name,
		string(configJSON),
		now,
		now,
	)

	if err != nil {
		// Check for unique constraint violation
		if isUniqueConstraintError(err) || (err != nil && err.Error() == "UNIQUE constraint failed: llm_provider_templates.name") {
			return fmt.Errorf("%w: template with name '%s' already exists", ErrConflict, name)
		}
		return fmt.Errorf("failed to insert template: %w", err)
	}

	s.logger.Info("LLM provider template saved",
		zap.String("id", template.ID),
		zap.String("name", name))

	return nil
}

// UpdateLLMProviderTemplate updates an existing LLM provider template
func (s *SQLiteStorage) UpdateLLMProviderTemplate(template *models.StoredLLMProviderTemplate) error {
	// Check if template exists
	_, err := s.GetLLMProviderTemplate(template.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("cannot update non-existent template: %w", err)
		}
		return err
	}

	// Serialize configuration to JSON
	configJSON, err := json.Marshal(template.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal template configuration: %w", err)
	}

	name := template.GetName()

	query := `
		UPDATE llm_provider_templates
		SET name = ?, configuration = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := s.db.Exec(query,
		name,
		string(configJSON),
		time.Now(),
		template.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("%w: id=%s", ErrNotFound, template.ID)
	}

	s.logger.Info("LLM provider template updated",
		zap.String("id", template.ID),
		zap.String("name", name))

	return nil
}

// DeleteLLMProviderTemplate removes an LLM provider template by ID
func (s *SQLiteStorage) DeleteLLMProviderTemplate(id string) error {
	query := `DELETE FROM llm_provider_templates WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("%w: id=%s", ErrNotFound, id)
	}

	s.logger.Info("LLM provider template deleted", zap.String("id", id))

	return nil
}

// GetLLMProviderTemplate retrieves an LLM provider template by ID
func (s *SQLiteStorage) GetLLMProviderTemplate(id string) (*models.StoredLLMProviderTemplate, error) {
	query := `
		SELECT id, configuration, created_at, updated_at
		FROM llm_provider_templates
		WHERE id = ?
	`

	var template models.StoredLLMProviderTemplate
	var configJSON string

	err := s.db.QueryRow(query, id).Scan(
		&template.ID,
		&configJSON,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id=%s", ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to query template: %w", err)
	}

	// Deserialize JSON configuration
	if err := json.Unmarshal([]byte(configJSON), &template.Configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template configuration: %w", err)
	}

	return &template, nil
}

// GetLLMProviderTemplateByName retrieves an LLM provider template by name
func (s *SQLiteStorage) GetLLMProviderTemplateByName(name string) (*models.StoredLLMProviderTemplate, error) {
	query := `
		SELECT id, configuration, created_at, updated_at
		FROM llm_provider_templates
		WHERE name = ?
	`

	var template models.StoredLLMProviderTemplate
	var configJSON string

	err := s.db.QueryRow(query, name).Scan(
		&template.ID,
		&configJSON,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: name=%s", ErrNotFound, name)
		}
		return nil, fmt.Errorf("failed to query template: %w", err)
	}

	// Deserialize JSON configuration
	if err := json.Unmarshal([]byte(configJSON), &template.Configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template configuration: %w", err)
	}

	return &template, nil
}

// GetAllLLMProviderTemplates retrieves all LLM provider templates
func (s *SQLiteStorage) GetAllLLMProviderTemplates() ([]*models.StoredLLMProviderTemplate, error) {
	query := `
		SELECT id, configuration, created_at, updated_at
		FROM llm_provider_templates
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []*models.StoredLLMProviderTemplate

	for rows.Next() {
		var template models.StoredLLMProviderTemplate
		var configJSON string

		err := rows.Scan(
			&template.ID,
			&configJSON,
			&template.CreatedAt,
			&template.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Deserialize JSON configuration
		if err := json.Unmarshal([]byte(configJSON), &template.Configuration); err != nil {
			return nil, fmt.Errorf("failed to unmarshal template configuration: %w", err)
		}

		templates = append(templates, &template)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return templates, nil
}
