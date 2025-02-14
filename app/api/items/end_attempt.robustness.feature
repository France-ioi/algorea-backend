Feature: End an attempt (itemAttemptEnd) - robustness
  Background:
    Given the database has the following table "groups":
      | id  | type                |
      | 102 | Team                |
      | 201 | ContestParticipants |
      | 202 | ContestParticipants |
      | 203 | ContestParticipants |
    And the database has the following users:
      | group_id | login |
      | 101      | john  |
      | 111      | jane  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | expires_at          |
      | 102             | 101            | 9999-12-31 23:59:59 |
      | 201             | 101            | 9999-12-31 23:59:59 |
      | 201             | 111            | 2019-12-31 23:59:59 |
      | 202             | 101            | 9999-12-31 23:59:59 |
      | 202             | 102            | 9999-12-31 23:59:59 |
      | 202             | 111            | 9999-12-31 23:59:59 |
      | 203             | 101            | 9999-12-31 23:59:59 |
      | 203             | 102            | 2019-12-31 23:59:59 |
      | 203             | 111            | 9999-12-31 23:59:59 |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id | url                                                                     | type    | allows_multiple_attempts | participants_group_id | default_language_tag |
      | 10 | null                                                                    | Chapter | 0                        | 201                   | fr                   |
      | 50 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task    | 0                        | 202                   | fr                   |
      | 60 | http://taskplatform.mblockelet.info/task.html?taskId=403449543672183936 | Task    | 1                        | 203                   | fr                   |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 10             | 60            | 1           |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 10               | 60            |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       |
      | 101      | 50      | content                  |
      | 102      | 60      | content                  |
      | 111      | 10      | content_with_descendants |
      | 111      | 50      | content_with_descendants |
    And the database has the following table "attempts":
      | id | participant_id | root_item_id | parent_attempt_id | allows_submissions_until | ended_at            |
      | 0  | 101            | 10           | null              | 9999-12-31 23:59:59      | null                |
      | 0  | 102            | 10           | null              | 9999-12-31 23:59:59      | null                |
      | 0  | 111            | 10           | null              | 9999-12-31 23:59:59      | null                |
      | 1  | 101            | 50           | null              | 9999-12-31 23:59:59      | null                |
      | 1  | 102            | 50           | null              | 2019-12-31 23:59:59      | null                |
      | 1  | 111            | 50           | null              | 9999-12-31 23:59:59      | null                |
      | 2  | 101            | 60           | 0                 | 9999-12-31 23:59:59      | null                |
      | 2  | 102            | 60           | 0                 | 9999-12-31 23:59:59      | 2019-12-31 23:59:59 |
      | 2  | 111            | 60           | 0                 | 9999-12-31 23:59:59      | null                |

  Scenario: Wrong attempt_id
    Given I am the user with id "111"
    When I send a POST request to "/attempts/abc/end"
    Then the response code should be 400
    And the response error message should contain "Wrong value for attempt_id (should be int64)"
    And the table "attempts" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Zero attempt_id
    Given I am the user with id "111"
    When I send a POST request to "/attempts/0/end"
    Then the response code should be 403
    And the response error message should contain "Implicit attempts cannot be ended"
    And the table "attempts" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Wrong as_team_id
    Given I am the user with id "101"
    When I send a POST request to "/attempts/1/end?as_team_id=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for as_team_id (should be int64)"
    And the table "attempts" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: The user is not a member of the team
    Given I am the user with id "111"
    When I send a POST request to "/attempts/1/end?as_team_id=102"
    Then the response code should be 403
    And the response error message should contain "Can't use given as_team_id as a user's team"
    And the table "attempts" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: The attempt is expired
    Given I am the user with id "101"
    When I send a POST request to "/attempts/1/end?as_team_id=102"
    Then the response code should be 403
    And the response error message should contain "Active attempt not found"
    And the table "attempts" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: The attempt is ended
    Given I am the user with id "101"
    When I send a POST request to "/attempts/2/end?as_team_id=102"
    Then the response code should be 403
    And the response error message should contain "Active attempt not found"
    And the table "attempts" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: No attempt
    Given I am the user with id "111"
    When I send a POST request to "/attempts/3/end"
    Then the response code should be 403
    And the response error message should contain "Active attempt not found"
    And the table "attempts" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
