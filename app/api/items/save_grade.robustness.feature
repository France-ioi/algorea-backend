Feature: Save grading result - robustness
  Background:
    Given the database has the following users:
      | login | group_id |
      | john  | 101      |
    And the database has the following table "platforms":
      | id | regexp                                             | priority | public_key                |
      | 10 | http://taskplatform.mblockelet.info/task.html\?.*  | 2        | {{taskPlatformPublicKey}} |
      | 20 | http://taskplatform1.mblockelet.info/task.html\?.* | 1        | null                      |
    And the database has the following table "items":
      | id | platform_id | url                                                                     | read_only | validation_type | default_language_tag |
      | 50 | 10          | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | 1         | All             | fr                   |
      | 70 | 20          | http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839  | 0         | All             | fr                   |
      | 80 | 10          | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183937 | 0         | All             | fr                   |
      | 10 | null        | null                                                                    | 0         | AllButOne       | fr                   |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 10             | 50            | 0           |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 10               | 50            |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 101      | 50      | content            |
      | 101      | 70      | content            |
      | 101      | 80      | content            |
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 101            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | hints_requested        |
      | 0          | 101            | 50      | [0,  1, "hint" , null] |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 123 | 101       | 101            | 100        | 50      | 2017-05-29 06:38:38 |
    And the server time is frozen

  Scenario: Wrong JSON in request
    When I send a POST request to "/items/save-grade" with the following body:
      """
      []
      """
    Then the response code should be 400
    And the response error message should contain "Json: cannot unmarshal array into Go value of type items.saveGradeRequest"
    And the table "answers" should remain unchanged
    And the table "attempts" should remain unchanged

  Scenario: Invalid score_token
    Given I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "abcdef"
      }
      """
    Then the response code should be 400
    And the response error message should contain "Invalid score_token"
    And the table "answers" should remain unchanged
    And the table "attempts" should remain unchanged

  Scenario: Expired score_token
    Given the server time now is "2020-01-01T00:00:00Z"
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
    Then the server time now is "2020-01-03T00:00:00Z"
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score_token": "{{scoreToken}}"
      }
      """
    Then the response code should be 400
    And the response error message should contain "Invalid score_token: the token has expired"

  Scenario: Falsified score_token
    Given "scoreToken" is a falsified token signed by the task platform with the following payload:
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
    Then the response code should be 400
    And the response error message should contain "Invalid score_token: invalid token: crypto/rsa: verification error"

  Scenario: Platform doesn't use tokens and answer_token is missing
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score": 100.0
      }
      """
    Then the response code should be 400
    And the response error message should contain "Missing answer_token"
    And the table "answers" should remain unchanged
    And the table "attempts" should remain unchanged

  Scenario: Platform doesn't use tokens and answer_token is invalid
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score": 100.0,
        "answer_token": "abc"
      }
      """
    Then the response code should be 400
    And the response error message should contain "Invalid answer_token"
    And the table "answers" should remain unchanged
    And the table "attempts" should remain unchanged

  Scenario: Platform doesn't use tokens and answer_token is expired
    Given the server time now is "2020-01-01T00:00:00Z"
    And "answerToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "70",
        "itemURL": "http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839",
        "idUserAnswer": "123",
        "idAttempt": "101/0",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    Then the server time now is "2020-01-03T00:00:00Z"
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "score": 100.0,
        "answer_token": "{{answerToken}}"
      }
      """
    Then the response code should be 400
    And the response error message should contain "Invalid answer_token: the token has expired"

  Scenario: Platform doesn't use tokens and answer_token is falsified
    Given "answerToken" is a falsified token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "70",
        "itemURL": "http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839",
        "idUserAnswer": "123",
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
    Then the response code should be 400
    And the response error message should contain "Invalid answer_token: invalid token: crypto/rsa: verification error"

  Scenario: Platform doesn't use tokens and idAttempt in answer_token is wrong (should not be null)
    Given "answerToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "70",
        "itemURL": "http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839",
        "idUserAnswer": "123",
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
    Then the response code should be 400
    And the response error message should contain "Invalid answer_token: wrong idAttempt"
    And the table "answers" should remain unchanged
    And the table "attempts" should remain unchanged

  Scenario: Platform doesn't use tokens and idAttempt in answer_token is wrong (format should be number/number)
    Given "answerToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "70",
        "itemURL": "http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839",
        "idAttempt": "110-0",
        "idUserAnswer": "123",
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
    Then the response code should be 400
    And the response error message should contain "Invalid answer_token: wrong idAttempt"
    And the table "answers" should remain unchanged
    And the table "attempts" should remain unchanged

  Scenario: Platform doesn't use tokens and score is missing
    Given "answerToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "70",
        "idAttempt": "101/0",
        "itemURL": "http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839",
        "idUserAnswer": "123",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "answer_token": "{{answerToken}}"
      }
      """
    Then the response code should be 400
    And the response error message should contain "Missing score"
    And the table "answers" should remain unchanged
    And the table "attempts" should remain unchanged

  Scenario: Platform doesn't use tokens and idUserAnswer in answer_token is invalid
    Given "answerToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "70",
        "idAttempt": "101/0",
        "itemURL": "http://taskplatform1.mblockelet.info/task.html?taskId=4034495436721839",
        "idUserAnswer": "abc",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    When I send a POST request to "/items/save-grade" with the following body:
      """
      {
        "answer_token": "{{answerToken}}",
        "score": 99.0
      }
      """
    Then the response code should be 400
    And the response error message should contain "Invalid answer_token: wrong idUserAnswer"
    And the table "answers" should remain unchanged
    And the table "attempts" should remain unchanged

  Scenario: The answer has been already graded
    Given the database table "attempts" also has the following row:
      | id | participant_id |
      | 1  | 101            |
    And the database table "results" also has the following row:
      | attempt_id | participant_id | item_id | validated_at        |
      | 1          | 101            | 80      | 2018-05-29 06:38:38 |
    And the database has the following table "answers":
      | id  | author_id | participant_id | attempt_id | item_id | created_at          |
      | 124 | 101       | 101            | 105        | 80      | 2017-05-29 06:38:38 |
    And the database has the following table "gradings":
      | answer_id | score | graded_at           |
      | 124       | 0     | 2017-05-29 06:38:38 |
    And "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "80",
        "idAttempt": "101/1",
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
    Then the response code should be 403
    And the response error message should contain "The answer has been already graded or is not found"
    And logs should contain:
    """
    {{ quote(`A user tries to replay a score token with a different score value ({"idAttempt":"101/1","idItem":"80","idUser":"101","idUserAnswer":"124","newScore":100,"oldScore":0})`) }}
    """
    And the table "answers" should remain unchanged
    And the table "attempts" should remain unchanged

  Scenario: The answer is not found
    Given "scoreToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "80",
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
    Then the response code should be 403
    And the response error message should contain "The answer has been already graded or is not found"
    And the table "answers" should remain unchanged
    And the table "attempts" should remain unchanged
