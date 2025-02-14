Feature: Accept group requests
  Background:
    Given the database has the following table "groups":
      | id  | type    | require_personal_info_access_approval | enforce_max_participants | max_participants |
      | 11  | Class   | none                                  | false                    | null             |
      | 13  | Team    | none                                  | false                    | null             |
      | 14  | Friends | view                                  | true                     | 7                |
      | 31  | User    | none                                  | false                    | null             |
      | 111 | User    | none                                  | false                    | null             |
      | 121 | User    | none                                  | false                    | null             |
      | 122 | User    | none                                  | false                    | null             |
      | 123 | User    | none                                  | false                    | null             |
      | 131 | User    | none                                  | false                    | null             |
      | 141 | User    | none                                  | false                    | null             |
      | 151 | User    | none                                  | false                    | null             |
      | 161 | User    | none                                  | false                    | null             |
    And the database has the following user:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
    And the database has the following table "items":
      | id   | default_language_tag |
      | 1234 | fr                   |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | personal_info_view_approved_at | lock_membership_approved_at | watch_approved_at |
      | 14              | 111            | null                           | null                        | null              |
      | 14              | 121            | null                           | null                        | null              |
      | 14              | 123            | null                           | null                        | null              |
      | 14              | 151            | null                           | null                        | null              |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id | default_language_tag |
      | 20 | fr                   |
      | 30 | fr                   |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 20               | 30            |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 20             | 30            | 1           |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 13       | 20      | content            |
      | 31       | 30      | content            |
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 31             |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id |
      | 0          | 31             | 30      |

  Scenario: Accept requests
    Given I am the user with id "21"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 14       | 21         | memberships |
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         | personal_info_view_approved | lock_membership_approved | watch_approved | at                      |
      | 13       | 11        | invitation   | 0                           | 0                        | 0              | 2019-06-05 00:00:00.000 |
      | 13       | 21        | join_request | 0                           | 0                        | 0              | 2019-06-06 00:00:00.000 |
      | 14       | 21        | invitation   | 0                           | 0                        | 0              | 2019-06-01 00:00:00.000 |
      | 14       | 31        | join_request | 1                           | 0                        | 0              | 2019-06-02 00:00:00.000 |
      | 14       | 141       | join_request | 1                           | 1                        | 1              | 2019-06-03 00:00:00.000 |
      | 14       | 161       | join_request | 0                           | 0                        | 0              | 2019-06-04 00:00:00.000 |
    When I send a POST request to "/groups/14/join-requests/accept?group_ids=31,141,21,11,13,122,151"
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
    And the table "groups_groups" should remain unchanged, regardless of the row with parent_group_id "14"
    And the table "groups_groups" at parent_group_id "14" should be:
      | parent_group_id | child_group_id | personal_info_view_approved_at | lock_membership_approved_at | watch_approved_at   |
      | 14              | 31             | 2019-06-02 00:00:00            | null                        | null                |
      | 14              | 111            | null                           | null                        | null                |
      | 14              | 121            | null                           | null                        | null                |
      | 14              | 123            | null                           | null                        | null                |
      | 14              | 141            | 2019-06-03 00:00:00            | 2019-06-03 00:00:00         | 2019-06-03 00:00:00 |
      | 14              | 151            | null                           | null                        | null                |
    And the table "group_pending_requests" should be:
      | group_id | member_id | type         |
      | 13       | 11        | invitation   |
      | 13       | 21        | join_request |
      | 14       | 21        | invitation   |
      | 14       | 161       | join_request |
    And the table "group_membership_changes" should be:
      | group_id | member_id | action                | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 14       | 31        | join_request_accepted | 21           | 1                                         |
      | 14       | 141       | join_request_accepted | 21           | 1                                         |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self |
      | 11                | 11             | 1       |
      | 13                | 13             | 1       |
      | 14                | 14             | 1       |
      | 14                | 31             | 0       |
      | 14                | 111            | 0       |
      | 14                | 121            | 0       |
      | 14                | 123            | 0       |
      | 14                | 141            | 0       |
      | 14                | 151            | 0       |
      | 21                | 21             | 1       |
      | 31                | 31             | 1       |
      | 111               | 111            | 1       |
      | 121               | 121            | 1       |
      | 122               | 122            | 1       |
      | 123               | 123            | 1       |
      | 131               | 131            | 1       |
      | 141               | 141            | 1       |
      | 151               | 151            | 1       |
      | 161               | 161            | 1       |
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: The group is full
    Given I am the user with id "21"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 14       | 21         | memberships |
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         | personal_info_view_approved | lock_membership_approved | watch_approved | at                      |
      | 14       | 11        | invitation   | 0                           | 0                        | 0              | 2019-06-05 00:00:00.000 |
      | 14       | 21        | invitation   | 0                           | 0                        | 0              | 2019-06-01 00:00:00.000 |
      | 14       | 31        | join_request | 1                           | 0                        | 0              | 2019-06-02 00:00:00.000 |
      | 14       | 141       | join_request | 1                           | 1                        | 1              | 2019-06-03 00:00:00.000 |
      | 14       | 161       | join_request | 0                           | 0                        | 0              | 2019-06-04 00:00:00.000 |
    When I send a POST request to "/groups/14/join-requests/accept?group_ids=31,141,21,11,13,122,151"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "data": {
        "141": "full",
        "31": "full",
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
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario Outline: Accept team requests
    Given I am the user with id "21"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 13       | 21         | memberships |
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         | personal_info_view_approved   | lock_membership_approved   | watch_approved   | at                      |
      | 13       | 21        | invitation   | 0                             | 0                          | 0                | 2019-06-01 00:00:00.000 |
      | 13       | 31        | join_request | <personal_info_view_approved> | <lock_membership_approved> | <watch_approved> | <at>                    |
    When I send a POST request to "/groups/13/join-requests/accept?group_ids=31"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "data": {
        "31": "<result>"
      },
      "message": "updated",
      "success": true
    }
    """
    And the table "groups_groups" should remain unchanged, regardless of the row with parent_group_id "13"
    And the table "groups_groups" at parent_group_id "13" should be:
      | parent_group_id | child_group_id | personal_info_view_approved_at   | lock_membership_approved_at   | watch_approved_at   |
      | 13              | 31             | <personal_info_view_approved_at> | <lock_membership_approved_at> | <watch_approved_at> |
    And the table "group_pending_requests" should be:
      | group_id | member_id | type         |
      | 13       | 21        | invitation   |
    And the table "group_membership_changes" should be:
      | group_id | member_id | action                | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 13       | 31        | join_request_accepted | 21           | 1                                         |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self |
      | 11                | 11             | 1       |
      | 13                | 13             | 1       |
      | 14                | 14             | 1       |
      | 14                | 111            | 0       |
      | 14                | 121            | 0       |
      | 14                | 123            | 0       |
      | 14                | 151            | 0       |
      | 21                | 21             | 1       |
      | 31                | 31             | 1       |
      | 111               | 111            | 1       |
      | 121               | 121            | 1       |
      | 122               | 122            | 1       |
      | 123               | 123            | 1       |
      | 131               | 131            | 1       |
      | 141               | 141            | 1       |
      | 151               | 151            | 1       |
      | 161               | 161            | 1       |
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged
  Examples:
    | result  | personal_info_view_approved | lock_membership_approved | watch_approved | at                      | personal_info_view_approved_at | lock_membership_approved_at | watch_approved_at   |
    | success | 1                           | 0                        | 0              | 2019-06-01 00:00:00.000 | 2019-06-01 00:00:00            | null                        | null                |
    | success | 0                           | 1                        | 0              | 2019-06-03 00:00:00.000 | null                           | 2019-06-03 00:00:00         | null                |
    | success | 0                           | 0                        | 1              | 2019-06-04 00:00:00.000 | null                           | null                        | 2019-06-04 00:00:00 |

  Scenario: Accept requests for a team while skipping members of other teams participating in the same contests
    Given I am the user with id "21"
    And the database table "groups" also has the following row:
      | id  | type |
      | 444 | Team |
    And the database table "groups_groups" also has the following rows:
      | parent_group_id | child_group_id |
      | 444             | 31             |
    And the groups ancestors are computed
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 13       | 21         | memberships_and_group |
    And the database table "attempts" also has the following rows:
      | participant_id | id | root_item_id |
      | 13             | 1  | 1234         |
      | 444            | 2  | 1234         |
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         | personal_info_view_approved | lock_membership_approved | watch_approved | at                      |
      | 13       | 31        | join_request | 1                           | 1                        | 1              | 2019-06-04 00:00:00.000 |
    When I send a POST request to "/groups/13/join-requests/accept?group_ids=31"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "data": {
          "31": "in_another_team"
        },
        "message": "updated",
        "success": true
      }
      """
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Accept request for a team for which entry conditions would become not satisfied
    Given I am the user with id "21"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 13       | 21         | memberships_and_group |
    And the database has the following table "items":
      | id | default_language_tag | entry_min_admitted_members_ratio |
      | 1  | fr                   | All                              |
    And the database has the following table "attempts":
      | participant_id | id | root_item_id |
      | 13             | 1  | 1            |
    And the database has the following table "results":
      | participant_id | attempt_id | item_id | started_at          |
      | 13             | 1          | 1       | 2019-05-30 11:00:00 |
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         | personal_info_view_approved | lock_membership_approved | watch_approved | at                      |
      | 13       | 31        | join_request | 1                           | 1                        | 1              | 2019-06-04 00:00:00.000 |
    When I send a POST request to "/groups/13/join-requests/accept?group_ids=31"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "data": {
          "31": "entry_condition_failed"
        },
        "message": "updated",
        "success": true
      }
      """
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Checks approvals if required
    Given I am the user with id "21"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 14       | 21         | memberships_and_group |
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         | personal_info_view_approved | lock_membership_approved | watch_approved | at                      |
      | 14       | 21        | join_request | 0                           | 0                        | 0              | 2019-06-06 00:00:00.000 |
    When I send a POST request to "/groups/14/join-requests/accept?group_ids=21"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "data": {
          "21": "approvals_missing"
        },
        "message": "updated",
        "success": true
      }
      """
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged
