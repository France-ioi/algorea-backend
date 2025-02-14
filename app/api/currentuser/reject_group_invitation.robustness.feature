Feature: User rejects an invitation to join a group - robustness
  Background:
    Given the database has the following table "groups":
      | id |
      | 11 |
      | 13 |
      | 14 |
      | 21 |
    And the database has the following user:
      | group_id | login |
      | 21       | john  |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         |
      | 11       | 21        | join_request |
      | 13       | 21        | invitation   |

  Scenario: User tries to reject an invitation that doesn't exist
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-invitations/11/reject"
    Then the response code should be 404
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Not Found",
      "error_text": "No such relation"
    }
    """
    And the table "group_pending_requests" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the group id is wrong
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-invitations/abc/reject"
    Then the response code should be 400
    And the response error message should contain "Wrong value for group_id (should be int64)"
    And the table "group_pending_requests" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails if the user doesn't exist
    Given I am the user with id "404"
    When I send a POST request to "/current-user/group-invitations/13/reject"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"

