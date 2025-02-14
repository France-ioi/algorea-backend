Feature: Generate a task token with a refreshed attempt for an item
  Background:
    Given the database has the following table "groups":
      | id  | type |
      | 102 | Team |
    And the database has the following users:
      | group_id | login |
      | 101      | john  |
      | 111      | jane  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 102             | 101            |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id | url                                                                     | type    | allows_multiple_attempts | hints_allowed | text_id | supported_lang_prog | default_language_tag |
      | 10 | null                                                                    | Chapter | 0                        | 0             | null    | null                | fr                   |
      | 50 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task    | 0                        | 1             | task1   | null                | fr                   |
      | 60 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task    | 1                        | 0             | null    | c,python            | fr                   |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 10               | 60            |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       |
      | 101      | 50      | content                  |
      | 101      | 60      | solution                 |
      | 102      | 60      | solution                 |
      | 111      | 50      | content_with_descendants |
    And the server time is frozen

  Scenario: User is able to fetch a task token
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
      | 0  | 102            |
      | 1  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | latest_activity_at  | started_at          | score_computed | score_obtained_at | validated_at        | hints_requested | hints_cached |
      | 0          | 101            | 50      | 2017-05-29 06:38:38 | null                | 0              | null              | 2019-05-30 11:00:00 | null            | 0            |
      | 0          | 101            | 51      | 2019-04-29 06:38:38 | null                | 0              | null              | null                | null            | 0            |
      | 0          | 102            | 50      | 2019-05-29 06:38:38 | null                | 0              | null              | null                | null            | 0            |
      | 1          | 101            | 50      | 2018-05-29 06:38:38 | 2018-05-29 05:00:00 | 0              | null              | null                | [1,2,3,4]       | 4            |
    When I send a POST request to "/items/50/attempts/1/generate-task-token"
    Then the response code should be 200
    And the response body decoded as "GenerateTaskTokenResponse" should be, in JSON:
      """
      {
        "data": {
          "task_token": {
            "date": "{{currentTimeInFormat("02-01-2006")}}",
            "bAccessSolutions": false,
            "bHintsAllowed": true,
            "bIsAdmin": false,
            "bReadAnswers": true,
            "bSubmissionPossible": true,
            "idAttempt": "101/1",
            "idUser": "101",
            "idItemLocal": "50",
            "idItem": "task1",
            "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
            "nbHintsGiven": "4",
            "sHintsRequested": "[1,2,3,4]",
            "randomSeed": "12601247502642542026",
            "platformName": "{{app().Config.GetString("token.platformName")}}",
            "sLogin": "john"
          }
        },
        "message": "updated",
        "success": true
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged, regardless of the row with attempt_id "1"
    And the table "results" at attempt_id "1" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, latest_submission_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, score_obtained_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, validated_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 1          | 101            | 50      | 0              | 0           | 1                                                         | null                                                        | null                                                     | null                                                | 0                                                 |
    And the table "results_propagate" should be empty

  Scenario: User is able to fetch a task token as a team
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
      | 0  | 102            |
      | 1  | 102            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | latest_activity_at  | started_at          | score_computed | score_obtained_at | validated_at | hints_requested | hints_cached |
      | 0          | 101            | 60      | 2019-05-29 06:38:38 | null                | 0              | null              | null         | null            | 0            |
      | 0          | 102            | 60      | 2017-05-29 06:38:38 | null                | 0              | null              | null         | null            | 0            |
      | 0          | 102            | 61      | 2019-04-29 06:38:38 | null                | 0              | null              | null         | null            | 0            |
      | 1          | 102            | 60      | 2018-05-29 06:38:38 | 2018-05-29 06:38:38 | 0              | null              | null         | [1,2,3,4]       | 4            |
    When I send a POST request to "/items/60/attempts/1/generate-task-token?as_team_id=102"
    Then the response code should be 200
    And the response body decoded as "GenerateTaskTokenResponse" should be, in JSON:
      """
      {
        "data": {
          "task_token": {
            "date": "{{currentTimeInFormat("02-01-2006")}}",
            "bAccessSolutions": true,
            "bHintsAllowed": false,
            "bIsAdmin": false,
            "bReadAnswers": true,
            "bSubmissionPossible": true,
            "idAttempt": "102/1",
            "idUser": "101",
            "idItemLocal": "60",
            "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
            "nbHintsGiven": "4",
            "sHintsRequested": "[1,2,3,4]",
            "sSupportedLangProg": "c,python",
            "randomSeed": "17292903417420170135",
            "platformName": "{{app().Config.GetString("token.platformName")}}",
            "sLogin": "john"
          }
        },
        "message": "updated",
        "success": true
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged, regardless of the row with attempt_id "1"
    And the table "results" at attempt_id "1" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, latest_submission_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, score_obtained_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, validated_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 1          | 102            | 60      | 0              | 0           | 1                                                         | null                                                        | null                                                     | null                                                | 0                                                 |
    And the table "results_propagate" should be empty

  Scenario: bAccessSolutions is true when the item is validated
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
      | 0  | 102            |
      | 1  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | latest_activity_at  | started_at          | score_computed | score_obtained_at | validated_at        | hints_requested | hints_cached |
      | 0          | 101            | 50      | 2017-05-29 06:38:38 | null                | 0              | null              | null                | null            | 0            |
      | 0          | 101            | 51      | 2019-04-29 06:38:38 | null                | 0              | null              | null                | null            | 0            |
      | 0          | 102            | 50      | 2019-05-29 06:38:38 | null                | 0              | null              | null                | null            | 0            |
      | 1          | 101            | 50      | 2018-05-29 06:38:38 | 2018-05-29 05:00:00 | 0              | null              | 2019-05-30 11:00:00 | [1,2,3,4]       | 4            |
    When I send a POST request to "/items/50/attempts/1/generate-task-token"
    Then the response code should be 200
    And the response body decoded as "GenerateTaskTokenResponse" should be, in JSON:
      """
      {
        "data": {
          "task_token": {
            "date": "{{currentTimeInFormat("02-01-2006")}}",
            "bAccessSolutions": true,
            "bHintsAllowed": true,
            "bIsAdmin": false,
            "bReadAnswers": true,
            "bSubmissionPossible": true,
            "idAttempt": "101/1",
            "idUser": "101",
            "idItemLocal": "50",
            "idItem": "task1",
            "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
            "nbHintsGiven": "4",
            "sHintsRequested": "[1,2,3,4]",
            "randomSeed": "12601247502642542026",
            "platformName": "{{app().Config.GetString("token.platformName")}}",
            "sLogin": "john"
          }
        },
        "message": "updated",
        "success": true
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged, regardless of the row with attempt_id "1"
    And the table "results" at attempt_id "1" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, latest_submission_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, score_obtained_at, NOW())) < 3 | validated_at        | ABS(TIMESTAMPDIFF(SECOND, started_at, NOW())) < 3 |
      | 1          | 101            | 50      | 0              | 0           | 1                                                         | null                                                        | null                                                     | 2019-05-30 11:00:00 | 0                                                 |
    And the table "results_propagate" should be empty
