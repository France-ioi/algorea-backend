Feature: Reject requests to leave a group
  Background:
    Given the database has the following table "groups":
      | id  | type    |
      | 11  | Class   |
      | 13  | Team    |
      | 14  | Friends |
      | 31  | User    |
      | 111 | User    |
      | 121 | User    |
      | 122 | User    |
      | 123 | User    |
      | 131 | User    |
      | 141 | User    |
      | 151 | User    |
      | 161 | User    |
      | 444 | Team    |
    And the database has the following user:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 13              | 31             |
      | 13              | 111            |
      | 13              | 121            |
      | 13              | 123            |
      | 13              | 141            |
      | 13              | 151            |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type          |
      | 13       | 21        | invitation    |
      | 13       | 31        | leave_request |
      | 13       | 141       | leave_request |
      | 13       | 161       | join_request  |
      | 14       | 11        | invitation    |
      | 14       | 21        | join_request  |

  Scenario: Reject requests to leave a group
    Given I am the user with id "21"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 13       | 21         | memberships |
    When I send a POST request to "/groups/13/leave-requests/reject?group_ids=31,141,21,11,13,122,151"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "data": {
        "141": "success",
        "31": "success",
        "11": "invalid",
        "13": "invalid",
        "21": "invalid",
        "122": "invalid",
        "151": "invalid"
      },
      "message": "updated",
      "success": true
    }
    """
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should be:
      | group_id | member_id | type         |
      | 13       | 21        | invitation   |
      | 13       | 161       | join_request |
      | 14       | 11        | invitation   |
      | 14       | 21        | join_request |
    And the table "group_membership_changes" should be:
      | group_id | member_id | action                | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 13       | 31        | leave_request_refused | 21           | 1                                         |
      | 13       | 141       | leave_request_refused | 21           | 1                                         |
    And the table "groups_ancestors" should remain unchanged
