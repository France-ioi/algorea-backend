Feature: List attempts for current user and item_id - robustness
  Background:
    Given the database has the following users:
      | login | group_id | first_name | last_name |
      | jdoe  | 11       | John       | Doe       |
      | jane  | 12       | Jane       | Doe       |
    And the database has the following table "groups":
      | id | type  |
      | 13 | Team  |
      | 14 | Class |
      | 15 | Team  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 13              | 12             |
      | 14              | 12             |
      | 15              | 12             |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id  | allows_multiple_attempts | default_language_tag |
      | 210 | 1                        | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 11       | 210     | content            |
      | 13       | 210     | info               |
      | 15       | 210     | solution           |
    And the database has the following table "attempts":
      | id | participant_id | created_at          | creator_id | parent_attempt_id | ended_at |
      | 0  | 11             | 2018-05-29 05:38:38 | null       | null              | null     |
      | 1  | 13             | 2018-05-29 05:38:38 | null       | null              | null     |

  Scenario: User doesn't exist
    Given I am the user with id "404"
    When I send a GET request to "/items/1/attempts"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"

  Scenario: Wrong item_id
    Given I am the user with id "11"
    When I send a GET request to "/items/abc/attempts"
    Then the response code should be 400
    And the response error message should contain "Wrong value for item_id (should be int64)"

  Scenario: Wrong sorting
    Given I am the user with id "11"
    When I send a GET request to "/items/210/attempts?parent_attempt_id=0&sort=login"
    Then the response code should be 400
    And the response error message should contain "Unallowed field in sorting parameters: "login""

  Scenario: User doesn't have access to the item
    Given I am the user with id "12"
    When I send a GET request to "/items/210/attempts?attempt_id=0&limit=1"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: Wrong as_team_id
    Given I am the user with id "12"
    When I send a GET request to "/items/210/attempts?attempt_id=0&as_team_id=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for as_team_id (should be int64)"

  Scenario: Team doesn't have access to the item
    Given I am the user with id "12"
    When I send a GET request to "/items/210/attempts?attempt_id=0&as_team_id=13"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: User is not a member of the team
    Given I am the user with id "11"
    When I send a GET request to "/items/210/attempts?attempt_id=0&as_team_id=13"
    Then the response code should be 403
    And the response error message should contain "Can't use given as_team_id as a user's team"

  Scenario: as_team_id is not a team
    Given I am the user with id "12"
    When I send a GET request to "/items/210/attempts?attempt_id=0&as_team_id=14"
    Then the response code should be 403
    And the response error message should contain "Can't use given as_team_id as a user's team"
    And the table "attempts" should remain unchanged

  Scenario: Wrong attempt_id
    Given I am the user with id "11"
    When I send a GET request to "/items/210/attempts?attempt_id=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for attempt_id (should be int64)"

  Scenario: Wrong parent_attempt_id
    Given I am the user with id "11"
    When I send a GET request to "/items/210/attempts?parent_attempt_id=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for parent_attempt_id (should be int64)"

  Scenario: Both attempt_id & parent_attempt_id are given
    Given I am the user with id "11"
    When I send a GET request to "/items/210/attempts?attempt_id=0&parent_attempt_id=0"
    Then the response code should be 400
    And the response error message should contain "Only one of attempt_id and parent_attempt_id can be given"

  Scenario: Neither attempt_id nor parent_attempt_id is given
    Given I am the user with id "11"
    When I send a GET request to "/items/210/attempts"
    Then the response code should be 400
    And the response error message should contain "One of attempt_id and parent_attempt_id should be given"

  Scenario: attempt_id doesn't exist
    Given I am the user with id "11"
    When I send a GET request to "/items/210/attempts?attempt_id=1"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: attempt_id doesn't exist for the team
    Given I am the user with id "12"
    When I send a GET request to "/items/210/attempts?attempt_id=1&as_team_id=15"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
