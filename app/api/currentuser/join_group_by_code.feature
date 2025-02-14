Feature: Join a group using a code (groupsJoinByCode)
  Background:
    Given the database has the following table "groups":
      | id | type  | code       | code_expires_at     | code_lifetime | require_watch_approval |
      | 11 | Team  | 3456789abc | 2037-05-29 06:38:38 | 3723          | 0                      |
      | 12 | Team  | abc3456789 | null                | 45296         | 0                      |
      | 14 | Team  | cba9876543 | null                | null          | 0                      |
      | 15 | Team  | 987654321a | null                | null          | 1                      |
      | 16 | Class | 2345668999 | null                | null          | 0                      |
      | 17 | Team  | null       | null                | null          | 0                      |
    And the database has the following user:
      | group_id |
      | 21       |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 17              | 21             |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         |
      | 11       | 21        | invitation   |
      | 14       | 21        | join_request |
    And the database has the following table "items":
      | id | default_language_tag | entry_min_admitted_members_ratio |
      | 20 | fr                   | All                              |
      | 30 | fr                   | All                              |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 20               | 30            |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 20             | 30            | 1           |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 11       | 20      | content            |
      | 12       | 20      | content            |
      | 14       | 20      | solution           |
      | 15       | 20      | info               |
      | 16       | 20      | info               |
      | 21       | 30      | content            |
    And the database has the following table "attempts":
      | id | participant_id | root_item_id |
      | 0  | 16             | 30           |
      | 0  | 17             | 30           |
      | 0  | 21             | null         |
      | 1  | 16             | 30           |
      | 1  | 21             | 30           |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | started_at          |
      | 0          | 17             | 30      | 2019-05-30 11:00:00 |
      | 0          | 21             | 30      | 2019-05-30 11:00:00 |
      | 1          | 16             | 30      | 2019-05-30 11:00:00 |

  Scenario: Successfully join a team
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-memberships/by-code?code=3456789abc"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created",
      "data": {"changed": true}
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 11              | 21             |
      | 17              | 21             |
    And the table "group_pending_requests" should be:
      | group_id | member_id | type         |
      | 14       | 21        | join_request |
    And the table "group_membership_changes" should be:
      | group_id | member_id | action         | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 11       | 21        | joined_by_code | 21           | 1                                         |
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Successfully join a group
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-memberships/by-code?code=2345668999"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created",
      "data": {"changed": true}
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 16              | 21             |
      | 17              | 21             |
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be:
      | group_id | member_id | action         | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 16       | 21        | joined_by_code | 21           | 1                                         |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | expires_at          |
      | 11                | 11             | 9999-12-31 23:59:59 |
      | 12                | 12             | 9999-12-31 23:59:59 |
      | 14                | 14             | 9999-12-31 23:59:59 |
      | 15                | 15             | 9999-12-31 23:59:59 |
      | 16                | 16             | 9999-12-31 23:59:59 |
      | 16                | 21             | 9999-12-31 23:59:59 |
      | 17                | 17             | 9999-12-31 23:59:59 |
      | 21                | 21             | 9999-12-31 23:59:59 |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | started_at          |
      | 0          | 17             | 30      | 2019-05-30 11:00:00 |
      | 0          | 21             | 20      | null                |
      | 0          | 21             | 30      | 2019-05-30 11:00:00 |
      | 1          | 16             | 30      | 2019-05-30 11:00:00 |
    And the table "results_propagate" should be empty

  Scenario: Updates the code_expires_at
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-memberships/by-code?code=abc3456789"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created",
      "data": {"changed": true}
    }
    """
    And the table "groups" should remain unchanged, regardless of the row with id "12"
    And the table "groups" at id "12" should be:
      | id | type | code       | code_lifetime | TIMESTAMPDIFF(SECOND, code_expires_at, DATE_ADD(NOW(), INTERVAL 45296 SECOND)) < 3 |
      | 12 | Team | abc3456789 | 45296         | 1                                                                                  |
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 12              | 21             |
      | 17              | 21             |
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be:
      | group_id | member_id | action         | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 12       | 21        | joined_by_code | 21           | 1                                         |
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Doesn't update the code_expires_at if code_lifetime is null
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-memberships/by-code?code=cba9876543"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created",
      "data": {"changed": true}
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 14              | 21             |
      | 17              | 21             |
    And the table "group_pending_requests" should be:
      | group_id | member_id | type       |
      | 11       | 21        | invitation |
    And the table "group_membership_changes" should be:
      | group_id | member_id | action         | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 14       | 21        | joined_by_code | 21           | 1                                         |
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Successfully join a group that requires approvals
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-memberships/by-code?code=987654321a&approvals=watch"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created",
      "data": {"changed": true}
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 15              | 21             |
      | 17              | 21             |
    And the table "group_pending_requests" should be:
      | group_id | member_id | type         |
      | 11       | 21        | invitation   |
      | 14       | 21        | join_request |
    And the table "group_membership_changes" should be:
      | group_id | member_id | action         | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 15       | 21        | joined_by_code | 21           | 1                                         |
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged
