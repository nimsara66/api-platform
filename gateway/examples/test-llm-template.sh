#!/bin/bash

# Test script for LLM Provider Template CRUD operations
# Prerequisites: Gateway Controller should be running on localhost:9090

BASE_URL="http://localhost:9090"

echo "=== Testing LLM Provider Template CRUD Operations ==="
echo ""

# 1. Create an LLM Provider Template
echo "1. Creating LLM Provider Template..."
CREATE_RESPONSE=$(curl -s -X POST "${BASE_URL}/llm-providers/templates" \
  -H "Content-Type: application/yaml" \
  -d @llm-provider-template.yaml)

echo "Response: ${CREATE_RESPONSE}"
echo ""

# Extract template ID from response
TEMPLATE_ID=$(echo "${CREATE_RESPONSE}" | jq -r '.id // empty')
TEMPLATE_NAME="openai_v1.0"

sleep 1

# 2. List all templates
echo "2. Listing all LLM Provider Templates..."
LIST_RESPONSE=$(curl -s -X GET "${BASE_URL}/llm-providers/templates")
echo "Response: ${LIST_RESPONSE}"
echo ""

sleep 1

# 3. Get template by name
echo "3. Getting LLM Provider Template by name: ${TEMPLATE_NAME}..."
GET_RESPONSE=$(curl -s -X GET "${BASE_URL}/llm-providers/templates/${TEMPLATE_NAME}")
echo "Response: ${GET_RESPONSE}"
echo ""

sleep 1

# 4. Get template as YAML
echo "4. Getting LLM Provider Template as YAML..."
GET_YAML_RESPONSE=$(curl -s -X GET "${BASE_URL}/llm-providers/templates/${TEMPLATE_NAME}" \
  -H "Accept: application/yaml")
echo "Response:"
echo "${GET_YAML_RESPONSE}"
echo ""

sleep 1

# 5. Update template
echo "5. Updating LLM Provider Template..."
cat > /tmp/updated-template.yaml << 'EOF'
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
      title: OpenAI Chat Completions API - Updated
      version: "1.0.0"
    paths:
      /v1/chat/completions:
        post:
          summary: Create chat completion
          responses:
            "200":
              description: Success
EOF

UPDATE_RESPONSE=$(curl -s -X PUT "${BASE_URL}/llm-providers/templates/${TEMPLATE_NAME}" \
  -H "Content-Type: application/yaml" \
  -d @/tmp/updated-template.yaml)

echo "Response: ${UPDATE_RESPONSE}"
echo ""

sleep 1

# 6. Verify update
echo "6. Verifying update..."
VERIFY_RESPONSE=$(curl -s -X GET "${BASE_URL}/llm-providers/templates/${TEMPLATE_NAME}")
echo "Response: ${VERIFY_RESPONSE}"
echo ""

sleep 1

# 7. Delete template
echo "7. Deleting LLM Provider Template..."
DELETE_RESPONSE=$(curl -s -X DELETE "${BASE_URL}/llm-providers/templates/${TEMPLATE_NAME}")
echo "Response: ${DELETE_RESPONSE}"
echo ""

sleep 1

# 8. Verify deletion
echo "8. Verifying deletion (should return 404)..."
VERIFY_DELETE_RESPONSE=$(curl -s -X GET "${BASE_URL}/llm-providers/templates/${TEMPLATE_NAME}")
echo "Response: ${VERIFY_DELETE_RESPONSE}"
echo ""

# Cleanup
rm -f /tmp/updated-template.yaml

echo "=== Test Complete ==="

