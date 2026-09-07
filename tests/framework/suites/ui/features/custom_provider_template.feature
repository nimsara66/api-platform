Feature: Custom LLM provider template lifecycle
  The journey ported from the product's own Cypress suite (005-custom-provider-template),
  creating a custom template, adding a second version, using it to create a provider, and
  confirming a version still referenced by a provider cannot be deleted until that provider
  is removed.

  Scenario: An administrator versions a custom template, uses it for a provider, and is blocked from deleting the version in use
    Given the user is signed in
    When the user creates the LLM provider template "E2E Custom Template" at "https://api.e2e-custom-template.example.com"
    And the user opens the LLM provider template "E2E Custom Template"
    Then the user sees "E2E Custom Template" on the page
    And the user sees a "v1.0" version button

    When the user creates version "v2.0" of the template from version "v1.0" at "https://api.e2e-custom-template.example.com"
    Then the user sees "E2E Custom Template" on the page
    And the user sees a "v2.0" version button

    When the user creates the provider "E2E Custom Template Provider" from the "E2E Custom Template" template's "v2.0" version
    Then the user is on the provider's overview page
    And the user sees "E2E Custom Template Provider" on the page

    When the user opens the LLM provider template "E2E Custom Template"
    And the user attempts to delete the current template version
    Then the user sees "Cannot delete: one or more providers were created from this template." on the page

    When the user deletes the provider "E2E Custom Template Provider" directly
    And the user deletes the template's "v2.0" version
    Then the user sees a "v1.0" version button

    When the user deletes the template's "v1.0" version
    Then the user no longer sees "E2E Custom Template"
