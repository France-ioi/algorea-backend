Feature: Create a temporary user - robustness
  Scenario: Authorization header is present
    Given the "Authorization" request header is "Bearer 1234567890"
    When I send a POST request to "/auth/temp-user"
    Then the response code should be 400
    And the response error message should contain "The 'Authorization' header must not be present"
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "sessions" should remain unchanged

  Scenario: default_language is too long
    When I send a POST request to "/auth/temp-user?default_language=russian"
    Then the response code should be 400
    And the response error message should contain "The length of default_language should be no more than 3 characters"
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "sessions" should remain unchanged

  Scenario Outline: Invalid cookie attributes
    Given I send a POST request to "/auth/temp-user<query>"
    Then the response code should be 400
    And the response error message should contain "<expected_error>"
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "sessions" should remain unchanged
  Examples:
    | query                                            | expected_error                                                                 |
    | ?use_cookie=1                                    | One of cookie_secure and cookie_same_site must be true when use_cookie is true |
    | ?use_cookie=1&cookie_same_site=0&cookie_secure=0 | One of cookie_secure and cookie_same_site must be true when use_cookie is true |
    | ?use_cookie=abc                                  | Wrong value for use_cookie (should have a boolean value (0 or 1))              |
    | ?cookie_same_site=abc                            | Wrong value for cookie_same_site (should have a boolean value (0 or 1))        |
    | ?cookie_secure=abc                               | Wrong value for cookie_secure (should have a boolean value (0 or 1))           |
