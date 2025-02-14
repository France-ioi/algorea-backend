Feature: Start a result for an item
  Background:
    Given the time is frozen
    And the database has the following table "groups":
      | id  | type  | root_activity_id | root_skill_id |
      | 90  | Class | 10               | null          |
      | 91  | Other | 50               | null          |
      | 101 | User  | null             | null          |
      | 102 | Team  | 60               | null          |
      | 111 | User  | null             | 80            |
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
      | 80 | null                                                                    | Skill   | 0                        | fr                   |
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
      | 111      | 80      | content                  |
    And the database has the following table "attempts":
      | id | participant_id | created_at          |
      | 0  | 101            | 2019-05-30 11:00:00 |
      | 0  | 102            | 2019-05-30 11:00:00 |
      | 0  | 111            | 2019-05-30 11:00:00 |
      | 1  | 102            | 2019-05-30 11:00:00 |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | started_at          | latest_activity_at  |
      | 1          | 102            | 10      | 2019-05-30 11:00:00 | 2019-05-30 11:00:00 |

  Scenario Outline: User is able to start a result for his self group
    Given I am the user with id "111"
    When I send a POST request to "/items/<item_id>/start-result?attempt_id=0"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "message": "updated",
        "success": true,
        "data": {
          "allows_submissions_until": "9999-12-31T23:59:59Z",
          "created_at": "2019-05-30T11:00:00Z",
          "ended_at": null,
          "help_requested": false,
          "id": "0",
          "latest_activity_at": "{{timeDBToRFC3339(currentTimeDB())}}",
          "score_computed": 0,
          "started_at": "{{timeDBToRFC3339(currentTimeDB())}}",
          "user_creator": null,
          "validated": false
        }
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id   | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | latest_submission_at | score_obtained_at | validated_at | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 0          | 111            | <item_id> | 0              | 0           | 1                                                         | null                 | null              | null         | 1                                                 |
      | 1          | 102            | 10        | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
    And the table "results_propagate" should be empty
  Examples:
    | item_id |
    | 50      |
    | 10      |
    | 80      |

  Scenario: User is able to start a result as a team
    Given I am the user with id "101"
    When I send a POST request to "/items/60/start-result?as_team_id=102&attempt_id=0"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "message": "updated",
        "success": true,
        "data": {
          "allows_submissions_until": "9999-12-31T23:59:59Z",
          "created_at": "2019-05-30T11:00:00Z",
          "ended_at": null,
          "help_requested": false,
          "id": "0",
          "latest_activity_at": "{{timeDBToRFC3339(currentTimeDB())}}",
          "score_computed": 0,
          "started_at": "{{timeDBToRFC3339(currentTimeDB())}}",
          "user_creator": null,
          "validated": false
        }
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | latest_submission_at | score_obtained_at | validated_at | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 0          | 102            | 10      | 0              | 0           | 1                                                         | null                 | null              | null         | null                                              |
      | 0          | 102            | 60      | 0              | 0           | 1                                                         | null                 | null              | null         | 1                                                 |
      | 1          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
    And the table "results_propagate" should be empty
    When I send a POST request to "/items/60/70/start-result?as_team_id=102&attempt_id=0"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "message": "updated",
        "success": true,
        "data": {
          "allows_submissions_until": "9999-12-31T23:59:59Z",
          "created_at": "2019-05-30T11:00:00Z",
          "ended_at": null,
          "help_requested": false,
          "id": "0",
          "latest_activity_at": "{{timeDBToRFC3339(currentTimeDB())}}",
          "score_computed": 0,
          "started_at": "{{timeDBToRFC3339(currentTimeDB())}}",
          "user_creator": null,
          "validated": false
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
    And the table "results_propagate" should be empty

  Scenario: Keeps the previous started_at value
    Given I am the user with id "101"
    And the database table "results" also has the following rows:
      | attempt_id | participant_id | item_id | started_at          | latest_activity_at  |
      | 1          | 102            | 60      | 2019-05-30 11:00:00 | 2019-05-30 11:00:00 |
    When I send a POST request to "/items/10/60/start-result?as_team_id=102&attempt_id=1"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "message": "updated",
        "success": true,
        "data": {
          "allows_submissions_until": "9999-12-31T23:59:59Z",
          "created_at": "2019-05-30T11:00:00Z",
          "ended_at": null,
          "help_requested": false,
          "id": "1",
          "latest_activity_at": "2019-05-30T11:00:00Z",
          "score_computed": 0,
          "started_at": "2019-05-30T11:00:00Z",
          "user_creator": null,
          "validated": false
        }
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | latest_submission_at | score_obtained_at | validated_at | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 1          | 102            | 10      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
      | 1          | 102            | 60      | 0              | 0           | 0                                                         | null                 | null              | null         | 0                                                 |
    And the table "results_propagate" should be empty

  Scenario: Sets started_at of an existing result
    Given I am the user with id "101"
    And the database table "results" also has the following rows:
      | attempt_id | participant_id | item_id | started_at | latest_activity_at  |
      | 1          | 102            | 60      | null       | 2019-05-30 11:00:00 |
    When I send a POST request to "/items/10/60/start-result?as_team_id=102&attempt_id=1"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "message": "updated",
        "success": true,
        "data": {
          "allows_submissions_until": "9999-12-31T23:59:59Z",
          "created_at": "2019-05-30T11:00:00Z",
          "ended_at": null,
          "help_requested": false,
          "id": "1",
          "latest_activity_at": "{{timeDBToRFC3339(currentTimeDB())}}",
          "score_computed": 0,
          "started_at": "{{timeDBToRFC3339(currentTimeDB())}}",
          "user_creator": null,
          "validated": false
        }
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | latest_submission_at | score_obtained_at | validated_at | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 1          | 102            | 10      | 0              | 0           | 1                                                         | null                 | null              | null         | 0                                                 |
      | 1          | 102            | 60      | 0              | 0           | 1                                                         | null                 | null              | null         | 1                                                 |
    And the table "results_propagate" should be empty
