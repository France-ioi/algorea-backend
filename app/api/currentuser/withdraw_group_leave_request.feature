Feature: User withdraws a request to leave a group
  Background:
    Given the database has the following table "groups":
      | id |
      | 11 |
      | 14 |
    And the database has the following user:
      | group_id |
      | 21       |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 11              | 21             |
      | 14              | 21             |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type          | at                  |
      | 11       | 21        | leave_request | 2019-05-30 11:00:00 |
      | 14       | 21        | leave_request | 2019-05-30 11:00:00 |

  Scenario: Successfully withdraw a request
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-leave-requests/11/withdraw"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "updated",
      "data": {"changed": true}
    }
    """
    And the table "group_pending_requests" should be:
      | group_id | member_id | type          | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 14       | 21        | leave_request | 0                                         |
    And the table "group_membership_changes" should be:
      | group_id | member_id | action                  | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 11       | 21        | leave_request_withdrawn | 21           | 1                                         |
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
