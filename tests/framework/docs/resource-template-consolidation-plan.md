# Resource Template Consolidation Plan

> Temporary implementation plan. Delete this file after the consolidation is complete and the permanent framework documentation has been updated.

## Objective

Reduce the integration-suite resource definitions from feature-specific YAML files to one canonical, configurable template per supported resource kind.

The target is to keep scenario-specific values in Gherkin tables or framework context while keeping resource structure in reusable templates.

## Target structure

```text
tests/framework/suites/it/resources/templates/
├── rest-api.yaml
├── llm-provider.yaml
├── llm-provider-template.yaml
├── llm-proxy.yaml
└── mcp-proxy.yaml
```

The exact set of templates must follow the resource kinds supported by the product and the suite.

## Initial inventory

The suite initially contained 216 API resource YAML files. The initial inventory by YAML `kind` was:

| Resource kind | Current files | Target template |
|---|---:|---|
| `LlmProxy` | 3 initially; complete | `resources/templates/llm-proxy.yaml` |
| `Mcp` | 5 initially; complete | `resources/templates/mcp.yaml` |
| `LlmProviderTemplate` | 6 initially; complete | `resources/templates/llm-provider-template.yaml` |
| `LlmProvider` | 13 initially; complete | `resources/templates/llm-provider.yaml` |
| `RestApi` | 189 | `resources/templates/rest-api.yaml` |
| **Total** | **216** | **5 canonical templates** |

The first conversion target was `LlmProxy`, because it had the smallest current inventory. It now has no remaining feature-specific resource files.

The first LlmProxy resilience variant has been migrated to `resources/templates/llm-proxy.yaml`, verified through the focused SQLite `llm-backend-timeout` runner, and deleted.
The second LlmProxy resilience variant has also been migrated to the canonical template, verified through the same focused runner, and deleted.
The startup LlmProxy variant has also been migrated to the canonical template, verified through the focused SQLite `startup-db-bootstrap` runner, and deleted.

The inventory must be regenerated after each migration batch. Counts must reach zero for obsolete feature-specific files before that resource kind is considered complete. LlmProxy and Mcp now have zero obsolete resource files in the migrated targets.

LlmProviderTemplate and LlmProvider have also been migrated. Their lazy-resource,
timeout, startup, and named-upstream scenarios now use the canonical templates,
and all initial feature-specific files for both kinds were deleted after focused
SQLite verification.

The first REST API runner batch is complete. The `metrics`, `log-message`, and
`startup-db-bootstrap` runners now use `resources/templates/rest-api.yaml`.
Their three runner-specific REST API files were deleted after each runner passed
both before and after deletion in focused SQLite validation.

The second REST API runner batch is complete. The `header-routing`,
`path-normalization`, `cors`, `upstream-connect-timeout`,
`backend-route-timeout`, `interceptor-service`, and `vhost-routing-single`
runners now use the canonical template. Their twenty runner-specific REST API
files were deleted after focused SQLite validation before and after deletion.

The third REST API runner batch is complete. The `vhost-routing-multi`,
`redirect`, `host-rewrite`, `search-deployments`, and `api-error-responses`
runners now use the canonical template for their REST API resources. Their
twenty-nine runner-specific REST API files were deleted after each runner
passed focused SQLite validation before and after deletion. The malformed API
error cases retain their original invalid configuration through table-driven
overrides, including the explicitly empty metadata name case.

The fourth REST API runner batch is complete. The `dynamic-endpoint`,
`route-path-matching`, and `api-keys` runners now use the canonical template.
Their twenty-five runner-specific REST API files were deleted after focused
SQLite validation before and after deletion. The API-key feature also regained
its missing `@api-keys` selection tag so its configured runner can be selected
independently.

The fifth REST API runner batch is complete. The `api-with-policies` and
`request-rewrite` runners now use the canonical template for API-level,
operation-level, update, empty-version, path, query, method, regex, and match
condition policy cases. Their twenty-one runner-specific REST API files were
deleted after focused SQLite validation before and after deletion.

The sixth REST API runner batch is complete. The `api-deploy` and `health`
runners were migrated together because both initially used `deploy-simple.yaml`.
All valid, labeled, version-placeholder, and invalid-upstream cases now use the
canonical template. All eight deployment resource files, including the shared
`deploy-simple.yaml`, were deleted only after both focused runners passed, and
both runners passed again after deletion.

The seventh REST API runner batch is complete. The `api-management`, `respond`,
and `sandbox-routing` runners now use the canonical template for management,
response-policy, sandbox-vhost, named-upstream, policy, update, and validation
cases. Their 82 runner-specific REST API files were deleted after focused
SQLite validation before and after deletion. The final `resources/apis`
inventory contains only the canonical `base.yaml` compatibility resource; no
feature-specific API resource files or stale references remain.

### Field-level inventory

The following comparison separates the stable resource envelope from fields that vary by scenario.

#### `LlmProxy`

Stable envelope:

- `apiVersion`, `kind`, and `metadata.name`.
- `spec.displayName`, `spec.version`, `spec.context`.
- `spec.provider.id`.

Variable or optional fields:

