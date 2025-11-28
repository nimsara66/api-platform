# LLM Provider and Template Support in API Platform Gateway

# What is the problem we are trying to solve and why should it be solved?

As organizations adopt Large Language Models (LLMs) at scale, platform teams must ensure secure, compliant, cost-aware access to multiple LLM providers (e.g., OpenAI, Anthropic) across business units. While developers need frictionless access, enterprises require consistent governance over credentials, routing, rate limits, token accounting, and policies. Today, this is often managed in a decentralized manner with ad-hoc per-app configuration, creating gaps in:

* Security and compliance: Centralized credential management, scoped access, and auditable policy enforcement.
* Cost controls and budgeting: Token-level accounting, request rate limits, and provider-specific quotas.
* Operational consistency: Standardized provider integrations, shared templates, and lifecycle governance.
* Observability and troubleshooting: Unified monitoring of usage, errors, and policy outcomes.

The API Platform Gateway enables platform administrators to centrally manage LLM provider configurations and reusable provider templates. This lets organizations standardize how LLMs are accessed, governed, and monitored without slowing down developer teams.

# Who are we solving it for?

* Platform Administrators who need to enforce governance, security, policy controls, and cost management across LLM access.
* AI Developers who consume centrally managed providers and benefit from consistent, secure, and policy-compliant integrations.

Personas accessing the Platform

Platform Administrator

* Ensures safe, compliant, and standardized access to all LLM providers.
* Security, governance, authentication, and policy enforcement.
* Provider-level configuration (credentials, network, rate limits).
* Organization-wide settings and monitoring.

# Proposed Solution

Support two first-class configuration artifacts managed by Platform Administrators:

* LLM Provider Templates (Blueprints)
  * Define reusable specifications for provider types (e.g., OpenAI API), including:
    * Provider specific OpenAPI (operations, paths).
    * Token extraction and accounting definitions (prompt, completion, total tokens).
    * Request model identification and metadata extraction.
  * Shared across the organization to drive consistency and reduce duplication.

* LLM Providers (Deployable Instances)
  * Reference a template by name and provide deployment-specific configuration:
    * Upstream endpoints, networks, and routing.
    * Authentication (API keys, bearer tokens) with secure header mapping.
    * Access control (allow/deny with path/method exceptions).
    * Organization policies (budget control, guardrails, routing).
  * Managed centrally by platform administrators and consumed by multiple AI developer applications subject to governance.

The Gateway will accept artifacts similar to the following. The Controller will validate and translate these artifacts into Router configuration.

LLM Provider Template (Blueprint)

```
version: api-platform.wso2.com/v1
kind: llm/provider-template
data:
  name: openai_v1.0
  # Provider-specific surface; used to derive routes and capabilities
  openapi: |
    openapi: 3.0.1
    ...

  # Token accounting definitions (used by organization policies)
  totalTokens:
    location: payload
    identifier: usage.total_tokens
  ...
```

LLM Provider (Deployment Instance)

```
version: api-platform.wso2.com/v1
kind: llm/provider
data:
  name: wso2-openai-provider
  version: v1.0
  template: openai_v1.0
  upstream:
    - url: https://api.openai.com/v1
      auth:
        type: bearer
        header: Authorization
        value: ${SECRET_OPENAI_TOKEN}
  accessControl:
    mode: deny_all
    exceptions:
      - path: /chat/completions
        methods: [POST]
  policies:
    budgetControl:
      - path: /chat/completions
        methods: [POST]
        params:
            requests: 
              type: string
              value: { count: 1000, unit: min }
            totalTokens:
              type: string
              value: { count: 50000, unit: min }
    PII:
      - path: /chat/completions
        methods: [POST]
        params:
          name:
            type: string
            value: PII Anonymizer
          jsonPath:
            type: string
            value: $.messages[-1].content
```

## Administrator Experience in the Standalone Gateway

* Central Creation and Management
  * Register provider templates to standardize how LLMs are accessed (surface, token extraction, model identification).
  * Create provider instances referencing templates; configure upstreams, credentials, access exceptions, and policies.
* Governance and Security
  * Enforce organization-wide policies: budget limits, PII handling, semantic guardrails.
  * Control egress by centrally managed credentials and scoped access rules.
* Operational Consistency
  * Templates eliminate duplication and configuration drift; providers apply context-specific details.
  * Standard naming/versioning for discoverability and lifecycle operations.
* Observability and Accountability
  * Monitor usage by provider (requests, tokens, costs) and policy outcomes.
  * Audit configuration changes and access outcomes across teams.

## User Flow

1. Platform Administrator creates or imports a provider template (e.g., `openai_v1.0`) that defines API surface and token extraction rules.
2. Platform Administrator creates one or more provider instances referencing the template, supplying upstream URLs, credentials, access exceptions, and policies.
3. The Gateway Controller validates submissions, ensuring both templates and providers meet organizational standards.
4. The Gateway translates validated configurations into Router settings for enforcement (routing, access control, and policy chains).
5. Developers use centrally published providers, benefiting from governance, consistent access patterns, and observability.

![][image1]