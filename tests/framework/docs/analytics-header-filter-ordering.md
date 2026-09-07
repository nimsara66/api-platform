# Analytics Header Filter Ordering

## Finding

The analytics header filter policy parses request and response configuration in the same way and returns the corresponding `DropHeaderAction`. The request and response policy implementations are functionally symmetric.

The different behavior is caused by response policy execution order and analytics metadata precedence.

## Request path

Request policies execute in declaration order. The analytics system policy captures request headers first, and `analytics-header-filter` applies its filtered `request_headers` metadata afterward. The filtered value therefore wins.

## Response path

Response policies execute in reverse declaration order so response policies can unwind request policy behavior. With the analytics system policy and the per-API filter in the same chain:

1. The response filter produces a filtered `DropHeaderAction`.
2. The analytics system policy captures the original response headers.
3. The translator merges both metadata results in execution order.
4. The unfiltered `response_headers` metadata can overwrite the filtered value.

The response translator applies the filter at [`translator.go`](../../../gateway/gateway-runtime/policy-engine/internal/kernel/translator.go), but the later analytics metadata merge can replace it. Response policy reversal is implemented in [`chain.go`](../../../gateway/gateway-runtime/policy-engine/internal/executor/chain.go).

## Evidence

The focused SQLite probe published the following results to the webhook collector:

- `authorization` and `x-api-key` were absent from the request event.
- `user-agent` remained in the request event.
- `x-denied-response` remained in the response event despite a response deny rule.

This demonstrates that request filtering works and that response filtering reaches the policy pipeline but loses during response analytics metadata assembly.

## Fix boundary

The correction belongs in the gateway response translation or analytics metadata merge logic. The implementation must ensure that a response filter’s finalized headers take precedence over unfiltered headers captured by the analytics system policy, regardless of policy declaration order.

The policy definition and testbench collector do not require changes for this ordering issue.

## Required regression coverage

Gateway tests should verify response filtering for:

- deny mode with one or more headers;
- allow mode with one or more headers;
- case-insensitive header names;
- `content-type` and custom response headers;
- response-header filtering when analytics system metadata is also present;
- both bodyless and body-bearing responses where their response translation paths differ.
