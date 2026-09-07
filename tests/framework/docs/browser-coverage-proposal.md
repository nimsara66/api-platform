# Browser UI Coverage Architecture

> Superseded: the framework now requires source-level Istanbul instrumentation and collects
> `raw-istanbul.json` from instrumented pages. Chromium CDP/V8 conversion is not supported.

## Objective

The framework provides true browser-side JavaScript coverage for AI Workspace and API Portal. The implementation is reusable, concurrency-safe, source-map aware, and produces Codecov-compatible and human-readable reports.

The existing Playwright browser lifecycle is the correct integration point. Playwright drives the journeys, while the framework owns coverage lifecycle, isolation, conversion, filtering, and reporting.

## Execution status

The shared collector and reporting pipeline are implemented. AI Workspace and API Portal browser
coverage have been exercised through Docker-backed tests, including source-map resolution,
third-party filtering, and concurrent scenario artifact isolation. Raw artifacts are stored
below `raw/blocks/` so generated reports and collected inputs are clearly separated.

## Coverage model

The framework distinguishes coverage by execution environment:

| Coverage type | Execution environment | Examples |
|---|---|---|
| Go | Go process | Gateway controller, Platform API, AI Workspace BFF |
| Node V8 | Node.js process | API Portal server |
| Browser JavaScript | Chromium page or browser context | AI Workspace UI, API Portal UI |

Add a browser-specific coverage type, for example:

```go
const (
    GoCoverage        CoverageType = "go"
    NodeV8Coverage    CoverageType = "node-v8"
    BrowserJSCoverage CoverageType = "browser-js"
)
```

Browser coverage must not be reported as BFF or Node-server coverage. Product summaries should expose these independently.

## Shared browser coverage contract

The catalog should declare browser coverage capabilities without containing collection logic:

```go
type BrowserCoverageSpec struct {
    Supported     bool
    Include       []string
    Exclude       []string
    SourceRoot    string
    SourceMapRoot string
    OutputDir     string
}
```

Example product configuration:

```go
Browser: builder.BrowserCoverageSpec{
    Supported:  true,
    SourceRoot: "portals/ai-workspace/src",
    Include:    []string{"**/*.js", "**/*.jsx", "**/*.ts", "**/*.tsx"},
    Exclude:    []string{"**/*.test.*", "**/node_modules/**"},
    OutputDir:  "/coverage/browser",
},
```

AI Workspace and API Portal use the same contract with different source roots and filters.

## Browser instrumentation and collection

The existing UI lifecycle is already suitable:

1. The `Before` hook creates a browser context and page.
2. Browser coverage starts immediately after page creation.
3. Existing Playwright steps execute the user journey unchanged.
4. The `After` hook collects coverage before closing the page and context.

The framework uses source-level Istanbul instrumentation through the existing Playwright connection:

Instrumented pages expose `globalThis.__coverage__`. The `After` hook collects that object
before the page closes and writes it as `raw-istanbul.json`. Missing instrumentation fails
the coverage run.

## Concurrency and artifact isolation

UI scenarios and blocks may run concurrently. No active test may append to a shared coverage file.

Recommended output layout:

```text
coverage-out/
└── blocks/
    └── <block-id>/
        └── browser/
            └── <runner-id>/
                └── <scenario-id>/
                    ├── raw-istanbul.json
                    ├── metadata.json
                    └── sources/
```

The collector should:

- assign every scenario a unique output directory;
- create directories safely under concurrent execution;
- write temporary files and rename them after completion;
- record block, runner, scenario, product, and browser metadata;
- continue collecting other scenarios if one collection fails;
- report collection errors with their block, runner, and scenario identity.

## Conversion and source maps

Chrome DevTools Protocol output is V8 coverage, not LCOV. The framework-owned conversion tool is under:

```text
tests/framework/tools/coverage/
├── package.json
├── package-lock.json
├── browser-to-istanbul.js
└── normalize-v8.js
```

The converter:

1. Read all scenario-level Istanbul coverage files.
2. Resolve instrumented source paths to repository source files.
3. Merge statement, function, and branch counters.
5. Apply product include and exclude rules.
6. Merge all scenarios and runners.
7. Generate Istanbul JSON, LCOV, and HTML.

The denominator must contain product-owned source statements, not third-party dependencies, Playwright helpers, injected test scripts, CDN scripts, or anonymous evaluation scripts.

