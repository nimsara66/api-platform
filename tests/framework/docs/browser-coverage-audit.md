# AI Workspace Browser Coverage Audit

Date: 2026-09-02

## Scope

This audit manually reviews the clean AI Workspace browser coverage report produced
by the UI suite. The original audit checked whether the 3,151 executed statements map to the tested
browser journeys and whether the 18,463-statement denominator includes the intended
first-party source inventory.

The navigation-boundary fix described below was subsequently implemented and
verified by a second clean UI-suite run. The post-fix report contains 3,166
executed statements.

The report is available at:

- `suites/ui/coverage-out/ai-workspace/ui/coverage-final.json`
- `suites/ui/coverage-out/ai-workspace/ui/summary.json`
- `suites/ui/coverage-out/ai-workspace/ui/index.html`

## Original audit results

| Check | Result | Assessment |
| --- | ---: | --- |
| AI Workspace source files | 234 | All non-test `.ts` and `.tsx` files are included |
| Files in `coverage-final.json` | 234 | Exact match with the source inventory |
| Files outside `portals/ai-workspace/src` | 0 | No dependency or generated-bundle contamination |
| Invalid statement source locations | 0 | Every statement maps to a valid source line |
| Raw Istanbul reports | 6 | One report per UI scenario |
| Union of covered statements from raw reports | 3,151 | Exactly matches the final report |
| Total statements | 18,463 | Internally consistent |
| Statement coverage | 3,151 / 18,463 = 17.06% | Correct for the current report definition |

The raw reports were independently unioned by source file and statement identifier.
The result was exactly 3,151 covered statements, so the aggregation does not
double-count statements or invent coverage.

## Journey mapping

| Scenario | Browser interaction | Coverage behavior | Statements reported |
| --- | --- | --- | ---: |
| Sign-in page renders | Opens the workspace and checks the username and password fields | The first step is reset afterward, leaving no measured journey step | 0 |
| Administrator signs in | Opens the page, enters credentials, clicks Sign In, and reaches home | Login performs a full-page navigation, replacing the page coverage object | 908 |
| Signed-in session is reusable | Opens the authenticated workspace and checks organization home | Authenticated application execution is collected | 908 |
| Provider/proxy lifecycle | Creates a project and provider, deploys them, creates and deploys a proxy, invokes the completion, and deletes the proxy | SPA interactions and API-backed UI paths are measured | 2,524 |
| Runtime configuration branch | Loads the workspace for setup, resets coverage, loads it again, and checks the sign-in form | The runtime-configuration path is exercised after reset | 440 |
| Runtime configuration fallback | Intercepts `runtime-config.js`, loads the workspace, checks the sign-in form, and verifies fallback configuration | The missing-runtime-configuration path is exercised after reset | 431 |

Scenario totals are not additive because scenarios cover many of the same
statements. Their union is exactly 3,151.

## Covered source areas

The largest covered areas align with the provider/proxy journey:

| Source area | Covered / total statements | Explanation |
| --- | ---: | --- |
| Provider overview | 231 / 396 | Provider overview, OpenAPI parsing, deployment, and navigation |
| Provider overview tab | 190 / 313 | Provider gateway, deployment, and API-key sections |
| Proxy overview tab | 169 / 253 | Proxy deployment, API-key, OpenAPI, and invocation sections |
| New proxy page | 148 / 255 | Proxy creation form and validation |
| Gateway deployment context | 145 / 249 | Deployment polling and gateway state |
| Swagger viewer | 136 / 232 | Provider and proxy API specification rendering |
| Provider creation page | 91 / 154 | Provider template and creation workflow |
| Application shell | 67 / 128 | Workspace navigation and authenticated shell |
| Projects list | 34 / 52 | Project creation and project-list verification |

The report leaves substantial unrelated areas uncovered, including applications,
MCP pages, gateway management, external servers, settings, and provider-template
administration. This is consistent with the scenarios and is evidence that the
instrumentation is not marking the entire application as executed.

## Branch verification

The deterministic runtime-configuration scenarios exercise both sides of the
runtime configuration decision in
`portals/ai-workspace/src/utils/getEnvOrDefault.ts`.

The resulting branch counters include both-sided execution, for example:

```text
[12, 68]
[80, 12, 12]
[0, 12]
```

This confirms that executed and unexecuted branches are preserved. It does not
mean that every branch in the product is tested.

## Lifecycle issue and resolution

The initial audit exposed a coverage continuity problem. The product performs a
full-page navigation through `window.location.replace` in
`portals/ai-workspace/src/AIWorkspace.tsx`.

A full navigation creates a new JavaScript global environment and a new
`globalThis.__coverage__` object. Consequently:

- Counters collected before login are lost.
- Counters generated by the login interaction are not preserved.
- The replacement page starts collecting into a new coverage object.
- Startup execution from the replacement page can appear in the final report.

The login scenario's raw Istanbul report is byte-for-byte identical to the
reusable-session scenario, despite their different interactions. This is strong
evidence that the login interaction itself is not being retained across the
navigation.

The framework now addresses this at the browser-context level. Its coverage
initialization script stores the Istanbul counter object in scenario-local
`sessionStorage` during `pagehide`, restores it before the next document starts,
and persists the zeroed object after the startup reset. This preserves counters
across full-page navigation without changing product code or sharing data between
scenarios.

The fix was verified by rerunning the UI suite. Before the fix, the login and
reusable-session reports were byte-for-byte identical and each recorded 908
covered statements. After the fix, they recorded 931 and 935 statements
respectively and were no longer identical. This is consistent with the login
journey retaining execution across its navigation boundary.

Before the fix, the lifecycle created:

- Under-counting of code executed before navigation, such as the login submit path.
- Potential inclusion of startup execution after navigation.

These specific navigation-boundary issues are now resolved for instrumented
Istanbul coverage. Source-level Istanbul counters are required; the framework does not
substitute page-boundary CDP/V8 metrics when they are unavailable.

## Denominator assessment

The 18,463-statement denominator is complete relative to the configured source
inventory:

- All 234 first-party source files are represented.
- Unvisited files are included with zero counters.
- No source file is silently omitted.
- No dependency or generated bundle file is included.

The denominator is an Istanbul statement denominator, not a raw source-line
denominator. Type-only declarations, comments, whitespace, CSS, and other
non-executable syntax are correctly excluded.

The earlier approximately 73,000-statement denominator came from bundle/V8
conversion semantics and is not directly comparable. It counted generated or
expression-level locations that are not Istanbul source statements.

## Confidence summary

| Area | Confidence | Conclusion |
| --- | --- | --- |
| Source-file inclusion | High | No first-party source files are missing |
| Dependency exclusion | High | No third-party or generated-bundle files are included |
| Statement aggregation | High | The raw union exactly equals 3,151 |
| Source mapping | High | All statement locations map to valid source lines |
| SPA journey measurement | High | Covered statements correspond to rendered and interacted application code |
| Full-page navigation measurement | High for Istanbul coverage | Counters are preserved across `window.location.replace` |
| Denominator completeness | High for Istanbul statements | 18,463 represents the configured source inventory |
| Overall authoritative-coverage claim | High within the stated scope | The report is authoritative for first-party Istanbul statements and measured journeys; it is not a claim that untested product paths are covered |

## Final conclusion

The regenerated report is `17.14%` statement coverage (`3,166 / 18,463`) for
the clean UI run. It is a source-level Istanbul result for the collected browser
execution, with a complete 234-file denominator. Coverage is now preserved across
full-page navigation, including the login flow. The metric remains a measure of
the tested journeys, not a claim that unvisited product paths are covered.
