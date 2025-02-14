Feature: Start results for an item path
  Background:
    Given the database has the following table "groups":
      | id  | type  | root_activity_id |
      | 90  | Class | 10               |
      | 91  | Other | 50               |
      | 102 | Team  | 60               |
    And the database has the following users:
      | group_id | login |
      | 101      | john  |
      | 111      | jane  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 90              | 111            |
      | 90              | 102            |
      | 91              | 111            |
      | 102             | 101            |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id | url                                                                     | type    | allows_multiple_attempts | default_language_tag |
      | 10 | null                                                                    | Chapter | 1                        | fr                   |
      | 50 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task    | 1                        | fr                   |
      | 60 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task    | 1                        | fr                   |
      | 70 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task    | 1                        | fr                   |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 10             | 60            | 1           |
      | 60             | 70            | 1           |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 10               | 60            |
      | 10               | 70            |
      | 60               | 70            |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       |
      | 101      | 50      | content                  |
      | 102      | 10      | content                  |
      | 102      | 60      | content                  |
      | 102      | 70      | content                  |
      | 111      | 10      | content_with_descendants |
      | 111      | 50      | content_with_descendants |
    And the database has the following table "attempts":
      | id | participant_id | root_item_id | created_at          | ended_at            | allows_submissions_until |
      | 0  | 101            | null         | 2019-05-30 11:00:00 | null                | 9999-12-31 23:59:59      |
      | 0  | 102            | null         | 2019-05-30 11:00:00 | null                | 9999-12-31 23:59:59      |
      | 0  | 111            | null         | 2019-05-30 11:00:00 | null                | 9999-12-31 23:59:59      |
      | 1  | 102            | 10           | 2019-05-30 11:00:00 | null                | 9999-12-31 23:59:59      |
      | 2  | 102            | 10           | 2019-05-30 11:00:00 | 2019-05-30 11:00:00 | 9999-12-31 23:59:59      |
      | 3  | 102            | 10           | 2019-05-30 11:00:00 | null                | 2019-05-30 11:00:00      |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | started_at          | latest_activity_at  |
      | 1          | 102            | 10      | 2019-05-30 11:00:00 | 2019-05-30 11:00:00 |
      | 2          | 102            | 10      | 2019-05-30 11:00:00 | 2019-05-30 11:00:00 |
      | 2          | 102            | 60      | 2019-05-30 11:00:00 | 2019-05-30 11:00:00 |
      | 3          | 102            | 10      | 2019-05-30 11:00:00 | 2019-05-30 11:00:00 |
      | 3          | 102            | 60      | 2019-05-30 11:00:00 | 2019-05-30 11:00:00 |

  Scenario Outline: User is able to start a result for his self group
    Given I am the user with id "111"
    When I send a POST request to "/items/<item_id>/start-result-path"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "message": "updated",
        "success": true,
        "data": {
          "attempt_id": "0"
        }
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id   | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | latest_submission_at | score_obtained_at | validated_at | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 0          | 111            | <item_id> | 0              | 0           | 1                                                         | null                 | null              | null         | 1                                                 |
      | 1          | 102            | 10        | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 2          | 102            | 10        | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 2          | 102            | 60        | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 3          | 102            | 10        | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 3          | 102            | 60        | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
    And the table "results_propagate" should be empty
  Examples:
    | item_id |
    | 50      |
    | 10      |

  Scenario: User is able to create results as a team
    Given I am the user with id "101"
    When I send a POST request to "/items/60/70/start-result-path?as_team_id=102"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "message": "updated",
        "success": true,
        "data": {
          "attempt_id": "0"
        }
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | latest_submission_at | score_obtained_at | validated_at | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 0          | 102            | 10      | 0              | 0           | 1                                                         | null                 | null              | null         | null                                              |
      | 0          | 102            | 60      | 0              | 0           | 1                                                         | null                 | null              | null         | 1                                                 |
      | 0          | 102            | 70      | 0              | 0           | 1                                                         | null                 | null              | null         | 1                                                 |
      | 1          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 2          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 2          | 102            | 60      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 3          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 3          | 102            | 60      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
    And the table "results_propagate" should be empty
    When I send a POST request to "/items/60/70/start-result-path?as_team_id=102"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "message": "updated",
        "success": true,
        "data": {
          "attempt_id": "0"
        }
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | latest_submission_at | score_obtained_at | validated_at | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 0          | 102            | 10      | 0              | 0           | 1                                                         | null                 | null              | null         | null                                              |
      | 0          | 102            | 60      | 0              | 0           | 1                                                         | null                 | null              | null         | 1                                                 |
      | 0          | 102            | 70      | 0              | 0           | 1                                                         | null                 | null              | null         | 1                                                 |
      | 1          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 2          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 2          | 102            | 60      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 3          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 3          | 102            | 60      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
    And the table "results_propagate" should be empty

  Scenario: Keeps the previous started_at value
    Given I am the user with id "101"
    And the database table "results" also has the following rows:
      | attempt_id | participant_id | item_id | started_at          | latest_activity_at  |
      | 1          | 102            | 60      | 2019-05-30 11:00:00 | 2019-05-30 11:00:00 |
    When I send a POST request to "/items/10/60/start-result-path?as_team_id=102"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "message": "updated",
        "success": true,
        "data": {
          "attempt_id": "3"
        }
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | latest_submission_at | score_obtained_at | validated_at | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 1          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 1          | 102            | 60      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 2          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 2          | 102            | 60      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 3          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 3          | 102            | 60      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
    And the table "results_propagate" should be empty

  Scenario: Can create new results for all the path
    Given I am the user with id "101"
    When I send a POST request to "/items/60/70/start-result-path?as_team_id=102"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "message": "updated",
        "success": true,
        "data": {
          "attempt_id": "0"
        }
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | latest_submission_at | score_obtained_at | validated_at | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 0          | 102            | 10      | 0              | 0           | 1                                                         | null                 | null              | null         | null                                              |
      | 0          | 102            | 60      | 0              | 0           | 1                                                         | null                 | null              | null         | 1                                                 |
      | 0          | 102            | 70      | 0              | 0           | 1                                                         | null                 | null              | null         | 1                                                 |
      | 1          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 2          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 2          | 102            | 60      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 3          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 3          | 102            | 60      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
    And the table "results_propagate" should be empty
