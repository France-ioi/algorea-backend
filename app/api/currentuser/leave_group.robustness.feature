Feature: User leaves a group - robustness
  Background:
    Given the database has the following table "groups":
      | id | type  | require_lock_membership_approval_until | frozen_membership |
      | 11 | Class | null                                   | false             |
      | 14 | Team  | null                                   | false             |
      | 15 | Club  | 2037-04-29                             | false             |
      | 16 | Club  | null                                   | true              |
      | 17 | Base  | null                                   | false             |
    And the database has the following users:
      | group_id | login |
      | 21       | john  |
      | 31       | jane  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | lock_membership_approved_at |
      | 14              | 21             | null                        |
      | 14              | 31             | null                        |
      | 15              | 31             | 2019-05-30 11:00:00         |
      | 16              | 31             | null                        |
      | 17              | 31             | null                        |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         |
      | 11       | 21        | join_request |

  Scenario: User tries to leave a group (s)he is not a member of
    Given I am the user with id "21"
    When I send a DELETE request to "/current-user/group-memberships/11"
    Then the response code should be 404
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Not Found",
      "error_text": "No such relation"
    }
    """
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the group id is wrong
    Given I am the user with id "21"
    When I send a DELETE request to "/current-user/group-memberships/abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails if the user doesn't exist
    Given I am the user with id "404"
    When I send a DELETE request to "/current-user/group-memberships/14"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"

  Scenario: Fails if require_lock_membership_approval_until = NOW() + 1
    Given the DB time now is "2037-04-28 23:59:59"
    And I am the user with id "31"
    When I send a DELETE request to "/current-user/group-memberships/15"
    Then the response code should be 403
    And the response error message should contain "User deletion is locked for this group"

  Scenario: Fails if lock_user_deletion_until > NOW()
    Given I am the user with id "31"
    When I send a DELETE request to "/current-user/group-memberships/15"
    Then the response code should be 403
    And the response error message should contain "User deletion is locked for this group"

  Scenario: Fails if the group membership is frozen
    Given I am the user with id "31"
    When I send a DELETE request to "/current-user/group-memberships/16"
    Then the response code should be 403
    And the response error message should contain "User deletion is locked for this group"

  Scenario: Fails if the group is a 'Base' group
    Given I am the user with id "31"
    When I send a DELETE request to "/current-user/group-memberships/17"
    Then the response code should be 403
    And the response error message should contain "User deletion is locked for this group"

  Scenario: Fails if leaving breaks entry conditions for the team
    Given I am the user with id "31"
    And the database has the following table "items":
      | id | default_language_tag | entry_min_admitted_members_ratio |
      | 2  | fr                   | All                              |
    And the database table "attempts" also has the following row:
      | participant_id | id | root_item_id |
      | 14             | 1  | 2            |
    And the database has the following table "results":
      | participant_id | attempt_id | item_id | started_at          |
      | 14             | 1          | 2       | 2019-05-30 11:00:00 |
    When I send a DELETE request to "/current-user/group-memberships/14"
    Then the response code should be 422
    And the response error message should contain "Entry conditions would not be satisfied"
