Feature: Reject group requests
  Background:
    Given the database has the following table "groups":
      | id  |
      | 11  |
      | 13  |
      | 14  |
      | 22  |
      | 31  |
      | 111 |
      | 121 |
      | 122 |
      | 123 |
      | 131 |
      | 141 |
      | 151 |
    And the database has the following user:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 13              | 111            |
      | 13              | 121            |
      | 13              | 123            |
      | 13              | 151            |
      | 22              | 13             |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         | at                          |
      | 13       | 21        | invitation   | {{relativeTimeDB("-170h")}} |
      | 13       | 31        | join_request | {{relativeTimeDB("-168h")}} |
      | 14       | 11        | invitation   | {{relativeTimeDB("-167h")}} |
      | 14       | 21        | join_request | {{relativeTimeDB("-166h")}} |
      | 13       | 141       | join_request | {{relativeTimeDB("-165h")}} |

  Scenario Outline: Reject requests
    Given I am the user with id "21"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage   |
      | 13       | 21         | <can_manage> |
    When I send a POST request to "/groups/13/join-requests/reject?group_ids=31,141,21,11,13,22,151"
    And the response body should be, in JSON:
    """
    {
      "data": {
        "141": "success",
        "31": "success",
        "11": "invalid",
        "13": "invalid",
        "21": "invalid",
        "22": "invalid",
        "151": "invalid"
      },
      "message": "updated",
      "success": true
    }
    """
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should be:
      | group_id | member_id | type         | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 13       | 21        | invitation   | 0                                         |
      | 14       | 11        | invitation   | 0                                         |
      | 14       | 21        | join_request | 0                                         |
    And the table "group_membership_changes" should be:
      | group_id | member_id | action               | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 13       | 31        | join_request_refused | 21           | 1                                         |
      | 13       | 141       | join_request_refused | 21           | 1                                         |
    And the table "groups_ancestors" should remain unchanged
  Examples:
    | can_manage            |
    | memberships           |
    | memberships_and_group |
