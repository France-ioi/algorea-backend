Feature: Generate a task token with a refreshed attempt for an item - robustness
  Background:
    Given the database has the following table "groups":
      | id  | type  |
      | 102 | Team  |
      | 103 | Class |
      | 104 | Team  |
    And the database has the following user:
      | group_id | login |
      | 101      | john  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 103             | 101            |
      | 104             | 101            |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id | url                                                                     | type    | entry_participant_type | default_language_tag |
      | 50 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task    | User                   | fr                   |
      | 60 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task    | Team                   | fr                   |
      | 90 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Chapter | Team                   | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 101      | 50      | info               |
      | 101      | 60      | content            |
      | 101      | 70      | content            |
      | 101      | 80      | content            |
      | 101      | 90      | content            |
      | 102      | 60      | content            |
      | 103      | 60      | content            |
      | 104      | 60      | content            |
    And the database has the following table "attempts":
      | id | participant_id | created_at          | allows_submissions_until |
      | 0  | 101            | 2017-05-29 05:38:38 | 9999-12-31 23:59:59      |
      | 0  | 102            | 2017-05-29 05:38:38 | 9999-12-31 23:59:59      |
      | 0  | 103            | 2017-05-29 05:38:38 | 9999-12-31 23:59:59      |
      | 0  | 104            | 2017-05-29 05:38:38 | 9999-12-31 23:59:59      |
      | 1  | 101            | 2017-05-29 05:38:38 | 2019-05-30 11:00:00      |
      | 2  | 101            | 2017-05-29 05:38:38 | 9999-12-31 23:59:59      |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | latest_activity_at  | started_at          | score_computed | score_obtained_at | validated_at |
      | 0          | 101            | 50      | 2018-05-29 06:38:38 | 2017-05-29 06:38:38 | 0              | null              | null         |
      | 0          | 101            | 90      | 2018-05-29 06:38:38 | 2017-05-29 06:38:38 | 0              | null              | null         |
      | 0          | 102            | 60      | 2018-05-29 06:38:38 | 2017-05-29 06:38:38 | 0              | null              | null         |
      | 0          | 103            | 60      | 2018-05-29 06:38:38 | 2017-05-29 06:38:38 | 0              | null              | null         |
      | 0          | 104            | 60      | 2018-05-29 06:38:38 | 2017-05-29 06:38:38 | 0              | null              | null         |
      | 1          | 101            | 60      | 2018-05-29 06:38:38 | 2017-05-29 06:38:38 | 0              | null              | null         |
      | 2          | 101            | 60      | 2018-05-29 06:38:38 | null                | 0              | null              | null         |

  Scenario: Invalid attempt_id
    Given I am the user with id "101"
    When I send a POST request to "/items/50/attempts/abc/generate-task-token"
    Then the response code should be 400
    And the response error message should contain "Wrong value for attempt_id (should be int64)"
    And the table "attempts" should remain unchanged

  Scenario: Invalid item_id
    Given I am the user with id "101"
    When I send a POST request to "/items/abc/attempts/0/generate-task-token"
    Then the response code should be 400
    And the response error message should contain "Wrong value for item_id (should be int64)"
    And the table "attempts" should remain unchanged

  Scenario: Invalid as_team_id
    Given I am the user with id "101"
    When I send a POST request to "/items/50/attempts/0/generate-task-token?as_team_id=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for as_team_id (should be int64)"
    And the table "attempts" should remain unchanged

  Scenario: User not found
    Given I am the user with id "404"
    When I send a POST request to "/items/50/attempts/0/generate-task-token"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "attempts" should remain unchanged

  Scenario: No attempt
    Given I am the user with id "101"
    When I send a POST request to "/items/50/attempts/404/generate-task-token"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "attempts" should remain unchanged

  Scenario: No access to the item (info access)
    Given I am the user with id "101"
    When I send a POST request to "/items/50/attempts/0/generate-task-token"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "attempts" should remain unchanged

  Scenario: No access to the item (type='Chapter')
    Given I am the user with id "101"
    When I send a POST request to "/items/90/attempts/0/generate-task-token"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "attempts" should remain unchanged

  Scenario: User is not a team member
    Given I am the user with id "101"
    When I send a POST request to "/items/60/attempts/0/generate-task-token?as_team_id=102"
    Then the response code should be 403
    And the response error message should contain "Can't use given as_team_id as a user's team"
    And the table "attempts" should remain unchanged

  Scenario: Attempt group is not a team
    Given I am the user with id "101"
    When I send a POST request to "/items/60/attempts/0/generate-task-token?as_team_id=103"
    Then the response code should be 403
    And the response error message should contain "Can't use given as_team_id as a user's team"
    And the table "attempts" should remain unchanged

  Scenario: No result in the DB
    Given I am the user with id "101"
    When I send a POST request to "/items/60/attempts/0/generate-task-token"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "attempts" should remain unchanged

  Scenario: The attempt is expired (doesn't allow submissions)
    Given I am the user with id "101"
    When I send a POST request to "/items/60/attempts/1/generate-task-token"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "attempts" should remain unchanged

  Scenario: The result is not started
    Given I am the user with id "101"
    When I send a POST request to "/items/60/attempts/2/generate-task-token"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "attempts" should remain unchanged
