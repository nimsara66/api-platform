#!/bin/bash
# Test script for LLM Provider CRUD operations (Phase 1 - Validation Only)

BASE_URL="http://localhost:9090"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}LLM Provider API Tests (Phase 1)${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# Test 1: Create LLM Provider (Valid)
echo -e "${YELLOW}Test 1: Create Valid LLM Provider${NC}"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/llm-providers" \
  -H "Content-Type: application/yaml" \
  -H "X-Correlation-ID: test-llm-provider-001" \
  --data-binary @../examples/llm-provider.yaml)

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 201 ]; then
    echo -e "${GREEN}✓ PASS${NC} - Status: $http_code"
    echo "$body" | jq '.'
else
    echo -e "${RED}✗ FAIL${NC} - Expected 201, got $http_code"
    echo "$body" | jq '.'
fi
echo ""

# Test 2: Invalid Provider (Missing Name)
echo -e "${YELLOW}Test 2: Create Provider with Missing Name${NC}"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/llm-providers" \
  -H "Content-Type: application/yaml" \
  -d 'version: api-platform.wso2.com/v1
kind: llm/provider
data:
  name: ""
  version: v1.0
  template: openai_v1.0
  upstreams:
    - url: https://api.openai.com/v1')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 400 ]; then
    echo -e "${GREEN}✓ PASS${NC} - Status: $http_code"
    echo "$body" | jq '.'
else
    echo -e "${RED}✗ FAIL${NC} - Expected 400, got $http_code"
    echo "$body" | jq '.'
fi
echo ""

# Test 3: Invalid Provider (Wrong Kind)
echo -e "${YELLOW}Test 3: Create Provider with Wrong Kind${NC}"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/llm-providers" \
  -H "Content-Type: application/yaml" \
  -d 'version: api-platform.wso2.com/v1
kind: llm/wrong-kind
data:
  name: test-provider
  version: v1.0
  template: openai_v1.0
  upstreams:
    - url: https://api.openai.com/v1')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 400 ]; then
    echo -e "${GREEN}✓ PASS${NC} - Status: $http_code"
    echo "$body" | jq '.'
else
    echo -e "${RED}✗ FAIL${NC} - Expected 400, got $http_code"
    echo "$body" | jq '.'
fi
echo ""

# Test 4: Invalid Provider (Invalid URL)
echo -e "${YELLOW}Test 4: Create Provider with Invalid URL${NC}"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/llm-providers" \
  -H "Content-Type: application/yaml" \
  -d 'version: api-platform.wso2.com/v1
kind: llm/provider
data:
  name: test-provider
  version: v1.0
  template: openai_v1.0
  upstreams:
    - url: invalid-url-format')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 400 ]; then
    echo -e "${GREEN}✓ PASS${NC} - Status: $http_code"
    echo "$body" | jq '.'
else
    echo -e "${RED}✗ FAIL${NC} - Expected 400, got $http_code"
    echo "$body" | jq '.'
fi
echo ""

# Test 5: Missing Template Reference
echo -e "${YELLOW}Test 5: Create Provider without Template Reference${NC}"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/llm-providers" \
  -H "Content-Type: application/yaml" \
  -d 'version: api-platform.wso2.com/v1
kind: llm/provider
data:
  name: test-provider
  version: v1.0
  template: ""
  upstreams:
    - url: https://api.openai.com/v1')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 400 ]; then
    echo -e "${GREEN}✓ PASS${NC} - Status: $http_code"
    echo "$body" | jq '.'
else
    echo -e "${RED}✗ FAIL${NC} - Expected 400, got $http_code"
    echo "$body" | jq '.'
fi
echo ""

# Test 6: No Upstreams
echo -e "${YELLOW}Test 6: Create Provider without Upstreams${NC}"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/llm-providers" \
  -H "Content-Type: application/yaml" \
  -d 'version: api-platform.wso2.com/v1
kind: llm/provider
data:
  name: test-provider
  version: v1.0
  template: openai_v1.0
  upstreams: []')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 400 ]; then
    echo -e "${GREEN}✓ PASS${NC} - Status: $http_code"
    echo "$body" | jq '.'
else
    echo -e "${RED}✗ FAIL${NC} - Expected 400, got $http_code"
    echo "$body" | jq '.'
fi
echo ""

# Test 7: List Providers (Empty in Phase 1)
echo -e "${YELLOW}Test 7: List All LLM Providers${NC}"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/llm-providers")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 200 ]; then
    echo -e "${GREEN}✓ PASS${NC} - Status: $http_code"
    echo "$body" | jq '.'
else
    echo -e "${RED}✗ FAIL${NC} - Expected 200, got $http_code"
    echo "$body" | jq '.'
fi
echo ""

# Test 8: Get Provider (Not Found in Phase 1)
echo -e "${YELLOW}Test 8: Get Provider by Name/Version${NC}"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/llm-providers/wso2-openai-provider/v1.0")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 404 ]; then
    echo -e "${GREEN}✓ PASS${NC} - Status: $http_code (Expected in Phase 1)"
    echo "$body" | jq '.'
else
    echo -e "${RED}✗ FAIL${NC} - Expected 404, got $http_code"
    echo "$body" | jq '.'
fi
echo ""

# Test 9: Valid Auth Configuration
echo -e "${YELLOW}Test 9: Create Provider with Auth${NC}"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/llm-providers" \
  -H "Content-Type: application/yaml" \
  -d 'version: api-platform.wso2.com/v1
kind: llm/provider
data:
  name: test-provider-auth
  version: v1.0
  template: openai_v1.0
  upstreams:
    - url: https://api.openai.com/v1
      auth:
        type: bearer
        header: Authorization
        value: Bearer sk-test-key')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 201 ]; then
    echo -e "${GREEN}✓ PASS${NC} - Status: $http_code"
    echo "$body" | jq '.'
else
    echo -e "${RED}✗ FAIL${NC} - Expected 201, got $http_code"
    echo "$body" | jq '.'
fi
echo ""

# Test 10: Invalid Auth Type
echo -e "${YELLOW}Test 10: Create Provider with Invalid Auth Type${NC}"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/llm-providers" \
  -H "Content-Type: application/yaml" \
  -d 'version: api-platform.wso2.com/v1
kind: llm/provider
data:
  name: test-provider-bad-auth
  version: v1.0
  template: openai_v1.0
  upstreams:
    - url: https://api.openai.com/v1
      auth:
        type: oauth2
        header: Authorization
        value: token')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 400 ]; then
    echo -e "${GREEN}✓ PASS${NC} - Status: $http_code"
    echo "$body" | jq '.'
else
    echo -e "${RED}✗ FAIL${NC} - Expected 400, got $http_code"
    echo "$body" | jq '.'
fi
echo ""

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Phase 1 Tests Complete${NC}"
echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Note: Phases 2 & 3 will implement actual storage and deployment${NC}"

