Feature: Start results for an item path - robustness
  Background:
    Given the database has the following table "groups":
      | id  | type  | root_activity_id | root_skill_id |
      | 101 | User  | 70               | null          |
      | 102 | Team  | null             | null          |
      | 103 | Class | 50               | 90            |
      | 104 | Team  | 50               | 90            |
    And the database has the following users:
      | group_id | login |
      | 101      | john  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 103             | 101            |
      | 104             | 101            |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id | url                                                                     | type   | allows_multiple_attempts | default_language_tag | requires_explicit_entry |
      | 50 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task   | 0                        | fr                   | false                   |
      | 60 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task   | 1                        | fr                   | false                   |
      | 70 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task   | 1                        | fr                   | true                    |
      | 90 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Skill  | 1                        | fr                   | false                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 101      | 50      | info               |
      | 101      | 60      | content            |
      | 101      | 70      | content            |
      | 101      | 90      | content            |
    And the database has the following table "attempts":
      | participant_id | id |
      | 101            | 0  |

  Scenario: Invalid item ids
    Given I am the user with id "101"
    When I send a POST request to "/items/11111111111111111111111111111/222222222222/start-result-path?attempt_id=0"
    Then the response code should be 400
    And the response error message should contain "Unable to parse one of the integers given as query args (value: '11111111111111111111111111111', param: 'ids')"
    And the table "results" should remain unchanged

  Scenario: The path is too long
    Given I am the user with id "101"
    When I send a POST request to "/items/1/2/3/4/5/6/7/8/9/10/11/start-result-path"
    Then the response code should be 400
    And the response error message should contain "No more than 10 ids expected"
    And the table "results" should remain unchanged

  Scenario: Invalid as_team_id
    Given I am the user with id "101"
    When I send a POST request to "/items/50/start-result-path?as_team_id=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for as_team_id (should be int64)"

  Scenario: User not found
    Given I am the user with id "404"
    When I send a POST request to "/items/50/start-result-path"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "results" should remain unchanged

  Scenario: No access to the item (no item)
    Given I am the user with id "101"
    When I send a POST request to "/items/404/start-result-path"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "results" should remain unchanged

  Scenario: No access to the item (info access)
    Given I am the user with id "101"
    When I send a POST request to "/items/50/start-result-path"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "results" should remain unchanged

  Scenario: No access to the item (as a team)
    Given I am the user with id "101"
    When I send a POST request to "/items/50/start-result-path?as_team_id=104"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "results" should remain unchanged

  Scenario: User is not a team member
    Given I am the user with id "101"
    When I send a POST request to "/items/60/start-result-path?as_team_id=102"
    Then the response code should be 403
    And the response error message should contain "Can't use given as_team_id as a user's team"
    And the table "results" should remain unchanged

  Scenario: as_team_id is not a team
    Given I am the user with id "101"
    When I send a POST request to "/items/60/start-result-path?as_team_id=103"
    Then the response code should be 403
    And the response error message should contain "Can't use given as_team_id as a user's team"
    And the table "results" should remain unchanged

  Scenario: Cannot start the path
    Given I am the user with id "101"
    When I send a POST request to "/items/70/start-result-path"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "results" should remain unchanged
