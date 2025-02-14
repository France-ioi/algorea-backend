Feature: Create item
  Background:
    Given the database has the following table "groups":
      | id | name    | type    | root_activity_id | root_skill_id |
      | 10 | Friends | Friends | null             | null          |
    And the database has the following user:
      | group_id | login |
      | 11       | jdoe  |
    And the database has the following table "items":
      | id | entry_frozen_teams | no_score | default_language_tag |
      | 21 | true               | false    | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated | can_edit_generated |
      | 11       | 21      | solution           | children           |
    And the database has the following table "permissions_granted":
      | group_id | item_id | can_view | can_edit | source_group_id | latest_update_at    |
      | 11       | 21      | solution | children | 11              | 2019-05-30 11:00:00 |
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 11             |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id |
      | 0          | 11             | 21      |
    And the database has the following table "languages":
      | tag |
      | sl  |

  Scenario: Valid
    Given I am the user with id "11"
    When I send a POST request to "/items" with the following body:
      """
      {
        "type": "Task",
        "language_tag": "sl",
        "title": "my title",
        "image_url":"http://bit.ly/1234",
        "subtitle": "hard task",
        "description": "the goal of this task is ...",
        "parent": {"item_id": "21"}
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "success": true,
        "message": "created",
        "data": { "id": "5577006791947779410" }
      }
      """
    And the table "items" at id "5577006791947779410" should be:
      | id                  | type   | url  | options | default_language_tag | entry_frozen_teams | no_score | text_id | title_bar_visible | display_details_in_parent | uses_api | read_only | full_screen | children_layout | hints_allowed | fixed_ranks | validation_type | entry_min_admitted_members_ratio | entry_max_team_size | allows_multiple_attempts | duration | requires_explicit_entry | show_user_infos | no_score | prompt_to_join_group_by_code | entering_time_min   | entering_time_max   | participants_group_id |
      | 5577006791947779410 | Task   | null | null    | sl                   | 0                  | 0        | null    | 1                 | 0                         | 1        | 0         | default     | List            | 0             | 0           | All             | None                             | 0                   | 0                        | null     | 0                       | 0               | 0        | 0                            | 1000-01-01 00:00:00 | 9999-12-31 23:59:59 | null                  |
    And the table "items_strings" should be:
      | item_id             | language_tag | title    | image_url          | subtitle  | description                  |
      | 5577006791947779410 | sl           | my title | http://bit.ly/1234 | hard task | the goal of this task is ... |
    And the table "items_items" should be:
      | parent_item_id | child_item_id       | child_order | content_view_propagation | upper_view_levels_propagation | grant_view_propagation | watch_propagation | edit_propagation | request_help_propagation | category  | score_weight |
      | 21             | 5577006791947779410 | 1           | as_info                  | as_is                         | 1                      | 1                 | 1                | 1                        | Undefined | 1            |
    And the table "items_ancestors" should be:
      | ancestor_item_id | child_item_id       |
      | 21               | 5577006791947779410 |
    And the table "permissions_granted" at group_id "11" should be:
      | group_id | item_id             | source_group_id | origin           | can_view | can_grant_view | can_watch | can_edit | is_owner | ABS(TIMESTAMPDIFF(SECOND, latest_update_at, NOW())) < 3 |
      | 11       | 21                  | 11              | group_membership | solution | none           | none      | children | 0        | 0                                                       |
      | 11       | 5577006791947779410 | 11              | self             | none     | none           | none      | none     | 1        | 1                                                       |
    And the table "permissions_generated" should be:
      | group_id | item_id             | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 11       | 21                  | solution           | none                     | none                | children           | 0                  |
      | 11       | 5577006791947779410 | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | 1                  |
    And the table "groups" should remain unchanged
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Valid when as_root_of_group_id is given, but parent_item_id is not given (not a skill)
    Given I am the user with id "11"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 10       | 11         | memberships_and_group |
    When I send a POST request to "/items" with the following body:
      """
      {
        "type": "Task",
        "language_tag": "sl",
        "title": "my title",
        "image_url":"http://bit.ly/1234",
        "subtitle": "hard task",
        "description": "the goal of this task is ...",
        "as_root_of_group_id": "10"
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "success": true,
        "message": "created",
        "data": { "id": "5577006791947779410" }
      }
      """
    And the table "items" at id "5577006791947779410" should be:
      | id                  | type   | url  | options | default_language_tag | entry_frozen_teams | no_score | text_id | title_bar_visible | display_details_in_parent | uses_api | read_only | full_screen | children_layout  | hints_allowed | fixed_ranks | validation_type | entry_min_admitted_members_ratio | entry_max_team_size | allows_multiple_attempts | duration | requires_explicit_entry | show_user_infos | no_score | prompt_to_join_group_by_code | entering_time_min   | entering_time_max   | participants_group_id |
      | 5577006791947779410 | Task   | null | null    | sl                   | 0                  | 0        | null    | 1                 | 0                         | 1        | 0         | default     | List             | 0             | 0           | All             | None                             | 0                   | 0                        | null     | 0                       | 0               | 0        | 0                            | 1000-01-01 00:00:00 | 9999-12-31 23:59:59 | null                  |
    And the table "items_strings" should be:
      | item_id             | language_tag | title    | image_url          | subtitle  | description                  |
      | 5577006791947779410 | sl           | my title | http://bit.ly/1234 | hard task | the goal of this task is ... |
    And the table "items_items" should be empty
    And the table "items_ancestors" should be empty
    And the table "permissions_granted" at group_id "11" should be:
      | group_id | item_id             | source_group_id | origin           | can_view | can_grant_view | can_watch | can_edit | is_owner | ABS(TIMESTAMPDIFF(SECOND, latest_update_at, NOW())) < 3 |
      | 11       | 21                  | 11              | group_membership | solution | none           | none      | children | 0        | 0                                                       |
      | 11       | 5577006791947779410 | 11              | self             | none     | none           | none      | none     | 1        | 1                                                       |
    And the table "permissions_generated" should be:
      | group_id | item_id             | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 11       | 21                  | solution           | none                     | none                | children           | 0                  |
      | 11       | 5577006791947779410 | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | 1                  |
    And the table "groups" should be:
      | id | name    | type    | root_activity_id    | root_skill_id |
      | 10 | Friends | Friends | 5577006791947779410 | null          |
      | 11 | jdoe    | User    | null                | null          |
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Valid when as_root_of_group_id is given, but parent_item_id is not given (skill with children)
    Given I am the user with id "11"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 10       | 11         | memberships_and_group |
    And the database table "items" also has the following rows:
      | id | default_language_tag |
      | 12 | fr                   |
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated       | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 10       | 21      | none                     | content                  | none                | none               | 0                  |
      | 11       | 12      | content_with_descendants | solution                 | answer              | all                | 0                  |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view                 | can_grant_view      | can_watch         | can_edit       | is_owner | source_group_id | latest_update_at    |
      | 11       | 12      | content_with_descendants | solution            | answer            | all            | 0        | 11              | 2019-05-30 11:00:00 |
    When I send a POST request to "/items" with the following body:
      """
      {
        "type": "Skill",
        "language_tag": "sl",
        "title": "my title",
        "image_url":"http://bit.ly/1234",
        "subtitle": "hard task",
        "description": "the goal of this task is ...",
        "as_root_of_group_id": "10",
        "children": [{"item_id": "12", "order": 0}]
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "success": true,
        "message": "created",
        "data": { "id": "5577006791947779410" }
      }
      """
    And the table "items" at id "5577006791947779410" should be:
      | id                  | type  | url  | options | default_language_tag | entry_frozen_teams | no_score | text_id | title_bar_visible | display_details_in_parent | uses_api | read_only | full_screen | children_layout | hints_allowed | fixed_ranks | validation_type | entry_min_admitted_members_ratio | entry_max_team_size | allows_multiple_attempts | duration | requires_explicit_entry | show_user_infos | no_score | prompt_to_join_group_by_code | entering_time_min   | entering_time_max   | participants_group_id |
      | 5577006791947779410 | Skill | null | null    | sl                   | 0                  | 0        | null    | 1                 | 0                         | 1        | 0         | default     | List            | 0             | 0           | All             | None                             | 0                   | 0                        | null     | 0                       | 0               | 0        | 0                            | 1000-01-01 00:00:00 | 9999-12-31 23:59:59 | null                  |
    And the table "items_strings" should be:
      | item_id             | language_tag | title    | image_url          | subtitle  | description                  |
      | 5577006791947779410 | sl           | my title | http://bit.ly/1234 | hard task | the goal of this task is ... |
    And the table "items_items" should be:
      | parent_item_id      | child_item_id | child_order | content_view_propagation | upper_view_levels_propagation | grant_view_propagation | watch_propagation | edit_propagation | request_help_propagation | category  | score_weight |
      | 5577006791947779410 | 12            | 0           | as_info                  | as_is                         | 0                      | 0                 | 0                | 1                        | Undefined | 1            |
    And the table "items_ancestors" should be:
      | ancestor_item_id    | child_item_id       |
      | 5577006791947779410 | 12                  |
    And the table "permissions_granted" at group_id "11" should be:
      | group_id | item_id             | source_group_id | origin           | can_view                 | can_grant_view | can_watch | can_edit | is_owner | ABS(TIMESTAMPDIFF(SECOND, latest_update_at, NOW())) < 3 |
      | 11       | 12                  | 11              | group_membership | content_with_descendants | solution       | answer    | all      | 0        | 0                                                       |
      | 11       | 21                  | 11              | group_membership | solution                 | none           | none      | children | 0        | 0                                                       |
      | 11       | 5577006791947779410 | 11              | self             | none                     | none           | none      | none     | 1        | 1                                                       |
    And the table "permissions_generated" should be:
      | group_id | item_id             | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 10       | 21                  | none               | content                  | none                | none               | 0                  |
      | 11       | 12                  | solution           | solution                 | answer              | all                | 0                  |
      | 11       | 21                  | solution           | none                     | none                | children           | 0                  |
      | 11       | 5577006791947779410 | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | 1                  |
    And the table "groups" should be:
      | id | name    | type    | root_activity_id | root_skill_id       |
      | 10 | Friends | Friends | null             | 5577006791947779410 |
      | 11 | jdoe    | User    | null             | null                |
    And the table "attempts" should remain unchanged
    And the table "results" should remain unchanged

  Scenario: Set can_request_help for children
    Given I am the user with id "11"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 10       | 11         | memberships_and_group |
    And the database table "items" also has the following rows:
      | id   | default_language_tag |
      | 1001 | fr                   |
      | 1002 | fr                   |
      | 1003 | fr                   |
      | 1004 | fr                   |
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated       | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 11       | 1001    | content_with_descendants | content                  | answer              | none               | 0                  |
      | 11       | 1002    | content_with_descendants | enter                    | answer              | none               | 0                  |
      | 11       | 1003    | content_with_descendants | content                  | answer              | none               | 0                  |
      | 11       | 1004    | content_with_descendants | enter                    | answer              | none               | 0                  |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view                 | can_grant_view | can_watch | can_edit | is_owner | source_group_id | latest_update_at    |
      | 11       | 1001    | content_with_descendants | content        | answer    | none     | 0        | 11              | 2019-05-30 11:00:00 |
      | 11       | 1002    | content_with_descendants | enter          | answer    | none     | 0        | 11              | 2019-05-30 11:00:00 |
      | 11       | 1003    | content_with_descendants | content        | answer    | none     | 0        | 11              | 2019-05-30 11:00:00 |
      | 11       | 1004    | content_with_descendants | enter          | answer    | none     | 0        | 11              | 2019-05-30 11:00:00 |
    When I send a POST request to "/items" with the following body:
      """
      {
        "type": "Skill",
        "language_tag": "sl",
        "title": "my title",
        "image_url":"http://bit.ly/1234",
        "subtitle": "hard task",
        "description": "the goal of this task is ...",
        "as_root_of_group_id": "10",
        "children": [
          {"item_id": "1001", "order": 0, "request_help_propagation": true},
          {"item_id": "1002", "order": 1, "request_help_propagation": false},
          {"item_id": "1003", "order": 2},
          {"item_id": "1004", "order": 3}
        ]
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "success": true,
        "message": "created",
        "data": { "id": "5577006791947779410" }
      }
      """
    And the table "items_items" should be:
      | parent_item_id      | child_item_id | request_help_propagation | # comment
      | 5577006791947779410 | 1001          | 1                        | # set to 1
      | 5577006791947779410 | 1002          | 0                        | # set to 0
      | 5577006791947779410 | 1003          | 1                        | # defaults to 1
      | 5577006791947779410 | 1004          | 0                        | # defaults to 0 because current-user has can_grant_view<content on child

  Scenario Outline: Valid (all the fields are set)
    Given I am the user with id "11"
    And the database table "items" also has the following rows:
      | id | default_language_tag |
      | 12 | fr                   |
      | 34 | fr                   |
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated       | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 10       | 21      | none                     | content                  | none                | none               | 0                  |
      | 11       | 12      | content_with_descendants | solution                 | answer              | all                | 0                  |
      | 11       | 34      | solution                 | solution_with_grant      | answer_with_grant   | all_with_grant     | 0                  |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view                 | can_grant_view      | can_watch         | can_edit       | is_owner | source_group_id | latest_update_at    |
      | 11       | 12      | content_with_descendants | solution            | answer            | all            | 0        | 11              | 2019-05-30 11:00:00 |
      | 11       | 34      | solution                 | solution_with_grant | answer_with_grant | all_with_grant | 0        | 11              | 2019-05-30 11:00:00 |
    And the database table "results" also has the following rows:
      | attempt_id | participant_id | item_id |
      | 0          | 11             | 12      |
    When I send a POST request to "/items" with the following body:
      """
      {
        "type": "Chapter",
        "url": "http://myurl.com/",
        "options": "{\"opt1\":\"value\"}",
        "text_id": "Tasknumber1",
        "title_bar_visible": true,
        "display_details_in_parent": true,
        "uses_api": true,
        "read_only": true,
        "full_screen": "forceYes",
        "children_layout": "Grid",
        "hints_allowed": true,
        "fixed_ranks": true,
        "validation_type": "AllButOne",
        "entry_min_admitted_members_ratio": "All",
        "entering_time_min": "2007-01-01T01:02:03Z",
        "entering_time_max": "3007-01-01T01:02:03Z",
        "entry_frozen_teams": false,
        "entry_max_team_size": 2345,
        "allows_multiple_attempts": true,
        "entry_participant_type": "Team",
        "duration": "01:02:03",
        "requires_explicit_entry": true,
        "show_user_infos": true,
        "no_score": true,
        "prompt_to_join_group_by_code": true,
        "language_tag": "sl",
        "title": "my title",
        "image_url":"http://bit.ly/1234",
        "subtitle": "hard task",
        "description": "the goal of this task is ...",
        "parent": {
          "item_id": "21",
          "category": "Challenge",
          "score_weight": 3,
          "content_view_propagation": "as_content",
          "upper_view_levels_propagation": "use_content_view_propagation",
          "grant_view_propagation": <grant_view_propagation>,
          "watch_propagation": <watch_propagation>,
          "edit_propagation": <edit_propagation>,
          "request_help_propagation": <request_help_propagation>
        },
        "children": [
          {"item_id": "12", "order": 0},
          {"item_id": "34", "order": 1, "category": "Application", "score_weight": 2}
        ]
      }
      """
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "success": true,
        "message": "created",
        "data": { "id": "5577006791947779410" }
      }
      """
    And the table "items" at id "5577006791947779410" should be:
      | id                  | type    | url               | options          | default_language_tag | entry_frozen_teams | no_score | text_id     | title_bar_visible | display_details_in_parent | uses_api | read_only | full_screen | children_layout | hints_allowed | fixed_ranks | validation_type | entry_min_admitted_members_ratio | entry_max_team_size | allows_multiple_attempts | entry_participant_type | duration | requires_explicit_entry | show_user_infos | no_score | prompt_to_join_group_by_code | entering_time_min   | entering_time_max   | participants_group_id |
      | 5577006791947779410 | Chapter | http://myurl.com/ | {"opt1":"value"} | sl                   | 0                  | 1        | Tasknumber1 | 1                 | 1                         | 1        | 1         | forceYes    | Grid            | 1             | 1           | AllButOne       | All                              | 2345                | 1                        | Team                   | 01:02:03 | 1                       | 1               | 1        | 1                            | 2007-01-01 01:02:03 | 3007-01-01 01:02:03 | 8674665223082153551   |
    And the table "items_strings" should be:
      | item_id             | language_tag | title    | image_url          | subtitle  | description                  |
      | 5577006791947779410 | sl           | my title | http://bit.ly/1234 | hard task | the goal of this task is ... |
    And the table "items_items" should be:
      | parent_item_id      | child_item_id       | child_order | content_view_propagation | upper_view_levels_propagation | grant_view_propagation   | watch_propagation   | edit_propagation   | request_help_propagation   | category    | score_weight |
      | 21                  | 5577006791947779410 | 1           | as_content               | use_content_view_propagation  | <grant_view_propagation> | <watch_propagation> | <edit_propagation> | <request_help_propagation> | Challenge   | 3            |
      | 5577006791947779410 | 12                  | 0           | as_info                  | as_is                         | 0                        | 0                   | 0                  | 1                          | Undefined   | 1            |
      | 5577006791947779410 | 34                  | 1           | as_info                  | as_is                         | 1                        | 1                   | 1                  | 1                          | Application | 2            |
    And the table "items_ancestors" should be:
      | ancestor_item_id    | child_item_id       |
      | 21                  | 12                  |
      | 21                  | 34                  |
      | 21                  | 5577006791947779410 |
      | 5577006791947779410 | 12                  |
      | 5577006791947779410 | 34                  |
    And the table "groups" should be:
      | id                  | type                | name                             |
      | 10                  | Friends             | Friends                          |
      | 11                  | User                | jdoe                             |
      | 8674665223082153551 | ContestParticipants | 5577006791947779410-participants |
    And the table "permissions_granted" should be:
      | group_id            | item_id             | source_group_id     | origin           | can_view                 | can_grant_view      | can_watch         | can_edit       | is_owner | ABS(TIMESTAMPDIFF(SECOND, latest_update_at, NOW())) < 3 |
      | 11                  | 12                  | 11                  | group_membership | content_with_descendants | solution            | answer            | all            | 0        | 0                                                       |
      | 11                  | 21                  | 11                  | group_membership | solution                 | none                | none              | children       | 0        | 0                                                       |
      | 11                  | 34                  | 11                  | group_membership | solution                 | solution_with_grant | answer_with_grant | all_with_grant | 0        | 0                                                       |
      | 11                  | 5577006791947779410 | 11                  | self             | none                     | none                | none              | none           | 1        | 1                                                       |
      | 8674665223082153551 | 5577006791947779410 | 8674665223082153551 | group_membership | content                  | none                | none              | none           | 0        | 1                                                       |
    And the table "permissions_generated" should be:
      | group_id            | item_id             | can_view_generated | can_grant_view_generated                          | can_watch_generated | can_edit_generated | is_owner_generated |
      | 10                  | 12                  | none               | none                                              | none                | none               | 0                  |
      | 10                  | 21                  | none               | content                                           | none                | none               | 0                  |
      | 10                  | 34                  | none               | {{<grant_view_propagation> ? "content" : "none"}} | none                | none               | 0                  |
      | 10                  | 5577006791947779410 | none               | {{<grant_view_propagation> ? "content" : "none"}} | none                | none               | 0                  |
      | 11                  | 12                  | solution           | solution                                          | answer              | all                | 0                  |
      | 11                  | 21                  | solution           | none                                              | none                | children           | 0                  |
      | 11                  | 34                  | solution           | solution_with_grant                               | answer_with_grant   | all_with_grant     | 0                  |
      | 11                  | 5577006791947779410 | solution           | solution_with_grant                               | answer_with_grant   | all_with_grant     | 1                  |
      | 8674665223082153551 | 12                  | info               | none                                              | none                | none               | 0                  |
      | 8674665223082153551 | 34                  | info               | none                                              | none                | none               | 0                  |
      | 8674665223082153551 | 5577006791947779410 | content            | none                                              | none                | none               | 0                  |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id |
      | 0          | 11             | 12      |
      | 0          | 11             | 21      |
    And the table "results_propagate" should be empty
  Examples:
    | grant_view_propagation | watch_propagation | edit_propagation | request_help_propagation |
    | true                   | false             | true             | true                     |
    | false                  | true              | true             | false                    |
    | false                  | false             | false            | false                    |

  Scenario: Valid when type=Skill
    Given I am the user with id "11"
    And the database table "items" also has the following rows:
      | id | default_language_tag | type    |
      | 12 | fr                   | Skill   |
      | 34 | fr                   | Chapter |
      | 50 | fr                   | Skill   |
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated       | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 10       | 21      | none                     | content                  | none                | none               | 0                  |
      | 11       | 12      | content_with_descendants | solution                 | answer              | all                | 0                  |
      | 11       | 34      | solution                 | solution_with_grant      | answer_with_grant   | all_with_grant     | 0                  |
      | 11       | 50      | solution                 | solution_with_grant      | answer_with_grant   | all_with_grant     | 0                  |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view                 | can_grant_view      | can_watch         | can_edit       | is_owner | source_group_id | latest_update_at    |
      | 11       | 12      | content_with_descendants | solution            | answer            | all            | 0        | 11              | 2019-05-30 11:00:00 |
      | 11       | 34      | solution                 | solution_with_grant | answer_with_grant | all_with_grant | 0        | 11              | 2019-05-30 11:00:00 |
      | 11       | 50      | solution                 | solution_with_grant | answer_with_grant | all_with_grant | 0        | 11              | 2019-05-30 11:00:00 |
    When I send a POST request to "/items" with the following body:
    """
    {
      "type": "Skill",
      "language_tag": "sl",
      "title": "my skill",
      "parent": {"item_id": "50"},
      "children": [
        {"item_id": "12", "order": 0, "category": "Application", "score_weight": 2},
        {"item_id": "34", "order": 1, "category": "Application", "score_weight": 2}
      ]
    }
    """
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created",
      "data": { "id": "5577006791947779410" }
    }
    """
    And the table "items" at id "5577006791947779410" should be:
      | id                  | type  | url  | options | default_language_tag | entry_frozen_teams | no_score | text_id | title_bar_visible | display_details_in_parent | uses_api | read_only | full_screen | children_layout | hints_allowed | fixed_ranks | validation_type | entry_min_admitted_members_ratio | entry_max_team_size | allows_multiple_attempts | duration | requires_explicit_entry | show_user_infos | no_score | prompt_to_join_group_by_code | participants_group_id |
      | 5577006791947779410 | Skill | null | null    | sl                   | 0                  | 0        | null    | 1                 | 0                         | 1        | 0         | default     | List            | 0             | 0           | All             | None                             | 0                   | 0                        | null     | 0                       | 0               | 0        | 0                            | null                  |
    And the table "items_strings" should be:
      | item_id             | language_tag | title    | image_url | subtitle | description |
      | 5577006791947779410 | sl           | my skill | null      | null     | null        |
    And the table "items_items" should be:
      | parent_item_id      | child_item_id       | child_order | content_view_propagation | upper_view_levels_propagation | grant_view_propagation | watch_propagation | edit_propagation | category    | score_weight |
      | 50                  | 5577006791947779410 | 1           | as_info                  | as_is                         | 1                      | 1                 | 1                | Undefined   | 1            |
      | 5577006791947779410 | 12                  | 0           | as_info                  | as_is                         | 0                      | 0                 | 0                | Application | 2            |
      | 5577006791947779410 | 34                  | 1           | as_info                  | as_is                         | 1                      | 1                 | 1                | Application | 2            |
    And the table "items_ancestors" should be:
      | ancestor_item_id    | child_item_id       |
      | 50                  | 12                  |
      | 50                  | 34                  |
      | 50                  | 5577006791947779410 |
      | 5577006791947779410 | 12                  |
      | 5577006791947779410 | 34                  |
    And the table "groups" should be:
      | id                  | type                | name                             |
      | 10                  | Friends             | Friends                          |
      | 11                  | User                | jdoe                             |
    And the table "permissions_granted" should be:
      | group_id            | item_id             | source_group_id     | origin           | can_view                 | can_grant_view      | can_watch         | can_edit       | is_owner | ABS(TIMESTAMPDIFF(SECOND, latest_update_at, NOW())) < 3 |
      | 11                  | 12                  | 11                  | group_membership | content_with_descendants | solution            | answer            | all            | 0        | 0                                                       |
      | 11                  | 21                  | 11                  | group_membership | solution                 | none                | none              | children       | 0        | 0                                                       |
      | 11                  | 34                  | 11                  | group_membership | solution                 | solution_with_grant | answer_with_grant | all_with_grant | 0        | 0                                                       |
      | 11                  | 50                  | 11                  | group_membership | solution                 | solution_with_grant | answer_with_grant | all_with_grant | 0        | 0                                                       |
      | 11                  | 5577006791947779410 | 11                  | self             | none                     | none                | none              | none           | 1        | 1                                                       |
    And the table "permissions_generated" should be:
      | group_id            | item_id             | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 10                  | 21                  | none               | content                  | none                | none               | 0                  |
      | 11                  | 12                  | solution           | solution                 | answer              | all                | 0                  |
      | 11                  | 21                  | solution           | none                     | none                | children           | 0                  |
      | 11                  | 34                  | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | 0                  |
      | 11                  | 50                  | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | 0                  |
      | 11                  | 5577006791947779410 | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | 1                  |

  Scenario: Recomputes the score of a parent item
    Given the database table "items" also has the following rows:
      | id | default_language_tag | type    |
      | 30 | fr                   | Chapter |
      | 31 | fr                   | Task    |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 30             | 31            | 1           |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 30               | 31            |
    And the database also has the following user:
      | group_id | login   |
      | 12       | manager |
    And the database table "results" also has the following rows:
      | attempt_id | participant_id | item_id | score_computed |
      | 0          | 11             | 30      | 50             |
      | 0          | 11             | 31      | 50             |
    And I am the user with id "12"
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated | can_edit_generated |
      | 12       | 30      | solution           | children           |
    When I send a POST request to "/items" with the following body:
    """
    {
      "type": "Task",
      "language_tag": "sl",
      "title": "my title",
      "image_url":"http://bit.ly/1234",
      "subtitle": "hard task",
      "description": "the goal of this task is ...",
      "parent": {"item_id": 30}
    }
    """
    Then the response code should be 201
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "created",
      "data": { "id": "5577006791947779410" }
    }
    """
    And the table "items" at id "5577006791947779410" should be:
      | id                  | type   | url  | options | default_language_tag | entry_frozen_teams | no_score | text_id | title_bar_visible | display_details_in_parent | uses_api | read_only | full_screen | children_layout  | hints_allowed | fixed_ranks | validation_type | entry_min_admitted_members_ratio | entry_max_team_size | allows_multiple_attempts | duration | requires_explicit_entry | show_user_infos | no_score | prompt_to_join_group_by_code | entering_time_min   | entering_time_max   | participants_group_id |
      | 5577006791947779410 | Task   | null | null    | sl                   | 0                  | 0        | null    | 1                 | 0                         | 1        | 0         | default     | List             |0             | 0           | All             | None                             | 0                   | 0                        | null     | 0                       | 0               | 0        | 0                            | 1000-01-01 00:00:00 | 9999-12-31 23:59:59 | null                  |
    And the table "items_items" should be:
      | parent_item_id | child_item_id       | child_order |
      | 30             | 31                  | 1           |
      | 30             | 5577006791947779410 | 2           |
    And the table "items_ancestors" should be:
      | ancestor_item_id | child_item_id       |
      | 30               | 31                  |
      | 30               | 5577006791947779410 |
    And the table "results" should be:
      | attempt_id | participant_id | item_id | score_computed |
      | 0          | 11             | 21      | 0              |
      | 0          | 11             | 30      | 25             |
      | 0          | 11             | 31      | 50             |
