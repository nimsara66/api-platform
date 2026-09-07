# Legacy Feature Migration Guide

This guide defines how a legacy `gateway/it` feature becomes a framework integration
feature. The migration must preserve coverage, strengthen weak assertions, and keep
product behavior separate from framework behavior.

## 1. Understand the legacy behavior

- Read the complete legacy feature file.
- Read every legacy step definition it uses.
- Trace each step to the actual product or API behavior.
- Identify the intended user journey and its completion point.
- Record the original scenario count.
- Determine whether each scenario tests product behavior or framework behavior.
- Move framework-only tests to framework unit or integration tests, not `it-suite.yaml`.

## 2. Validate scenario intent

- Ensure feature and scenario names describe the asserted behavior.
- Ensure every step contributes to the scenario's purpose.
- Ensure the scenario reaches the end of its user journey.
- Extend partial journeys where appropriate.
- Keep API lifecycle journeys through readiness, invocation, response validation, and cleanup.
- Split scenarios that test unrelated capabilities.
- Remove unrelated setup and assertions.

## 3. Preserve and strengthen assertions

- Never weaken an existing assertion to make migration pass.
- Preserve all meaningful legacy assertions.
- Assert management-plane responses and persisted state where relevant.
- Assert deployment and readiness before invocation.
- Assert invocation status, body, headers, and routing behavior where applicable.
- Assert negative status and error details explicitly.
- Validate every declared operation when the scenario concerns multiple operations.
- Add malformed-input, missing-value, duplicate, unsupported-value, and boundary cases when
  they protect meaningful product behavior.
- Verify failed operations do not leave unintended resources or routes.
- Keep assertions within the scenario's responsibility.

### Version-aware assertions

- Write Gherkin steps in version-agnostic, behavior-oriented language. Do not
  expose a response representation such as a string or object in the step when
  the user behavior is the same across product versions.
- Resolve the effective product version per block and inject it into the
  block context available to Gherkin bindings before scenarios execute.
- Implement version-specific request and response handling in the relevant
  product step package. The binding must explicitly support every version the
  framework supports and fail clearly for an unsupported version.
- Keep generic JSON assertions strict. They must not silently coerce or accept
  different JSON types to hide a product contract difference.
- Add unit coverage for every supported version contract and for missing,
  malformed, and unsupported version information.

## 4. Use reusable configuration

- Never embed API YAML or other product configuration in a feature file.
- Store configurations in reusable YAML or other resource files that form a
  shared artifact and capability catalog.
- Use Gherkin tables or parameterized values for scenario-specific data.
- Name resources after the product artifact, capability, or contract behavior
  they represent, not after a feature file, scenario, runner, migration, or
  ordinal number. Names such as `rest-api-basic.yaml`, `mcp-proxy.yaml`,
  `llm-provider.yaml`, and `cors-preflight.yaml` are preferred.
- Reuse an existing resource across features whenever its payload structure and
  tested contract are the same. Parameterize scenario-specific values instead
  of copying the resource.
- Create a separate resource only when the payload structure or contract
  behavior genuinely differs. Malformed and negative configurations remain
  valid separate resources when named after the condition they represent, such
  as `rest-api-missing-context.yaml`.
- Avoid feature-specific prefixes and suffixes such as `management-`,
  `lazy-3-1-`, or `-1.yaml`.

### Testbench infrastructure

- Do not introduce custom Docker containers to mock backends during migration.
- Reuse an existing testbench service whenever it provides the required
  behavior.
- If no existing service is sufficient, add the required mock backend to the
  testbench and provide its service implementation, container endpoint,
  readiness check, and framework catalog registration.
- Keep product scenarios dependent only on services declared by the framework
  topology.

### Infrastructure and contract fidelity

- Treat the repository-local legacy feature and the current framework catalog as two
  separate inputs: the legacy file defines the behavior to preserve, while the catalog
  defines the services and addresses available to the migrated test.
- Audit every upstream URL during migration. Do not carry legacy container names, ports,
  paths, or network aliases into a framework resource. Resolve the target through the
  registered testbench service and verify its handler path and response shape.
- Verify each resource file independently before running the feature. Its `kind`, API
  version, context, operation path, upstream URL, upstream base path, and policy references
  must agree with the step that creates it and with the testbench service it invokes.
- Preserve context semantics exactly. If the legacy context contains `$version`, retain it
  and invoke the expanded path accordingly. Do not append a version segment merely because
  another migrated feature uses one; do not remove a legacy version placeholder without
  checking the resulting management and invocation paths.
- Do not replace dynamic framework values with legacy literals. Container-published ports,
  generated names, host headers, redirect locations, and testbench response URLs must be
  asserted through generated values, patterns, or stable response fields.
- When a scenario uses multiple upstreams, routes, or listeners, establish readiness for
  the route needed by the scenario before asserting the alternate route. A successful
  management response is not data-plane readiness, and a generic component or snapshot
  readiness check is not proof that the expected resource or route is visible.
- Poll the exact observable condition under test. For lazy resources, poll the config dump
  until the named resource has the expected type, mapping, display value, or absence. For
  routes, poll the expected host/path/status combination. A version or snapshot equality
  check alone is insufficient evidence that the payload has propagated.
- Give a runner its own block and overlay when it restarts or mutates a shared component,
  or requires a product configuration that other runners must not inherit. Do not create a
  separate block when the behavior is already covered by the common defaults.
- Enable required diagnostic endpoints, such as controller and policy-engine config dumps,
  in the framework overlay when the migrated feature asserts propagated state. Keep those
  settings scoped to the relevant test topology.

## 5. Enforce isolation

