Feature: Create a group (groupCreate)

  Background:
    Given the database has the following user:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |

  Scenario Outline: Create a group
    Given I am the user with id "21"
    When I send a POST request to "/groups" with the following body:
    """
    {
      "name": "some name",
      "type": "<group_type>"
    }
    """
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created",
      "data": {"id":"5577006791947779410"}
    }
    """
    And the table "groups" should remain unchanged, regardless of the row with id "5577006791947779410"
    And the table "groups" at id "5577006791947779410" should be:
      | id                  | name      | type         | TIMESTAMPDIFF(SECOND, NOW(), created_at) < 3 |
      | 5577006791947779410 | some name | <group_type> | true                                         |
    And the table "group_managers" should be:
      | manager_id | group_id            | can_manage            | can_grant_group_access | can_watch_members |
      | 21         | 5577006791947779410 | memberships_and_group | 1                      | 1                 |
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should be:
      | ancestor_group_id   | child_group_id      | is_self |
      | 21                  | 21                  | 1       |
      | 5577006791947779410 | 5577006791947779410 | 1       |
  Examples:
    | group_type |
    | Class      |
    | Team       |
    | Club       |
    | Friends    |
    | Other      |
    | Session    |
