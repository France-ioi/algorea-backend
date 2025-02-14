Feature: Remove a direct parent-child relation between two groups
  Background:
    Given the database has the following table "groups":
      | id | name    | type  |
      | 11 | Group A | Class |
      | 13 | Group B | Class |
      | 14 | Group C | Class |
      | 21 | Self    | User  |
      | 22 | Group   | Class |
    And the database has the following user:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 13       | 21         | memberships |
      | 14       | 21         | none        |
      | 22       | 21         | memberships |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 13              | 11             |
      | 22              | 11             |
      | 22              | 13             |
      | 22              | 14             |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type       |
      | 13       | 11        | invitation |
      | 22       | 11        | invitation |
      | 22       | 13        | invitation |
      | 22       | 14        | invitation |
    And the database has the following table "group_membership_changes":
      | group_id | member_id |
      | 13       | 11        |
      | 22       | 11        |
      | 22       | 13        |
      | 22       | 14        |

  Scenario: User deletes a relation
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/13/relations/11"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "deleted"
    }
    """
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 22              | 11             |
      | 22              | 13             |
      | 22              | 14             |
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should remain unchanged
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self |
      | 11                | 11             | 1       |
      | 13                | 13             | 1       |
      | 14                | 14             | 1       |
      | 21                | 21             | 1       |
      | 22                | 11             | 0       |
      | 22                | 13             | 0       |
      | 22                | 14             | 0       |
      | 22                | 22             | 1       |
    And the table "groups" should remain unchanged

  Scenario: User deletes a relation and an orphaned child group
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/22/relations/13?delete_orphans=1"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "deleted"
    }
    """
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 22              | 11             |
      | 22              | 14             |
    And the table "group_membership_changes" should be:
      | group_id | member_id |
      | 22       | 11        |
      | 22       | 14        |
    And the table "group_pending_requests" should be:
      | group_id | member_id |
      | 22       | 11        |
      | 22       | 14        |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self |
      | 11                | 11             | 1       |
      | 14                | 14             | 1       |
      | 21                | 21             | 1       |
      | 22                | 11             | 0       |
      | 22                | 14             | 0       |
      | 22                | 22             | 1       |
    And the table "groups" should remain unchanged, regardless of the row with id "13"
    And the table "groups" should not contain id "13"

  Scenario: Delete a relation making a child group an orphan
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/22/relations/13"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "deleted"
    }
    """
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 13              | 11             |
      | 22              | 11             |
      | 22              | 14             |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self |
      | 11                | 11             | 1       |
      | 13                | 11             | 0       |
      | 13                | 13             | 1       |
      | 14                | 14             | 1       |
      | 21                | 21             | 1       |
      | 22                | 11             | 0       |
      | 22                | 14             | 0       |
      | 22                | 22             | 1       |
    And the table "groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
