Feature: AI gateway lifecycle
  The journey ported from the product's own Cypress suite (003-ai-gateway), registering
  and retiring an AI gateway entirely through the UI.

  Scenario: An administrator registers an AI gateway, then removes it
    Given the user is signed in
    When the user creates the AI gateway "e2e-ai-gateway" at "https://localhost:8443"
    Then the user is on the AI gateway's overview page
    And the user sees "e2e-ai-gateway" on the page

    When the user opens AI Gateways
    And the user deletes the AI gateway "e2e-ai-gateway"
    Then the user no longer sees "e2e-ai-gateway"
