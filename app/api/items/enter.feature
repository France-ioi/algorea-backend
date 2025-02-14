Feature: Enters a contest as a group (user self or team) (contestEnter)
  Background:
    Given the database has the following table "groups":
      | id | name         | type                | root_activity_id |
      | 10 | Class        | Class               | 10               |
      | 11 | Team 2       | Team                | 60               |
      | 21 | owner        | User                | null             |
      | 31 | john         | User                | 50               |
      | 41 | jane         | User                | null             |
      | 51 | jack         | User                | null             |
      | 98 | item60-group | ContestParticipants | null             |
      | 99 | item50-group | ContestParticipants | null             |
    And the database has the following users:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
      | 31       | john  | John        | Doe       |
      | 41       | jane  | Jane        | null      |
      | 51       | jack  | Jack        | Daniel    |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 10              | 31             |
      | 11              | 31             |
      | 11              | 41             |
      | 11              | 51             |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id | default_language_tag |
      | 10 | fr                   |
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
      | 11       | 30      | content            |
      | 31       | 30      | content            |
      | 98       | 10      | info               |
      | 98       | 20      | info               |
      | 99       | 10      | info               |
      | 99       | 20      | info               |
    And the database has the following table "attempts":
      | id | participant_id | created_at          |
      | 0  | 11             | 2019-05-30 11:00:00 |
      | 0  | 31             | 2019-05-30 11:00:00 |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | started_at          |
      | 0          | 11             | 30      | null                |
      | 0          | 31             | 10      | 2019-05-30 11:00:00 |
      | 0          | 31             | 30      | null                |
    And the DB time now is "3019-10-10 10:10:10"

  Scenario: Enter an individual contest
    Given the database table "items" also has the following row:
      | id | duration | requires_explicit_entry | entry_participant_type | entry_min_admitted_members_ratio | participants_group_id | default_language_tag | entering_time_min   | entering_time_max   |
      | 50 | 01:01:01 | 1                       | User                   | None                             | 99                    | fr                   | 2007-01-01 00:00:00 | 5000-01-01 00:00:00 |
    And the database table "items_ancestors" also has the following row:
      | ancestor_item_id | child_item_id |
      | 10               | 50            |
    And the database table "items_items" also has the following row:
      | parent_item_id | child_item_id | child_order |
      | 10             | 50            | 1           |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_enter_from      | can_enter_until     | source_group_id |
      | 11       | 50      | 2007-01-01 10:21:21 | 9999-12-31 23:59:59 | 11              |
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated       |
      | 11       | 50      | none                     |
      | 21       | 50      | solution                 |
      | 31       | 10      | content                  |
      | 31       | 50      | content_with_descendants |
    And the database has the following table "groups_contest_items":
      | group_id | item_id | additional_time |
      | 11       | 50      | 02:02:02        |
    And I am the user with id "31"
    When I send a POST request to "/items/10/50/enter?parent_attempt_id=0"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "message": "created",
      "success": true,
      "data": {
        "duration": "01:01:01",
        "entered_at": "3019-10-10T10:10:10Z"
      }
    }
    """
    And the table "attempts" should be:
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id | allows_submissions_until |
      | 0  | 11             | 2019-05-30 11:00:00 | null       | null              | null         | 9999-12-31 23:59:59      |
      | 0  | 31             | 2019-05-30 11:00:00 | null       | null              | null         | 9999-12-31 23:59:59      |
      | 1  | 31             | 3019-10-10 10:10:10 | 31         | 0                 | 50           | 3019-10-10 11:11:11      |
    And the table "results" should be:
      | attempt_id | participant_id | item_id | started_at          |
      | 0          | 11             | 30      | null                |
      | 0          | 31             | 10      | 2019-05-30 11:00:00 |
      | 0          | 31             | 20      | null                |
      | 0          | 31             | 30      | null                |
      | 1          | 31             | 50      | 3019-10-10 10:10:10 |
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id | expires_at          |
      | 10              | 31             | 9999-12-31 23:59:59 |
      | 11              | 31             | 9999-12-31 23:59:59 |
      | 11              | 41             | 9999-12-31 23:59:59 |
      | 11              | 51             | 9999-12-31 23:59:59 |
      | 99              | 31             | 3019-10-10 11:11:11 |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self | expires_at          |
      | 10                | 10             | 1       | 9999-12-31 23:59:59 |
      | 10                | 31             | 0       | 9999-12-31 23:59:59 |
      | 11                | 11             | 1       | 9999-12-31 23:59:59 |
      | 21                | 21             | 1       | 9999-12-31 23:59:59 |
      | 31                | 31             | 1       | 9999-12-31 23:59:59 |
      | 41                | 41             | 1       | 9999-12-31 23:59:59 |
      | 51                | 51             | 1       | 9999-12-31 23:59:59 |
      | 98                | 98             | 1       | 9999-12-31 23:59:59 |
      | 99                | 31             | 0       | 3019-10-10 11:11:11 |
      | 99                | 99             | 1       | 9999-12-31 23:59:59 |

  Scenario: Enter a team-only contest
    Given the database table "items" also has the following row:
      | id | duration | requires_explicit_entry | entry_participant_type | entry_min_admitted_members_ratio | entry_max_team_size | participants_group_id | default_language_tag |
      | 60 | 05:05:05 | 1                       | Team                   | Half                             | 3                   | 98                    | fr                   |
    And the database table "items_ancestors" also has the following row:
      | ancestor_item_id | child_item_id |
      | 10               | 60            |
    And the database table "items_items" also has the following row:
      | parent_item_id | child_item_id | child_order |
      | 10             | 60            | 1           |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | source_group_id | can_enter_from      | can_enter_until     |
      | 11       | 60      | 11              | 2007-01-01 10:21:21 | 9999-12-31 23:59:59 |
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated       |
      | 11       | 60      | content                  |
      | 21       | 60      | content_with_descendants |
    And the database has the following table "groups_contest_items":
      | group_id | item_id | additional_time |
      | 11       | 60      | 01:01:01        |
      | 31       | 60      | 02:02:02        |
      | 41       | 60      | 03:03:03        |
    And I am the user with id "31"
    When I send a POST request to "/items/60/enter?as_team_id=11&parent_attempt_id=0"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "message": "created",
      "success": true,
      "data": {
        "duration": "05:05:05",
        "entered_at": "3019-10-10T10:10:10Z"
      }
    }
    """
    And the table "attempts" should be:
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id | allows_submissions_until |
      | 0  | 11             | 2019-05-30 11:00:00 | null       | null              | null         | 9999-12-31 23:59:59      |
      | 0  | 31             | 2019-05-30 11:00:00 | null       | null              | null         | 9999-12-31 23:59:59      |
      | 1  | 11             | 3019-10-10 10:10:10 | 31         | 0                 | 60           | 3019-10-10 16:16:16      |
    And the table "results" should be:
      | attempt_id | participant_id | item_id | started_at          |
      | 0          | 11             | 10      | null                |
      | 0          | 11             | 20      | null                |
      | 0          | 11             | 30      | null                |
      | 0          | 31             | 10      | 2019-05-30 11:00:00 |
      | 0          | 31             | 30      | null                |
      | 1          | 11             | 60      | 3019-10-10 10:10:10 |
    And the table "results_propagate" should be empty
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id | expires_at          |
      | 10              | 31             | 9999-12-31 23:59:59 |
      | 11              | 31             | 9999-12-31 23:59:59 |
      | 11              | 41             | 9999-12-31 23:59:59 |
      | 11              | 51             | 9999-12-31 23:59:59 |
      | 98              | 11             | 3019-10-10 16:16:16 |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self | expires_at          |
      | 10                | 10             | 1       | 9999-12-31 23:59:59 |
      | 10                | 31             | 0       | 9999-12-31 23:59:59 |
      | 11                | 11             | 1       | 9999-12-31 23:59:59 |
      | 21                | 21             | 1       | 9999-12-31 23:59:59 |
      | 31                | 31             | 1       | 9999-12-31 23:59:59 |
      | 41                | 41             | 1       | 9999-12-31 23:59:59 |
      | 51                | 51             | 1       | 9999-12-31 23:59:59 |
      | 98                | 11             | 0       | 3019-10-10 16:16:16 |
      | 98                | 98             | 1       | 9999-12-31 23:59:59 |
      | 99                | 99             | 1       | 9999-12-31 23:59:59 |

  Scenario: Reenter a contest as a team
    Given the database table "items" also has the following row:
      | id | duration | requires_explicit_entry | entry_participant_type | allows_multiple_attempts | entry_min_admitted_members_ratio | entry_max_team_size | participants_group_id | default_language_tag |
      | 60 | 01:01:01 | 1                       | Team                   | 1                        | None                             | 10                  | 99                    | fr                   |
    And the database table "groups_groups" also has the following row:
      | parent_group_id | child_group_id | expires_at          |
      | 99              | 11             | 2019-05-30 11:00:00 |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | source_group_id | can_enter_from      | can_enter_until     |
      | 11       | 60      | 11              | 2007-01-01 10:21:21 | 9999-12-31 23:59:59 |
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated       |
      | 11       | 60      | solution                 |
      | 31       | 60      | content_with_descendants |
    And the database has the following table "groups_contest_items":
      | group_id | item_id | additional_time |
      | 11       | 60      | 02:02:02        |
    And the database table "attempts" also has the following row:
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id | allows_submissions_until |
      | 1  | 11             | 2019-05-29 11:00:00 | 31         | 0                 | 60           | 2019-05-30 11:00:00      |
    And the database table "results" also has the following row:
      | attempt_id | participant_id | item_id | started_at          |
      | 1          | 11             | 60      | 2019-05-29 11:00:00 |
    And I am the user with id "31"
    When I send a POST request to "/items/60/enter?as_team_id=11&parent_attempt_id=0"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "message": "created",
      "success": true,
      "data": {
        "duration": "01:01:01",
        "entered_at": "3019-10-10T10:10:10Z"
      }
    }
    """
    And the table "attempts" should be:
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id | allows_submissions_until |
      | 0  | 11             | 2019-05-30 11:00:00 | null       | null              | null         | 9999-12-31 23:59:59      |
      | 0  | 31             | 2019-05-30 11:00:00 | null       | null              | null         | 9999-12-31 23:59:59      |
      | 1  | 11             | 2019-05-29 11:00:00 | 31         | 0                 | 60           | 2019-05-30 11:00:00      |
      | 2  | 11             | 3019-10-10 10:10:10 | 31         | 0                 | 60           | 3019-10-10 13:13:13      |
    And the table "results" should be:
      | attempt_id | participant_id | item_id | started_at          |
      | 0          | 11             | 20      | null                |
      | 0          | 11             | 30      | null                |
      | 0          | 31             | 10      | 2019-05-30 11:00:00 |
      | 0          | 31             | 30      | null                |
      | 1          | 11             | 60      | 2019-05-29 11:00:00 |
      | 2          | 11             | 60      | 3019-10-10 10:10:10 |
    And the table "results_propagate" should be empty
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id | expires_at          |
      | 10              | 31             | 9999-12-31 23:59:59 |
      | 11              | 31             | 9999-12-31 23:59:59 |
      | 11              | 41             | 9999-12-31 23:59:59 |
      | 11              | 51             | 9999-12-31 23:59:59 |
      | 99              | 11             | 3019-10-10 13:13:13 |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self | expires_at          |
      | 10                | 10             | 1       | 9999-12-31 23:59:59 |
      | 10                | 31             | 0       | 9999-12-31 23:59:59 |
      | 11                | 11             | 1       | 9999-12-31 23:59:59 |
      | 21                | 21             | 1       | 9999-12-31 23:59:59 |
      | 31                | 31             | 1       | 9999-12-31 23:59:59 |
      | 41                | 41             | 1       | 9999-12-31 23:59:59 |
      | 51                | 51             | 1       | 9999-12-31 23:59:59 |
      | 98                | 98             | 1       | 9999-12-31 23:59:59 |
      | 99                | 11             | 0       | 3019-10-10 13:13:13 |
      | 99                | 99             | 1       | 9999-12-31 23:59:59 |

  Scenario: Enter a contest that doesn't have items.participants_group_id set
    Given the database table "items" also has the following row:
      | id | duration | requires_explicit_entry | entry_participant_type | entry_min_admitted_members_ratio | default_language_tag |
      | 50 | 01:01:01 | 1                       | User                   | None                             | fr                   |
    And the database table "permissions_granted" also has the following row:
      | group_id | item_id | source_group_id | can_enter_from      | can_enter_until     |
      | 11       | 50      | 11              | 2007-01-01 10:21:21 | 9999-12-31 23:59:59 |
    And the database table "permissions_generated" also has the following row:
      | group_id | item_id | can_view_generated       |
      | 11       | 50      | none                     |
      | 21       | 50      | solution                 |
      | 31       | 50      | content_with_descendants |
    And the database has the following table "groups_contest_items":
      | group_id | item_id | additional_time |
      | 11       | 50      | 02:02:02        |
    And I am the user with id "31"
    When I send a POST request to "/items/50/enter?parent_attempt_id=0"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "message": "created",
      "success": true,
      "data": {
        "duration": "01:01:01",
        "entered_at": "3019-10-10T10:10:10Z"
      }
    }
    """
    And the table "attempts" should be:
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id | allows_submissions_until |
      | 0  | 11             | 2019-05-30 11:00:00 | null       | null              | null         | 9999-12-31 23:59:59      |
      | 0  | 31             | 2019-05-30 11:00:00 | null       | null              | null         | 9999-12-31 23:59:59      |
      | 1  | 31             | 3019-10-10 10:10:10 | 31         | 0                 | 50           | 3019-10-10 11:11:11      |
    And the table "results" should be:
      | attempt_id | participant_id | item_id | started_at          |
      | 0          | 11             | 30      | null                |
      | 0          | 31             | 10      | 2019-05-30 11:00:00 |
      | 0          | 31             | 30      | null                |
      | 1          | 31             | 50      | 3019-10-10 10:10:10 |
    And the table "results_propagate" should be empty
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And logs should contain:
      """
      items.participants_group_id is not set for the item with id = 50
      """

  Scenario: Enter a contest with empty duration
    Given the database table "items" also has the following row:
      | id | duration | requires_explicit_entry | entry_participant_type | entry_min_admitted_members_ratio | participants_group_id | default_language_tag |
      | 50 | null     | 1                       | User                   | None                             | 99                    | fr                   |
    And the database table "items_ancestors" also has the following row:
      | ancestor_item_id | child_item_id |
      | 10               | 50            |
    And the database table "items_items" also has the following row:
      | parent_item_id | child_item_id | child_order |
      | 10             | 50            | 1           |
    And the database table "permissions_granted" also has the following row:
      | group_id | item_id | source_group_id | can_enter_from      | can_enter_until     |
      | 11       | 50      | 11              | 2007-01-01 10:21:21 | 9999-12-31 23:59:59 |
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated       |
      | 11       | 50      | none                     |
      | 21       | 50      | solution                 |
      | 31       | 50      | content_with_descendants |
    And the database has the following table "groups_contest_items":
      | group_id | item_id | additional_time |
      | 11       | 50      | 02:02:02        |
    And I am the user with id "31"
    When I send a POST request to "/items/50/enter?parent_attempt_id=0"
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "message": "created",
      "success": true,
      "data": {
        "duration": null,
        "entered_at": "3019-10-10T10:10:10Z"
      }
    }
    """
    And the table "attempts" should be:
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id | allows_submissions_until |
      | 0  | 11             | 2019-05-30 11:00:00 | null       | null              | null         | 9999-12-31 23:59:59      |
      | 0  | 31             | 2019-05-30 11:00:00 | null       | null              | null         | 9999-12-31 23:59:59      |
      | 1  | 31             | 3019-10-10 10:10:10 | 31         | 0                 | 50           | 9999-12-31 23:59:59      |
    And the table "results" should be:
      | attempt_id | participant_id | item_id | started_at          |
      | 0          | 11             | 30      | null                |
      | 0          | 31             | 10      | 2019-05-30 11:00:00 |
      | 0          | 31             | 20      | null                |
      | 0          | 31             | 30      | null                |
      | 1          | 31             | 50      | 3019-10-10 10:10:10 |
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id | expires_at          |
      | 10              | 31             | 9999-12-31 23:59:59 |
      | 11              | 31             | 9999-12-31 23:59:59 |
      | 11              | 41             | 9999-12-31 23:59:59 |
      | 11              | 51             | 9999-12-31 23:59:59 |
      | 99              | 31             | 9999-12-31 23:59:59 |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self | expires_at          |
      | 10                | 10             | 1       | 9999-12-31 23:59:59 |
      | 10                | 31             | 0       | 9999-12-31 23:59:59 |
      | 11                | 11             | 1       | 9999-12-31 23:59:59 |
      | 21                | 21             | 1       | 9999-12-31 23:59:59 |
      | 31                | 31             | 1       | 9999-12-31 23:59:59 |
      | 41                | 41             | 1       | 9999-12-31 23:59:59 |
      | 51                | 51             | 1       | 9999-12-31 23:59:59 |
      | 98                | 98             | 1       | 9999-12-31 23:59:59 |
      | 99                | 31             | 0       | 9999-12-31 23:59:59 |
      | 99                | 99             | 1       | 9999-12-31 23:59:59 |
