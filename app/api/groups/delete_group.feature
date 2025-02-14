Feature: Delete a group
  Background:
    Given the database has the following table "groups":
      | id | name              | type  |
      | 11 | Group A           | Class |
      | 13 | Group B           | Class |
      | 14 | Group C           | Class |
      | 15 | Group D           | Class |
      | 21 | Self              | User  |
      | 22 | Group             | Class |
      | 30 | ThreadHelperGroup | Class |
      | 31 | AllUsers          | Base  |
    And the application config is:
      """
      domains:
        -
          domains: [127.0.0.1]
          allUsersGroup: 31
      """
    And the database has the following user:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 14       | 21         | none                  |
      | 15       | 14         | memberships_and_group |
      | 22       | 21         | memberships           |
      | 30       | 21         | memberships_and_group |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | expires_at          |
      | 13              | 11             | 2019-05-30 11:00:00 |
      | 14              | 21             | 9999-12-31 23:59:59 |
      | 15              | 11             | 9999-12-31 23:59:59 |
      | 15              | 13             | 9999-12-31 23:59:59 |
      | 22              | 13             | 9999-12-31 23:59:59 |
      | 22              | 14             | 9999-12-31 23:59:59 |
      | 31              | 21             | 9999-12-31 23:59:59 |
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
    And the database has the following table "items":
      | id | default_language_tag |
      | 1  | fr                   |
      | 2  | fr                   |
    And the database has the following table "threads":
      | participant_id | item_id | status                  | helper_group_id |
      | 21             | 1       | waiting_for_participant | 30              |
      | 21             | 2       | waiting_for_participant | 22              |

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
    And the table "groups_groups" should remain unchanged, except that the rows with child_group_id "11" should be deleted
    And the table "group_pending_requests" should remain unchanged, except that the rows with group_id,member_id "11" should be deleted
    And the table "group_membership_changes" should remain unchanged, except that the rows with group_id,member_id "11" should be deleted
    And the table "groups_ancestors" should remain unchanged, except that the rows with ancestor_group_id,child_group_id "11" should be deleted
    And the table "groups" should remain unchanged, except that the rows with id "11" should be deleted

  Scenario: User deletes a group ignoring an expired parent-child relation
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
    And the table "groups_groups" should remain unchanged, except that the rows with parent_group_id,child_group_id "13" should be deleted
    And the table "group_membership_changes" should remain unchanged, except that the rows with group_id,member_id "13" should be deleted
    And the table "group_pending_requests" should remain unchanged, except that the rows with group_id,member_id "13" should be deleted
    And the table "groups_ancestors" should remain unchanged, except that the rows with ancestor_group_id,child_group_id "13" should be deleted
    And the table "groups" should remain unchanged, except that the row with id "13" should be deleted

  Scenario: User deletes a group that is the helper_group_id of a thread should change the helper group to AllUsers
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/30"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "deleted"
    }
    """
    And the table "threads" should remain unchanged, regardless of the row with item_id "1"
    And the table "threads" at item_id "1" should be:
      | participant_id | item_id | status                  | helper_group_id |
      | 21             | 1       | waiting_for_participant | 31               |
