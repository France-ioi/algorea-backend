Feature: Enters a contest as a group (user self or team) (contestEnter) - robustness
  Background:
    Given the database has the following table "groups":
      | id | name         | type                | root_activity_id |
      | 9  | Class        | Class               | 50               |
      | 10 | Team 1       | Team                | null             |
      | 11 | Team 2       | Team                | 60               |
      | 99 | item50-group | ContestParticipants | null             |
    And the database has the following users:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
      | 31       | john  | John        | Doe       |
      | 41       | jane  | Jane        | null      |
      | 51       | jack  | Jack        | Daniel    |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 9               | 31             |
      | 10              | 31             |
      | 10              | 41             |
      | 10              | 51             |
      | 11              | 31             |
      | 11              | 41             |
      | 11              | 51             |
    And the groups ancestors are computed

  Scenario: Wrong ids
    Given I am the user with id "31"
    When I send a POST request to "/items/11111111111111111111111111111/22222222222222222/enter?parent_attempt_id=0"
    Then the response code should be 400
    And the response error message should contain "Unable to parse one of the integers given as query args (value: '11111111111111111111111111111', param: 'ids')"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario: Wrong parent_attempt_id
    Given I am the user with id "31"
    When I send a POST request to "/items/50/enter?parent_attempt_id=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for parent_attempt_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario: Wrong as_team_id
    Given I am the user with id "31"
    When I send a POST request to "/items/50/enter?parent_attempt_id=0&as_team_id=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for as_team_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario: The item is not visible to the team
    Given I am the user with id "31"
    Given the database has the following table "items":
      | id | requires_explicit_entry | default_language_tag |
      | 50 | 1                       | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 31       | 50      | content            |
    When I send a POST request to "/items/50/enter?as_team_id=11&parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario: The item is visible, but it doesn't exist
    Given I am the user with id "31"
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 31       | 50      | content            |
    When I send a POST request to "/items/50/enter?parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario: The item is not visible to the user (can_view = none)
    Given the database has the following table "items":
      | id | requires_explicit_entry | default_language_tag |
      | 50 | 1                       | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 31       | 50      | none               |
    And I am the user with id "31"
    When I send a POST request to "/items/50/enter?parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario: The item is visible, but it's not a contest
    Given the database has the following table "items":
      | id | default_language_tag |
      | 50 | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       |
      | 21       | 50      | solution                 |
      | 31       | 50      | content_with_descendants |
    And I am the user with id "31"
    When I send a POST request to "/items/50/enter?parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario: as_team_id is given while the item's entry_participant_type = User
    Given the database has the following table "items":
      | id | requires_explicit_entry | entry_participant_type | default_language_tag |
      | 50 | 1                       | User                   | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       |
      | 11       | 50      | solution                 |
      | 31       | 50      | content_with_descendants |
    And I am the user with id "31"
    When I send a POST request to "/items/50/enter?as_team_id=11&parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario: as_team_id is not a team related to the item while the item's entry_participant_type = Team
    Given the database has the following table "items":
      | id | requires_explicit_entry | entry_participant_type | default_language_tag |
      | 60 | 1                       | Team                   | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       |
      | 11       | 60      | info                     |
      | 21       | 60      | content_with_descendants |
    And I am the user with id "31"
    When I send a POST request to "/items/60/enter?as_team_id=10&parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario: as_team_id is not given while the item's entry_participant_type = Team
    Given the database has the following table "items":
      | id | requires_explicit_entry | entry_participant_type | default_language_tag |
      | 60 | 1                       | Team                   | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       |
      | 11       | 60      | info                     |
    And I am the user with id "31"
    When I send a POST request to "/items/60/enter?parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario: The current user is not a member of as_team_id while the item's entry_participant_type = Team
    Given the database has the following table "items":
      | id | requires_explicit_entry | entry_participant_type | default_language_tag |
      | 60 | 1                       | Team                   | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       |
      | 11       | 60      | info                     |
      | 21       | 60      | content_with_descendants |
    And I am the user with id "21"
    When I send a POST request to "/items/60/enter?as_team_id=11&parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Can't use given as_team_id as a user's team"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario: The contest is not ready
    Given the database has the following table "items":
      | id | requires_explicit_entry | entry_participant_type | entry_min_admitted_members_ratio | entry_max_team_size | default_language_tag |
      | 60 | 1                       | Team                   | All                              | 3                   | fr                   |
    And the database table "permissions_granted" also has the following row:
      | group_id | item_id | source_group_id | can_enter_from      | can_enter_until     |
      | 11       | 60      | 11              | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 |
      | 41       | 60      | 41              | 2007-01-01 10:21:21 | 9999-12-31 23:59:59 |
      | 51       | 60      | 51              | 2007-01-01 10:21:21 | 2008-12-31 23:59:59 |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       |
      | 11       | 60      | info                     |
      | 21       | 60      | content_with_descendants |
    Given the database has the following table "groups_contest_items":
      | group_id | item_id |
      | 11       | 60      |
      | 41       | 60      |
      | 51       | 60      |
    And I am the user with id "31"
    When I send a POST request to "/items/60/enter?as_team_id=11&parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should be empty

  Scenario Outline: Reenter a non-team contest
    Given the database has the following table "items":
      | id | duration | requires_explicit_entry | entry_participant_type | entry_min_admitted_members_ratio | participants_group_id | default_language_tag |
      | 50 | 01:01:01 | 1                       | User                   | None                             | 99                    | fr                   |
    And the database table "groups_groups" also has the following row:
      | parent_group_id | child_group_id | expires_at   |
      | 99              | 31             | <expires_at> |
    And the database table "permissions_granted" also has the following row:
      | group_id | item_id | source_group_id | can_enter_from      | can_enter_until     |
      | 31       | 50      | 31              | 2007-01-01 10:21:21 | 9999-12-31 23:59:59 |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       |
      | 11       | 50      | none                     |
      | 21       | 50      | solution                 |
      | 31       | 50      | content_with_descendants |
    And the database has the following table "groups_contest_items":
      | group_id | item_id | additional_time |
      | 31       | 50      | 02:02:02        |
    And the database has the following table "attempts":
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id |
      | 1  | 31             | 2019-05-29 11:00:00 | 31         | 0                 | 50           |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | started_at          |
      | 1          | 31             | 50      | 2019-05-29 11:00:00 |
    And I am the user with id "31"
    When I send a POST request to "/items/50/enter?parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
  Examples:
    | expires_at          |
    | 2019-05-30 11:00:00 |
    | 9999-12-31 23:59:59 |

  Scenario: Reenter an already entered (not expired) contest as a team
    Given the database has the following table "items":
      | id | duration | requires_explicit_entry | entry_participant_type | entry_min_admitted_members_ratio | entry_max_team_size | participants_group_id | default_language_tag |
      | 60 | 01:01:01 | 1                       | Team                   | None                             | 10                  | 99                    | fr                   |
    And the database table "groups_groups" also has the following row:
      | parent_group_id | child_group_id |
      | 99              | 11             |
    And the database table "permissions_granted" also has the following row:
      | group_id | item_id | source_group_id | can_enter_from      | can_enter_until     |
      | 11       | 60      | 11              | 2007-01-01 10:21:21 | 9999-12-31 23:59:59 |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       |
      | 11       | 60      | solution                 |
      | 31       | 60      | content_with_descendants |
    And the database has the following table "groups_contest_items":
      | group_id | item_id | additional_time |
      | 11       | 60      | 02:02:02        |
    And the database has the following table "attempts":
      | id | participant_id | created_at          | creator_id | parent_attempt_id | root_item_id |
      | 1  | 11             | 2019-05-29 11:00:00 | 31         | 0                 | 60           |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id | started_at          |
      | 1          | 11             | 60      | 2019-05-29 11:00:00 |
    And I am the user with id "31"
    When I send a POST request to "/items/60/enter?as_team_id=11&parent_attempt_id=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And the table "attempts" should remain unchanged
