Feature: LLM provider and app LLM proxy lifecycle
  The journey ported from the product's own Cypress suite (001-provider-and-proxy), driven
  entirely through the UI against the real control plane and the block's real gateway.
  Every assertion is something the user SEES — a page reached, a deployment turning
  Active, the model's answer coming back — never a database row or a config dump.

  Two provider templates are exercised because they render genuinely different forms:

  Azure AI Foundry is the built-in that leaves the upstream URL to the user (here the
  block's mock LLM). Built-ins are the only templates a real gateway can serve —
  workspace-defined templates never reach it, so providers built from them fail to
  deploy — which is why the deploy-and-invoke half of the journey rides on this template.

  OpenAI is the built-in that bakes in its OWN upstream endpoint, so its form never asks
  for one at all — the same template the legacy Cypress spec exercised. Its provider is
  never deployed here (the real api.openai.com is not reachable from this block), but the
  creation form itself and the full proxy create/delete lifecycle are still exercised end
  to end. Both templates also verify the context field auto-filling from the provider's
  name before submission — the same fact the legacy suite asserted with its own
  independently-computed slug.

  Scenario: An administrator publishes a provider and proxy, invokes the LLM through the gateway, then retires the proxy
    Given the user is signed in

    When the user creates a project named "E2E Project"
    Then the user sees "E2E Project" among the projects

    When the user starts adding a provider from the "Azure AI Foundry" template
    And the user creates the provider "E2E OpenAI Provider" pointed at the mock LLM
    Then the user is on the provider's overview page
    And the user sees "E2E OpenAI Provider" on the page

    When the user deploys it to the gateway
    Then the user sees the deployment is active
    When the user returns to the provider overview
    And the user generates an API key named "e2e-provider-key"

    When the user creates an app LLM proxy "E2E OpenAI Proxy" in project "E2E Project" using that key
    Then the user is on the proxy's overview page
    And the user sees "E2E OpenAI Proxy" on the page

    When the user deploys it to the gateway
    Then the user sees the deployment is active
    When the user returns to the proxy overview
    And the user generates an API key named "e2e-proxy-key"
    And the user invokes the proxy's chat completions endpoint with that key
    Then the completion answers "Hello! How can I assist you today?"

    When the user deletes the proxy
    Then the user is back on the proxy list
    And the user no longer sees "E2E OpenAI Proxy"

  Scenario: An administrator creates a provider from the OpenAI template and its proxy, then removes the proxy
    Given the user is signed in

    When the user creates a project named "E2E Classic Project"
    Then the user sees "E2E Classic Project" among the projects

    When the user starts adding a provider from the "OpenAI" template
    And the user creates the provider "E2E Classic OpenAI Provider" using the template's built-in endpoint
    Then the user is on the provider's overview page
    And the user sees "E2E Classic OpenAI Provider" on the page

    When the user creates an app LLM proxy "E2E Classic OpenAI Proxy" in project "E2E Classic Project" using the API key "sk-e2e-classic-openai-proxy-key"
    Then the user is on the proxy's overview page
    And the user sees "E2E Classic OpenAI Proxy" on the page

    When the user deletes the proxy
    Then the user is back on the proxy list
    And the user no longer sees "E2E Classic OpenAI Proxy"