- `spec.resilience.timeout` appears only in resilience scenarios.
- Provider identity, display name, version, context, and resilience values are scenario-specific.

The canonical template contains the stable envelope and provider reference. Resilience must be inserted only when supplied.

#### `Mcp`

Stable envelope:

- `apiVersion`, `kind`, and `metadata.name`.
- `spec.displayName`, `spec.version`, `spec.context`, and `spec.specVersion`.
- `spec.tools`, `spec.resources`, and `spec.prompts`.

Variable or optional fields:

- `spec.upstream.url` for a direct upstream.
- `spec.upstreamDefinitions` and `spec.upstream.ref` for named upstreams and connection settings.
- Upstream URLs, connection timeouts, display names, versions, contexts, and MCP definition values.

The direct-upstream form is the canonical base. The renderer must replace it with the named-upstream form when that structure is supplied.

#### `LlmProviderTemplate`

Stable envelope:

- `apiVersion`, `kind`, and `metadata.name`.
- `spec.displayName`.

Variable or optional fields:

- `spec.version`.
- `spec.upstream.url`.
- `spec.accessControl.mode`.
- Token and model extraction definitions: `promptTokens`, `completionTokens`, `totalTokens`, `remainingTokens`, `requestModel`, and `responseModel`.
- Every extraction definition's `location` and `identifier`.

The canonical template intentionally contains only the stable envelope. Extraction, upstream, version, and access-control nodes must be added only when supplied.

#### `LlmProvider`

Stable envelope:

- `apiVersion`, `kind`, and `metadata.name`.
- `spec.displayName`, `spec.version`, `spec.template`, and `spec.upstream`.
- `spec.accessControl.mode`.

Variable or optional fields:

- `spec.context` is absent in some provider resources.
- `spec.upstream.url` or the named-upstream form using `spec.upstreamDefinitions` and `spec.upstream.ref`.
- `spec.accessControl.exceptions` for selective access-control rules.
- `spec.resilience.timeout`.
- Provider identity, template, upstream, context, access mode, exceptions, and resilience values.

The canonical template uses the direct-upstream and basic access-control forms. Named upstreams, exceptions, and resilience must be inserted through structured overrides.

#### `RestApi`

Stable envelope:

- `apiVersion`, `kind`, and `metadata.name`.
- `spec.displayName`, `spec.version`, `spec.context`.
- `spec.upstream` and `spec.operations`.

Variable or optional fields:

- `metadata.labels`.
- `spec.vhosts`.
- `spec.upstream.main` and optional `spec.upstream.sandbox`.
- Direct upstream URLs or named upstreams using `spec.upstreamDefinitions` and references.
- `spec.policies`.
- `spec.resilience.timeout`.
- Operation method, path, operation policies, and operation resilience.
- Display name, version, context, labels, hosts, upstreams, policies, and operation values.

The canonical template contains the direct main upstream and an empty operations list. Labels, virtual hosts, named upstreams, policies, resilience, and operation values must be supplied through structured overrides.

Fields deliberately absent from a canonical template are not unsupported. They are optional nodes whose presence is controlled by the rendered input. The renderer must support adding, replacing, and removing them while preserving omitted-versus-empty semantics.

## Design requirements

The renderer must:

- Load a canonical YAML template.
- Resolve `${CTX:...}` and `${UNIQUE:...}` values.
- Apply typed values using validated field paths.
- Support nested scalar, map, and list values.
- Omit optional fields when no value is supplied.
- Preserve the distinction between an omitted field and an explicitly empty value.
- Reject unknown field paths and incompatible value types before making an HTTP request.
- Validate the rendered document before it is submitted.
- Remain safe for concurrent rendering; a shared template must never be mutated.

The implementation must use a YAML AST or typed representation. String replacement alone is insufficient because it cannot reliably represent omission, types, nested structures, or list updates.

## Gherkin configuration model

The existing template step should evolve from feature-specific files toward structured field overrides, for example:

```gherkin
Given I create an API from "resources/templates/rest-api.yaml" with values:
  | metadata.name          | ${CTX:apiName}        |
  | spec.displayName       | ${CTX:displayName}    |
  | spec.version           | ${CTX:apiVersion}     |
  | spec.context           | ${CTX:apiContext}     |
  | spec.upstream.main.url | http://testbench:3000 |
```

Operations, policies, and other complex values must be supplied through typed structured values or reusable builder steps. Large inline YAML documents must not be reintroduced into feature files.

## Implementation tasks

### 1. Inventory current resources

- Count resources by `kind`.
- Normalize YAML structures and identify duplicate templates.
- Record fields that vary between files.
- Identify deliberate malformed-payload resources and genuine schema differences.
- Produce a migration map from every existing resource file to its canonical template and override table.

### 2. Define the renderer contract

- Define supported field-path syntax.
- Define scalar, map, list, empty, and omitted value semantics.
- Define validation behavior for unknown paths and incompatible types.
- Define how context and unique-name expansion interacts with typed values.
- Define whether resource-kind schemas are validated by the renderer or by the existing product request path.

### 3. Implement typed rendering

