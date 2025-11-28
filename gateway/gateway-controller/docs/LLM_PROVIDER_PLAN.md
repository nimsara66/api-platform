# LLM Provider Implementation Plan

## Overview

Implement `llm-provider` entity that uses the existing API deployment flow by transforming LLM provider configurations into API configurations.

---

## Architecture

```
LLM Provider Config (YAML/JSON)
    ↓ Parse (generic parser)
    ↓ Validate (LLMValidator)
LLMProvider
    ↓ Transform (Phase 2)
APIConfiguration
    ↓ Store (Phase 3)
StoredAPIConfig {
    Configuration: APIConfiguration         (transformed - for deployment)
    OriginalConfiguration: LLMProvider      (original - for retrieval)
}
    ↓ Deploy (existing API deployment flow)
Envoy xDS
```

---

## Phase 1: CRUD Interface & Validation ✅ COMPLETE

### Implemented

#### 1. API Endpoints (`api/openapi.yaml`)
- ✅ `POST /llm-providers` - Create provider
- ✅ `GET /llm-providers` - List providers
- ✅ `GET /llm-providers/{name}/{version}` - Get provider
- ✅ `PUT /llm-providers/{name}/{version}` - Update provider
- ✅ `DELETE /llm-providers/{name}/{version}` - Delete provider

#### 2. Validation (`pkg/config/llm_validator.go`)
- ✅ `validateLLMProvider()` - Main validation
- ✅ `validateProviderData()` - Data section validation
- ✅ `validateUpstream()` - Upstream validation
- ✅ `validateAuth()` - Auth validation
- ✅ `validateAccessControl()` - Access control validation

#### 3. Handlers (`pkg/api/handlers/handlers.go`)
- ✅ `CreateLLMProvider()` - Parse & validate
- ✅ `ListLLMProviders()` - Empty list (Phase 3)
- ✅ `GetLLMProviderByNameVersion()` - Not found (Phase 3)
- ✅ `UpdateLLMProvider()` - Parse & validate (Phase 3)
- ✅ `DeleteLLMProvider()` - Not found (Phase 3)

#### 4. Parser Integration
- ✅ Reuses generic `parser.Parse()` (same as API and templates)

### Test Script
- ✅ `examples/test-llm-provider-phase1.sh` - Validation tests

### Phase 1 Status
**Build**: ✅ Success  
**Endpoints**: ✅ 5 endpoints  
**Validation**: ✅ Complete  
**Testing**: ✅ Test script ready

---

## Phase 2: Transformation Layer 🔄 UPDATED

Concrete responsibilities now implemented in `pkg/transformer/llm_provider_transformer.go`:

1. Retrieve referenced template from in-memory `ConfigStore` using `GetTemplateByName(templateName)`.
2. Parse the template's embedded OpenAPI (YAML) spec and build an `api.APIConfiguration` (no YAML output; pure Go structs):
   - Extract `info.title` (fallback: provider name) → API name
   - Extract `info.version` (fallback: provider version) → API version
   - Derive context from first `servers[].url` path or fallback to cleaned provider name (e.g. `/wso2-openai-provider`)
   - Extract operations from `paths` by enumerating HTTP methods
   - Initialize `Kind = http/rest`, `Version = api-platform.wso2.com/v1`
3. Merge LLM Provider instance specifics:
   - Upstreams: replace `apiCfg.Data.Upstream` with mapped `[]api.Upstream` from provider `data.upstreams`
   - Access Control:
     - If `mode = deny_all` keep only operations that match exceptions (path + method)
     - If `mode = allow_all` remove operations that match exceptions
     - Matching rule: exact path or prefix match with patterns ending `/*`; method list must contain the operation method (empty methods array means match all)
   - Policies:
     - For each policy group (budgetControl, PII, SemanticPromptGuardrail) attach a generated `api.Policy` to matching operations
     - Matching by path (same logic as access control) + allowed HTTP methods
     - Policy naming (initial baseline):
       - budgetControl → `budget-control`
       - PII → `pii`
       - SemanticPromptGuardrail → `semantic-guardrail`
     - Version placeholder: `1.0.0` (future: resolve from policy definitions catalog)
     - Params: flatten each policy's `params` values into a `map[string]interface{}`
4. Return final `*api.APIConfiguration` for deployment. The original `LLMProvider` will be stored alongside transformed API (Phase 3) via `StoredAPIConfig.OriginalConfiguration`.

