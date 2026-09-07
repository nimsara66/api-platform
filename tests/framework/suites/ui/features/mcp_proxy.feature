Feature: MCP proxy lifecycle from the sample endpoint
  The journey ported from the product's own Cypress suite (002-mcp-proxy-sample-url),
  creating an MCP proxy from the form's built-in sample endpoint and retiring both the
  proxy and its owning project through the UI.

  Scenario: An administrator creates an MCP proxy from the sample endpoint, then removes it and its project
    Given the user is signed in
    When the user creates a project named "E2E MCP Project"
    Then the user sees "E2E MCP Project" among the projects

    When the user opens the project "E2E MCP Project"
    And the user opens MCP Proxies
    And the user creates an MCP proxy "E2E MCP Proxy" using the sample URL
    Then the user is on the MCP proxy's overview page
    And the user sees "E2E MCP Proxy" on the page

    When the user opens MCP Proxies
    And the user deletes the MCP proxy "E2E MCP Proxy"
    Then the user no longer sees "E2E MCP Proxy"

    When the user returns to the organization level
    And the user opens the projects list
    And the user deletes the project "E2E MCP Project"
    Then the user no longer sees "E2E MCP Project"
