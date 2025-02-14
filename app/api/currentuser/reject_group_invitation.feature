Feature: User rejects an invitation to join a group
  Background:
    Given the database has the following table "groups":
      | id |
      | 11 |
      | 14 |
      | 21 |
    And the database has the following users:
      | group_id |
      | 21       |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type       |
      | 11       | 21        | invitation |

  Scenario: Successfully reject an invitation
    Given I am the user with id "21"
    When I send a POST request to "/current-user/group-invitations/11/reject"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "updated",
      "data": {"changed": true}
    }
    """
    And the table "group_pending_requests" should be empty
    And the table "group_membership_changes" should be:
      | group_id | member_id | action             | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 11       | 21        | invitation_refused | 21           | 1                                         |
    And the table "groups_ancestors" should remain unchanged
