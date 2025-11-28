# Quick Reference: LLM Provider Template API

## Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/llm-providers/templates` | Create a new template |
| GET | `/llm-providers/templates` | List all templates |
| GET | `/llm-providers/templates/{name}` | Get template by name |
| PUT | `/llm-providers/templates/{name}` | Update a template |
| DELETE | `/llm-providers/templates/{name}` | Delete a template |

## Quick Start

### 1. Create a Template

```bash
curl -X POST http://localhost:9090/llm-providers/templates \
  -H "Content-Type: application/yaml" \
  --data-binary @- << 'EOF'
version: api-platform.wso2.com/v1
kind: llm/provider-template
data:
  name: openai_v1.0
  promptTokens:
    location: payload
    identifier: $.usage.prompt_tokens
  completionTokens:
    location: payload
    identifier: $.usage.completion_tokens
  totalTokens:
    location: payload
    identifier: $.usage.total_tokens
  requestModel:
    location: payload
    identifier: $.model
  openapi: |
    openapi: 3.0.1
    info:
      title: OpenAI Chat Completions API
      version: "1.0.0"
    paths:
      /v1/chat/completions:
        post:
          summary: Create chat completion
          responses:
            "200":
              description: Success
EOF
```

### 2. List All Templates

```bash
curl http://localhost:9090/llm-providers/templates | jq
```

### 3. Get Specific Template (JSON)

```bash
curl http://localhost:9090/llm-providers/templates/openai_v1.0 | jq
```

### 4. Get Specific Template (YAML)

```bash
curl -H "Accept: application/yaml" \
  http://localhost:9090/llm-providers/templates/openai_v1.0
```

### 5. Update Template

```bash
curl -X PUT http://localhost:9090/llm-providers/templates/openai_v1.0 \
  -H "Content-Type: application/yaml" \
  --data-binary @updated-template.yaml
```

### 6. Delete Template

```bash
curl -X DELETE http://localhost:9090/llm-providers/templates/openai_v1.0
```

## Template Structure

```yaml
version: api-platform.wso2.com/v1  # Required
kind: llm/provider-template        # Required
data:
  name: template_name              # Required, unique, alphanumeric_-.
  
  # Optional: Token tracking identifiers
  promptTokens:
    location: payload              # "payload" or "header"
    identifier: $.usage.inputTokens  # JSONPath or header name
  
  completionTokens:
    location: payload
    identifier: $.usage.outputTokens
  
  totalTokens:
    location: payload
    identifier: $.usage.totalTokens
  
  requestModel:
    location: payload
    identifier: $.model
  
  # Required: OpenAPI specification for the provider
  openapi: |
    openapi: 3.0.1
    info:
      title: Provider API
      version: "1.0.0"
    paths:
      # Define provider endpoints...
```

## Response Codes

| Code | Meaning |
|------|---------|
| 201 | Template created successfully |
| 200 | Success (get, update, delete, list) |
| 400 | Invalid request or validation error |
| 404 | Template not found |
| 409 | Template with same name already exists |
| 500 | Internal server error |

## Example Responses

### Success (Create)
```json
{
  "status": "success",
  "message": "LLM provider template created successfully",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-11-25T10:30:00Z"
}
```

### List
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

### Error (Validation)
```json
{
  "status": "error",
  "message": "Template validation failed",
  "errors": [
    {
      "field": "data.name",
      "message": "Template name is required"
    },
    {
      "field": "data.openapi",
      "message": "OpenAPI specification is required"
    }
  ]
}
```

### Error (Conflict)
```json
{
  "status": "error",
  "message": "Template with name 'openai_v1.0' already exists"
}
```

### Error (Not Found)
```json
{
  "status": "error",
  "message": "Template with name 'unknown' not found"
}
```

## Testing

Run the comprehensive test suite:

```bash
cd /Users/nimsara/wso2/api-platform/gateway
./examples/test-llm-template.sh
```

## Files Reference

- **Example**: `examples/llm-provider-template.yaml`
- **Documentation**: `gateway-controller/docs/llm-provider-template.md`
- **Test Script**: `examples/test-llm-template.sh`
- **Implementation Summary**: `IMPLEMENTATION_SUMMARY.md`

