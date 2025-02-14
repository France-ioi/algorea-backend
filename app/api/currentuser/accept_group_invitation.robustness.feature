Feature: User accepts an invitation to join a group - robustness
  Background:
    Given the database has the following table "groups":
      | id | type    | require_personal_info_access_approval | frozen_membership |
      | 11 | Class   | none                                  | false             |
      | 13 | Friends | none                                  | false             |
      | 14 | Team    | none                                  | false             |
      | 15 | Team    | none                                  | false             |
      | 16 | Team    | view                                  | false             |
      | 17 | Team    | none                                  | true              |
    And the database has the following users:
      | group_id | login | temp_user |
      | 21       | john  | false     |
      | 22       | tmp   | true      |
    And the database has the following table "items":
      | id   | default_language_tag |
      | 1234 | fr                   |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 14              | 21             |
      | 21              | 13             |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         |
      | 11       | 21        | join_request |
      | 13       | 21        | invitation   |
      | 15       | 21        | invitation   |
      | 14       | 22        | invitation   |
      | 16       | 21        | invitation   |
      | 17       | 21        | invitation   |
    And the database has the following table "attempts":
      | participant_id | id | root_item_id |
      | 14             | 1  | 1234         |
      | 15             | 2  | 1234         |

  Scenario: User tries to create a cycle in the group relations graph
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-invitations/13/accept"
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
    And the table "groups_ancestors" should remain unchanged

  Scenario: User tries to accept an invitation that doesn't exist
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-invitations/11/accept"
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

  Scenario: User tries to accept an invitation to join a team while being a member of another team participating in same contests
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-invitations/15/accept"
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
    And the table "groups_ancestors" should remain unchanged

  Scenario: User tries to accept an invitation to join a team while entry conditions would not be met if he joins
    Given I am the user with id "21"
    And the database has the following table "items":
      | id | default_language_tag | entry_max_team_size |
      | 2  | fr                   | 0                   |
    And the database table "attempts" also has the following row:
      | participant_id | id | root_item_id |
      | 16             | 1  | 2            |
    And the database has the following table "results":
      | participant_id | attempt_id | item_id | started_at          |
      | 16             | 1          | 2       | 2019-05-30 11:00:00 |
    When I send a POST request to "/current-user/group-invitations/16/accept?approvals=personal_info_view"
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
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the group id is wrong
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-invitations/abc/accept"
    Then the response code should be 400
    And the response error message should contain "Wrong value for group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails if the user doesn't exist
    Given I am the user with id "404"
    When I send a POST request to "/current-user/group-invitations/14/accept"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"

  Scenario: Fails if the user is temporary
    Given I am the user with id "22"
    When I send a POST request to "/current-user/group-invitations/14/accept"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: Fails if the group is a user
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-invitations/21/accept"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: User tries to accept an invitation to join a group that requires approvals which are not given
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-invitations/16/accept"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Missing required approvals",
      "data": {
        "missing_approvals": ["personal_info_view"]
      }
    }
    """
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: User tries to accept an invitation to join a group with frozen membership
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-invitations/17/accept"
    Then the response code should be 422
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Unprocessable Entity",
      "error_text": "Group membership is frozen"
    }
    """
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
