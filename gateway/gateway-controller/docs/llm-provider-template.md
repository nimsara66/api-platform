# LLM Provider Template Implementation

This document describes the implementation of CRUD operations for the `llm-provider-template` entity in the Gateway Controller.

## Overview

The LLM Provider Template entity allows users to define and manage templates for different LLM providers (like OpenAI, Anthropic, etc.). These templates specify how to interact with the provider's API, including token tracking identifiers and OpenAPI specifications.

## Architecture

The implementation follows the same pattern as the API entity (`CreateAPI`) with proper separation of concerns:

### Components

1. **Models** (`pkg/models/llm_provider_template.go`)
   - `StoredLLMProviderTemplate`: Represents the template in storage
   - Contains ID, Configuration, CreatedAt, UpdatedAt fields

2. **Validator** (`pkg/config/llm_validator.go`)
   - `LLMValidator`: Validates LLM-related configurations using the `Validator` interface
   - Uses type switching to support multiple LLM entity types
   - Currently supports: `LLMProviderTemplate`
   - Extensible for future entities: `LLMProvider`, `LLMProxy`
   - Checks required fields, formats, and token identifiers

3. **Storage**
   - **Database Schema** (`pkg/storage/gateway-controller-db.sql`)
     - Added `llm_provider_templates` table (schema version 3)
     - Fields: id, name (unique), configuration, created_at, updated_at
   - **SQLite Storage** (`pkg/storage/sqlite.go`)
     - Migration logic for schema version 3
     - CRUD methods: Save, Update, Delete, Get, GetByName, GetAll
   - **In-Memory Store** (`pkg/storage/memory.go`)
     - Extended `ConfigStore` to include LLM template storage
     - Thread-safe operations using same mutex as API configs
     - Supports lookup by ID and by name
     - Methods: AddTemplate, UpdateTemplate, DeleteTemplate, GetTemplate, GetTemplateByName, GetAllTemplates

4. **API Handlers** (`pkg/api/handlers/handlers.go`)
   - `CreateLLMProviderTemplate`: POST /llm-providers/templates
   - `ListLLMProviderTemplates`: GET /llm-providers/templates
   - `GetLLMProviderTemplateByName`: GET /llm-providers/templates/{name}
   - `UpdateLLMProviderTemplate`: PUT /llm-providers/templates/{name}
   - `DeleteLLMProviderTemplate`: DELETE /llm-providers/templates/{name}

5. **OpenAPI Specification** (`api/openapi.yaml`)
   - Added `/llm-providers/templates` endpoints
   - Schema definitions for templates and responses
   - Added "LLM Provider Management" tag

## API Endpoints

### Create Template
```bash
POST /llm-providers/templates
Content-Type: application/yaml

version: api-platform.wso2.com/v1
kind: llm/provider-template
data:
  name: openai_v1.0
  promptTokens:
    location: payload
    identifier: $.usage.inputTokens
  completionTokens:
    location: payload
    identifier: $.usage.outputTokens
  totalTokens:
    location: payload
    identifier: $.usage.totalTokens
  requestModel:
    location: payload
    identifier: $.model
  openapi: |
    openapi: 3.0.1
    ...
```

**Response (201 Created):**
```json
{
  "status": "success",
  "message": "LLM provider template created successfully",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-11-25T10:30:00Z"
}
```

### List Templates
```bash
GET /llm-providers/templates
```

**Response (200 OK):**
```json
{
  "status": "success",
  "count": 2,
  "templates": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "openai_v1.0",
      "created_at": "2025-11-25T10:30:00Z",
      "updated_at": "2025-11-25T10:30:00Z"
    }
  ]
}
```

### Get Template by Name
```bash
GET /llm-providers/templates/{name}
Accept: application/json  # or application/yaml
```

**Response (200 OK):**
```json
{
  "status": "success",
  "template": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "configuration": { ... },
    "metadata": {
      "created_at": "2025-11-25T10:30:00Z",
      "updated_at": "2025-11-25T10:30:00Z"
    }
  }
}
```

### Update Template
```bash
PUT /llm-providers/templates/{name}
Content-Type: application/yaml

# Updated template configuration
```

**Response (200 OK):**
```json
{
  "status": "success",
  "message": "LLM provider template updated successfully",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "updated_at": "2025-11-25T11:45:00Z"
}
```

### Delete Template
```bash
DELETE /llm-providers/templates/{name}
```

**Response (200 OK):**
```json
{
  "status": "success",
  "message": "LLM provider template deleted successfully",
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## Database Schema

### Table: `llm_provider_templates`

```sql
CREATE TABLE IF NOT EXISTS llm_provider_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    configuration TEXT NOT NULL,  -- JSON-serialized template
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_template_name ON llm_provider_templates(name);
```

## Validation Rules

1. **Version**: Must be `api-platform.wso2.com/v1`
2. **Kind**: Must be `llm/provider-template`
3. **Name**: Required, alphanumeric with underscores, dots, and hyphens
4. **OpenAPI**: Required field containing the provider's API specification
5. **Token Identifiers**: Optional, but if present must have:
   - `location`: Either "payload" or "header"
   - `identifier`: Non-empty string (JSONPath or header name)

## Logging

All operations include structured logging similar to API deployment:

- **Create**: Logs template creation with ID and name
- **Update**: Logs update with ID and name
- **Delete**: Logs deletion with ID and name
- **Validation Errors**: Logs validation failures with error count
- **Database Errors**: Logs persistence failures

Example:
```
INFO  LLM provider template created successfully  id=550e8400-... name=openai_v1.0
WARN  Template validation failed  name=openai_v1.0 error_count=2
ERROR Failed to save template to database  error="constraint violation"
```

## Storage Strategy

The implementation uses a dual-storage approach:

1. **SQLite Database** (Persistent)
   - Primary storage for durability
   - Survives controller restarts
   - Atomic operations with proper error handling

2. **In-Memory Store** (Fast Access)
   - Thread-safe cache using sync.RWMutex
   - Indexed by both ID and name
   - Loaded from database on startup

## Differences from API Entity

Unlike the API entity, LLM Provider Templates:

1. **No Deployment**: Templates are configuration only, not deployed to xDS
2. **Simpler Storage**: No status tracking (pending/deployed/failed)
3. **Name-Based Lookup**: Primary access is by name, not name+version
4. **No xDS Integration**: No snapshot updates or deployment callbacks

## Testing

Use the provided test script:

```bash
# Start the gateway controller
./bin/controller

# In another terminal, run tests
./examples/test-llm-template.sh
```

The test script demonstrates:
- Creating a template
- Listing all templates
- Getting a specific template (JSON and YAML)
- Updating a template
- Deleting a template
- Verifying 404 responses

## Migration

The database schema migration from version 2 to 3 is automatic:

1. On startup, the controller checks `PRAGMA user_version`
2. If version is 2, creates the `llm_provider_templates` table
3. Updates schema version to 3
4. Logs migration success

Existing data in `api_configs` and `policy_definitions` is preserved.

## Error Handling

- **409 Conflict**: Template with same name already exists
- **404 Not Found**: Template not found for get/update/delete
- **400 Bad Request**: Invalid configuration or validation failure
- **500 Internal Server Error**: Database or system errors

All errors include descriptive messages and proper HTTP status codes.

## Future Enhancements

Potential improvements:

1. **Versioning**: Add version field to support multiple template versions
2. **Import/Export**: Bulk import/export of templates
3. **Template Validation**: Validate OpenAPI spec structure
4. **Usage Tracking**: Track which LLM providers use which templates
5. **Template Inheritance**: Support template extension/composition

