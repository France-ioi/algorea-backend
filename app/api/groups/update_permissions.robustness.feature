Feature: Change item access rights for a group - robustness
  Background:
    Given the database has the following table "groups":
      | id | name          | type  |
      | 25 | some class    | Class |
      | 26 | another class | Class |
      | 27 | third class   | Class |
    And the database has the following users:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
      | 23       | user  | John        | Doe       |
      | 31       | admin | Allie       | Grater    |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_grant_group_access |
      | 23       | 21         | 1                      |
      | 25       | 21         | 0                      |
      | 25       | 31         | 1                      |
      | 26       | 21         | 1                      |
      | 26       | 31         | 1                      |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 25              | 23             |
      | 25              | 31             |
      | 26              | 23             |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id  | default_language_tag |
      | 100 | fr                   |
      | 101 | fr                   |
      | 102 | fr                   |
      | 103 | fr                   |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | content_view_propagation | child_order |
      | 100            | 101           | as_info                  | 0           |
      | 101            | 102           | as_content               | 0           |
      | 102            | 103           | as_content               | 0           |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 100              | 101           |
      | 100              | 102           |
      | 100              | 103           |
      | 101              | 102           |
      | 101              | 103           |
      | 102              | 103           |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated | can_grant_view_generated | is_owner_generated |
      | 21       | 100     | solution           | solution_with_grant      | 1                  |
      | 21       | 101     | none               | none                     | 0                  |
      | 21       | 102     | none               | solution                 | 0                  |
      | 21       | 103     | none               | solution                 | 0                  |
      | 25       | 100     | content            | none                     | 0                  |
      | 25       | 101     | info               | none                     | 0                  |
      | 31       | 102     | none               | content_with_descendants | 0                  |
    And the database has the following table "permissions_granted":
      | group_id | item_id | can_view | can_grant_view           | can_enter_until     | is_owner | source_group_id | latest_update_at    |
      | 21       | 100     | none     | none                     | 9999-12-31 23:59:59 | 1        | 23              | 2019-05-30 11:00:00 |
      | 21       | 102     | none     | solution                 | 9999-12-31 23:59:59 | 1        | 23              | 2019-05-30 11:00:00 |
      | 23       | 101     | none     | none                     | 2018-05-30 11:00:00 | 0        | 26              | 2019-05-30 11:00:00 |
      | 25       | 100     | content  | none                     | 9999-12-31 23:59:59 | 0        | 23              | 2019-05-30 11:00:00 |
      | 25       | 101     | info     | none                     | 9999-12-31 23:59:59 | 0        | 23              | 2019-05-30 11:00:00 |
      | 31       | 102     | none     | content_with_descendants | 9999-12-31 23:59:59 | 0        | 31              | 2019-05-30 11:00:00 |

  Scenario: Invalid source_group_id
    Given I am the user with id "21"
    When I send a PUT request to "/groups/abc/permissions/23/102" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 400
    And the response error message should contain "Wrong value for source_group_id (should be int64)"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: Invalid group_id
    Given I am the user with id "21"
    When I send a PUT request to "/groups/25/permissions/abc/102" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 400
    And the response error message should contain "Wrong value for group_id (should be int64)"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: Invalid item_id
    Given I am the user with id "21"
    When I send a PUT request to "/groups/25/permissions/23/abc" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 400
    And the response error message should contain "Wrong value for item_id (should be int64)"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: Invalid can_view
    Given I am the user with id "21"
    When I send a PUT request to "/groups/26/permissions/23/102" with the following body:
    """
    {
      "can_view": "unknown"
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "error_text": "Invalid input data",
      "errors": {
        "can_view": ["can_view must be one of [none info content content_with_descendants solution]"]
      },
      "message": "Bad Request",
      "success": false
    }
    """
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: The user doesn't exist
    Given I am the user with id "404"
    When I send a PUT request to "/groups/25/permissions/23/102" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: The user doesn't have enough rights to set permissions
    Given I am the user with id "31"
    When I send a PUT request to "/groups/26/permissions/23/101" with the following body:
    """
    {
      "can_view": "info",
      "can_grant_view": "solution",
      "can_watch": "answer",
      "can_edit": "all",
      "can_make_session_official": true,
      "is_owner": true,
      "can_enter_from": "2019-05-30T11:00:00Z",
      "can_enter_until": "2019-05-30T11:00:00Z"
    }
    """
    Then the response code should be 400
    And the response error message should contain "Invalid input data"
    And the response body should be, in JSON:
    """
    {
      "error_text": "Invalid input data",
      "errors": {
        "can_view": ["the value is not permitted"],
        "can_grant_view": ["the value is not permitted"],
        "can_watch": ["the value is not permitted"],
        "can_edit": ["the value is not permitted"],
        "can_make_session_official": ["the value is not permitted"],
        "is_owner": ["the value is not permitted"],
        "can_enter_from": ["the value is not permitted"],
        "can_enter_until": ["the value is not permitted"]
      },
      "message": "Bad Request",
      "success": false
    }
    """
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: The item doesn't exist
    Given I am the user with id "21"
    When I send a PUT request to "/groups/25/permissions/23/404" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: The user is not a manager of the source_group_id
    Given I am the user with id "21"
    When I send a PUT request to "/groups/27/permissions/27/102" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: The user is a manager of the source_group_id, but he doesn't have 'can_grant_group_access' permission
    Given I am the user with id "21"
    When I send a PUT request to "/groups/25/permissions/25/102" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: source_group_id is not an ancestor of group_id
    Given I am the user with id "31"
    When I send a PUT request to "/groups/25/permissions/26/102" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: source_group_id is a user group
    Given I am the user with id "31"
    When I send a PUT request to "/groups/31/permissions/31/102" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: The source group doesn't exist
    Given I am the user with id "21"
    When I send a PUT request to "/groups/404/permissions/21/102" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: The group doesn't exist
    Given I am the user with id "21"
    When I send a PUT request to "/groups/25/permissions/404/102" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: There are no item's parents visible to the group
    Given I am the user with id "31"
    When I send a PUT request to "/groups/25/permissions/23/103" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged
