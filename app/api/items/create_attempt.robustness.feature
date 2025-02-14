Feature: Create an attempt for an item - robustness
  Background:
    Given the database has the following table "groups":
      | id  | type  | root_activity_id | root_skill_id |
      | 102 | Team  | null             | null          |
      | 103 | Class | 50               | 90            |
      | 104 | Team  | 50               | 90            |
    And the database has the following user:
      | group_id | login |
      | 101      | john  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 103             | 101            |
      | 104             | 101            |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id | url                                                                     | type   | allows_multiple_attempts | default_language_tag |
      | 50 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task   | 0                        | fr                   |
      | 60 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task   | 1                        | fr                   |
      | 90 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Skill  | 1                        | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 101      | 50      | info               |
      | 101      | 60      | content            |
      | 101      | 90      | content            |

  Scenario: Invalid item ids
    Given I am the user with id "101"
    When I send a POST request to "/items/11111111111111111111111111111/222222222222/attempts?parent_attempt_id=0"
    Then the response code should be 400
    And the response error message should contain "Unable to parse one of the integers given as query args (value: '11111111111111111111111111111', param: 'ids')"
    And the table "attempts" should remain unchanged

  Scenario: Invalid parent_attempt_id
    Given I am the user with id "101"
    When I send a POST request to "/items/50/attempts?parent_attempt_id=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for parent_attempt_id (should be int64)"
    And the table "attempts" should remain unchanged

  Scenario: Invalid as_team_id
    Given I am the user with id "101"
    When I send a POST request to "/items/50/attempts?as_team_id=abc&parent_attempt_id=0"
    Then the response code should be 400
    And the response error message should contain "Wrong value for as_team_id (should be int64)"

  Scenario: User not found
    Given I am the user with id "404"
    When I send a POST request to "/items/50/attempts?parent_attempt_id=0"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "attempts" should remain unchanged

  Scenario: No access to the item (no item)
    Given I am the user with id "101"
    When I send a POST request to "/items/404/attempts?parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "attempts" should remain unchanged

  Scenario: No access to the item (info access)
    Given I am the user with id "101"
    When I send a POST request to "/items/50/attempts?parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "attempts" should remain unchanged

  Scenario: No access to the item (as a team)
    Given I am the user with id "101"
    When I send a POST request to "/items/50/attempts?as_team_id=104&parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "attempts" should remain unchanged

  Scenario: No access to the item (type='Skill')
    Given I am the user with id "101"
    When I send a POST request to "/items/90/attempts?parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "attempts" should remain unchanged

  Scenario: User is not a team member
    Given I am the user with id "101"
    When I send a POST request to "/items/60/attempts?as_team_id=102&parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Can't use given as_team_id as a user's team"
    And the table "attempts" should remain unchanged

  Scenario: as_team_id is not a team
    Given I am the user with id "101"
    When I send a POST request to "/items/60/attempts?as_team_id=103&parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Can't use given as_team_id as a user's team"
    And the table "attempts" should remain unchanged

  Scenario: There is an attempt for the (group, item) pair already, but items.allows_multiple_attempts = 0
    Given I am the user with id "101"
    And the database table "permissions_generated" also has the following row:
      | group_id | item_id | can_view_generated |
      | 104      | 50      | content            |
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 104            |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | started_at          |
      | 0          | 104            | 50      | 2019-05-30 11:00:00 |
    When I send a POST request to "/items/50/attempts?as_team_id=104&parent_attempt_id=0"
    Then the response code should be 422
    And the response error message should contain "The item doesn't allow multiple attempts"
    And the table "attempts" should remain unchanged

  Scenario: Not enough permissions for the last item in the path
    Given I am the user with id "101"
    And the database table "permissions_generated" also has the following row:
      | group_id | item_id | can_view_generated |
      | 104      | 50      | info               |
    When I send a POST request to "/items/50/attempts?as_team_id=104&parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "attempts" should remain unchanged
