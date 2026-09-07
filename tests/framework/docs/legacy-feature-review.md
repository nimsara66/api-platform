# Legacy Feature Review

This ledger records evidence for migrations from the repository-local
`gateway/it/features` tree. That directory is the sole authoritative legacy source. No
migration entries have been completed in the current iteration.

## Review record

Add one section per feature containing:

- authoritative legacy and framework paths;
- legacy and migrated scenario counts;
- rehomed, removed, skipped, and added scenarios with reasons;
- reusable resources and cleanup ownership;
- focused and database-matrix validation results;
- remaining product or framework gaps.

## Deletion ledger

Record every removed scenario, assertion, step, or resource and its reason. Do not remove
coverage without proving redundancy or documenting an explicit product-contract exception.

## Dynamic endpoint routing

- Authoritative source: `gateway/it/features/dynamic-endpoint.feature`.
- Framework feature: `tests/framework/suites/it/features/dynamic_endpoint.feature`.
- Scenario count: 8 legacy, 8 migrated.
- Configuration: the canonical `resources/templates/rest-api.yaml` is parameterized through
  Gherkin tables.
- Isolation: API names, contexts, and configured vhosts use runner-local generated values.
- Assertions: operation routing, upstream base paths, sandbox behavior, policy composition,
  and invalid required-policy parameters are retained.
- Rehoming: none; all scenarios own dynamic endpoint routing behavior.
- Validation: focused Docker validation is tracked in the resource-template consolidation plan.

## Routing and endpoint behavior

- Authoritative source: `gateway/it/features`.
- Framework features: `tests/framework/suites/it/features`.
- Scenario inventory: 118 legacy scenarios across 13 feature files; all 118 remain represented
  in the framework suite.
- Configuration: routing scenarios use the canonical
  `tests/framework/suites/it/resources/templates/rest-api.yaml` and Gherkin value tables. No
  routing feature embeds YAML configuration.
- Isolation: test-owned API names and contexts are generated per scenario; custom vhosts are
  generated where the API owns them. Gateway default vhosts remain fixed only where the
  scenario explicitly verifies the product default-vhost contract.
- Ownership: dynamic endpoint, path matching, host/vhost routing, sandbox routing, request
  transformation, CORS, response, redirect, and timeout scenarios remain in their matching
  feature files.
- Timeout support: the incomplete-header scenario uses the shared raw HTTP step and publishes
  its response through the HTTP funnel.
- Rehoming/removal: no scenarios were removed or weakened. Duplicate waits and historical
  comments were removed as mechanics; assertions were retained.
- Validation: focused unit tests, feature standards checks, suite-shape checks, formatting,
  diff checks, and focused Docker-backed execution are tracked with the migration batch records.
