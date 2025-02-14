Feature: User sends a request to join a group - robustness
  Background:
    Given the database has the following table "groups":
      | id | is_public | type    | require_personal_info_access_approval | require_lock_membership_approval_until | require_watch_approval | frozen_membership | enforce_max_participants | max_participants |
      | 11 | 1         | Class   | none                                  | null                                   | 0                      | false             | false                    | 0                |
      | 13 | 1         | Friends | none                                  | null                                   | 0                      | false             | false                    | 0                |
      | 14 | 1         | Team    | none                                  | null                                   | 0                      | false             | false                    | 0                |
      | 15 | 0         | Club    | none                                  | null                                   | 0                      | false             | false                    | 0                |
      | 16 | 1         | Team    | edit                                  | 9999-12-31 23:59:59                    | 1                      | false             | false                    | 0                |
      | 17 | 1         | Team    | none                                  | null                                   | 0                      | false             | false                    | 0                |
      | 18 | 1         | Team    | none                                  | null                                   | 0                      | true              | false                    | 0                |
      | 19 | 1         | Team    | none                                  | null                                   | 0                      | false             | false                    | 0                |
      | 20 | 1         | Team    | none                                  | null                                   | 0                      | false             | true                     | 0                |
      | 21 | 0         | User    | none                                  | null                                   | 0                      | false             | false                    | 0                |
      | 22 | 0         | User    | none                                  | null                                   | 0                      | false             | false                    | 0                |
      | 23 | 1         | User    | none                                  | null                                   | 0                      | false             | false                    | 0                |
    And the database has the following users:
      | group_id | login | temp_user |
      | 21       | john  | false     |
      | 22       | tmp   | true      |
      | 23       | jane  | false     |
    And the database has the following table "items":
      | id   | default_language_tag |
      | 1234 | fr                   |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 17       | 21         | memberships |
      | 19       | 21         | memberships |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 16              | 21             |
      | 21              | 13             |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         |
      | 11       | 21        | invitation   |
      | 14       | 21        | join_request |
    And the database has the following table "attempts":
      | participant_id | id | root_item_id |
      | 14             | 1  | 1234         |
      | 16             | 2  | 1234         |
      | 17             | 3  | 1234         |

  Scenario: User tries to create a cycle in the group relations graph
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-requests/13"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Cycles in the group relations graph are not allowed"
    }
    """
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged

  Scenario: User tries to send a request while a conflicting relation exists
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-requests/11"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "A conflicting relation exists"
    }
    """
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged

  Scenario Outline: User tries to send a request while while entry conditions would not be met if he joins
    Given I am the user with id "21"
    And the database has the following table "items":
      | id | default_language_tag | entry_max_team_size |
      | 2  | fr                   | 0                   |
    And the database table "attempts" also has the following row:
      | participant_id | id | root_item_id |
      | <team_id>      | 1  | 2            |
    And the database has the following table "results":
      | participant_id | attempt_id | item_id | started_at          |
      | <team_id>      | 1          | 2       | 2019-05-30 11:00:00 |
    When I send a POST request to "/current-user/group-requests/<team_id>"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Entry conditions would not be satisfied"
    }
    """
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged
  Examples:
    | team_id |
    | 16      |
    | 19      |

  Scenario: User tries to send a request to join a team while being a member of another team participating in same contests
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-requests/14"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Team's participations are in conflict with the user's participations"
    }
    """
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Team owner tries to send a request to join a team while being a member of another team participating in same contests
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-requests/17"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Team's participations are in conflict with the user's participations"
    }
    """
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the group id is wrong
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-requests/abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails if the user doesn't exist
    Given I am the user with id "404"
    When I send a POST request to "/current-user/group-requests/14"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"

  Scenario: Can't send request to a group having is_public=0
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-requests/15"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: Can't send request to a group when all approvals are missing
    Given I am the user with id "23"
    When I send a POST request to "/current-user/group-requests/16"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Missing required approvals",
      "data": {"missing_approvals": ["personal_info_view","lock_membership","watch"]}
    }
    """

  Scenario: Can't send request to a group when lock_membership & watch approvals are missing
    Given I am the user with id "23"
    When I send a POST request to "/current-user/group-requests/16?approvals=personal_info_view"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Missing required approvals",
      "data": {"missing_approvals": ["lock_membership","watch"]}
    }
    """

  Scenario: Can't send request to a group when watch approval is missing
    Given I am the user with id "23"
    When I send a POST request to "/current-user/group-requests/16?approvals=personal_info_view,lock_membership"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Missing required approvals",
      "data": {"missing_approvals": ["watch"]}
    }
    """

  Scenario: Can't send request to a group when an approval is missing even while being a group manager
    Given I am the user with id "23"
    And the database table "group_managers" also has the following rows:
      | group_id | manager_id | can_manage  |
      | 16       | 21         | memberships |
    When I send a POST request to "/current-user/group-requests/16?approvals=personal_info_view,lock_membership"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Missing required approvals",
      "data": {"missing_approvals": ["watch"]}
    }
    """

  Scenario: Can't send request to a user
    Given I am the user with id "23"
    When I send a POST request to "/current-user/group-requests/23?approvals=personal_info_view,lock_membership"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: Can't send request to a user even while being a group manager
    Given I am the user with id "23"
    And the database table "group_managers" also has the following rows:
      | group_id | manager_id | can_manage  |
      | 23       | 23         | memberships |
    When I send a POST request to "/current-user/group-requests/23?approvals=personal_info_view,lock_membership"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: Can't send request to a group with frozen membership
    Given I am the user with id "23"
    When I send a POST request to "/current-user/group-requests/18"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Group membership is frozen"
    }
    """

  Scenario: Can't send request to a group when enforce_max_participants=1 and the limit is exceeded
    Given I am the user with id "23"
    When I send a POST request to "/current-user/group-requests/20"
    Then the response code should be 409
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Conflict",
      "error_text": "The group is full"
    }
    """

  Scenario: Can't send request to a group with frozen membership even while being a group manager
    Given I am the user with id "23"
    And the database table "group_managers" also has the following rows:
      | group_id | manager_id | can_manage  |
      | 18       | 21         | memberships |
    When I send a POST request to "/current-user/group-requests/18"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Group membership is frozen"
    }
    """

  Scenario: A temporary user cannot send a request to a group
    Given I am the user with id "22"
    When I send a POST request to "/current-user/group-requests/17"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
