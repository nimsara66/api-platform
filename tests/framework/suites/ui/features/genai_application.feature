Feature: GenAI application lifecycle
  The journey ported from the product's own Cypress suite (004-genai-application), creating
  a GenAI application and retiring both the application and its owning project through the
  UI.

  Scenario: An administrator creates a GenAI application, then removes it and its project
    Given the user is signed in
    When the user creates a project named "E2E GenAI Project"
    Then the user sees "E2E GenAI Project" among the projects

    When the user opens the project "E2E GenAI Project"
    And the user opens GenAI Applications
    And the user creates the GenAI application "E2E GenAI Assistant"
    Then the user is on the GenAI application's overview page
    And the user sees "E2E GenAI Assistant" on the page

    When the user opens GenAI Applications
    And the user deletes the GenAI application "E2E GenAI Assistant"
    Then the user no longer sees "E2E GenAI Assistant"

    When the user returns to the organization level
    And the user opens the projects list
    And the user deletes the project "E2E GenAI Project"
    Then the user no longer sees "E2E GenAI Project"
