Feature: User sends a request to leave a group
  Background:
    Given the database has the following table "groups":
      | id | require_lock_membership_approval_until |
      | 11 | 3000-01-01 00:00:00                    |
      | 14 | 4000-01-01 00:00:00                    |
    And the database has the following user:
      | group_id |
      | 21       |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | lock_membership_approved_at |
      | 11              | 21             | 2019-05-30 11:00:00         |
      | 14              | 21             | 2019-06-30 11:00:00         |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type          | at                      |
      | 14       | 21        | leave_request | 2019-05-30 11:00:00.001 |

  Scenario: Successfully send a request
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-leave-requests/11"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created",
      "data": {"changed": true}
    }
    """
    And the table "group_pending_requests" should be:
      | group_id | member_id | type          | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 11       | 21        | leave_request | 1                                         |
      | 14       | 21        | leave_request | 0                                         |
    And the table "group_membership_changes" should be:
      | group_id | member_id | action                | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 11       | 21        | leave_request_created | 21           | 1                                         |
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Try to recreate a request that already exists
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-leave-requests/14"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "unchanged",
      "data": {"changed": false}
    }
    """
    And the table "group_pending_requests" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
