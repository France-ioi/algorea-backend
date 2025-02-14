Feature: Create a group (groupCreate) - robustness

  Background:
    Given the database has the following users:
      | group_id | login  | temp_user |
      | 21       | owner  | 0         |
      | 31       | tmp12  | 1         |
      | 51       | john   | 0         |
    And the groups ancestors are computed

  Scenario: No name
    Given I am the user with id "21"
    When I send a POST request to "/groups" with the following body:
    """
    {"type": "Class"}
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Bad Request",
      "error_text": "Invalid input data",
      "errors": {"name": ["missing field"]}
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Empty name
    Given I am the user with id "21"
    When I send a POST request to "/groups" with the following body:
    """
    {"name": "", "type": "Class"}
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Bad Request",
      "error_text": "Invalid input data",
      "errors": {"name": ["name must be at least 1 character in length"]}
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: No type
    Given I am the user with id "21"
    When I send a POST request to "/groups" with the following body:
    """
    {"name": "some name"}
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Bad Request",
      "error_text": "Invalid input data",
      "errors": {"type": ["missing field"]}
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario Outline: Empty or wrong type
    Given I am the user with id "21"
    When I send a POST request to "/groups" with the following body:
    """
    {"name": "some name", "type": "<type>"}
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Bad Request",
      "error_text": "Invalid input data",
      "errors": {"type": ["type must be one of [Class Team Club Friends Other Session]"]}
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
  Examples:
    | type     |
    |          |
    | Unknown  |
    | User     |
    | Base     |

  Scenario: Temporary user
    Given I am the user with id "31"
    When I send a POST request to "/groups" with the following body:
    """
    {"name": "some name", "type": "Class"}
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
