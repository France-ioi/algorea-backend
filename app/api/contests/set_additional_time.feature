Feature: Set additional time in the contest for the group (contestSetAdditionalTime)
  Background:
    Given the database has the following table "groups":
      | id | name    | type    |
      | 10 | Parent  | Club    |
      | 11 | Group A | Class   |
      | 13 | Group B | Other   |
      | 14 | Group B | Friends |
      | 15 | Club    | Club    |
      | 16 | Team    | Team    |
      | 33 | item10  | Other   |
      | 34 | item50  | Other   |
      | 35 | item60  | Other   |
      | 36 | item70  | Other   |
    And the database has the following users:
      | group_id | login |
      | 21       | owner |
      | 31       | john  |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_grant_group_access | can_watch_members |
      | 13       | 21         | true                   | true              |
      | 14       | 21         | false                  | false             |
      | 15       | 21         | true                   | true              |
      | 16       | 21         | true                   | true              |
      | 31       | 21         | true                   | true              |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | expires_at          |
      | 10              | 11             | 9999-12-31 23:59:59 |
      | 10              | 21             | 9999-12-31 23:59:59 |
      | 11              | 13             | 9999-12-31 23:59:59 |
      | 13              | 14             | 9999-12-31 23:59:59 |
      | 34              | 13             | 9999-12-31 23:59:59 |
      | 34              | 14             | 9999-12-31 23:59:59 |
      | 36              | 13             | 9999-12-31 23:59:59 |
      | 36              | 14             | 2018-12-31 23:59:59 |
      | 36              | 15             | 2018-12-31 23:59:59 |
      | 36              | 16             | 2018-12-31 23:59:59 |
      | 36              | 31             | 9999-12-31 23:59:59 |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id | duration | participants_group_id | default_language_tag |
      | 10 | 00:00:02 | 33                    | fr                   |
      | 50 | 00:00:00 | 34                    | fr                   |
      | 60 | 00:00:01 | 35                    | fr                   |
      | 70 | 00:00:03 | 36                    | fr                   |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 10               | 50            |
      | 10               | 70            |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 10             | 50            | 1           |
      | 10             | 70            | 1           |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       | can_grant_view_generated | can_watch_generated |
      | 10       | 50      | none                     | enter                    | none                |
      | 11       | 50      | none                     | none                     | none                |
      | 11       | 60      | none                     | none                     | none                |
      | 11       | 70      | content_with_descendants | none                     | none                |
      | 13       | 50      | content                  | none                     | none                |
      | 13       | 60      | info                     | none                     | none                |
      | 21       | 50      | content                  | none                     | result              |
      | 21       | 60      | content_with_descendants | none                     | none                |
      | 21       | 70      | content_with_descendants | content                  | answer              |
      | 36       | 10      | info                     | none                     | none                |
    And the database has the following table "groups_contest_items":
      | group_id | item_id | additional_time |
      | 10       | 50      | 01:00:00        |
      | 11       | 50      | 00:01:00        |
      | 13       | 50      | 00:00:01        |
      | 13       | 60      | 00:00:30        |
      | 21       | 50      | 00:01:00        |
      | 21       | 60      | 00:01:00        |
      | 21       | 70      | 00:01:00        |
    And the database has the following table "attempts":
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id | allows_submissions_until | ended_at            |
      | 1  | 13             | 3018-05-29 06:38:38 | 21         | 0                 | 50           | 2018-12-31 23:59:59      | null                |
      | 1  | 14             | 3019-05-29 06:38:38 | 21         | 0                 | 50           | 9999-12-31 23:59:59      | 2019-05-30 11:00:00 |
      | 1  | 15             | 2018-05-29 06:38:38 | 21         | 0                 | 70           | 2018-12-31 23:59:59      | null                |
      | 1  | 16             | 3019-05-29 06:38:38 | 21         | 0                 | 70           | 2018-12-31 23:59:59      | null                |
      | 1  | 31             | 3017-05-29 06:38:38 | 21         | 0                 | 70           | 9999-12-31 23:59:59      | null                |
      | 2  | 14             | 3019-05-29 06:38:38 | 21         | 0                 | 50           | 9999-12-31 23:59:59      | null                |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | started_at          |
      | 1          | 13             | 50      | 3018-05-29 06:38:38 |
      | 1          | 13             | 70      | 3018-05-29 06:38:38 |
      | 1          | 14             | 50      | 3019-05-29 06:38:38 |
      | 1          | 15             | 70      | 2018-10-31 23:59:59 |
      | 1          | 16             | 70      | 3019-10-31 23:59:59 |
      | 1          | 31             | 70      | 3017-05-29 06:38:38 |

  Scenario: Updates an existing row
    Given I am the user with id "21"
    When I send a PUT request to "/contests/50/groups/13/additional-times?seconds=3020399"
    Then the response code should be 200
    And the response should be "updated"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should be:
      | group_id | item_id | additional_time |
      | 10       | 50      | 01:00:00        |
      | 11       | 50      | 00:01:00        |
      | 13       | 50      | 838:59:59       |
      | 13       | 60      | 00:00:30        |
      | 21       | 50      | 00:01:00        |
      | 21       | 60      | 00:01:00        |
      | 21       | 70      | 00:01:00        |
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id | expires_at          |
      | 10              | 11             | 9999-12-31 23:59:59 |
      | 10              | 21             | 9999-12-31 23:59:59 |
      | 11              | 13             | 9999-12-31 23:59:59 |
      | 13              | 14             | 9999-12-31 23:59:59 |
      | 34              | 13             | 3018-07-03 06:39:37 |
      | 34              | 14             | 9999-12-31 23:59:59 |
      | 36              | 13             | 9999-12-31 23:59:59 |
      | 36              | 14             | 2018-12-31 23:59:59 |
      | 36              | 15             | 2018-12-31 23:59:59 |
      | 36              | 16             | 2018-12-31 23:59:59 |
      | 36              | 31             | 9999-12-31 23:59:59 |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self | expires_at          |
      | 10                | 10             | 1       | 9999-12-31 23:59:59 |
      | 10                | 11             | 0       | 9999-12-31 23:59:59 |
      | 10                | 13             | 0       | 9999-12-31 23:59:59 |
      | 10                | 14             | 0       | 9999-12-31 23:59:59 |
      | 10                | 21             | 0       | 9999-12-31 23:59:59 |
      | 11                | 11             | 1       | 9999-12-31 23:59:59 |
      | 11                | 13             | 0       | 9999-12-31 23:59:59 |
      | 11                | 14             | 0       | 9999-12-31 23:59:59 |
      | 13                | 13             | 1       | 9999-12-31 23:59:59 |
      | 13                | 14             | 0       | 9999-12-31 23:59:59 |
      | 14                | 14             | 1       | 9999-12-31 23:59:59 |
      | 15                | 15             | 1       | 9999-12-31 23:59:59 |
      | 16                | 16             | 1       | 9999-12-31 23:59:59 |
      | 21                | 21             | 1       | 9999-12-31 23:59:59 |
      | 31                | 31             | 1       | 9999-12-31 23:59:59 |
      | 33                | 33             | 1       | 9999-12-31 23:59:59 |
      | 34                | 13             | 0       | 3018-07-03 06:39:37 |
      | 34                | 14             | 0       | 9999-12-31 23:59:59 |
      | 34                | 34             | 1       | 9999-12-31 23:59:59 |
      | 35                | 35             | 1       | 9999-12-31 23:59:59 |
      | 36                | 13             | 0       | 9999-12-31 23:59:59 |
      | 36                | 14             | 0       | 9999-12-31 23:59:59 |
      | 36                | 31             | 0       | 9999-12-31 23:59:59 |
      | 36                | 36             | 1       | 9999-12-31 23:59:59 |
    And the table "attempts" should be:
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id | allows_submissions_until |
      | 1  | 13             | 3018-05-29 06:38:38 | 21         | 0                 | 50           | 3018-07-03 06:39:37      |
      | 1  | 14             | 3019-05-29 06:38:38 | 21         | 0                 | 50           | 9999-12-31 23:59:59      |
      | 1  | 15             | 2018-05-29 06:38:38 | 21         | 0                 | 70           | 2018-12-31 23:59:59      |
      | 1  | 16             | 3019-05-29 06:38:38 | 21         | 0                 | 70           | 2018-12-31 23:59:59      |
      | 1  | 31             | 3017-05-29 06:38:38 | 21         | 0                 | 70           | 9999-12-31 23:59:59      |
      | 2  | 14             | 3019-05-29 06:38:38 | 21         | 0                 | 50           | 9999-12-31 23:59:59      |
    And the table "results" should remain unchanged

  Scenario: Creates a new row
    Given I am the user with id "21"
    When I send a PUT request to "/contests/70/groups/13/additional-times?seconds=-3020399"
    Then the response code should be 200
    And the response should be "updated"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should be:
      | group_id | item_id | additional_time |
      | 10       | 50      | 01:00:00        |
      | 11       | 50      | 00:01:00        |
      | 13       | 50      | 00:00:01        |
      | 13       | 60      | 00:00:30        |
      | 13       | 70      | -838:59:59      |
      | 21       | 50      | 00:01:00        |
      | 21       | 60      | 00:01:00        |
      | 21       | 70      | 00:01:00        |
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id | expires_at          |
      | 10              | 11             | 9999-12-31 23:59:59 |
      | 10              | 21             | 9999-12-31 23:59:59 |
      | 11              | 13             | 9999-12-31 23:59:59 |
      | 13              | 14             | 9999-12-31 23:59:59 |
      | 34              | 13             | 9999-12-31 23:59:59 |
      | 34              | 14             | 9999-12-31 23:59:59 |
      | 36              | 13             | 3018-04-24 07:38:42 |
      | 36              | 14             | 2018-12-31 23:59:59 |
      | 36              | 15             | 2018-12-31 23:59:59 |
      | 36              | 16             | 2018-12-31 23:59:59 |
      | 36              | 31             | 9999-12-31 23:59:59 |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self | expires_at          |
      | 10                | 10             | 1       | 9999-12-31 23:59:59 |
      | 10                | 11             | 0       | 9999-12-31 23:59:59 |
      | 10                | 13             | 0       | 9999-12-31 23:59:59 |
      | 10                | 14             | 0       | 9999-12-31 23:59:59 |
      | 10                | 21             | 0       | 9999-12-31 23:59:59 |
      | 11                | 11             | 1       | 9999-12-31 23:59:59 |
      | 11                | 13             | 0       | 9999-12-31 23:59:59 |
      | 11                | 14             | 0       | 9999-12-31 23:59:59 |
      | 13                | 13             | 1       | 9999-12-31 23:59:59 |
      | 13                | 14             | 0       | 9999-12-31 23:59:59 |
      | 14                | 14             | 1       | 9999-12-31 23:59:59 |
      | 15                | 15             | 1       | 9999-12-31 23:59:59 |
      | 16                | 16             | 1       | 9999-12-31 23:59:59 |
      | 21                | 21             | 1       | 9999-12-31 23:59:59 |
      | 31                | 31             | 1       | 9999-12-31 23:59:59 |
      | 33                | 33             | 1       | 9999-12-31 23:59:59 |
      | 34                | 13             | 0       | 9999-12-31 23:59:59 |
      | 34                | 14             | 0       | 9999-12-31 23:59:59 |
      | 34                | 34             | 1       | 9999-12-31 23:59:59 |
      | 35                | 35             | 1       | 9999-12-31 23:59:59 |
      | 36                | 13             | 0       | 3018-04-24 07:38:42 |
      | 36                | 14             | 0       | 3018-04-24 07:38:42 |
      | 36                | 31             | 0       | 9999-12-31 23:59:59 |
      | 36                | 36             | 1       | 9999-12-31 23:59:59 |
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Doesn't create a new row when seconds=0
    Given I am the user with id "21"
    When I send a PUT request to "/contests/70/groups/13/additional-times?seconds=0"
    Then the response code should be 200
    And the response should be "updated"
    And the table "groups_contest_items" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Doesn't update time columns of groups_groups/attempts for ended attempts
    Given I am the user with id "21"
    When I send a PUT request to "/contests/50/groups/14/additional-times?seconds=10000"
    Then the response code should be 200
    And the response should be "updated"
    And the table "groups_contest_items" should be:
      | group_id | item_id | additional_time |
      | 10       | 50      | 01:00:00        |
      | 11       | 50      | 00:01:00        |
      | 13       | 50      | 00:00:01        |
      | 13       | 60      | 00:00:30        |
      | 14       | 50      | 02:46:40        |
      | 21       | 50      | 00:01:00        |
      | 21       | 60      | 00:01:00        |
      | 21       | 70      | 00:01:00        |
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Doesn't update attempts.allows_submissions_until if both old and new values are in the past
    Given I am the user with id "21"
    When I send a PUT request to "/contests/70/groups/15/additional-times?seconds=-10"
    Then the response code should be 200
    And the response should be "updated"
    And the table "groups_contest_items" should be:
      | group_id | item_id | additional_time |
      | 10       | 50      | 01:00:00        |
      | 11       | 50      | 00:01:00        |
      | 13       | 50      | 00:00:01        |
      | 13       | 60      | 00:00:30        |
      | 15       | 70      | -00:00:10       |
      | 21       | 50      | 00:01:00        |
      | 21       | 60      | 00:01:00        |
      | 21       | 70      | 00:01:00        |
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id | expires_at          |
      | 10              | 11             | 9999-12-31 23:59:59 |
      | 10              | 21             | 9999-12-31 23:59:59 |
      | 11              | 13             | 9999-12-31 23:59:59 |
      | 13              | 14             | 9999-12-31 23:59:59 |
      | 34              | 13             | 9999-12-31 23:59:59 |
      | 34              | 14             | 9999-12-31 23:59:59 |
      | 36              | 13             | 9999-12-31 23:59:59 |
      | 36              | 14             | 2018-12-31 23:59:59 |
      | 36              | 15             | 2018-10-31 23:59:52 |
      | 36              | 16             | 2018-12-31 23:59:59 |
      | 36              | 31             | 9999-12-31 23:59:59 |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self | expires_at          |
      | 10                | 10             | 1       | 9999-12-31 23:59:59 |
      | 10                | 11             | 0       | 9999-12-31 23:59:59 |
      | 10                | 13             | 0       | 9999-12-31 23:59:59 |
      | 10                | 14             | 0       | 9999-12-31 23:59:59 |
      | 10                | 21             | 0       | 9999-12-31 23:59:59 |
      | 11                | 11             | 1       | 9999-12-31 23:59:59 |
      | 11                | 13             | 0       | 9999-12-31 23:59:59 |
      | 11                | 14             | 0       | 9999-12-31 23:59:59 |
      | 13                | 13             | 1       | 9999-12-31 23:59:59 |
      | 13                | 14             | 0       | 9999-12-31 23:59:59 |
      | 14                | 14             | 1       | 9999-12-31 23:59:59 |
      | 15                | 15             | 1       | 9999-12-31 23:59:59 |
      | 16                | 16             | 1       | 9999-12-31 23:59:59 |
      | 21                | 21             | 1       | 9999-12-31 23:59:59 |
      | 31                | 31             | 1       | 9999-12-31 23:59:59 |
      | 33                | 33             | 1       | 9999-12-31 23:59:59 |
      | 34                | 13             | 0       | 9999-12-31 23:59:59 |
      | 34                | 14             | 0       | 9999-12-31 23:59:59 |
      | 34                | 34             | 1       | 9999-12-31 23:59:59 |
      | 35                | 35             | 1       | 9999-12-31 23:59:59 |
      | 36                | 13             | 0       | 9999-12-31 23:59:59 |
      | 36                | 14             | 0       | 9999-12-31 23:59:59 |
      | 36                | 31             | 0       | 9999-12-31 23:59:59 |
      | 36                | 36             | 1       | 9999-12-31 23:59:59 |
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Activates an expired participation
    Given I am the user with id "21"
    When I send a PUT request to "/contests/70/groups/16/additional-times?seconds=3020399"
    Then the response code should be 200
    And the response should be "updated"
    And the table "groups_contest_items" should be:
      | group_id | item_id | additional_time |
      | 10       | 50      | 01:00:00        |
      | 11       | 50      | 00:01:00        |
      | 13       | 50      | 00:00:01        |
      | 13       | 60      | 00:00:30        |
      | 16       | 70      | 838:59:59       |
      | 21       | 50      | 00:01:00        |
      | 21       | 60      | 00:01:00        |
      | 21       | 70      | 00:01:00        |
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id | expires_at          |
      | 10              | 11             | 9999-12-31 23:59:59 |
      | 10              | 21             | 9999-12-31 23:59:59 |
      | 11              | 13             | 9999-12-31 23:59:59 |
      | 13              | 14             | 9999-12-31 23:59:59 |
      | 34              | 13             | 9999-12-31 23:59:59 |
      | 34              | 14             | 9999-12-31 23:59:59 |
      | 36              | 13             | 9999-12-31 23:59:59 |
      | 36              | 14             | 2018-12-31 23:59:59 |
      | 36              | 15             | 2018-12-31 23:59:59 |
      | 36              | 16             | 3019-12-05 23:00:01 |
      | 36              | 31             | 9999-12-31 23:59:59 |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self | expires_at          |
      | 10                | 10             | 1       | 9999-12-31 23:59:59 |
      | 10                | 11             | 0       | 9999-12-31 23:59:59 |
      | 10                | 13             | 0       | 9999-12-31 23:59:59 |
      | 10                | 14             | 0       | 9999-12-31 23:59:59 |
      | 10                | 21             | 0       | 9999-12-31 23:59:59 |
      | 11                | 11             | 1       | 9999-12-31 23:59:59 |
      | 11                | 13             | 0       | 9999-12-31 23:59:59 |
      | 11                | 14             | 0       | 9999-12-31 23:59:59 |
      | 13                | 13             | 1       | 9999-12-31 23:59:59 |
      | 13                | 14             | 0       | 9999-12-31 23:59:59 |
      | 14                | 14             | 1       | 9999-12-31 23:59:59 |
      | 15                | 15             | 1       | 9999-12-31 23:59:59 |
      | 16                | 16             | 1       | 9999-12-31 23:59:59 |
      | 21                | 21             | 1       | 9999-12-31 23:59:59 |
      | 31                | 31             | 1       | 9999-12-31 23:59:59 |
      | 33                | 33             | 1       | 9999-12-31 23:59:59 |
      | 34                | 13             | 0       | 9999-12-31 23:59:59 |
      | 34                | 14             | 0       | 9999-12-31 23:59:59 |
      | 34                | 34             | 1       | 9999-12-31 23:59:59 |
      | 35                | 35             | 1       | 9999-12-31 23:59:59 |
      | 36                | 13             | 0       | 9999-12-31 23:59:59 |
      | 36                | 14             | 0       | 9999-12-31 23:59:59 |
      | 36                | 16             | 0       | 3019-12-05 23:00:01 |
      | 36                | 31             | 0       | 9999-12-31 23:59:59 |
      | 36                | 36             | 1       | 9999-12-31 23:59:59 |
    And the table "attempts" should be:
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id | allows_submissions_until |
      | 1  | 13             | 3018-05-29 06:38:38 | 21         | 0                 | 50           | 2018-12-31 23:59:59      |
      | 1  | 14             | 3019-05-29 06:38:38 | 21         | 0                 | 50           | 9999-12-31 23:59:59      |
      | 1  | 15             | 2018-05-29 06:38:38 | 21         | 0                 | 70           | 2018-12-31 23:59:59      |
      | 1  | 16             | 3019-05-29 06:38:38 | 21         | 0                 | 70           | 3019-12-05 23:00:01      |
      | 1  | 31             | 3017-05-29 06:38:38 | 21         | 0                 | 70           | 9999-12-31 23:59:59      |
      | 2  | 14             | 3019-05-29 06:38:38 | 21         | 0                 | 50           | 9999-12-31 23:59:59      |
    And the table "results" should remain unchanged

  Scenario: Creates a new row for a user
    Given I am the user with id "21"
    When I send a PUT request to "/contests/70/groups/31/additional-times?seconds=-3020399"
    Then the response code should be 200
    And the response should be "updated"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should be:
      | group_id | item_id | additional_time |
      | 10       | 50      | 01:00:00        |
      | 11       | 50      | 00:01:00        |
      | 13       | 50      | 00:00:01        |
      | 13       | 60      | 00:00:30        |
      | 21       | 50      | 00:01:00        |
      | 21       | 60      | 00:01:00        |
      | 21       | 70      | 00:01:00        |
      | 31       | 70      | -838:59:59      |
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id | expires_at          |
      | 10              | 11             | 9999-12-31 23:59:59 |
      | 10              | 21             | 9999-12-31 23:59:59 |
      | 11              | 13             | 9999-12-31 23:59:59 |
      | 13              | 14             | 9999-12-31 23:59:59 |
      | 34              | 13             | 9999-12-31 23:59:59 |
      | 34              | 14             | 9999-12-31 23:59:59 |
      | 36              | 13             | 9999-12-31 23:59:59 |
      | 36              | 14             | 2018-12-31 23:59:59 |
      | 36              | 15             | 2018-12-31 23:59:59 |
      | 36              | 16             | 2018-12-31 23:59:59 |
      | 36              | 31             | 3017-04-24 07:38:42 |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self | expires_at          |
      | 10                | 10             | 1       | 9999-12-31 23:59:59 |
      | 10                | 11             | 0       | 9999-12-31 23:59:59 |
      | 10                | 13             | 0       | 9999-12-31 23:59:59 |
      | 10                | 14             | 0       | 9999-12-31 23:59:59 |
      | 10                | 21             | 0       | 9999-12-31 23:59:59 |
      | 11                | 11             | 1       | 9999-12-31 23:59:59 |
      | 11                | 13             | 0       | 9999-12-31 23:59:59 |
      | 11                | 14             | 0       | 9999-12-31 23:59:59 |
      | 13                | 13             | 1       | 9999-12-31 23:59:59 |
      | 13                | 14             | 0       | 9999-12-31 23:59:59 |
      | 14                | 14             | 1       | 9999-12-31 23:59:59 |
      | 15                | 15             | 1       | 9999-12-31 23:59:59 |
      | 16                | 16             | 1       | 9999-12-31 23:59:59 |
      | 21                | 21             | 1       | 9999-12-31 23:59:59 |
      | 31                | 31             | 1       | 9999-12-31 23:59:59 |
      | 33                | 33             | 1       | 9999-12-31 23:59:59 |
      | 34                | 13             | 0       | 9999-12-31 23:59:59 |
      | 34                | 14             | 0       | 9999-12-31 23:59:59 |
      | 34                | 34             | 1       | 9999-12-31 23:59:59 |
      | 35                | 35             | 1       | 9999-12-31 23:59:59 |
      | 36                | 13             | 0       | 9999-12-31 23:59:59 |
      | 36                | 14             | 0       | 9999-12-31 23:59:59 |
      | 36                | 31             | 0       | 3017-04-24 07:38:42 |
      | 36                | 36             | 1       | 9999-12-31 23:59:59 |
    And the table "attempts" should be:
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id | allows_submissions_until |
      | 1  | 13             | 3018-05-29 06:38:38 | 21         | 0                 | 50           | 2018-12-31 23:59:59      |
      | 1  | 14             | 3019-05-29 06:38:38 | 21         | 0                 | 50           | 9999-12-31 23:59:59      |
      | 1  | 15             | 2018-05-29 06:38:38 | 21         | 0                 | 70           | 2018-12-31 23:59:59      |
      | 1  | 16             | 3019-05-29 06:38:38 | 21         | 0                 | 70           | 2018-12-31 23:59:59      |
      | 1  | 31             | 3017-05-29 06:38:38 | 21         | 0                 | 70           | 3017-04-24 07:38:42      |
      | 2  | 14             | 3019-05-29 06:38:38 | 21         | 0                 | 50           | 9999-12-31 23:59:59      |
    And the table "results" should remain unchanged