- Use unique names for every test-owned API, context, version, policy, application,
  credential, host, and other resource.
- Do not rely on different names per runner as the only isolation mechanism.
- Use the framework unique-name mechanism consistently.
- Do not depend on another scenario or runner's data.
- Keep setup fixtures safe to reuse within one runner.
- Prevent parallel runners from colliding in names, routes, files, or external state.
- Do not use shared mutable global state.

## 6. Define cleanup ownership

- Register every created resource immediately after successful creation.
- Use scenario-level cleanup for self-contained scenarios.
- Use runner-level cleanup for reusable setup fixtures shared by later scenarios.
- Ensure cleanup runs after success and failure.
- Verify APIs, routes, policies, applications, subscriptions, tenants, keys, and other
  created resources are removed.
- Do not use inline teardown as the primary cleanup mechanism.
- Ensure cleanup registration is neither duplicated nor omitted.
- Preserve contextual cleanup failure logging and bounded retry behavior.

## 7. Reuse and simplify steps

- Search existing step definitions before adding a new step.
- Reuse an existing generic step where possible.
- Parameterize or extend a near-fit step instead of duplicating it.
- Keep Gherkin steps behavior-oriented and readable.
- Avoid multiple step variations for the same operation.
- Route management-plane HTTP calls through the approved request funnel.
- Route gateway invocations through the approved invocation funnel.
- Reuse common request, assertion, JSON, URL, and polling helpers.
- Remove dead steps, methods, registrations, and stale references completely.
- Keep product-specific steps in the relevant product package.

## 8. Use deterministic synchronization

- Replace sleeps with polling on the condition actually required.
- Use framework retry utilities for eventual consistency.
- Poll for deployment, route availability, policy propagation, and backend readiness.
- Use bounded timeouts with useful failure messages.
- Do not assume a fixed local startup duration.
- Verify readiness before sending traffic.

## 9. Organize features appropriately

- Separate reusable setup features from user-journey features where appropriate.
- Keep setup fixtures clearly identifiable and reusable.
- Keep each scenario within its feature's capability boundary.
- Move routing scenarios to routing features.
- Move API lifecycle scenarios to `api_management.feature`.
- Move policy-application scenarios to `api_with_policies.feature`.
- Keep analytics, health, authentication, and other concerns in their respective features.
- Do not add runner-isolation or framework-concurrency tests to the product suite YAML.

## 10. Apply annotations and selection rules

- Add capability and scenario annotations when that migration phase begins.
- Use descriptive skip annotations for known product-contract gaps.
- Do not hide framework failures with product-scenario skip tags.
- Document intentionally skipped scenarios with their reason and expected contract.
- Ensure runner tags select only the intended scenarios.

## 11. Verify implementation correctness

For every migrated feature:

- Run the standards checker against the specific feature. In this Go repository,
  this is the PMD/Checkstyle-equivalent migration gate:

  ```bash
  go run ./cmd/standards \
    -root . \
    -unit-root core \
    -steps suites/it/steps \
    -features suites/it/features/<feature>.feature \
    -scripts tools \
    -suites suites/it/it-suite.yaml,suites/ui/ui-suite.yaml \
    -docs core
  ```

  Do not proceed while the checker reports an issue.
- Confirm there is no inline YAML.
- Confirm resource names describe reusable artifacts, capabilities, or contract
  conditions rather than feature files or scenarios.
- Confirm an existing resource was reused before a new resource was added and
  that equivalent resources are not duplicated across features.
- Confirm recognized resource fields use unique naming.
- Confirm there is no `time.Sleep`, shell sleep, or ad hoc polling.
- Confirm approved HTTP, retry, and parser utilities are used.
- Confirm cleanup registration is present.
- Confirm framework-only tests are absent from `it-suite.yaml`.
- Run `gofmt -d` and format changed Go files with `gofmt -w` when needed.
- Run `go test ./path/to/changed/package -count=1`.
- Run `go test -race ./path/to/changed/package -count=1`.
- Run `go vet ./path/to/changed/package`.
- Run the focused runner with SQLite first.
- Run the complete feature file, not only selected scenarios.
- Run the relevant database matrix after the focused run passes.
- Use the concurrency declared by `it-suite.yaml` for normal verification; do not pass
  ad hoc block or runner parallelism values unless diagnosing a framework scheduling issue.
- Use block selection to isolate topology. A Gherkin tag filter selects scenarios but does
  not by itself isolate shared components or prevent unrelated runners from being scheduled.
- Treat startup longer than the expected minute-scale boot as a diagnostic failure. Stop a
  run that approaches its timeout, inspect the active containers and last readiness response,
  and fix the cause instead of increasing the timeout or allowing repeated propagation polls
  to conceal the original response.
- Separate scenario results from process status. A run may report all product scenarios
  passed while a suite-shape, scheduling, or other harness test fails; record both results
  and fix the harness failure before declaring the run green.
- Run `git diff --check` and review the complete diff.
- Search for stale names, comments, registrations, and references.
- Classify every failure as a migration/framework, testbench, environment, or product issue.
- Do not modify product source code during feature migration.

## 12. Complete the migration record

Record the following in `legacy-feature-review.md`:

- Legacy and new feature paths.
- Original and migrated scenario counts.
- Added scenarios and assertions.
- Intentionally skipped scenarios.
- Reusable templates introduced.
- Cleanup ownership.
- Focused, full-feature, and database-matrix results.
- Known product-contract gaps.
- Remaining follow-up work.

A feature is migrated only when its behavior has been traced, its scenarios form complete
user journeys, its assertions are sufficiently strong, its resources are isolated and
cleaned up, its configuration is reusable, and its validation results are passing or have
documented product-contract exceptions.