### Helper behaviors
- OpenAPI parsing tolerant of JSON-as-YAML.
- Operations sorted deterministically by path then method.
- Context normalized (leading slash, no trailing slash unless root).
- Access control absent → retain all operations.

### Edge Cases & Safeguards
- Missing template → error.
- Empty/malformed OpenAPI paths → zero operations (deployment will subsequently fail validation or produce a no-op API).
- deny_all with no exceptions → zero operations.
- allow_all with no exceptions → all operations kept.
- Overlapping policy path patterns can stack policies on the same operation.

### Deferred / Future Enhancements
- Policy version resolution from registered policy definitions.
- Advanced path matching (template parameters, regex support).
- Upstream auth mapping into API-level security artifacts.
- Deriving description into optional metadata or tags.

## Phase 3: Storage & Deployment 📦 (unchanged high-level, will now consume transformer output)

### To Implement

#### 1. Update CreateLLMProvider Handler

```go
func (s *APIServer) CreateLLMProvider(c *gin.Context) {
    // ...existing parse and validate...
    
    // Transform to API configuration
    apiConfig, err := s.transformer.Transform(&llmProviderConfig)
    if err != nil {
        // Handle transformation error
        return
    }
    
    // Create StoredAPIConfig with both original and transformed
    storedConfig := &models.StoredAPIConfig{
        ID:                    uuid.New().String(),
        Configuration:         *apiConfig,
        Status:                models.StatusPending,
        OriginalConfiguration: llmProviderConfig,  // Store original
        CreatedAt:             time.Now(),
        UpdatedAt:             time.Now(),
    }
    
    // Deploy using existing API deployment service
    result, err := s.deploymentService.DeployAPIConfiguration(utils.APIDeploymentParams{
        Data:          body,
        ContentType:   contentType,
        APIID:         storedConfig.ID,
        CorrelationID: correlationID,
        Logger:        log,
        // NEW: Pass original config
        OriginalConfig: &llmProviderConfig,
    })
    
    // Return success
    c.JSON(http.StatusCreated, api.LLMProviderCreateResponse{...})
}
```

#### 2. Update ListLLMProviders Handler

```go
func (s *APIServer) ListLLMProviders(c *gin.Context) {
    // Get all configs
    allConfigs := s.store.GetAll()
    
    // Filter by original_kind = "llm/provider"
    providers := []*models.StoredAPIConfig{}
    for _, cfg := range allConfigs {
        if cfg.GetOriginalKind() == "llm/provider" {
            providers = append(providers, cfg)
        }
    }
    
    // Build response
    items := make([]api.LLMProviderListItem, len(providers))
    for i, p := range providers {
        original := p.OriginalConfiguration.(api.LLMProvider)
        items[i] = api.LLMProviderListItem{
            Id:        p.ID,
            Name:      &original.Data.Name,
            Version:   &original.Data.Version,
            Template:  &original.Data.Template,
            Status:    (*api.LLMProviderListItemStatus)(&p.Status),
            CreatedAt: &p.CreatedAt,
            UpdatedAt: &p.UpdatedAt,
        }
    }
    
    c.JSON(http.StatusOK, gin.H{
        "status":    "success",
        "count":     len(items),
        "providers": items,
    })
}
```

#### 3. Update GetLLMProviderByNameVersion Handler

```go
func (s *APIServer) GetLLMProviderByNameVersion(c *gin.Context, name string, version string) {
    // Get from api_configs
    cfg, err := s.store.GetByNameVersion(name, version)
    if err != nil {
        c.JSON(http.StatusNotFound, ...)
        return
    }
    
    // Verify it's an LLM provider
    if cfg.GetOriginalKind() != "llm/provider" {
        c.JSON(http.StatusNotFound, ...)
        return
    }
    
    // Return original configuration
    original := cfg.OriginalConfiguration.(api.LLMProvider)
    
    c.JSON(http.StatusOK, api.LLMProviderDetailResponse{
        Status: "success",
        Provider: {
            Id:               cfg.ID,
            Configuration:    &original,
            DeploymentStatus: string(cfg.Status),
            Metadata: {
                CreatedAt:  cfg.CreatedAt,
                UpdatedAt:  cfg.UpdatedAt,
                DeployedAt: cfg.DeployedAt,
            },
        },
    })
}
```

#### 4. Update UpdateLLMProvider Handler

