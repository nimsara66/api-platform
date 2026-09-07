Feature: MCP proxy credential preservation across policy updates
  As a security-conscious operator
  I want saving an MCP proxy's policies to never touch its stored credential
  So that editing unrelated configuration can never leak or discard a secret

  Scenario: Saving policies keeps an existing credential's header and type without minting a new secret
    Given the user is signed in
    And the user creates a project named "TC96 MCP Update Project"
    And the user opens the project "TC96 MCP Update Project"
    And the user opens MCP Proxies
    And the user creates the MCP proxy "TC96 MCP Update Server" at "https://sample.mcp.example.com/mcp" with the auth header "Authorization" set to "Bearer tok-setup-key"
    Then the user is on the MCP proxy's overview page

    When the user opens the MCP proxy's Policies tab
    And the user adds a CORS policy and saves
    Then the MCP proxy update kept the auth header "Authorization" and type "header"
    And no secret was created for that credential
    And the MCP proxy update body does not include "tok-setup-key"

  Scenario: Saving policies on a proxy without a credential keeps the update free of auth
    Given the user is signed in
    And the user creates a project named "TC97 MCP Update Project"
    And the user opens the project "TC97 MCP Update Project"
    And the user opens MCP Proxies
    And the user creates the MCP proxy "TC97 MCP Update Server" at "https://sample.mcp.example.com/mcp" with no credential
    Then the user is on the MCP proxy's overview page

    When the user opens the MCP proxy's Policies tab
    And the user adds a CORS policy and saves
    Then the MCP proxy update had no auth block
    And no secret was created for that credential
    And the MCP proxy update body does not include "{{ secret"
