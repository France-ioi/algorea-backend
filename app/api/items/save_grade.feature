Feature: Save grading result
  Background:
    Given the database has the following user:
      | login | group_id | default_language |
      | john  | 101      | en               |
    And the database has the following table "groups":
      | id  | name | type |
      | 201 | team | Team |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 201             | 101            |
    And the groups ancestors are computed
    And the database has the following table "platforms":
      | id | regexp                                             | priority | public_key                |
      | 10 | http://taskplatform.mblockelet.info/task.html\?.*  | 2        | {{taskPlatformPublicKey}} |
      | 20 | http://taskplatform1.mblockelet.info/task.html\?.* | 1        | null                      |
    And the database has the following table "items":
      | id | platform_id | url                                                                     | validation_type | default_language_tag |
      | 50 | 10          | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | All             | fr                   |
      | 60 | 10          | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183937 | All             | fr                   |
      | 10 | null        | null                                                                    | AllButOne       | fr                   |
      | 70 | 20          | http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839  | All             | fr                   |
    And the database has the following table "items_strings":
      | item_id | language_tag | title        |
      | 50      | fr           | Chapitre A   |
      | 50      | en           | Chapter A    |
    And the database has the following table "item_dependencies":
      | item_id | dependent_item_id | score |
      | 60      | 50                | 98    |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 10             | 50            | 0           |
      | 10             | 60            | 1           |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 10               | 50            |
      | 10               | 60            |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 101      | 50      | content            |
      | 101      | 60      | content            |
      | 101      | 70      | content            |
      | 201      | 50      | content            |
      | 201      | 60      | content            |
      | 201      | 70      | content            |
    And the server time is frozen

  Scenario: User is able to save the grading result with a high score and attempt_id
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
      | 1  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | latest_activity_at  | hints_requested        |
      | 0          | 101            | 10      | 2019-05-30 11:00:00 | null                   |
      | 0          | 101            | 50      | 2019-05-30 11:00:00 | [0,  1, "hint" , null] |
      | 1          | 101            | 60      | 2019-05-29 11:00:00 | [0,  1, "hint" , null] |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 123 | 101       | 101            | 0          | 50      | 2017-05-29 06:38:38 |
      | 124 | 101       | 101            | 0          | 60      | 2017-05-29 06:38:38 |
    And "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "score": "100",
        "idUserAnswer": "123"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "{{scoreToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": true,
          "unlocked_items": []
        },
        "message": "created",
        "success": true
      }
      """
    And the table "answers" should remain unchanged
    And the table "gradings" should be:
      | answer_id | score | ABS(TIMESTAMPDIFF(SECOND, graded_at, NOW())) < 3 |
      | 123       | 100   | 1                                                |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | validated | latest_activity_at  | latest_submission_at | score_obtained_at   | validated_at        |
      | 0          | 101            | 10      | 50             | 1           | 1         | 2019-05-30 11:00:00 | null                 | null                | 2017-05-29 06:38:38 |
      | 0          | 101            | 50      | 100            | 1           | 1         | 2019-05-30 11:00:00 | null                 | 2017-05-29 06:38:38 | 2017-05-29 06:38:38 |
      | 1          | 101            | 60      | 0              | 0           | 0         | 2019-05-29 11:00:00 | null                 | null                | null                |
    And the table "results_propagate" should be empty
    And the table "results_propagate_sync" should be empty

  Scenario: User is able to save the grading result for a team (participant_id is the first integer in idAttempt in the score token)
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 201            |
      | 1  | 201            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | latest_activity_at  | hints_requested        |
      | 0          | 201            | 10      | 2019-05-30 11:00:00 | null                   |
      | 0          | 201            | 50      | 2019-05-30 11:00:00 | [0,  1, "hint" , null] |
      | 1          | 201            | 60      | 2019-05-29 11:00:00 | [0,  1, "hint" , null] |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 123 | 101       | 201            | 0          | 50      | 2017-05-29 06:38:38 |
      | 124 | 101       | 201            | 0          | 60      | 2017-05-29 06:38:38 |
    And "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "201/0",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "score": "100",
        "idUserAnswer": "123"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "{{scoreToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": true,
          "unlocked_items": []
        },
        "message": "created",
        "success": true
      }
      """
    And the table "answers" should remain unchanged
    And the table "gradings" should be:
      | answer_id | score | ABS(TIMESTAMPDIFF(SECOND, graded_at, NOW())) < 3 |
      | 123       | 100   | 1                                                |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | validated | latest_activity_at  | latest_submission_at | score_obtained_at   | validated_at        |
      | 0          | 201            | 10      | 50             | 1           | 1         | 2019-05-30 11:00:00 | null                 | null                | 2017-05-29 06:38:38 |
      | 0          | 201            | 50      | 100            | 1           | 1         | 2019-05-30 11:00:00 | null                 | 2017-05-29 06:38:38 | 2017-05-29 06:38:38 |
      | 1          | 201            | 60      | 0              | 0           | 0         | 2019-05-29 11:00:00 | null                 | null                | null                |
    And the table "results_propagate" should be empty
    And the table "results_propagate_sync" should be empty

  Scenario Outline: User is able to save the grading result with a low score and idAttempt
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
      | 1  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | hints_requested        | latest_activity_at  | score_edit_rule   | score_edit_value   |
      | 0          | 101            | 10      | null                   | 2019-05-30 11:00:00 | null              | null               |
      | 0          | 101            | 50      | [0,  1, "hint" , null] | 2019-05-30 11:00:00 | <score_edit_rule> | <score_edit_value> |
      | 1          | 101            | 60      | [0,  1, "hint" , null] | 2019-05-29 11:00:00 | null              | null               |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 123 | 101       | 101            | 0          | 50      | 2017-05-29 06:38:38 |
      | 124 | 101       | 101            | 1          | 60      | 2017-05-29 06:38:38 |
    And "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "score": "<score>",
        "idUserAnswer": "123"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "{{scoreToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": false,
          "unlocked_items": []
        },
        "message": "created",
        "success": true
      }
      """
    And the table "answers" should remain unchanged
    And the table "gradings" should be:
      | answer_id | score   | ABS(TIMESTAMPDIFF(SECOND, graded_at, NOW())) < 3 |
      | 123       | <score> | 1                                                |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed   | tasks_tried | validated | latest_activity_at  | latest_submission_at | score_obtained_at   | validated_at |
      | 0          | 101            | 10      | <parent_score>   | 1           | 0         | 2019-05-30 11:00:00 | null                 | null                | null         |
      | 0          | 101            | 50      | <score_computed> | 1           | 0         | 2019-05-30 11:00:00 | null                 | 2017-05-29 06:38:38 | null         |
      | 1          | 101            | 60      | 0                | 0           | 0         | 2019-05-29 11:00:00 | null                 | null                | null         |
    And the table "results_propagate" should be empty
    And the table "results_propagate_sync" should be empty
  Examples:
    | score | score_edit_rule | score_edit_value | score_computed | parent_score |
    | 99    | null            | null             | 99             | 49.5         |
    | 89    | set             | 10               | 10             | 5            |
    | 89    | set             | -10              | 0              | 0            |
    | 79    | diff            | -10              | 69             | 34.5         |
    | 79    | diff            | -80              | 0              | 0            |
    | 79    | diff            | 80               | 100            | 50           |

  Scenario: User is able to save the grading result with a low score, but still obtaining a key (with idAttempt)
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
      | 1  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | score_obtained_at   | latest_activity_at  |
      | 0          | 101            | 50      | 2017-04-29 06:38:38 | 2019-05-30 11:00:00 |
      | 1          | 101            | 60      | 2017-05-29 06:38:38 | 2019-05-29 11:00:00 |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 123 | 101       | 101            | 0          | 50      | 2017-05-29 06:38:38 |
      | 124 | 101       | 101            | 1          | 60      | 2017-05-29 06:38:38 |
    And "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "60",
        "idAttempt": "101/1",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183937",
        "score": "99",
        "idUserAnswer": "124"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "{{scoreToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": false,
          "unlocked_items": [
            {
              "item_id": "50",
              "language_tag": "en",
              "title": "Chapter A",
              "type": "Chapter"
            }
          ]
        },
        "message": "created",
        "success": true
      }
      """
    And the table "answers" should remain unchanged
    And the table "gradings" should be:
      | answer_id | score | ABS(TIMESTAMPDIFF(SECOND, graded_at, NOW())) < 3 |
      | 124       | 99    | 1                                                |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | participant_id | attempt_id | item_id | score_computed | tasks_tried | validated | latest_activity_at  | latest_submission_at | score_obtained_at   | validated_at |
      | 101            | 0          | 50      | 0              | 0           | 0         | 2019-05-30 11:00:00 | null                 | 2017-04-29 06:38:38 | null         |
      | 101            | 1          | 60      | 99             | 1           | 0         | 2019-05-29 11:00:00 | null                 | 2017-05-29 06:38:38 | null         |
    And the table "results_propagate" should be empty
    And the table "results_propagate_sync" should be empty

  Scenario Outline: Should keep previous score if it is greater
    Given I am the user with id "101"
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 123 | 101       | 101            | 0          | 10      | 2018-05-29 06:38:38 |
      | 124 | 101       | 101            | 0          | 50      | 2018-05-29 06:38:38 |
      | 125 | 101       | 101            | 0          | 10      | 2018-05-29 06:38:38 |
    And the database has the following table "gradings":
      | answer_id | score | graded_at           |
      | 123       | 5     | 2018-05-29 06:38:38 |
      | 125       | 20    | 2018-05-29 06:38:38 |
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
    And the database has the following table "results":
      | participant_id | attempt_id | item_id | score_computed | score_obtained_at   | score_edit_rule   | score_edit_value   |
      | 101            | 0          | 10      | 20             | 2018-05-29 06:38:38 | null              | null               |
      | 101            | 0          | 50      | 20             | 2018-05-29 06:38:38 | <score_edit_rule> | <score_edit_value> |
      | 101            | 0          | 60      | 20             | 2018-05-29 06:38:38 | null              | null               |
    And "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "60",
        "idAttempt": "101/0",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183937",
        "score": "<score>",
        "idUserAnswer": "124"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "{{scoreToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": false,
          "unlocked_items": []
        },
        "message": "created",
        "success": true
      }
      """
    And the table "answers" should remain unchanged
    And the table "gradings" should be:
      | answer_id | score   | ABS(TIMESTAMPDIFF(SECOND, graded_at, NOW())) < 3 |
      | 123       | 5       | 0                                                |
      | 124       | <score> | 1                                                |
      | 125       | 20      | 0                                                |
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged
    Examples:
      | score | score_edit_rule | score_edit_value |
      | 19    | null            | null             |
      | 19    | set             | 10               |
      | 19    | set             | -10              |
      | 20    | diff            | -1               |
      | 15    | diff            | -80              |

  Scenario: Should keep previous validated_at if it is earlier
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | validated_at        |
      | 0          | 101            | 10      | 2016-05-29 06:38:37 |
      | 0          | 101            | 50      | 2016-05-29 06:38:37 |
      | 0          | 101            | 60      | 2015-05-29 06:38:37 |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 123 | 101       | 101            | 0          | 50      | 2017-05-29 06:38:38 |
      | 124 | 101       | 101            | 0          | 60      | 2017-05-29 06:38:38 |
    And "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "60",
        "idAttempt": "101/0",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183937",
        "score": "100",
        "idUserAnswer": "124"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "{{scoreToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": true,
          "unlocked_items": [
            {
              "item_id": "50",
              "language_tag": "en",
              "title": "Chapter A",
              "type": "Chapter"
            }
          ]
        },
        "message": "created",
        "success": true
      }
      """
    And the table "answers" should remain unchanged
    And the table "gradings" should be:
      | answer_id | score | ABS(TIMESTAMPDIFF(SECOND, graded_at, NOW())) < 3 |
      | 124       | 100   | 1                                                |
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Should set bAccessSolutions=1 if the task has been validated
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | validated_at        |
      | 0          | 101            | 50      | 2018-05-29 06:38:38 |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 123 | 101       | 101            | 100        | 50      | 2017-05-29 06:38:38 |
    And "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "score": "100",
        "idUserAnswer": "123"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "{{scoreToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": true,
          "unlocked_items": []
        },
        "message": "created",
        "success": true
      }
      """

  Scenario: Should set bAccessSolutions=1 if the previous task task token had bAccessSolutions=1
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | validated_at        |
      | 0          | 101            | 50      | 2018-05-29 06:38:38 |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 123 | 101       | 101            | 100        | 50      | 2017-05-29 06:38:38 |
    And "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "score": "10",
        "idUserAnswer": "123"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "{{scoreToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": false,
          "unlocked_items": []
        },
        "message": "created",
        "success": true
      }
      """

  Scenario: Platform doesn't support tokens
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 1  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | validated_at        |
      | 1          | 101            | 70      | 2018-05-29 06:38:38 |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 125 | 101       | 101            | 100        | 70      | 2017-05-29 06:38:38 |
    And "answerToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "70",
        "idAttempt": "101/1",
        "itemURL": "http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839",
        "idUserAnswer": "125",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score": 100.0,
        "answer_token": "{{answerToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": true,
          "unlocked_items": []
        },
        "message": "created",
        "success": true
      }
      """
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed | tasks_tried | validated |
      | 1          | 101            | 70      | 100            | 1           | 1         |

  Scenario: Platform doesn't support tokens for team (participant_id is the first integer in idAttempt in the answer token)
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 1  | 201            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | validated_at        |
      | 1          | 201            | 70      | 2018-05-29 06:38:38 |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 125 | 101       | 201            | 100        | 70      | 2017-05-29 06:38:38 |
    And "answerToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "70",
        "idAttempt": "201/1",
        "itemURL": "http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839",
        "idUserAnswer": "125",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score": 100.0,
        "answer_token": "{{answerToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": true,
          "unlocked_items": []
        },
        "message": "created",
        "success": true
      }
      """
  And the table "results" should be:
    | attempt_id | participant_id | item_id | score_computed | tasks_tried | validated |
    | 1          | 201            | 70      | 100            | 1           | 1         |

  Scenario: Should ignore score_token when provided if the platform doesn't have a key. Make sure the right score is used.
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 1  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | validated_at        |
      | 1          | 101            | 70      | 2018-05-29 06:38:38 |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 125 | 101       | 101            | 100        | 70      | 2017-05-29 06:38:38 |
    And "answerToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "70",
        "idAttempt": "101/1",
        "itemURL": "http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839",
        "idUserAnswer": "125",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "70",
        "idAttempt": "101/1",
        "itemURL": "http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839",
        "score": "99",
        "idUserAnswer": "125"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "{{scoreToken}}",
        "score": 100.0,
        "answer_token": "{{answerToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": true,
          "unlocked_items": []
        },
        "message": "created",
        "success": true
      }
      """

  Scenario: Unlocks multiple items recursively
    Given I am the user with id "101"
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
      | 1  | 101            |
    And the database table "items" also has the following row:
      | id  | type    | validation_type | default_language_tag |
      | 80  | Chapter | All             | fr                   |
      | 90  | Chapter | All             | fr                   |
      | 100 | Chapter | All             | de                   |
    And the database table "item_dependencies" also has the following row:
      | item_id | dependent_item_id | score |
      | 60      | 80                | 0     |
      | 80      | 100               | 0     |
    And the database table "items_items" also has the following row:
      | parent_item_id | child_item_id | child_order |
      | 80             | 90            | 0           |
    And the database table "items_ancestors" also has the following row:
      | ancestor_item_id | child_item_id |
      | 80               | 90            |
    And the database table "items_strings" also has the following rows:
      | item_id | language_tag | title      |
      | 80      | fr           | Chapitre B |
      | 100     | de           | Kapitel C  |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | score_obtained_at   | latest_activity_at  | score_computed |
      | 0          | 101            | 50      | 2017-04-29 06:38:38 | 2019-05-30 11:00:00 | 0              |
      | 1          | 101            | 60      | 2017-05-29 06:38:38 | 2019-05-29 11:00:00 | 0              |
      | 1          | 101            | 80      | 2017-05-29 06:38:38 | 2019-05-29 11:00:00 | 0              |
      | 1          | 101            | 90      | 2017-05-29 06:38:38 | 2019-05-29 11:00:00 | 20             |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 123 | 101       | 101            | 0          | 50      | 2017-05-29 06:38:38 |
      | 124 | 101       | 101            | 0          | 60      | 2017-05-29 06:38:38 |
    And "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "60",
        "idAttempt": "101/1",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183937",
        "score": "99",
        "idUserAnswer": "124"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "{{scoreToken}}"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "data": {
          "validated": false,
          "unlocked_items": [
            {
              "item_id": "50",
              "language_tag": "en",
              "title": "Chapter A",
              "type": "Chapter"
            },
            {
              "item_id": "80",
              "language_tag": "fr",
              "title": "Chapitre B",
              "type": "Chapter"
            },
            {
              "item_id": "100",
              "language_tag": "de",
              "title": "Kapitel C",
              "type": "Chapter"
            }
          ]
        },
        "message": "created",
        "success": true
      }
      """
    And the table "answers" should remain unchanged
    And the table "gradings" should be:
      | answer_id | score | ABS(TIMESTAMPDIFF(SECOND, graded_at, NOW())) < 3 |
      | 124       | 99    | 1                                                |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | participant_id | attempt_id | item_id | score_computed | tasks_tried | validated | latest_activity_at  | latest_submission_at | score_obtained_at   | validated_at |
      | 101            | 0          | 50      | 0              | 0           | 0         | 2019-05-30 11:00:00 | null                 | 2017-04-29 06:38:38 | null         |
      | 101            | 1          | 60      | 99             | 1           | 0         | 2019-05-29 11:00:00 | null                 | 2017-05-29 06:38:38 | null         |
      | 101            | 1          | 80      | 20             | 0           | 0         | 2019-05-29 11:00:00 | null                 | 2017-05-29 06:38:38 | null         |
      | 101            | 1          | 90      | 20             | 0           | 0         | 2019-05-29 11:00:00 | null                 | 2017-05-29 06:38:38 | null         |
    And the table "results_propagate" should be empty
    And the table "results_propagate_sync" should be empty
