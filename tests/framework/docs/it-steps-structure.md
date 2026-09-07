# Integration Step Structure

The integration suite keeps common Gherkin behavior separate from product-specific
step definitions. `base.go` remains the entry point for registering the suite's
steps and product packages.

## Proposed layout

```text
suites/it/steps/
├── base.go                  # Suite entry point and common step registration
├── request.go               # Common request state and execution
├── assertions.go            # Common response and JSON assertions
├── health.go                # Common health assertions
│
├── common/
│   ├── request.go           # Shared request helpers
│   ├── json.go              # Shared JSON helpers
│   └── polling.go           # Shared polling helpers
│
└── gateway/
    ├── gateway.go           # Gateway registration and orchestration
    ├── endpoint.go          # Gateway endpoint resolution
    ├── health.go            # Gateway health probes
    ├── resource.go          # API and controller resources
    ├── policy.go            # Policy synchronization and config-dump waits
    └── template.go          # Out-of-box template comparison
```

## Responsibilities

The root `steps` package owns suite construction and common Gherkin steps. The
`common` package contains reusable test mechanics and must not depend on gateway
terminology. The `gateway` package contains gateway-specific endpoints, resources,
health checks, policy behavior, and template assertions.

Future products can add sibling packages such as `platformapi`, `aiworkspace`, and
`apiportal` without expanding the common package with product-specific behavior.

## Dependency direction

```text
base.go
   └── gateway.Register(...)
          └── steps/common
                 └── framework core packages
```

The gateway package must not import the parent `steps` package. Shared dependencies
should be passed explicitly through small dependency structures or interfaces. This
avoids import cycles and makes each product package independently testable.

The structure separates reusable mechanics from product semantics, keeps the suite
entry point small, and provides a consistent extension point for additional products.

## API templates

Reusable API definitions belong under `suites/it/resources/apis`. Template files use
`${VALUE:key}` for scenario values. The `I create API from` and `I update API from`
steps accept a two-column table, apply those values, and then resolve the standard
`${UNIQUE:...}` and `${CTX:...}` placeholders. Malformed tables, unresolved values,
absolute paths, and paths outside the suite root are rejected.

Keep intentionally malformed payloads inline when the payload itself is the subject
of the test. Use templates for valid definitions and repeated setup or update shapes.
