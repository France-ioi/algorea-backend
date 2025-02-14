Feature: Remove members from a group (groupRemoveMembers)
  Background:
    Given the database has the following table "groups":
      | id  |
      | 13  |
      | 14  |
      | 131 |
      | 132 |
    And the database has the following users:
      | group_id | login  |
      | 21       | owner  |
      | 31       | john   |
      | 41       | jane   |
      | 51       | jack   |
      | 61       | james  |
      | 71       | jacob  |
      | 81       | janis  |
      | 91       | jeff   |
      | 101      | jenna  |
      | 111      | jannet |
      | 121      | judith |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 13              | 51             |
      | 13              | 61             |
      | 13              | 91             |
      | 13              | 111            |
      | 13              | 131            |
      | 14              | 41             |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type         |
      | 13       | 21        | invitation   |
      | 13       | 41        | join_request |
      | 13       | 101       | join_request |
      | 14       | 51        | join_request |

  Scenario Outline: Remove members
    Given I am the user with id "21"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage   |
      | 13       | 21         | <can_manage> |
    When I send a DELETE request to "/groups/13/members?user_ids=31,41,51,61,71,81,91,101,111,121,131,404"
    And the response body should be, in JSON:
    """
    {
      "data": {
        "31":  "invalid",
        "41":  "invalid",
        "51":  "success",
        "61":  "success",
        "71":  "invalid",
        "81":  "invalid",
        "91":  "success",
        "101": "invalid",
        "111": "success",
        "121": "invalid",
        "131": "not_found",
        "404": "not_found"
      },
      "message": "deleted",
      "success": true
    }
    """
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 13              | 131            |
      | 14              | 41             |
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be:
      | group_id | member_id | action  | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 13       | 51        | removed | 21           | 1                                         |
      | 13       | 61        | removed | 21           | 1                                         |
      | 13       | 91        | removed | 21           | 1                                         |
      | 13       | 111       | removed | 21           | 1                                         |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self |
      | 13                | 13             | 1       |
      | 13                | 131            | 0       |
      | 14                | 14             | 1       |
      | 14                | 41             | 0       |
      | 21                | 21             | 1       |
      | 31                | 31             | 1       |
      | 41                | 41             | 1       |
      | 51                | 51             | 1       |
      | 61                | 61             | 1       |
      | 71                | 71             | 1       |
      | 81                | 81             | 1       |
      | 91                | 91             | 1       |
      | 101               | 101            | 1       |
      | 111               | 111            | 1       |
      | 121               | 121            | 1       |
      | 131               | 131            | 1       |
      | 132               | 132            | 1       |
  Examples:
    | can_manage            |
    | memberships           |
    | memberships_and_group |
