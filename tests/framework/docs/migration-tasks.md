# Legacy Feature Migration Tasks

This is the execution checklist for migrating legacy product scenarios into
`tests/framework/suites/it`. The migration is restricted to test and framework code.

## Authoritative source

The only authoritative legacy input is the repository-local directory
`gateway/it/features`. Every migration must begin by reading the corresponding feature from
that directory. Do not use external copies, backups, scratch directories, generated files, or
previous migration outputs as source material.

## Migration baseline

- [ ] Read each legacy feature from `gateway/it/features` and its referenced step definitions.
- [ ] Preserve the legacy scenario and assertion inventory unless redundancy is demonstrated.
- [ ] Record every rehomed, removed, skipped, or strengthened scenario in the audit ledger.
- [ ] Keep framework-concurrency and runner-isolation tests out of the product suite YAML.

## Lessons from initial runner verification

| Lesson | Practical implication |
|---|---|
| Product response schemas vary by gateway version | Use version-aware product assertions instead of generic assertions that assume one response shape. |
| Legacy API contexts may contain `/$version` | Preserve context semantics during migration; do not automatically append `/v1.0` to every invocation URL. |
| Reusable YAML templates require complete value tables | Validate every `${VALUE:...}` placeholder before running scenarios. |
| Focused scenario runs are not full feature verification | A selected scenario passing does not prove that the complete runner passes. |
| Gateway startup was healthy and fast | Distinguish startup failures from readiness or route-propagation failures. |
| Unique names and contexts are essential | Fixed resource names can collide across parallel runners and obscure failures. |
| Migration must compare behavior, not only Gherkin text | Differences in contexts, paths, and response contracts can change runtime behavior. |
| Framework changes should remain separate from product changes | Migration fixes should be limited to test resources, feature files, and framework bindings. |
| Final verification must include all databases | SQLite success alone does not establish PostgreSQL and SQL Server compatibility. |
| Migration records must be updated immediately | Keep task and audit documents synchronized so completed work is not reported as pending. |

## API lifecycle and management

- [ ] Migrate `api-error-responses.feature`.
- [ ] Migrate `api-management.feature`.
- [ ] Migrate `api_deploy.feature`.
- [ ] Migrate `search-deployments.feature`.
- [ ] Migrate `api-keys.feature`.
- [ ] Migrate `api-with-policies.feature`.
- [ ] Migrate `startup-db-bootstrap.feature`.
- [ ] Migrate `lazy-resources-xds.feature`.

## Health, reliability, and service behavior

- [x] Migrate `health.feature`.
- [x] Migrate `backend-timeout.feature`.
- [x] Migrate `llm-backend-timeout.feature`.
- [x] Migrate `interceptor-service.feature`.
- [x] Migrate `metrics.feature`.
- [ ] Migrate `log-message.feature`.

## Per-feature requirements

- [ ] Keep each scenario within the owning product capability and complete its user journey.
- [ ] Move behavior to the feature that owns it when the legacy grouping is wrong.
- [ ] Use reusable YAML resources and Gherkin tables; never embed YAML configuration in features.
- [ ] Generate every test-owned name, context, route, and identifier uniquely.
- [ ] Register cleanup at creation time with the correct scenario or runner lifetime.
- [ ] Replace fixed sleeps with bounded readiness or propagation polling.
- [ ] Reuse existing steps and route HTTP calls through the approved funnels.
- [ ] Remove dead steps, duplicate mechanics, and historical comments.
- [ ] Strengthen assertions within the scenario’s stated user journey without relaxing existing
  assertions.

## Validation for every migrated feature

- [ ] Run the feature-specific standards check.
- [ ] Run focused unit tests, `go test -race`, and `go vet`.
- [ ] Run the focused SQLite integration feature.
- [ ] Run the complete SQLite, PostgreSQL, and SQL Server matrix.
- [ ] Verify cleanup leaves no test-owned resources or generated residue.
- [ ] Run `gofmt` and `git diff --check`.

## Final validation

- [ ] Register each migrated feature exactly once in `it-suite.yaml`.
- [ ] Confirm framework-only tests are absent from product suite YAML.
- [ ] Run the framework unit, race, vet, and integration checks.
- [ ] Review the complete diff and search for stale paths, names, comments, and registrations.
- [ ] Complete the audit ledger and deletion ledger.

## Routing and endpoint behavior migration

The authoritative source for this phase remains `gateway/it/features`. The target is
the framework suite under `tests/framework/suites/it`; product source files are out of
scope.

### Inventory and ownership

- [x] Inventory the 13 routing and endpoint feature files and their 118 scenarios.
- [x] Trace every legacy step to its repository-local step definition and product behavior.
- [x] Assign each scenario to one behavioral owner before migration.
- [x] Record every rehomed, removed, skipped, and strengthened scenario in the review ledger.

### Target behavioral groups

- [x] Endpoint selection: `dynamic-endpoint.feature`.
- [x] Path matching and normalization: `route-path-matching.feature` and
  `path-normalization.feature`.
- [x] Header and host selection: `header-routing.feature`, `host-rewrite.feature`,
  and the vhost features.
- [x] Sandbox routing: `sandbox-routing.feature`.
- [x] Request transformation: `request-rewrite.feature`.
- [x] CORS: `cors.feature`.
- [x] Response behavior: `respond.feature`.
- [x] Redirect behavior: `redirect.feature`.
- [x] Upstream and gateway timeouts: `upstream-connect-timeout.feature`.
- [x] Create focused feature files when a scenario does not belong to any existing owner.

### Migration requirements

- [x] Preserve the complete legacy scenario and assertion inventory unless redundancy is
  demonstrated and recorded.
- [x] Replace inline YAML with reusable resource files and Gherkin value tables.
- [x] Generate all test-owned names, contexts, hosts, and identifiers uniquely.
- [x] Register every created resource at creation time with the correct cleanup lifetime.
- [x] Replace fixed sleeps with bounded polling or deterministic socket coordination.
- [x] Reuse generic request, invocation, assertion, JSON, and polling steps.
- [x] Complete each user journey through deployment/readiness, invocation, assertions, and
  cleanup where those behaviors are part of the scenario.
- [x] Keep framework scheduling, runner isolation, and concurrency tests out of product suite
  YAML.

### Validation gate

- [ ] Run the feature-specific standards checker after each migration group.
- [ ] Run focused unit tests, race tests, and `go vet` for changed framework packages.
- [ ] Run each complete migrated feature with SQLite.
- [ ] Run the routing feature group across SQLite, PostgreSQL, and SQL Server.
- [ ] Verify cleanup and search for stale legacy paths, names, comments, and registrations.
- [ ] Run formatting and `git diff --check`.
