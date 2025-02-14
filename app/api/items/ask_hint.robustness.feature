Feature: Ask for a hint - robustness
  Background:
    Given the database has the following users:
      | login | group_id |
      | john  | 101      |
    And the database has the following table "platforms":
      | id | regexp                     | public_key                | priority |
      | 10 | https://platformwithkey    | {{taskPlatformPublicKey}} | 0        |
      | 11 | https://nokeyplatform.test |                           | 1        |
    And the database has the following table "items":
      | id | platform_id | url                           | read_only | default_language_tag |
      | 50 | 10          | https://platformwithkey/50    | 1         | fr                   |
      | 10 | 10          | https://platformwithkey/10    | 0         | fr                   |
      | 51 | 11          | https://nokeyplatform.test/51 | 1         | fr                   |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 10             | 50            | 0           |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 10               | 50            |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 101      | 10      | content            |
      | 101      | 50      | content            |
      | 101      | 51      | content            |
    And the database has the following table "attempts":
      | id | participant_id | allows_submissions_until |
      | 0  | 101            | 9999-12-31 23:59:59      |
      | 1  | 101            | 2019-05-30 11:00:00      |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | hints_requested        |
      | 0          | 101            | 50      | [0,  1, "hint" , null] |
      | 0          | 101            | 51      | [0,  1, "hint" , null] |
      | 0          | 101            | 10      | null                   |
      | 1          | 101            | 10      | null                   |
    And the server time is frozen

  Scenario: Wrong JSON in request
    Given I send a POST request to "/items/ask-hint" with the following body:
      """
      []
      """
    Then the response code should be 400
    And the response error message should contain "Json: cannot unmarshal array into Go value of type items.askHintRequestWrapper"
    And the table "attempts" should remain unchanged

  Scenario: Expired task_token
    Given the server time now is "2020-01-01T00:00:00Z"
    And "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    Then the server time now is "2020-01-03T00:00:00Z"
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemUrl": "https://platformwithkey/404",
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
    Then the response code should be 400
    And the response error message should contain "Invalid task_token: the token has expired"

  Scenario: Falsified task_token with non-matching signature
    Given "priorUserTaskToken" is a falsified token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemUrl": "https://platformwithkey/404",
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
    Then the response code should be 400
    And the response error message should contain "Invalid task_token: invalid token: crypto/rsa: verification error"

  Scenario: itemUrls of task_token and hint_requested don't match
    Given "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemUrl": "https://platformwithkey/404",
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
    Then the response code should be 400
    And the response error message should contain "Wrong itemUrl in hint_requested token"
    And the table "attempts" should remain unchanged

  Scenario: idUser in hint_requested token doesn't match the idUser in the task token
    Given "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "20",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50",
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
    Then the response code should be 400
    And the response error message should contain "Token in hint_requested doesn't correspond to user session: got idUser=20, expected 10"
    And the table "attempts" should remain unchanged

  Scenario: idAttempt in hint_requested & task_token don't match
    Given "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/1",
        "itemURL": "https://platformwithkey/50",
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
    Then the response code should be 400
    And the response error message should contain "Wrong idAttempt in hint_requested token"
    And the table "attempts" should remain unchanged

  Scenario: idItemLocal in hint_requested & task_token don't match
    Given "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "10",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/10",
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
    Then the response code should be 400
    And the response error message should contain "Wrong idItemLocal in hint_requested token"
    And the table "attempts" should remain unchanged

  Scenario: No submission rights
    Given "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50",
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
    Then the response code should be 403
    And the response error message should contain "Item is read-only"
    And the table "attempts" should remain unchanged

  Scenario: idAttempt not found
    Given "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "10",
        "idAttempt": "101/2",
        "itemURL": "https://platformwithkey/10",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "10",
        "idAttempt": "101/2",
        "itemURL": "https://platformwithkey/10",
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
    Then the response code should be 404
    And the response error message should contain "No result or the attempt is expired"
    And the table "attempts" should remain unchanged

  Scenario: missing askedHint
    Given "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50"
      }
      """
    When I send a POST request to "/items/ask-hint" with the following body:
      """
      {
        "task_token": "{{priorUserTaskToken}}",
        "hint_requested": "{{hintRequestToken}}"
      }
      """
    Then the response code should be 400
    And the response error message should contain "Asked hint should not be empty"
    And the table "attempts" should remain unchanged

  Scenario: The attempt is expired (doesn't allow submissions anymore)
    Given "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "10",
        "idAttempt": "101/1",
        "itemURL": "https://platformwithkey/10",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    And "hintRequestToken" is a token signed by the task platform with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "10",
        "idAttempt": "101/1",
        "itemURL": "https://platformwithkey/10",
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
    Then the response code should be 404
    And the response error message should contain "No result or the attempt is expired"
    And the table "attempts" should remain unchanged

  Scenario: Should return an error if there is a public key and the hint token's content is sent in clear JSON
    Given "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "50",
        "idAttempt": "101/0",
        "itemURL": "https://platformwithkey/50",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    When I send a POST request to "/items/ask-hint" with the following body:
      """
      {
        "task_token": "{{priorUserTaskToken}}",
        "hint_requested": {
          "idUser": "101",
          "idItemLocal": "50",
          "idAttempt": "101/0",
          "itemURL": "https://platformwithkey/50",
          "askedHint": {"rotorIndex":1}
        }
      }
      """
    Then the response code should be 400
    And the response error message should contain "Invalid hint_requested: json: cannot unmarshal object into Go value of type string"

  Scenario: Should return an error if there is no public key and the hint token's content is sent in clear JSON
    Given "priorUserTaskToken" is a token signed by the app with the following payload:
      """
      {
        "idUser": "101",
        "idItemLocal": "51",
        "idAttempt": "101/0",
        "itemURL": "https://nokeyplatform.test/51",
        "platformName": "{{app().Config.GetString("token.platformName")}}"
      }
      """
    When I send a POST request to "/items/ask-hint" with the following body:
      """
      {
        "task_token": "{{priorUserTaskToken}}",
        "hint_requested": {
          "idUser": "101",
          "idItemLocal": "51",
          "idAttempt": "101/0",
          "itemURL": "https://nokeyplatform.test/51",
          "askedHint": {"rotorIndex":1}
        }
      }
      """
    Then the response code should be 400
    And the response error message should contain "No public key available for item 51"