- Replace or extend text-only substitution with YAML-node or typed-map rendering.
- Keep template loading path validation and resource-kind validation.
- Ensure each render operates on an independent copy of the template.
- Preserve concurrent safety and avoid package-level mutable template state.
- Return errors that identify the template, field path, expected type, and supplied value.

### 4. Add renderer unit tests

Cover at least:

- Scalar replacement.
- Nested field replacement.
- Boolean and numeric values.
- Map replacement.
- List replacement.
- Multiple operations.
- API-level policies.
- Operation-level policies.
- Missing optional values and node omission.
- Explicit empty values.
- Unknown field paths.
- Invalid value types.
- Missing required values.
- Duplicate table keys.
- Invalid YAML templates.
- Resource-kind mismatch.
- Concurrent rendering with shared template input.
- Verification that one render cannot mutate another render or the source template.

### 5. Create canonical templates

Create one template for each supported resource kind and keep only stable structural fields in those templates. Do not encode scenario names, runner names, ordinal numbers, or policy combinations in filenames or template content.

### 6. Migrate API resources

Migrate the API lifecycle and policy feature groups first:

- API management.
- API deployment.
- API error responses.
- API keys.
- APIs with policies.
- Search and deployment behavior.
- Routing and endpoint behavior.

For each resource:

- Replace the feature-specific file with the canonical template.
- Move variable values into Gherkin tables or context values.
- Preserve unique naming and version generation.
- Preserve omitted-versus-empty behavior.
- Keep all existing assertions strict.
- Re-home scenarios when their responsibility belongs in another feature.

Migration must be iterative by resource kind. For each kind:

1. List every existing resource file and every feature/step reference.
2. Design or refine the canonical template.
3. Convert every reference to the canonical template and structured overrides.
4. Run unit tests for the renderer and focused Docker-backed scenarios.
5. Search for stale references and verify the old-file count is zero for that kind.
6. Delete the obsolete files for that kind.
7. Run the focused scenarios again after deletion.
8. Record the completed kind and validation results in this plan.

Do not delete a resource file while any feature, runner, step, test, or documentation still references it.

The intended conversion order is:

1. `LlmProxy` — complete.
2. `Mcp` — complete.
3. `LlmProviderTemplate` — complete.
4. `LlmProvider` — complete.
5. `RestApi` — migrate by runner ownership, beginning with metrics, log-message,
   and startup-db-bootstrap.

Migration progress:

- `LlmProxy`: complete; all three initial files have been migrated and deleted.
- `Mcp`: complete; all five initial files have been migrated and deleted.
- `LlmProviderTemplate`: complete; all six initial files have been migrated and deleted.
- `LlmProvider`: complete; all thirteen initial files have been migrated and deleted.
- `RestApi`: complete; all 189 initial feature-specific files were migrated by
  runner batch and deleted after focused SQLite validation.

The final inventory must contain only the canonical templates and explicitly documented malformed-payload exceptions.

### 7. Migrate remaining resource kinds

Convert LLM providers, provider templates, LLM proxies, MCP proxies, and other supported resource kinds using the same renderer contract.

### 8. Handle intentional exceptions

Keep separate files only when one of the following is true:

- The resource kind differs.
- The wire schema genuinely differs.
- The test intentionally sends malformed or unparsable YAML.
- The request exercises a product contract that cannot be represented by the valid typed resource model.

Every exception must be documented and must not be created merely because a scenario has different values.

### 9. Remove duplicate resources

- Verify that every migrated feature references a canonical template.
- Search for references to obsolete files.
- Delete obsolete feature-specific resource files.
- Reject feature-named, runner-named, ordinal, and migration-specific resource files after migration.

### 10. Update standards and documentation

Update `CLAUDE.md` and the migration guideline to require canonical templates and structured overrides. Replace the current rule that effectively encourages one YAML file for every configuration variation.

Document the intentional exceptions for malformed-payload tests and genuinely different schemas.

## Validation sequence

Run the narrowest checks first:

```bash
gofmt -w <changed Go files>
go test ./tests/framework/suites/it/steps -count=1
go test -race ./tests/framework/suites/it/steps -count=1
go vet ./tests/framework/suites/it/steps
git diff --check
```

Then run:

- Focused feature tests for each migrated feature group.
- Docker-backed tests for each supported resource kind.
- Concurrent rendering tests with `go test -race`.
- The SQLite integration suite with the configured Docker/Testcontainers environment.
- Full framework unit tests and validation checks.
- A stale-reference and duplicate-resource scan.

The suite must preserve existing product assertions and cleanup behavior. A failed migration must be corrected rather than hidden by weakening assertions or skipping scenarios.

## Completion criteria

This temporary plan is complete when:

- Canonical templates cover all supported resource kinds.
- Scenario-specific resource YAML duplication has been removed.
- Optional fields can be omitted reliably.
- Complex policies and operations can be configured without feature-specific resource files.
- Renderer unit, race, and integration tests pass.
- All migrated feature files use canonical templates.
- No stale resource references remain.
- `CLAUDE.md` and the migration guideline contain the permanent rule.
- The full relevant suite passes.

After these criteria are met, update permanent documentation and delete this file.
