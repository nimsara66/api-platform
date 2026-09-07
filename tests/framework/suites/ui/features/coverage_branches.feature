Feature: Browser source coverage branches
  These scenarios verify that source-level browser counters preserve branches selected by
  browser behavior after scenario setup coverage has been discarded.

  Scenario: The runtime configuration branch executes
    When the user opens the workspace
    And the user opens the workspace
    Then the user sees the sign-in form

  Scenario: The missing runtime configuration branch executes
    Given the browser has no runtime configuration
    When the user opens the workspace
    Then the user sees the sign-in form
    And the runtime configuration fallback is active