```go
func (s *APIServer) UpdateLLMProvider(c *gin.Context, name string, version string) {
    // Get existing config
    cfg, err := s.store.GetByNameVersion(name, version)
    
    // Parse and validate new config
    // ...
    
    // Transform to API
    apiConfig, err := s.transformer.Transform(&llmProviderConfig)
    
    // Update both original and transformed
    cfg.Configuration = *apiConfig
    cfg.OriginalConfiguration = llmProviderConfig
    cfg.Status = models.StatusPending
    cfg.UpdatedAt = time.Now()
    
    // Update in database and memory
    s.db.UpdateConfig(cfg)
    s.store.Update(cfg)
    
    // Trigger xDS snapshot update
    s.snapshotManager.UpdateSnapshot(...)
}
```

#### 5. Update DeleteLLMProvider Handler

```go
func (s *APIServer) DeleteLLMProvider(c *gin.Context, name string, version string) {
    // Get config
    cfg, err := s.store.GetByNameVersion(name, version)
    
    // Verify it's an LLM provider
    if cfg.GetOriginalKind() != "llm/provider" {
        c.JSON(http.StatusNotFound, ...)
        return
    }
    
    // Delete using existing flow
    s.db.DeleteConfig(cfg.ID)
    s.store.Delete(cfg.ID)
    
    // Trigger xDS snapshot update
    s.snapshotManager.UpdateSnapshot(...)
}
```

### Phase 3 Deliverables

- [ ] Update all handler methods
- [ ] Add filtering by original_kind
- [ ] Add type assertions for original configs
- [ ] Update deployment service to accept OriginalConfiguration
- [ ] Integration tests for end-to-end flow
- [ ] Documentation updates

---

## Benefits of This Approach

### 1. Code Reuse
- ✅ Reuses existing API deployment pipeline
- ✅ Reuses xDS snapshot generation
- ✅ Reuses database persistence
- ✅ Reuses status tracking
- ✅ No duplicate deployment logic

### 2. Consistency
- ✅ Same deployment flow for all entities
- ✅ Same status tracking (pending → deployed → failed)
- ✅ Same xDS update mechanism
- ✅ Same error handling patterns

### 3. Storage Efficiency
- ✅ Single table for all deployed entities
- ✅ Unified indexing (name + version)
- ✅ Original config preserved for retrieval
- ✅ Transformed config for deployment

### 4. Extensibility
- ✅ Easy to add more entities (llm-proxy, etc.)
- ✅ Each entity just needs a transformer
- ✅ Minimal changes to existing code

---

## Testing Plan

### Phase 1 Tests (Current)
- ✅ Valid provider creation
- ✅ Validation error handling
- ✅ Parser integration
- ✅ Error responses

### Phase 2 Tests (Next)
- [ ] Template lookup
- [ ] Transformation success
- [ ] Transformation errors
- [ ] OpenAPI parsing
- [ ] Upstream mapping
- [ ] Policy mapping

### Phase 3 Tests (Future)
- [ ] End-to-end provider creation
- [ ] Storage verification
- [ ] Retrieval of original config
- [ ] Update flow
- [ ] Delete flow
- [ ] xDS snapshot generation
- [ ] Envoy deployment

---

## Current Status

### ✅ Completed
- [x] Phase 1: CRUD Interface & Validation
  - [x] OpenAPI spec with endpoints and schemas
  - [x] LLMValidator with provider validation
  - [x] Handler methods (validation only)
  - [x] Test script for Phase 1
  - [x] Build verified

### 🔄 In Progress
- [ ] Phase 2: Transformation Layer

### 📋 Planned
- [ ] Phase 3: Storage & Deployment

---

## File Summary

### Created
1. ✅ `examples/test-llm-provider-phase1.sh` - Phase 1 test script
2. ✅ `docs/LLM_PROVIDER_PHASE1_COMPLETE.md` - Phase 1 documentation
3. ✅ `docs/LLM_PROVIDER_PLAN.md` - Complete implementation plan

### Modified
1. ✅ `api/openapi.yaml` - Added provider endpoints and schemas
2. ✅ `pkg/config/llm_validator.go` - Added provider validation
3. ✅ `pkg/api/handlers/handlers.go` - Added handler methods
4. ✅ `pkg/models/api_config.go` - Added LLMProvider to GetOriginalKind()

### Generated
1. ✅ `pkg/api/generated/generated.go` - LLMProvider types

---

## Next Steps

Ready to start **Phase 2: Transformation Layer**!

This will implement:
1. Transformer interface definition
2. LLMProviderTransformer implementation
3. Template lookup and OpenAPI parsing
4. Mapping logic for upstreams, policies, and access control
5. Unit tests for transformation logic

Once Phase 2 is complete, Phase 3 will connect everything to the deployment pipeline.