## Product source configuration

| Product | Browser source root | Included files |
|---|---|---|
| AI Workspace | `portals/ai-workspace/src` | `.js`, `.jsx`, `.ts`, `.tsx` |
| API Portal | `portals/api-portal/src/scripts` | Browser JavaScript sources |

The framework fails clearly if all captured scripts are excluded. Source inventories include
unvisited product files so the denominator is not limited to loaded bundles.

## Navigation and browser targets

Coverage must account for navigations because browser coverage may reset during navigation. The safe design is to collect coverage before every page or context closes and merge all fragments afterward.

The implementation should define support for:

- the primary page;
- additional pages and popup windows;
- same-origin frames;
- cross-origin frames;
- web workers;
- service workers.

The first implementation may focus on the primary page and same-browser page targets, but unsupported targets must be reported explicitly rather than silently included in a claim of complete UI coverage.

## Report outputs

Each browser coverage result should contain:

```text
coverage-out/
└── browser-report/
    ├── coverage-final.json
    ├── lcov.info
    ├── html/
    │   └── index.html
    └── summary.json
```

The report directory also contains product-specific reports under
`browser-report/ai-workspace-ui/` and `browser-report/api-portal-ui/` when those products
participate in the run. Raw scenario artifacts remain under
`raw/blocks/<block>/browser/<scenario>/raw-istanbul.json`.

Outputs have separate purposes:

- `lcov.info`: upload to Codecov;
- `coverage-final.json`: machine-readable Istanbul data;
- `html/index.html`: source-level report for developers;
- `summary.json`: statement, branch, function, and line totals.

Codecov flags should distinguish products and execution environments, for example:

```text
ai-workspace-ui
ai-workspace-bff
api-portal-ui
api-portal-server
```

This prevents the AI Workspace BFF percentage from being presented as overall product UI coverage.

## Responsibility boundaries

| Responsibility | Framework | Catalog | Product |
|---|---:|---:|---:|
| Start and stop browser coverage | Yes | No | No |
| CDP communication | Yes | No | No |
| Scenario and block isolation | Yes | No | No |
| V8-to-Istanbul conversion | Yes | No | No |
| LCOV, JSON, and HTML generation | Yes | No | No |
| Source roots and include/exclude patterns | No | Yes | No |
| Source-map availability and frontend build configuration | No | No | Yes |
| UI journeys | No | No | Yes |
| Product-specific coverage hooks | No | No | No |

Products should not contain custom coverage collection logic. They must preserve usable source maps and make the built frontend sources resolvable by the framework converter.

## Implementation tasks

1. [x] Add `BrowserJSCoverage` and `BrowserCoverageSpec` to the shared coverage contract.
2. [x] Add catalog declarations for AI Workspace and API Portal browser source roots.
3. [x] Add a reusable collector around the existing Playwright page lifecycle.
4. [x] Add raw CDP serialization with scenario isolation.
5. [x] Add the V8-to-Istanbul conversion tool.
6. [x] Add source-map resolution and include/exclude handling.
7. [x] Add LCOV, JSON, and HTML generation.
8. [x] Add unit tests for conversion, filtering, missing files, and error handling.
9. [x] Add concurrency tests for multiple scenarios and blocks.
10. [x] Add Docker-backed browser coverage tests for AI Workspace and API Portal.
11. [x] Verify that backend and browser coverage remain separate in summaries.
12. [x] Verify artifact isolation, report validity, repository-relative source paths, and Codecov-compatible LCOV.

## Acceptance criteria

The implementation is complete when:

- AI Workspace produces separate UI and BFF coverage results.
- API Portal produces separate UI and Node-server coverage results.
- browser coverage is collected through the existing Playwright browser session;
- parallel blocks and runners do not overwrite one another's artifacts;
- source maps resolve generated bundles to repository sources;
- LCOV paths are valid for Codecov;
- HTML reports show product-owned source files;
- missing or unsupported coverage is reported explicitly;
- unit, race, integration, and concurrent UI tests pass.

## References

- [Playwright JavaScript and CSS coverage](https://playwright.dev/docs/next/api/class-coverage)
- [Chrome DevTools Protocol Profiler domain](https://chromedevtools.github.io/devtools-protocol/1-3/Profiler/)
