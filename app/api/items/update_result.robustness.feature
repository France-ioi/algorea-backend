Feature: Update an attempt result - robustness
  Background:
    Given the database has the following table "groups":
      | id | name    | type  |
      | 13 | Group B | Class |
      | 23 | Group C | Team  |
      | 25 | Group D | Club  |
    And the database has the following users:
      | group_id | login | first_name | last_name |
      | 11       | jdoe  | John       | Doe       |
      | 21       | other | George     | Bush      |
      | 24       | jane  | Jane       | Joe       |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | personal_info_view_approved_at |
      | 13              | 11             | null                           |
      | 13              | 21             | null                           |
      | 23              | 21             | 2019-05-30 11:00:00            |
      | 23              | 24             | null                           |
      | 23              | 31             | null                           |
      | 25              | 24             | 2019-05-30 11:00:00            |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id  | allows_multiple_attempts | default_language_tag |
      | 200 | 0                        | fr                   |
      | 210 | 1                        | fr                   |
    And the database has the following table "attempts":
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id | ended_at            |
      | 1  | 11             | 2018-05-29 05:38:38 | 21         | 0                 | 210          | 2018-05-29 05:38:38 |
      | 2  | 11             | 2018-05-29 05:38:38 | 11         | 1                 | 200          | 2018-05-29 05:38:38 |
      | 0  | 11             | 2018-05-29 05:38:38 | null       | null              | null         | null                |
      | 0  | 23             | 2019-05-29 05:38:38 | 11         | null              | null         | null                |
      | 1  | 23             | 2019-05-29 05:38:38 | 24         | 0                 | 210          | null                |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | score_computed | validated_at        | started_at          | latest_activity_at  | help_requested |
      | 0          | 11             | 200     | 99             | null                | 2018-05-29 06:38:38 | 2018-05-29 06:38:39 | false          |
      | 0          | 23             | 210     | 99             | 2018-05-29 08:00:00 | 2019-05-29 06:38:38 | 2019-05-29 06:38:39 | true           |
      | 1          | 11             | 200     | 100            | 2018-05-29 07:00:00 | 2018-05-29 06:38:38 | 2018-05-29 06:38:39 | false          |
      | 1          | 11             | 210     | 99             | 2018-05-29 08:00:00 | 2019-05-29 06:38:38 | 2019-05-29 06:38:39 | false          |
      | 1          | 23             | 210     | 99             | 2018-05-29 08:00:00 | 2019-05-29 06:38:38 | 2019-05-29 06:38:39 | false          |
      | 2          | 11             | 200     | 100            | 2018-05-29 07:00:00 | 2018-05-29 06:38:38 | 2018-05-29 06:38:39 | false          |
      | 2          | 11             | 210     | 99             | 2018-05-29 08:00:00 | 2019-05-29 06:38:38 | 2019-05-29 06:38:39 | true           |

  Scenario: Invalid item_id
    Given I am the user with id "21"
    When I send a PUT request to "/items/abc/attempts/1"
    Then the response code should be 400
    And the response error message should contain "Wrong value for item_id (should be int64)"

  Scenario: Invalid attempt_id
    Given I am the user with id "21"
    When I send a PUT request to "/items/200/attempts/abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for attempt_id (should be int64)"

  Scenario: Invalid as_team_id
    Given I am the user with id "21"
    When I send a PUT request to "/items/200/attempts/1?as_team_id=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for as_team_id (should be int64)"

  Scenario: as_team_id is not a user's team
    Given I am the user with id "11"
    When I send a PUT request to "/items/200/attempts/1?as_team_id=13"
    Then the response code should be 403
    And the response error message should contain "Can't use given as_team_id as a user's team"

  Scenario: No result with given participant_id, attempt_id, item_id
    Given I am the user with id "11"
    When I send a PUT request to "/items/200/attempts/3" with the following body:
      """
      {
        "help_requested": true
      }
      """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: Wrong data (wrong value for help_requested)
    Given I am the user with id "11"
    When I send a PUT request to "/items/200/attempts/1" with the following body:
      """
      {
        "help_requested": 1,
        "attempt_id": "2",
        "unknown": "abc"
      }
      """
    Then the response code should be 400
    And the response body should be, in JSON:
      """
      {
        "success": false,
        "message": "Bad Request",
        "error_text": "Invalid input data",
        "errors":{
          "help_requested": ["expected type 'bool', got unconvertible type 'float64'"]
        }
      }
      """
    And the table "results" should remain unchanged

  Scenario: Wrong data (unknown fields)
    Given I am the user with id "11"
    When I send a PUT request to "/items/200/attempts/1" with the following body:
      """
      {
        "attempt_id": "2",
        "unknown": "abc"
      }
      """
    Then the response code should be 400
    And the response body should be, in JSON:
      """
      {
        "success": false,
        "message": "Bad Request",
        "error_text": "Invalid input data",
        "errors":{
          "attempt_id": ["unexpected field"],
          "unknown": ["unexpected field"]
        }
      }
      """
    And the table "results" should remain unchanged
