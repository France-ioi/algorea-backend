Feature: User withdraws a request to leave a group - robustness
  Background:
    Given the database has the following table "groups":
      | id |
      | 11 |
      | 14 |
    And the database has the following users:
      | group_id | login |
      | 21       | john  |
      | 22       | jane  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 11              | 21             |
      | 14              | 22             |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type          | at                      |
      | 14       | 21        | leave_request | 2019-05-30 11:00:00.001 |

  Scenario: User tries to withdraw a non-existing leave request
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-leave-requests/11/withdraw"
    Then the response code should be 404
    And the response error message should contain "No such relation"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the group id is wrong
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-leave-requests/abc/withdraw"
    Then the response code should be 400
    And the response error message should contain "Wrong value for group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged
