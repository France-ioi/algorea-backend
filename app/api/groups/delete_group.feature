Feature: Delete a group
  Background:
    Given the database has the following table 'groups':
      | id | name    | type  |
      | 11 | Group A | Class |
      | 13 | Group B | Class |
      | 14 | Group C | Class |
      | 21 | Self    | User  |
      | 22 | Group   | Class |
    And the database has the following table 'users':
      | login | group_id | first_name  | last_name | allow_subgroups |
      | owner | 21       | Jean-Michel | Blanquer  | 1               |
    And the database has the following table 'group_managers':
      | group_id | manager_id | can_manage            |
      | 13       | 14         | memberships_and_group |
      | 14       | 21         | none                  |
      | 22       | 21         | memberships           |
    And the database has the following table 'groups_groups':
      | parent_group_id | child_group_id |
      | 13              | 11             |
      | 14              | 21             |
      | 22              | 13             |
      | 22              | 14             |
    And the groups ancestors are computed
    And the database has the following table 'group_pending_requests':
      | group_id | member_id | type       |
      | 13       | 11        | invitation |
      | 22       | 11        | invitation |
      | 22       | 13        | invitation |
      | 22       | 14        | invitation |
    And the database has the following table 'group_membership_changes':
      | group_id | member_id |
      | 13       | 11        |
      | 22       | 11        |
      | 22       | 13        |
      | 22       | 14        |

  Scenario: User deletes a group
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/11"
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
      | 14              | 21             |
      | 22              | 13             |
      | 22              | 14             |
    And the table "group_pending_requests" should be:
      | group_id | member_id |
      | 22       | 13        |
      | 22       | 14        |
    And the table "group_membership_changes" should be:
      | group_id | member_id |
      | 22       | 13        |
      | 22       | 14        |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self |
      | 13                | 13             | 1       |
      | 14                | 14             | 1       |
      | 14                | 21             | 0       |
      | 21                | 21             | 1       |
      | 22                | 13             | 0       |
      | 22                | 14             | 0       |
      | 22                | 21             | 0       |
      | 22                | 22             | 1       |
    And the table "groups" should stay unchanged but the row with id "11"
    And the table "groups" should not contain id "11"

  Scenario: User deletes a group and an orphaned child group
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/13"
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
      | 14              | 21             |
      | 22              | 14             |
    And the table "group_membership_changes" should be:
      | group_id | member_id |
      | 22       | 14        |
    And the table "group_pending_requests" should be:
      | group_id | member_id |
      | 22       | 14        |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self |
      | 14                | 14             | 1       |
      | 14                | 21             | 0       |
      | 21                | 21             | 1       |
      | 22                | 14             | 0       |
      | 22                | 21             | 0       |
      | 22                | 22             | 1       |
    And the table "groups" should be:
      | id | name    | type  |
      | 14 | Group C | Class |
      | 21 | Self    | User  |
      | 22 | Group   | Class |
