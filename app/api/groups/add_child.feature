Feature: Add a parent-child relation between two groups
  Background:
    Given the database has the following table "groups":
      | id | name    | type  |
      | 11 | Group A | Class |
      | 13 | Group B | Class |
      | 14 | Group C | Class |
      | 21 | Self    | User  |
    And the database has the following user:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 11       | 21         | memberships_and_group |
      | 13       | 21         | memberships           |
      | 14       | 21         | memberships_and_group |
    And the database has the following table "items":
      | id | default_language_tag |
      | 20 | fr                   |
      | 30 | fr                   |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 20               | 30            |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 20             | 30            | 1           |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 13       | 20      | content            |
      | 21       | 30      | content            |
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 11             |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id |
      | 0          | 11             | 30      |

  Scenario: User is a manager of the two groups, has the needed permissions, and is allowed to create sub-groups
    Given I am the user with id "21"
    When I send a POST request to "/groups/13/relations/11"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created"
    }
    """
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 13              | 11             |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self |
      | 11                | 11             | 1       |
      | 13                | 11             | 0       |
      | 13                | 13             | 1       |
      | 14                | 14             | 1       |
      | 21                | 21             | 1       |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id |
      | 0          | 11             | 20      |
      | 0          | 11             | 30      |
    And the table "results_propagate" should be empty
    When I send a POST request to "/groups/13/relations/14"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created"
    }
    """
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 13              | 11             |
      | 13              | 14             |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self |
      | 11                | 11             | 1       |
      | 13                | 11             | 0       |
      | 13                | 13             | 1       |
      | 13                | 14             | 0       |
      | 14                | 14             | 1       |
      | 21                | 21             | 1       |
    When I send a POST request to "/groups/13/relations/11"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created"
    }
    """
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 13              | 11             |
      | 13              | 14             |
