Feature: Ask for a hint
  Background:
    Given the database has the following users:
      | login | group_id |
      | john  | 101      |
    And the database has the following table "groups":
      | id  | name | type |
      | 201 | team | Team |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 201             | 101            |
    And the groups ancestors are computed
    And the database has the following table "platforms":
      | id | regexp                                            | public_key                |
      | 10 | http://taskplatform.mblockelet.info/task.html\?.* | {{taskPlatformPublicKey}} |
    And the database has the following table "items":
      | id | platform_id | url                                                                     | default_language_tag |
      | 50 | 10          | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | fr                   |
      | 10 | null        | null                                                                    | fr                   |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 10             | 50            | 0           |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 10               | 50            |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 101      | 50      | content            |
      | 201      | 50      | content            |
    And the server time is frozen

  Scenario: User is able to ask for a hint
    Given the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | hints_requested        | hints_cached | started_at          |
      | 0          | 101            | 50      | [0,  1, "hint" , null] | 4            | 2019-05-30 11:00:00 |
      | 0          | 101            | 10      | null                   | 0            | 2019-05-30 11:00:00 |
    And "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "askedHint": {"rotorIndex":1}
      }
      """
    When I send a POST request to "/items/ask-hint" with the following body:
      """
      {
        "task_token": "{{priorUserTaskToken}}",
        "hint_requested": "{{hintRequestToken}}"
      }
      """
    Then the response code should be 201
    And the response body decoded as "AskHintResponse" should be, in JSON:
      """
      {
        "data": {
          "task_token": {
            "date": "{{currentTimeInFormat("02-01-2006")}}",
            "idUser": "101",
            "idItemLocal": "50",
            "idAttempt": "101/0",
            "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
            "randomSeed": "",
            "platformName": "{{app().Config.GetString("token.platformName")}}",
            "sHintsRequested": "[0,1,\"hint\",null,{\"rotorIndex\":1}]",
            "nbHintsGiven": "5"
          }
        },
        "message": "created",
        "success": true
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | tasks_with_help | hints_cached | hints_requested                    | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, latest_hint_at, NOW())) < 3 |
      | 0          | 101            | 10      | 1               | 0            | null                               | 1                                                         | null                                                  |
      | 0          | 101            | 50      | 1               | 5            | [0,1,"hint",null,{"rotorIndex":1}] | 1                                                         | 1                                                     |
    And the table "results_propagate" should be empty

  Scenario: User is able to ask for a hint for a team (participant_id is the first integer in idAttempt in the task token)
    Given the database has the following table "attempts":
      | id | participant_id |
      | 0  | 201            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | hints_requested        | hints_cached | started_at          |
      | 0          | 201            | 50      | [0,  1, "hint" , null] | 4            | 2020-01-01 00:00:00 |
      | 0          | 201            | 10      | null                   | 0            | 2020-01-01 00:00:00 |
    And "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "201/0",
        "itemURL": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "201/0",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "askedHint": {"rotorIndex":1}
      }
      """
    When I send a POST request to "/items/ask-hint" with the following body:
      """
      {
        "task_token": "{{priorUserTaskToken}}",
        "hint_requested": "{{hintRequestToken}}"
      }
      """
    Then the response code should be 201
    And the response body decoded as "AskHintResponse" should be, in JSON:
      """
      {
        "data": {
          "task_token": {
            "date": "{{currentTimeInFormat("02-01-2006")}}",
            "idUser": "101",
            "idItemLocal": "50",
            "idAttempt": "201/0",
            "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
            "randomSeed": "",
            "platformName": "{{app().Config.GetString("token.platformName")}}",
            "sHintsRequested": "[0,1,\"hint\",null,{\"rotorIndex\":1}]",
            "nbHintsGiven": "5"
          }
        },
        "message": "created",
        "success": true
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | tasks_with_help | hints_cached | hints_requested                    | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, latest_hint_at, NOW())) < 3 |
      | 0          | 201            | 10      | 1               | 0            | null                               | 1                                                         | null                                                  |
      | 0          | 201            | 50      | 1               | 5            | [0,1,"hint",null,{"rotorIndex":1}] | 1                                                         | 1                                                     |
    And the table "results_propagate" should be empty

  Scenario: User is able to ask for a hint with a minimal hint token
    Given the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | hints_requested        | started_at          |
      | 0          | 101            | 10      | null                   | 2019-05-30 11:00:00 |
      | 0          | 101            | 50      | [0,  1, "hint" , null] | 2019-05-30 11:00:00 |
    And "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "askedHint": {"rotorIndex":1}
      }
      """
    When I send a POST request to "/items/ask-hint" with the following body:
      """
      {
        "task_token": "{{priorUserTaskToken}}",
        "hint_requested": "{{hintRequestToken}}"
      }
      """
    Then the response code should be 201
    And the response body decoded as "AskHintResponse" should be, in JSON:
      """
      {
        "data": {
          "task_token": {
            "date": "{{currentTimeInFormat("02-01-2006")}}",
            "idUser": "101",
            "idItemLocal": "50",
            "idAttempt": "101/0",
            "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
            "randomSeed": "",
            "platformName": "{{app().Config.GetString("token.platformName")}}",
            "sHintsRequested": "[0,1,\"hint\",null,{\"rotorIndex\":1}]",
            "nbHintsGiven": "5"
          }
        },
        "message": "created",
        "success": true
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | tasks_with_help | hints_cached | hints_requested                    | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, latest_hint_at, NOW())) < 3 |
      | 0          | 101            | 10      | 1               | 0            | null                               | 1                                                         | null                                                  |
      | 0          | 101            | 50      | 1               | 5            | [0,1,"hint",null,{"rotorIndex":1}] | 1                                                         | 1                                                     |
    And the table "results_propagate" should be empty

  Scenario: User is able to ask for an already given hint
    Given the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | hints_requested        | started_at          |
      | 0          | 101            | 50      | [0,  1, "hint" , null] | 2019-05-30 11:00:00 |
      | 0          | 101            | 10      | null                   | 2019-05-30 11:00:00 |
    And "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "askedHint": "hint"
      }
      """
    When I send a POST request to "/items/ask-hint" with the following body:
      """
      {
        "task_token": "{{priorUserTaskToken}}",
        "hint_requested": "{{hintRequestToken}}"
      }
      """
    Then the response code should be 201
    And the response body decoded as "AskHintResponse" should be, in JSON:
      """
      {
        "data": {
          "task_token": {
            "date": "{{currentTimeInFormat("02-01-2006")}}",
            "idUser": "101",
            "idItemLocal": "50",
            "idAttempt": "101/0",
            "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
            "randomSeed": "",
            "platformName": "{{app().Config.GetString("token.platformName")}}",
            "sHintsRequested": "[0,1,\"hint\",null]",
            "nbHintsGiven": "4"
          }
        },
        "message": "created",
        "success": true
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | tasks_with_help | hints_cached | hints_requested   | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, latest_hint_at, NOW())) < 3 |
      | 0          | 101            | 10      | 1               | 0            | null              | 1                                                         | null                                                  |
      | 0          | 101            | 50      | 1               | 4            | [0,1,"hint",null] | 1                                                         | 1                                                     |
    And the table "results_propagate" should be empty

  Scenario: Can't parse hints_requested
    Given the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | hints_requested | started_at          |
      | 0          | 101            | 50      | not an array    | 2019-05-30 11:00:00 |
      | 0          | 101            | 10      | null            | 2019-05-30 11:00:00 |
    And "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
        "askedHint": {"rotorIndex":1}
      }
      """
    When I send a POST request to "/items/ask-hint" with the following body:
      """
      {
        "task_token": "{{priorUserTaskToken}}",
        "hint_requested": "{{hintRequestToken}}"
      }
      """
    Then the response code should be 201
    And the response body decoded as "AskHintResponse" should be, in JSON:
      """
      {
        "data": {
          "task_token": {
            "date": "{{currentTimeInFormat("02-01-2006")}}",
            "idUser": "101",
            "idItemLocal": "50",
            "idAttempt": "101/0",
            "itemUrl": "http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936",
            "randomSeed": "",
            "platformName": "{{app().Config.GetString("token.platformName")}}",
            "sHintsRequested": "[{\"rotorIndex\":1}]",
            "nbHintsGiven": "1"
          }
        },
        "message": "created",
        "success": true
      }
      """
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id | tasks_with_help | hints_cached | hints_requested    | ABS(TIMESTAMPDIFF(SECOND, latest_activity_at, NOW())) < 3 | ABS(TIMESTAMPDIFF(SECOND, latest_hint_at, NOW())) < 3 |
      | 0          | 101            | 10      | 1               | 0            | null               | 1                                                         | null                                                  |
      | 0          | 101            | 50      | 1               | 1            | [{"rotorIndex":1}] | 1                                                         | 1                                                     |
    And the table "results_propagate" should be empty
    And logs should contain:
      """
      {{ quote(`Unable to parse hints_requested ({"idAttempt":"101/0","idItemLocal":"50","idUser":"101"}) having value "not an array": invalid character 'o' in literal null (expecting 'u')`) }}
      """
