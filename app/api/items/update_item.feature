Feature: Update item
Background:
  Given the database has the following table "groups":
    | id | name | type |
    | 10 | Club | Club |
  And the database has the following user:
    | group_id | login |
    | 11       | jdoe  |
  And the database has the following table "items":
    | id | type    | url                  | options   | default_language_tag | no_score | text_id | title_bar_visible | display_details_in_parent | uses_api | read_only | full_screen | children_layout | hints_allowed | fixed_ranks | validation_type | entry_min_admitted_members_ratio | entry_frozen_teams | entry_max_team_size | allows_multiple_attempts | duration | requires_explicit_entry | show_user_infos | prompt_to_join_group_by_code | entering_time_min   | entering_time_max   | participants_group_id |
    | 21 | Chapter | http://someurl1.com/ | {"opt":1} | en                   | 1        | Task1   | 0                 | 1                         | 0        | 1         | forceNo     | List            | 1             | 1           | One             | Half                             | 0                  | 10                  | 1                        | 01:20:30 | 1                       | 1               | 1                            | 2007-01-01 01:02:03 | 3007-01-01 01:02:03 | null                  |
    | 50 | Chapter | http://someurl2.com/ | {"opt":2} | en                   | 1        | Task2   | 0                 | 1                         | 0        | 1         | forceNo     | List            | 1             | 1           | One             | Half                             | 0                  | 10                  | 1                        | 01:20:30 | 1                       | 1               | 1                            | 2007-01-01 01:02:03 | 3007-01-01 01:02:03 | null                  |
    | 60 | Chapter | http://someurl2.com/ | {"opt":2} | en                   | 1        | Task3   | 0                 | 1                         | 0        | 1         | forceNo     | List            | 1             | 1           | One             | Half                             | 0                  | 10                  | 1                        | 01:20:30 | 1                       | 1               | 1                            | 2007-01-01 01:02:03 | 3007-01-01 01:02:03 | 1234                  |
    | 70 | Skill   | http://someurl3.com/ | {"opt":3} | en                   | 0        | null    | 0                 | 1                         | 0        | 1         | forceNo     | List            | 1             | 1           | One             | Half                             | 0                  | 10                  | 1                        | 01:20:30 | 1                       | 1               | 1                            | 2007-01-01 01:02:03 | 3007-01-01 01:02:03 | 1234                  |
  And the database has the following table "items_items":
    | parent_item_id | child_item_id | child_order |
    | 21             | 60            | 0           |
    | 50             | 21            | 0           |
  And the database has the following table "items_ancestors":
    | ancestor_item_id | child_item_id |
    | 21               | 60            |
    | 50               | 21            |
    | 50               | 60            |
  And the database has the following table "permissions_generated":
    | group_id | item_id | can_view_generated       | can_grant_view_generated | can_edit_generated | is_owner_generated |
    | 10       | 50      | content_with_descendants | solution_with_grant      | all                | true               |
    | 11       | 21      | content                  | none                     | children           | false              |
    | 11       | 50      | none                     | none                     | none               | false              |
    | 11       | 60      | solution                 | solution_with_grant      | all_with_grant     | true               |
    | 11       | 70      | content                  | solution_with_grant      | all_with_grant     | true               |
  And the database has the following table "permissions_granted":
    | group_id | item_id | can_view | is_owner | source_group_id | latest_update_at    |
    | 10       | 50      | none     | true     | 11              | 2019-05-30 11:00:00 |
    | 11       | 21      | content  | false    | 11              | 2019-05-30 11:00:00 |
    | 11       | 50      | none     | false    | 11              | 2019-05-30 11:00:00 |
    | 11       | 60      | none     | true     | 11              | 2019-05-30 11:00:00 |
    | 11       | 70      | none     | true     | 11              | 2019-05-30 11:00:00 |
  And the database has the following table "groups_groups":
    | parent_group_id | child_group_id |
    | 10              | 11             |
  And the groups ancestors are computed
  And the database has the following table "attempts":
    | id | participant_id |
    | 0  | 11             |
  And the database has the following table "results":
    | attempt_id | participant_id | item_id | score_computed |
    | 0          | 11             | 21      | 1              |
    | 0          | 11             | 50      | 10             |
    | 0          | 11             | 70      | 20             |
  And the database has the following table "languages":
    | tag |
    | en  |
    | sl  |

  Scenario: Valid
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "url": "http://someurl3.com/",
        "options": "{\"opt\":3}"
      }
      """
    Then the response should be "updated"
    And the table "items" should remain unchanged, regardless of the row with id "50"
    And the table "items" at id "50" should be:
      | id | type    | url                  | options   | default_language_tag | no_score | text_id | title_bar_visible | display_details_in_parent | uses_api | read_only | full_screen | children_layout | hints_allowed | fixed_ranks | validation_type | entry_min_admitted_members_ratio | entry_frozen_teams | entry_max_team_size | allows_multiple_attempts | duration | show_user_infos | prompt_to_join_group_by_code | entering_time_min   | entering_time_max   | participants_group_id |
      | 50 | Chapter | http://someurl3.com/ | {"opt":3} | en                   | 1        | Task2   | 0                 | 1                         | 0        | 1         | forceNo     | List            | 1             | 1           | One             | Half                             | 0                  | 10                  | 1                        | 01:20:30 | 1               | 1                            | 2007-01-01 01:02:03 | 3007-01-01 01:02:03 | null                  |
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: Valid (set nullable fields to null)
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "url": null, "options": null, "text_id": null, "duration": null
      }
      """
    Then the response should be "updated"
    And the table "items" should remain unchanged, regardless of the row with id "50"
    And the table "items" at id "50" should be:
      | id | type    | url  | options | default_language_tag | no_score | text_id | title_bar_visible | display_details_in_parent | uses_api | read_only | full_screen | children_layout | hints_allowed | fixed_ranks | validation_type | entry_min_admitted_members_ratio | entry_frozen_teams | entry_max_team_size | allows_multiple_attempts | duration | show_user_infos | prompt_to_join_group_by_code | entering_time_min   | entering_time_max   | participants_group_id |
      | 50 | Chapter | null | null    | en                   | 1        | null    | 0                 | 1                         | 0        | 1         | forceNo     | List            | 1             | 1           | One             | Half                             | 0                  | 10                  | 1                        | null     | 1               | 1                            | 2007-01-01 01:02:03 | 3007-01-01 01:02:03 | null                  |
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: Valid (all the fields are set)
    Given I am the user with id "11"
    And the database has the following table "items":
      | id  | default_language_tag |
      | 112 | fr                   |
      | 134 | fr                   |
    And the database has the following table "items_strings":
      | language_tag | item_id |
      | sl           | 50      |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 11       | 112     | solution           | content                  | answer              | all                | false              |
      | 11       | 134     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
    And the database has the following table "permissions_granted":
      | group_id | item_id | can_view | can_grant_view | can_watch | can_edit | is_owner | source_group_id | latest_update_at    |
      | 11       | 112     | solution | content        | answer    | all      | false    | 11              | 2019-05-30 11:00:00 |
      | 11       | 134     | none     | none           | none      | none     | true     | 11              | 2019-05-30 11:00:00 |
    And the database table "results" also has the following rows:
      | attempt_id | participant_id | item_id | score_computed |
      | 0          | 11             | 112     | 50             |
      | 0          | 11             | 134     | 60             |
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "url": "http://myurl.com/",
        "options": "{\"opt\":true}",
        "text_id": "Tasknumber1",
        "title_bar_visible": true,
        "display_details_in_parent": false,
        "uses_api": true,
        "read_only": false,
        "full_screen": "forceYes",
        "children_layout": "Grid",
        "hints_allowed": false,
        "fixed_ranks": false,
        "validation_type": "AllButOne",
        "entry_min_admitted_members_ratio": "All",
        "entry_frozen_teams": true,
        "entry_max_team_size": 2345,
        "allows_multiple_attempts": false,
        "duration": "01:02:03",
        "requires_explicit_entry": true,
        "show_user_infos": false,
        "no_score": false,
        "prompt_to_join_group_by_code": false,
        "default_language_tag": "sl",
        "children": [
          {"item_id": "112", "order": 0, "category": "Discovery", "score_weight": 1},
          {"item_id": "134", "order": 1, "category": "Application", "score_weight": 2}
        ]
      }
      """
    Then the response should be "updated"
    And the table "items" should remain unchanged, regardless of the row with id "50"
    And the table "items" at id "50" should be:
      | id | type    | url               | options      | default_language_tag | entry_frozen_teams | no_score | text_id     | title_bar_visible | display_details_in_parent | uses_api | read_only | full_screen | children_layout | hints_allowed | fixed_ranks | validation_type | entry_min_admitted_members_ratio | entry_frozen_teams | entry_max_team_size | allows_multiple_attempts | duration | requires_explicit_entry | show_user_infos | prompt_to_join_group_by_code | participants_group_id |
      | 50 | Chapter | http://myurl.com/ | {"opt":true} | sl                   | 1                  | 0        | Tasknumber1 | 1                 | 0                         | 1        | 0         | forceYes    | Grid            | 0             | 0           | AllButOne       | All                              | 1                  | 2345                | 0                        | 01:02:03 | 1                       | 0               | 0                            | 5577006791947779410   |
    And the table "items_strings" should remain unchanged
    And the table "items_items" should be:
      | parent_item_id | child_item_id | category    | score_weight | content_view_propagation | upper_view_levels_propagation | grant_view_propagation | watch_propagation | edit_propagation |
      | 21             | 60            | Undefined   | 1            | none                     | use_content_view_propagation  | 0                      | 0                 | 0                |
      | 50             | 112           | Discovery   | 1            | as_info                  | use_content_view_propagation  | 0                      | 0                 | 0                |
      | 50             | 134           | Application | 2            | as_info                  | as_is                         | 1                      | 1                 | 1                |
    And the table "items_ancestors" should be:
      | ancestor_item_id | child_item_id |
      | 21               | 60            |
      | 50               | 112           |
      | 50               | 134           |
    And the table "groups" should be:
      | id                  | type                | name            |
      | 10                  | Club                | Club            |
      | 11                  | User                | jdoe            |
      | 5577006791947779410 | ContestParticipants | 50-participants |
    And the table "permissions_granted" should be:
      | group_id            | item_id | can_view | can_grant_view | can_watch | can_edit | is_owner | source_group_id     | ABS(TIMESTAMPDIFF(SECOND, latest_update_at, NOW())) < 3 |
      | 10                  | 50      | none     | none           | none      | none     | true     | 11                  | 0                                                       |
      | 11                  | 21      | content  | none           | none      | none     | false    | 11                  | 0                                                       |
      | 11                  | 50      | none     | none           | none      | none     | false    | 11                  | 0                                                       |
      | 11                  | 60      | none     | none           | none      | none     | true     | 11                  | 0                                                       |
      | 11                  | 70      | none     | none           | none      | none     | true     | 11                  | 0                                                       |
      | 11                  | 112     | solution | content        | answer    | all      | false    | 11                  | 0                                                       |
      | 11                  | 134     | none     | none           | none      | none     | true     | 11                  | 0                                                       |
      | 5577006791947779410 | 50      | content  | none           | none      | none     | false    | 5577006791947779410 | 1                                                       |
    And the table "permissions_generated" should be:
      | group_id            | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 10                  | 50      | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 10                  | 112     | info               | none                     | none                | none               | false              |
      | 10                  | 134     | solution           | solution                 | answer              | all                | false              |
      | 11                  | 21      | content            | none                     | none                | none               | false              |
      | 11                  | 50      | none               | none                     | none                | none               | false              |
      | 11                  | 60      | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 11                  | 70      | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 11                  | 112     | solution           | content                  | answer              | all                | false              |
      | 11                  | 134     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 5577006791947779410 | 50      | content            | none                     | none                | none               | false              |
      | 5577006791947779410 | 112     | info               | none                     | none                | none               | false              |
      | 5577006791947779410 | 134     | info               | none                     | none                | none               | false              |
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged, regardless of the row with item_id "50"
    And the table "results" at item_id "50" should be:
      | attempt_id | participant_id | item_id | score_computed |
      | 0          | 11             | 50      | 56.666668      |
    And the table "results_propagate" should be empty

  Scenario: Valid (with skill items)
    Given I am the user with id "11"
    And the database has the following table "items":
      | id  | default_language_tag | type    |
      | 112 | fr                   | Skill   |
      | 134 | fr                   | Chapter |
    And the database has the following table "items_strings":
      | language_tag | item_id |
      | sl           | 50      |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 11       | 112     | solution           | content                  | answer              | all                | false              |
      | 11       | 134     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
    And the database has the following table "permissions_granted":
      | group_id | item_id | can_view | can_grant_view | can_watch | can_edit | is_owner | source_group_id | latest_update_at    |
      | 11       | 112     | solution | content        | answer    | all      | false    | 11              | 2019-05-30 11:00:00 |
      | 11       | 134     | none     | none           | none      | none     | true     | 11              | 2019-05-30 11:00:00 |
    And the database table "results" also has the following rows:
      | attempt_id | participant_id | item_id | score_computed |
      | 0          | 11             | 112     | 50             |
      | 0          | 11             | 134     | 60             |
    When I send a PUT request to "/items/70" with the following body:
      """
      {
        "children": [
          {"item_id": "112", "order": 0, "category": "Discovery", "score_weight": 1},
          {"item_id": "134", "order": 1, "category": "Application", "score_weight": 2}
        ]
      }
      """
    Then the response should be "updated"
    And the table "items" should remain unchanged, regardless of the row with id "70"
    And the table "items" at id "70" should be:
      | id | type  | url                  | options   | default_language_tag | entry_frozen_teams | no_score | text_id | title_bar_visible | display_details_in_parent | uses_api | read_only | full_screen | children_layout | hints_allowed | fixed_ranks | validation_type | entry_min_admitted_members_ratio | entry_frozen_teams | entry_max_team_size | allows_multiple_attempts | duration | show_user_infos | prompt_to_join_group_by_code | participants_group_id |
      | 70 | Skill | http://someurl3.com/ | {"opt":3} | en                   | 0                  | 0        | null    | 0                 | 1                         | 0        | 1         | forceNo     | List            | 1             | 1           | One             | Half                             | 0                  | 10                  | 1                        | 01:20:30 | 1               | 1                            | 1234                  |
    And the table "items_strings" should remain unchanged
    And the table "items_items" should be:
      | parent_item_id | child_item_id | category    | score_weight | content_view_propagation | upper_view_levels_propagation | grant_view_propagation | watch_propagation | edit_propagation |
      | 21             | 60            | Undefined   | 1            | none                     | use_content_view_propagation  | 0                      | 0                 | 0                |
      | 50             | 21            | Undefined   | 1            | none                     | use_content_view_propagation  | 0                      | 0                 | 0                |
      | 70             | 112           | Discovery   | 1            | as_info                  | use_content_view_propagation  | 0                      | 0                 | 0                |
      | 70             | 134           | Application | 2            | as_info                  | as_is                         | 1                      | 1                 | 1                |
    And the table "items_ancestors" should be:
      | ancestor_item_id | child_item_id |
      | 21               | 60            |
      | 50               | 21            |
      | 50               | 60            |
      | 70               | 112           |
      | 70               | 134           |
    And the table "groups" should be:
      | id | type | name |
      | 10 | Club | Club |
      | 11 | User | jdoe |
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should be:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 10       | 21      | none               | none                     | none                | none               | false              |
      | 10       | 50      | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 10       | 60      | none               | none                     | none                | none               | false              |
      | 11       | 21      | content            | none                     | none                | none               | false              |
      | 11       | 50      | none               | none                     | none                | none               | false              |
      | 11       | 60      | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 11       | 70      | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 11       | 112     | solution           | content                  | answer              | all                | false              |
      | 11       | 134     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged, regardless of the row with item_id "70"
    And the table "results" at item_id "70" should be:
      | attempt_id | participant_id | item_id | score_computed |
      | 0          | 11             | 70      | 56.666668      |
    And the table "results_propagate" should be empty

  Scenario: Should set content_view_propagation to 'none' by default if can_grant_view = 'none' for the parent item
    Given I am the user with id "11"
    And the database has the following table "items":
      | id  | default_language_tag |
      | 112 | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 11       | 112     | solution           | content                  | answer              | all                | false              |
    And the database has the following table "permissions_granted":
      | group_id | item_id | can_view | can_grant_view | can_watch | can_edit | is_owner | source_group_id | latest_update_at    |
      | 11       | 112     | solution | content        | answer    | all      | false    | 11              | 2019-05-30 11:00:00 |
    When I send a PUT request to "/items/21" with the following body:
      """
      {
        "children": [
          {"item_id": "112", "order": 0}
        ]
      }
      """
    Then the response should be "updated"
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should be:
      | parent_item_id | child_item_id | category  | score_weight | content_view_propagation | upper_view_levels_propagation | grant_view_propagation | watch_propagation | edit_propagation |
      | 21             | 112           | Undefined | 1            | as_info                  | use_content_view_propagation  | 0                      | 0                 | 0                |
      | 50             | 21            | Undefined | 1            | none                     | use_content_view_propagation  | 0                      | 0                 | 0                |
    And the table "items_ancestors" should be:
      | ancestor_item_id | child_item_id |
      | 21               | 112           |
      | 50               | 21            |
      | 50               | 112           |
    And the table "groups" should be:
      | id | type | name |
      | 10 | Club | Club |
      | 11 | User | jdoe |
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should be:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 10       | 21      | none               | none                     | none                | none               | false              |
      | 10       | 50      | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 10       | 112     | none               | none                     | none                | none               | false              |
      | 11       | 21      | content            | none                     | none                | none               | false              |
      | 11       | 50      | none               | none                     | none                | none               | false              |
      | 11       | 60      | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 11       | 70      | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 11       | 112     | solution           | content                  | answer              | all                | false              |
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged, regardless of the rows with item_ids "21,50"
    And the table "results" at item_ids "21,50" should be:
      | attempt_id | participant_id | item_id | score_computed |
      | 0          | 11             | 21      | 0              |
      | 0          | 11             | 50      | 0              |
    And the table "results_propagate" should be empty

  Scenario: Valid without any fields
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
    """
    {
    }
    """
    Then the response should be "updated"
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: Valid with empty children array
    Given I am the user with id "11"
    When I send a PUT request to "/items/21" with the following body:
    """
    {
      "children": []
    }
    """
    Then the response should be "updated"
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should be:
      | parent_item_id | child_item_id |
      | 50             | 21            |
    And the table "items_ancestors" should be:
      | ancestor_item_id | child_item_id |
      | 50               | 21            |
    And the table "groups" should remain unchanged
    And the table "permissions_granted" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged, regardless of the rows with item_ids "21,50"
    And the table "results" at item_ids "21,50" should be:
      | attempt_id | participant_id | item_id | score_computed |
      | 0          | 11             | 21      | 0              |
      | 0          | 11             | 50      | 0              |
    And the table "results_propagate" should be empty

  Scenario: Keep existing contest participants group
    Given I am the user with id "11"
    When I send a PUT request to "/items/60" with the following body:
    """
    {
      "requires_explicit_entry": false,
      "duration": null
    }
    """
    Then the response should be "updated"
    And the table "items" should remain unchanged, regardless of the row with id "60"
    And the table "items" at id "60" should be:
      | id | duration | requires_explicit_entry | participants_group_id |
      | 60 | null     | false                   | 1234                  |
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "permissions_granted" should remain unchanged
    When I send a PUT request to "/items/60" with the following body:
    """
    {
      "requires_explicit_entry": true
    }
    """
    Then the response should be "updated"
    And the table "items" should remain unchanged, regardless of the row with id "60"
    And the table "items" at id "60" should be:
      | id | duration | requires_explicit_entry | participants_group_id |
      | 60 | null     | true                    | 1234                  |
    And the table "groups" should remain unchanged

  Scenario: Recomputes results if no_score is given
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
    """
    {
      "no_score": false
    }
    """
    Then the response should be "updated"
    And the table "items" should remain unchanged, regardless of the row with id "50"
    And the table "items" at id "50" should be:
      | id | no_score |
      | 50 | false    |
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "permissions_granted" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged, regardless of the row with item_id "50"
    And the table "results" at item_id "50" should be:
      | attempt_id | participant_id | item_id | score_computed |
      | 0          | 11             | 50      | 0              |
    And the table "results_propagate" should be empty

  Scenario: Recomputes results if validation_type is given
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
    """
    {
      "validation_type": "All"
    }
    """
    Then the response should be "updated"
    And the table "items" should remain unchanged, regardless of the row with id "50"
    And the table "items" at id "50" should be:
      | id | validation_type |
      | 50 | All             |
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "permissions_granted" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged, regardless of the row with item_id "50"
    And the table "results" at item_id "50" should be:
      | attempt_id | participant_id | item_id | score_computed |
      | 0          | 11             | 50      | 0              |
    And the table "results_propagate" should be empty

  Scenario Outline: Sets default values of items_items.content_view_propagation/upper_view_levels_propagation/grant_view_propagation correctly for each can_grant_view
    Given I am the user with id "11"
    And the database has the following table "items":
      | id  | default_language_tag |
      | 112 | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated | can_grant_view_generated |
      | 11       | 112     | info               | <can_grant_view>         |
    And the database has the following table "permissions_granted":
      | group_id | item_id | can_view | can_grant_view   | source_group_id |
      | 11       | 112     | info     | <can_grant_view> | 11              |
    When I send a PUT request to "/items/21" with the following body:
      """
      {
        "children": [{"item_id": 112, "order": 1}]
      }
      """
    Then the response should be "updated"
    And the table "items_items" should be:
      | parent_item_id | child_item_id | child_order | content_view_propagation   | upper_view_levels_propagation   | grant_view_propagation   |
      | 21             | 112           | 1           | <content_view_propagation> | <upper_view_levels_propagation> | <grant_view_propagation> |
      | 50             | 21            | 0           | none                       | use_content_view_propagation    | false                    |
    Examples:
      | can_grant_view           | content_view_propagation | upper_view_levels_propagation | grant_view_propagation |
      | solution_with_grant      | as_info                  | as_is                         | true                   |
      | solution                 | as_info                  | as_is                         | false                  |
      | content_with_descendants | as_info                  | as_content_with_descendants   | false                  |
      | content                  | as_info                  | use_content_view_propagation  | false                  |
      | none                     | none                     | use_content_view_propagation  | false                  |

  Scenario Outline: Sets default values of items_items.watch_propagation/edit_propagation correctly
    Given I am the user with id "11"
    And the database has the following table "items":
      | id  | default_language_tag |
      | 112 | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated | <parent_permission_column> |
      | 11       | 112     | info               | <parent_permission_value>  |
    When I send a PUT request to "/items/21" with the following body:
      """
      {
        "children": [{"item_id": 112, "order": 1}]
      }
      """
    Then the response should be "updated"
    And the table "items_items" at parent_item_id "21" should be:
      | parent_item_id | child_item_id | child_order | <propagation_column> |
      | 21             | 112           | 1           | <propagation_value>  |
    Examples:
      | parent_permission_column | parent_permission_value | propagation_column | propagation_value |
      | can_watch_generated      | answer_with_grant       | watch_propagation  | true              |
      | can_watch_generated      | answer                  | watch_propagation  | false             |
      | can_watch_generated      | result                  | watch_propagation  | false             |
      | can_watch_generated      | none                    | watch_propagation  | false             |
      | can_edit_generated       | all_with_grant          | edit_propagation   | true              |
      | can_edit_generated       | all                     | edit_propagation   | false             |
      | can_edit_generated       | children                | edit_propagation   | false             |
      | can_edit_generated       | none                    | edit_propagation   | false             |

  Scenario Outline: Sets items_items.content_view_propagation/upper_view_levels_propagation/grant_view_propagation correctly
    Given I am the user with id "11"
    And the database has the following table "items":
      | id  | default_language_tag |
      | 112 | fr                   |
    And the database table "permissions_generated" also has the following row:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 11       | 112     | info               | <can_grant_view>         | none                | none               | 0                  |
    When I send a PUT request to "/items/21" with the following body:
      """
      {
        "children": [{
          "item_id": 112,
          "order": 1,
          "<field_name>": {{"<value>" != "true" && "<value>" != "false" ? "\"<value>\"" : <value>}}
        }]
      }
      """
    Then the response should be "updated"
    And the table "items_items" at parent_item_id "21" should be:
      | parent_item_id | child_item_id | child_order | <field_name> |
      | 21             | 112           | 1           | <value>      |
    Examples:
      | can_grant_view           | field_name                    | value                        |
      | solution_with_grant      | content_view_propagation      | as_content                   |
      | solution                 | content_view_propagation      | as_content                   |
      | content_with_descendants | content_view_propagation      | as_content                   |
      | content                  | content_view_propagation      | as_content                   |
      | solution_with_grant      | content_view_propagation      | as_info                      |
      | solution                 | content_view_propagation      | as_info                      |
      | content_with_descendants | content_view_propagation      | as_info                      |
      | content                  | content_view_propagation      | as_info                      |
      | enter                    | content_view_propagation      | as_info                      |
      | solution_with_grant      | content_view_propagation      | none                         |
      | solution                 | content_view_propagation      | none                         |
      | content_with_descendants | content_view_propagation      | none                         |
      | content                  | content_view_propagation      | none                         |
      | enter                    | content_view_propagation      | none                         |
      | none                     | content_view_propagation      | none                         |
      | solution_with_grant      | upper_view_levels_propagation | as_is                        |
      | solution                 | upper_view_levels_propagation | as_is                        |
      | solution_with_grant      | upper_view_levels_propagation | as_content_with_descendants  |
      | solution                 | upper_view_levels_propagation | as_content_with_descendants  |
      | content_with_descendants | upper_view_levels_propagation | as_content_with_descendants  |
      | content                  | upper_view_levels_propagation | use_content_view_propagation |
      | enter                    | upper_view_levels_propagation | use_content_view_propagation |
      | none                     | upper_view_levels_propagation | use_content_view_propagation |
      | solution_with_grant      | grant_view_propagation        | true                         |
      | solution_with_grant      | grant_view_propagation        | false                        |
      | solution                 | grant_view_propagation        | false                        |
      | content_with_descendants | grant_view_propagation        | false                        |
      | content                  | grant_view_propagation        | false                        |
      | enter                    | grant_view_propagation        | false                        |
      | none                     | grant_view_propagation        | false                        |

  Scenario Outline: Sets items_items.watch_propagation/edit_propagation correctly
    Given I am the user with id "11"
    And the database has the following table "items":
      | id  | default_language_tag |
      | 112 | fr                   |
    And the database table "permissions_generated" also has the following row:
      | group_id | item_id | can_view_generated | <parent_permission_column> |
      | 11       | 112     | info               | <parent_permission_value>  |
    When I send a PUT request to "/items/21" with the following body:
      """
      {
        "children": [{
          "item_id": 112,
          "order": 1,
          "<field_name>": {{"<value>" != "true" && "<value>" != "false" ? "\"<value>\"" : <value>}}
        }]
      }
      """
    Then the response should be "updated"
    And the table "items_items" at parent_item_id "21" should be:
      | parent_item_id | child_item_id | child_order | <field_name> |
      | 21             | 112           | 1           | <value>      |
    Examples:
      | parent_permission_column | parent_permission_value | field_name        | value |
      | can_watch_generated      | answer_with_grant       | watch_propagation | true  |
      | can_watch_generated      | answer_with_grant       | watch_propagation | false |
      | can_watch_generated      | answer                  | watch_propagation | false |
      | can_watch_generated      | result                  | watch_propagation | false |
      | can_watch_generated      | none                    | watch_propagation | false |
      | can_edit_generated       | all_with_grant          | edit_propagation  | true  |
      | can_edit_generated       | all_with_grant          | edit_propagation  | false |
      | can_edit_generated       | all                     | edit_propagation  | false |
      | can_edit_generated       | children                | edit_propagation  | false |
      | can_edit_generated       | none                    | edit_propagation  | false |

  Scenario: Sets items_items.request_help_propagation correctly
    Given I am the user with id "11"
    And the database has the following table "items":
      | id  | default_language_tag |
      | 112 | fr                   |
      | 113 | fr                   |
      | 114 | fr                   |
      | 115 | fr                   |
      | 116 | fr                   |
      | 117 | fr                   |
      | 118 | fr                   |
      | 119 | fr                   |
      | 120 | fr                   |
      | 121 | fr                   |
      | 122 | fr                   |
      | 123 | fr                   |
      | 124 | fr                   |
      | 125 | fr                   |
      | 126 | fr                   |
      | 127 | fr                   |
    And the database table "permissions_generated" also has the following row:
      | group_id | item_id | can_grant_view_generated | can_view_generated |
      | 11       | 112     | content                  | content            |
      | 11       | 113     | content                  | content            |
      | 11       | 114     | content                  | content            |
      | 11       | 115     | content_with_descendants | content            |
      | 11       | 116     | content_with_descendants | content            |
      | 11       | 117     | content_with_descendants | content            |
      | 11       | 118     | solution                 | content            |
      | 11       | 119     | solution                 | content            |
      | 11       | 120     | solution                 | content            |
      | 11       | 121     | solution_with_grant      | content            |
      | 11       | 122     | solution_with_grant      | content            |
      | 11       | 123     | solution_with_grant      | content            |
      | 11       | 124     | none                     | content            |
      | 11       | 125     | none                     | content            |
      | 11       | 126     | enter                    | content            |
      | 11       | 127     | enter                    | content            |
    When I send a PUT request to "/items/21" with the following body:
      """
      {
        "children": [{
          "item_id": 112,
          "order": 1,
          "request_help_propagation": true
        },{
          "item_id": 113,
          "order": 2,
          "request_help_propagation": false
        },{
          "item_id": 114,
          "order": 3
        },{
          "item_id": 115,
          "order": 4,
          "request_help_propagation": true
        },{
          "item_id": 116,
          "order": 5,
          "request_help_propagation": false
        },{
          "item_id": 117,
          "order": 6
        },{
          "item_id": 118,
          "order": 7,
          "request_help_propagation": true
        },{
          "item_id": 119,
          "order": 8,
          "request_help_propagation": false
        },{
          "item_id": 120,
          "order": 9
        },{
          "item_id": 121,
          "order": 10,
          "request_help_propagation": true
        },{
          "item_id": 122,
          "order": 11,
          "request_help_propagation": false
        },{
          "item_id": 123,
          "order": 12
        },{
          "item_id": 124,
          "order": 13,
          "request_help_propagation": false
        },{
          "item_id": 125,
          "order": 14
        },{
          "item_id": 126,
          "order": 15,
          "request_help_propagation": false
        },{
          "item_id": 127,
          "order": 16
        }]
      }
      """
    Then the response should be "updated"
    And the table "items_items" at parent_item_id "21" should be:
      | parent_item_id | child_item_id | child_order | request_help_propagation |
      | 21             | 112           | 1           | true                     |
      | 21             | 113           | 2           | false                    |
      | 21             | 114           | 3           | true                     |
      | 21             | 115           | 4           | true                     |
      | 21             | 116           | 5           | false                    |
      | 21             | 117           | 6           | true                     |
      | 21             | 118           | 7           | true                     |
      | 21             | 119           | 8           | false                    |
      | 21             | 120           | 9           | true                     |
      | 21             | 121           | 10          | true                     |
      | 21             | 122           | 11          | false                    |
      | 21             | 123           | 12          | true                     |
      | 21             | 124           | 13          | false                    |
      | 21             | 125           | 14          | false                    |
      | 21             | 126           | 15          | false                    |
      | 21             | 127           | 16          | false                    |

  Scenario Outline: Allows setting items_items.content_view_propagation/upper_view_levels_propagation/grant_view_propagation/watch_propagation/edit_propagation to the same of a lower value
    Given I am the user with id "11"
    And the database table "items" also has the following rows:
      | id  | default_language_tag |
      | 112 | fr                   |
    And the database table "items_items" also has the following rows:
      | parent_item_id | child_item_id | child_order | <field_name> |
      | 21             | 112           | 1           | <old_value>  |
    And the database table "permissions_generated" also has the following row:
      | group_id | item_id | can_view_generated |
      | 11       | 112     | info               |
    When I send a PUT request to "/items/21" with the following body:
      """
      {
        "children": [{
          "item_id": 112,
          "order": 1,
          "<field_name>": {{"<value>" != "true" && "<value>" != "false" ? "\"<value>\"" : <value>}}
        }]
      }
      """
    Then the response should be "updated"
    And the table "items_items" at parent_item_id "21" should be:
      | parent_item_id | child_item_id | child_order | <field_name> |
      | 21             | 112           | 1           | <value>      |
    Examples:
      | field_name                    | old_value                   | value                       |
      | content_view_propagation      | as_content                  | as_content                  |
      | content_view_propagation      | as_content                  | as_info                     |
      | content_view_propagation      | as_info                     | as_info                     |
      | upper_view_levels_propagation | as_is                       | as_is                       |
      | upper_view_levels_propagation | as_is                       | as_content_with_descendants |
      | upper_view_levels_propagation | as_content_with_descendants | as_content_with_descendants |
      | grant_view_propagation        | true                        | true                        |
      | watch_propagation             | true                        | true                        |

  Scenario: Allows keeping old values in items_items
    Given I am the user with id "11"
    And the database table "items" also has the following rows:
      | id  | default_language_tag |
      | 112 | fr                   |
    And the database table "items_items" also has the following rows:
      | parent_item_id | child_item_id | child_order | category  | score_weight | content_view_propagation | upper_view_levels_propagation | grant_view_propagation | watch_propagation | edit_propagation |
      | 21             | 112           | 1           | Challenge | 2            | as_content               | as_is                         | true                   | true              | true             |
    When I send a PUT request to "/items/21" with the following body:
      """
      {
        "children": [{
          "item_id": 112,
          "order": 1
        }]
      }
      """
    Then the response should be "updated"
    And the table "items_items" at parent_item_id "21" should be:
      | parent_item_id | child_item_id | child_order | category  | score_weight | content_view_propagation | upper_view_levels_propagation | grant_view_propagation | watch_propagation | edit_propagation |
      | 21             | 112           | 1           | Challenge | 2            | as_content               | as_is                         | true                   | true              | true             |
