# Coverage architecture

The framework collects runtime coverage from images built from the checked-out source.
Coverage is enabled at build time and is not exposed through a product HTTP endpoint.

## Image modes

| Version source | Build behavior | Coverage behavior |
|---|---|---|
| Suite YAML or `-gateway-version` | Do not build; use the requested image | `-gateway-version` cannot be combined with `-coverage` |
| Product `VERSION` | Build from the checkout through `core/builder` | Images are instrumented only when `-coverage` is selected; image names stay unchanged |

The catalog supplies one build specification per source-built product. Each specification owns
its supported formats, Go package patterns, JavaScript include patterns, build arguments, and
runtime environment. Product Dockerfiles are the canonical Dockerfiles and are never copied
into the framework. An explicit product version selects an existing image and cannot be combined
with `-coverage`.

## Collection flow

1. The framework passes instrumentation arguments to the product's canonical Dockerfile.
2. The framework passes `GOCOVERDIR` and/or `NODE_V8_COVERAGE` only in coverage mode.
3. The suite runs blocks concurrently on isolated networks.
4. Coverage services receive a graceful stop request after the block finishes.
5. The framework copies each service's coverage directory from the stopped container.
6. Data is stored under separate block and service directories.
7. `tools/coverage-report.sh` merges Go counters, Node/V8 artifacts, and browser artifacts
	independently. It writes Go text/HTML output, a c8 text/HTML/JSON/LCOV report, and browser
	Istanbul JSON/HTML/LCOV output. Browser output also has product-specific directories such as
	`ai-workspace/ui` and `api-portal/ui`.

Collection is attempted independently for every service. A collection error is reported with
the block and service name and does not prevent sibling cleanup. Forced termination can lose
counters that were not flushed before the process was killed.

## Product capabilities

- Platform Gateway: Go controller and policy engine; build-time Go coverage is supported.
- Platform API: Go service with atomic coverage counters.
- AI Workspace: Go BFF plus source-instrumented browser JavaScript. Istanbul counters are
	collected from the Playwright page and mapped back to `portals/ai-workspace/src`.
- API Portal: Node.js service with Node/V8 coverage plus source-instrumented browser
	JavaScript. c8 reports the server separately, while Istanbul counters are mapped to
	`portals/api-portal/src/scripts`.

Browser coverage is collected per scenario under
`raw/blocks/<block>/browser/<scenario>/raw-istanbul.json`. Source-level Istanbul reports are
required and produce Codecov-compatible
LCOV, machine-readable JSON, and HTML reports. Third-party URLs and Playwright-injected
anonymous scripts are excluded. Source inventories add unvisited product files to the
denominator, so a report is not limited to loaded bundles.

Go HTML generation filters out generated or external source files that are not present in the
checkout. The complete profile and statement totals are retained separately for upload to a
coverage service.

The framework must fail clearly when a requested coverage mode is not supported by a product;
it must never label an ordinary image as instrumented.

`IT_COVERAGE_OUT` controls where suites write collected artifacts. `COVERAGE_OUT` controls
which artifact directory `tools/coverage-report.sh` reads; set both when using a non-default
directory.

## Report layout

The report command creates a navigable dashboard at `<coverage-root>/index.html`. Component
reports are grouped by product and service:

```text
<coverage-root>/
├── index.html
├── platform-gateway/{controller,runtime}/
├── platform-api/
├── ai-workspace/{bff,ui}/
├── api-portal/{server,ui}/
└── raw/                       # collected block/service artifacts, when present
```

Each component directory contains its own machine-readable coverage data and HTML report. The
root dashboard links to every available component report and omits components that did not
participate in the run. The combined browser report remains under `browser-report/` for
cross-product analysis; it is not used as a substitute for component reports.
