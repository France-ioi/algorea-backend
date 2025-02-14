Feature: Change item access rights for a group
  Background:
    Given the database has the following table "groups":
      | id | name       | type  |
      | 25 | some class | Class |
    And the database has the following users:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
      | 23       | user  | John        | Doe       |
      | 31       | jane  | Jane        | Doe       |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_grant_group_access |
      | 25       | 21         | 1                      |
      | 31       | 21         | 0                      |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 25              | 23             |
      | 25              | 31             |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id  | default_language_tag |
      | 100 | fr                   |
      | 101 | fr                   |
      | 102 | fr                   |
      | 103 | fr                   |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | content_view_propagation | grant_view_propagation | watch_propagation | edit_propagation | child_order |
      | 100            | 101           | as_info                  | false                  | false             | false            | 0           |
      | 101            | 102           | as_content               | false                  | false             | false            | 0           |
      | 102            | 103           | as_content               | true                   | true              | true             | 0           |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 100              | 101           |
      | 100              | 102           |
      | 100              | 103           |
      | 101              | 102           |
      | 101              | 103           |
      | 102              | 103           |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated        |
      | 23       | 100     | content_with_descendants  |
      | 23       | 101     | info                      |
      | 23       | 103     | info                      |
    And the database has the following table "permissions_granted":
      | group_id | item_id | can_view | source_group_id | latest_update_at    |
      | 23       | 100     | content  | 23              | 2019-05-30 11:00:00 |
    And the database has the following table "attempts":
      | id | participant_id |
      | 0  | 21             |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id |
      | 0          | 21             | 103     |

  Scenario Outline: Create a new permissions_granted row (with results propagation)
    Given I am the user with id "21"
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 21       | 102     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 21       | 103     | solution           | solution                 | answer              | all                | true               |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view | can_grant_view      | can_watch         | can_edit       | source_group_id | latest_update_at    |
      | 21       | 102     | solution | solution_with_grant | answer_with_grant | all_with_grant | 23              | 2019-05-30 11:00:00 |
      | 23       | 102     | none     | none                | none              | none           | 23              | 2019-05-30 11:00:00 |
    When I send a PUT request to "/groups/25/permissions/23/102" with the following body:
      """
      <json>
      """
    Then the response should be "updated"
    And the table "permissions_granted" should be:
      | group_id | item_id | source_group_id | origin           | can_view   | can_grant_view      | can_watch         | can_edit       | is_owner   | can_make_session_official   | can_enter_from      | can_enter_until     | TIMESTAMPDIFF(SECOND, latest_update_at, NOW()) < 3 |
      | 21       | 102     | 23              | group_membership | solution   | solution_with_grant | answer_with_grant | all_with_grant | false      | false                       | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 100     | 23              | group_membership | content    | none                | none              | none           | false      | false                       | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 102     | 23              | group_membership | none       | none                | none              | none           | false      | false                       | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 102     | 25              | group_membership | <can_view> | <can_grant_view>    | <can_watch>       | <can_edit>     | <is_owner> | <can_make_session_official> | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 1                                                  |
    And the table "permissions_generated" should be:
      | group_id | item_id | can_view_generated    | can_grant_view_generated    | can_watch_generated    | can_edit_generated    | is_owner_generated |
      | 21       | 102     | solution              | solution_with_grant         | answer_with_grant      | all_with_grant        | false              |
      | 21       | 103     | content               | solution                    | answer                 | all                   | false              |
      | 23       | 100     | content               | none                        | none                   | none                  | false              |
      | 23       | 101     | info                  | none                        | none                   | none                  | false              |
      | 23       | 102     | <can_view_generated>  | <can_grant_view_generated>  | <can_watch_generated>  | <can_edit_generated>  | <is_owner>         |
      | 23       | 103     | <can_view_propagated> | <can_grant_view_propagated> | <can_watch_propagated> | <can_edit_propagated> | false              |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id |
      | 0          | 21             | 102     |
      | 0          | 21             | 103     |
    And the table "results_propagate" should be empty
  Examples:
    | json                                                 | can_view | can_grant_view | can_watch | can_edit | is_owner | can_make_session_official | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | can_view_propagated | can_grant_view_propagated | can_watch_propagated | can_edit_propagated |
    | {"can_view":"solution"}                              | solution | none           | none      | none     | false    | false                     | solution           | none                     | none                | none               | content             | none                      | none                 | none                |
    | {"can_view":"info"}                                  | info     | none           | none      | none     | false    | false                     | info               | none                     | none                | none               | none                | none                      | none                 | none                |
    | {"can_view":"info","can_grant_view":"enter"}         | info     | enter          | none      | none     | false    | false                     | info               | enter                    | none                | none               | none                | enter                     | none                 | none                |
    | {"can_view":"content","can_watch":"answer"}          | content  | none           | answer    | none     | false    | false                     | content            | none                     | answer              | none               | content             | none                      | answer               | none                |
    | {"can_view":"content","can_edit":"all"}              | content  | none           | none      | all      | false    | false                     | content            | none                     | none                | all                | content             | none                      | none                 | all                 |
    | {"can_view":"info","can_make_session_official":true} | info     | none           | none      | none     | false    | true                      | info               | none                     | none                | none               | none                | none                      | none                 | none                |
    | {"is_owner":true}                                    | none     | none           | none      | none     | true     | false                     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | content             | solution                  | answer               | all                 |

  Scenario Outline: Create a new permissions_granted row (without results propagation)
    Given I am the user with id "21"
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 21       | 102     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 21       | 103     | solution           | solution                 | answer              | all                | true               |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view | can_grant_view      | can_watch         | can_edit       | source_group_id | latest_update_at    |
      | 21       | 102     | solution | solution_with_grant | answer_with_grant | all_with_grant | 23              | 2019-05-30 11:00:00 |
      | 23       | 102     | none     | none                | none              | none           | 23              | 2019-05-30 11:00:00 |
    When I send a PUT request to "/groups/25/permissions/23/102" with the following body:
      """
      <json>
      """
    Then the response should be "updated"
    And the table "permissions_granted" should be:
      | group_id | item_id | source_group_id | origin           | can_view | can_grant_view      | can_watch         | can_edit       | is_owner | can_make_session_official | can_enter_from      | can_enter_until     | TIMESTAMPDIFF(SECOND, latest_update_at, NOW()) < 3 |
      | 21       | 102     | 23              | group_membership | solution | solution_with_grant | answer_with_grant | all_with_grant | false    | false                     | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 100     | 23              | group_membership | content  | none                | none              | none           | false    | false                     | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 102     | 23              | group_membership | none     | none                | none              | none           | false    | false                     | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 102     | 25              | group_membership | none     | none                | none              | none           | false    | false                     | <can_enter_from>    | <can_enter_until>   | 1                                                  |
    And the table "permissions_generated" should be:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 21       | 102     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | false              |
      | 21       | 103     | content            | solution                 | answer              | all                | false              |
      | 23       | 100     | content            | none                     | none                | none               | false              |
      | 23       | 101     | info               | none                     | none                | none               | false              |
      | 23       | 102     | none               | none                     | none                | none               | false              |
      | 23       | 103     | none               | none                     | none                | none               | false              |
    And the table "attempts" should remain unchanged
    And the table "results_propagate" should be:
      | attempt_id | participant_id | item_id | state            |
      | 0          | 21             | 103     | to_be_propagated |
  Examples:
    | json                                       | can_enter_from      | can_enter_until     |
    | {"can_enter_from":"2019-05-30T11:00:00Z"}  | 2019-05-30 11:00:00 | 9999-12-31 23:59:59 |
    | {"can_enter_until":"2019-05-30T11:00:00Z"} | 9999-12-31 23:59:59 | 2019-05-30 11:00:00 |

  Scenario Outline: Update an existing permissions_granted row
    Given I am the user with id "21"
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 21       | 102     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 23       | 102     | none               | none                     | none                | none               | false              |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view | can_grant_view      | can_watch         | can_edit       | is_owner | origin           | source_group_id | latest_update_at    |
      | 21       | 102     | solution | solution_with_grant | answer_with_grant | all_with_grant | true     | group_membership | 23              | 2019-05-30 11:00:00 |
      | 23       | 102     | none     | none                | none              | none           | false    | group_membership | 23              | 2019-05-30 11:00:00 |
      | 23       | 102     | none     | none                | none              | none           | false    | group_membership | 25              | 2019-05-30 11:00:00 |
    When I send a PUT request to "/groups/25/permissions/23/102" with the following body:
    """
    <json>
    """
    Then the response should be "updated"
    And the table "permissions_granted" should be:
      | group_id | item_id | source_group_id | origin           | can_view   | can_grant_view      | can_watch         | can_edit       | can_make_session_official   | is_owner   | can_enter_from      | can_enter_until     | TIMESTAMPDIFF(SECOND, latest_update_at, NOW()) < 3 |
      | 21       | 102     | 23              | group_membership | solution   | solution_with_grant | answer_with_grant | all_with_grant | false                       | true       | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 100     | 23              | group_membership | content    | none                | none              | none           | false                       | false      | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 102     | 23              | group_membership | none       | none                | none              | none           | false                       | false      | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 102     | 25              | group_membership | <can_view> | <can_grant_view>    | <can_watch>       | <can_edit>     | <can_make_session_official> | <is_owner> | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 1                                                  |
    And the table "permissions_generated" should be:
      | group_id | item_id | can_view_generated    | can_grant_view_generated    | can_watch_generated    | can_edit_generated    | is_owner_generated |
      | 21       | 102     | solution              | solution_with_grant         | answer_with_grant      | all_with_grant        | true               |
      | 21       | 103     | content               | solution                    | answer                 | all                   | false              |
      | 23       | 100     | content               | none                        | none                   | none                  | false              |
      | 23       | 101     | info                  | none                        | none                   | none                  | false              |
      | 23       | 102     | <can_view_generated>  | <can_grant_view_generated>  | <can_watch_generated>  | <can_edit_generated>  | <is_owner>         |
      | 23       | 103     | <can_view_propagated> | <can_grant_view_propagated> | <can_watch_propagated> | <can_edit_propagated> | false              |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id |
      | 0          | 21             | 102     |
      | 0          | 21             | 103     |
    And the table "results_propagate" should be empty
  Examples:
    | json                                                           | can_view                 | can_grant_view      | can_watch | can_edit       | is_owner | can_make_session_official | can_view_generated       | can_grant_view_generated | can_watch_generated | can_edit_generated | can_view_propagated | can_grant_view_propagated | can_watch_propagated | can_edit_propagated |
    | {"can_view":"content_with_descendants"}                        | content_with_descendants | none                | none      | none           | false    | false                     | content_with_descendants | none                     | none                | none               | content             | none                      | none                 | none                |
    | {"can_view":"solution","can_grant_view":"solution_with_grant"} | solution                 | solution_with_grant | none      | none           | false    | false                     | solution                 | solution_with_grant      | none                | none               | content             | solution                  | none                 | none                |
    | {"can_view":"content","can_watch":"result"}                    | content                  | none                | result    | none           | false    | false                     | content                  | none                     | result              | none               | content             | none                      | result               | none                |
    | {"can_view":"content","can_edit":"all_with_grant"}             | content                  | none                | none      | all_with_grant | false    | false                     | content                  | none                     | none                | all_with_grant     | content             | none                      | none                 | all                 |
    | {"can_view":"info","can_make_session_official":true}           | info                     | none                | none      | none           | false    | true                      | info                     | none                     | none                | none               | none                | none                      | none                 | none                |
    | {"is_owner":true}                                              | none                     | none                | none      | none           | true     | false                     | solution                 | solution_with_grant      | answer_with_grant   | all_with_grant     | content             | solution                  | answer               | all                 |

  Scenario: Create a new permissions_granted row (the group has only 'content' access on the item's parent)
    Given I am the user with id "21"
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 21       | 102     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | 1                  |
      | 21       | 103     | none               | none                     | none                | none               | 0                  |
      | 31       | 101     | content            | none                     | none                | none               | 0                  |
      | 31       | 102     | none               | none                     | none                | none               | 0                  |
      | 31       | 103     | none               | none                     | none                | none               | 0                  |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view | is_owner | source_group_id | origin           | latest_update_at    |
      | 21       | 102     | none     | 1        | 23              | group_membership | 2019-05-30 11:00:00 |
      | 31       | 101     | content  | 0        | 23              | group_membership | 2019-05-30 11:00:00 |
    When I send a PUT request to "/groups/25/permissions/31/102" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response should be "updated"
    And the table "permissions_granted" should be:
      | group_id | item_id | can_view | is_owner | source_group_id | origin           | TIMESTAMPDIFF(SECOND, latest_update_at, NOW()) < 3 |
      | 21       | 102     | none     | 1        | 23              | group_membership | 0                                                  |
      | 23       | 100     | content  | 0        | 23              | group_membership | 0                                                  |
      | 31       | 101     | content  | 0        | 23              | group_membership | 0                                                  |
      | 31       | 102     | solution | 0        | 25              | group_membership | 1                                                  |
    And the table "permissions_generated" should be:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 21       | 102     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | 1                  |
      | 21       | 103     | content            | solution                 | answer              | all                | 0                  |
      | 23       | 100     | content            | none                     | none                | none               | 0                  |
      | 23       | 101     | info               | none                     | none                | none               | 0                  |
      | 23       | 102     | none               | none                     | none                | none               | 0                  |
      | 23       | 103     | none               | none                     | none                | none               | 0                  |
      | 31       | 101     | content            | none                     | none                | none               | 0                  |
      | 31       | 102     | solution           | none                     | none                | none               | 0                  |
      | 31       | 103     | content            | none                     | none                | none               | 0                  |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id |
      | 0          | 21             | 102     |
      | 0          | 21             | 103     |
    And the table "results_propagate" should be empty

  Scenario: Create a new permissions_granted row (the group has no access to the item's parents, but has full access to the item itself)
    Given I am the user with id "21"
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated       | can_grant_view_generated | is_owner_generated |
      | 21       | 100     | solution                 | solution                 | 1                  |
      | 21       | 101     | none                     | none                     | 0                  |
      | 21       | 102     | none                     | none                     | 0                  |
      | 21       | 103     | none                     | none                     | 0                  |
      | 31       | 100     | content_with_descendants | none                     | 0                  |
      | 31       | 101     | content_with_descendants | none                     | 0                  |
      | 31       | 102     | content_with_descendants | none                     | 0                  |
      | 31       | 103     | content_with_descendants | none                     | 0                  |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view                 | can_grant_view | is_owner | source_group_id | origin           | latest_update_at    |
      | 21       | 100     | solution                 | solution       | 1        | 23              | group_membership | 2019-05-30 11:00:00 |
      | 31       | 100     | content_with_descendants | none           | 0        | 23              | group_membership | 2019-05-30 11:00:00 |
    When I send a PUT request to "/groups/25/permissions/31/100" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response should be "updated"
    And the table "permissions_granted" should be:
      | group_id | item_id | can_view                 | is_owner | source_group_id | origin           | TIMESTAMPDIFF(SECOND, latest_update_at, NOW()) < 3 |
      | 21       | 100     | solution                 | 1        | 23              | group_membership | 0                                                  |
      | 23       | 100     | content                  | 0        | 23              | group_membership | 0                                                  |
      | 31       | 100     | content_with_descendants | 0        | 23              | group_membership | 0                                                  |
      | 31       | 100     | solution                 | 0        | 25              | group_membership | 1                                                  |
    And the table "permissions_generated" should be:
      | group_id | item_id | can_view_generated | is_owner_generated |
      | 21       | 100     | solution           | 1                  |
      | 21       | 101     | info               | 0                  |
      | 21       | 102     | none               | 0                  |
      | 21       | 103     | none               | 0                  |
      | 23       | 100     | content            | 0                  |
      | 23       | 101     | info               | 0                  |
      | 23       | 102     | none               | 0                  |
      | 23       | 103     | none               | 0                  |
      | 31       | 100     | solution           | 0                  |
      | 31       | 101     | info               | 0                  |
      | 31       | 102     | none               | 0                  |
      | 31       | 103     | none               | 0                  |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id |
      | 0          | 21             | 100     |
      | 0          | 21             | 101     |
      | 0          | 21             | 102     |
      | 0          | 21             | 103     |
    And the table "results_propagate" should be empty

  Scenario: Create a new permissions_granted row (the group has no access to the item's parents, but has 'content' access to the item itself)
    Given I am the user with id "21"
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated | can_grant_view_generated | is_owner_generated |
      | 21       | 100     | solution           | solution                 | 1                  |
      | 21       | 101     | none               | none                     | 0                  |
      | 21       | 102     | none               | none                     | 0                  |
      | 21       | 103     | none               | none                     | 0                  |
      | 31       | 100     | content            | none                     | 0                  |
      | 31       | 101     | content            | none                     | 0                  |
      | 31       | 102     | content            | none                     | 0                  |
      | 31       | 103     | content            | none                     | 0                  |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view | can_grant_view | is_owner | source_group_id | origin           | latest_update_at    |
      | 21       | 100     | none     | solution       | 1        | 23              | group_membership | 2019-05-30 11:00:00 |
      | 31       | 100     | content  | none           | 0        | 23              | group_membership | 2019-05-30 11:00:00 |
    When I send a PUT request to "/groups/25/permissions/31/100" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response should be "updated"
    And the table "permissions_granted" should be:
      | group_id | item_id | can_view | is_owner | source_group_id | origin           | TIMESTAMPDIFF(SECOND, latest_update_at, NOW()) < 3 |
      | 21       | 100     | none     | 1        | 23              | group_membership | 0                                                  |
      | 23       | 100     | content  | 0        | 23              | group_membership | 0                                                  |
      | 31       | 100     | content  | 0        | 23              | group_membership | 0                                                  |
      | 31       | 100     | solution | 0        | 25              | group_membership | 1                                                  |
    And the table "permissions_generated" should be:
      | group_id | item_id | can_view_generated | is_owner_generated |
      | 21       | 100     | solution           | 1                  |
      | 21       | 101     | info               | 0                  |
      | 21       | 102     | none               | 0                  |
      | 21       | 103     | none               | 0                  |
      | 23       | 100     | content            | 0                  |
      | 23       | 101     | info               | 0                  |
      | 23       | 102     | none               | 0                  |
      | 23       | 103     | none               | 0                  |
      | 31       | 100     | solution           | 0                  |
      | 31       | 101     | info               | 0                  |
      | 31       | 102     | none               | 0                  |
      | 31       | 103     | none               | 0                  |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id |
      | 0          | 21             | 100     |
      | 0          | 21             | 101     |
      | 0          | 21             | 102     |
      | 0          | 21             | 103     |
    And the table "results_propagate" should be empty

  Scenario: Create a new permissions_granted row (the group has no access to the item's parents, but has info access to the item itself)
    Given I am the user with id "21"
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated | can_grant_view_generated | is_owner_generated |
      | 21       | 100     | solution           | solution                 | 1                  |
      | 21       | 101     | none               | none                     | 0                  |
      | 21       | 102     | none               | none                     | 0                  |
      | 21       | 103     | none               | none                     | 0                  |
      | 31       | 100     | info               | none                     | 0                  |
      | 31       | 101     | info               | none                     | 0                  |
      | 31       | 102     | info               | none                     | 0                  |
      | 31       | 103     | info               | none                     | 0                  |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view | can_grant_view | is_owner | source_group_id | origin           | latest_update_at    |
      | 21       | 100     | none     | solution       | 1        | 23              | group_membership | 2019-05-30 11:00:00 |
      | 31       | 100     | info     | none           | 0        | 23              | group_membership | 2019-05-30 11:00:00 |
    When I send a PUT request to "/groups/25/permissions/31/100" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response should be "updated"
    And the table "permissions_granted" should be:
      | group_id | item_id | can_view | is_owner | source_group_id | origin           | TIMESTAMPDIFF(SECOND, latest_update_at, NOW()) < 3 |
      | 21       | 100     | none     | 1        | 23              | group_membership | 0                                                  |
      | 23       | 100     | content  | 0        | 23              | group_membership | 0                                                  |
      | 31       | 100     | info     | 0        | 23              | group_membership | 0                                                  |
      | 31       | 100     | solution | 0        | 25              | group_membership | 1                                                  |
    And the table "permissions_generated" should be:
      | group_id | item_id | can_view_generated | is_owner_generated |
      | 21       | 100     | solution           | 1                  |
      | 21       | 101     | info               | 0                  |
      | 21       | 102     | none               | 0                  |
      | 21       | 103     | none               | 0                  |
      | 23       | 100     | content            | 0                  |
      | 23       | 101     | info               | 0                  |
      | 23       | 102     | none               | 0                  |
      | 23       | 103     | none               | 0                  |
      | 31       | 100     | solution           | 0                  |
      | 31       | 101     | info               | 0                  |
      | 31       | 102     | none               | 0                  |
      | 31       | 103     | none               | 0                  |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id |
      | 0          | 21             | 100     |
      | 0          | 21             | 101     |
      | 0          | 21             | 102     |
      | 0          | 21             | 103     |
    And the table "results_propagate" should be empty

  Scenario: Drops invalid permissions from an existing permissions_granted row
    Given I am the user with id "21"
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 21       | 102     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 23       | 102     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | false              |
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view | can_grant_view      | can_watch         | can_edit       | can_make_session_official | is_owner | origin           | source_group_id | latest_update_at    |
      | 21       | 102     | solution | solution_with_grant | answer_with_grant | all_with_grant | true                      | true     | group_membership | 23              | 2019-05-30 11:00:00 |
      | 23       | 102     | none     | none                | none              | none           | false                     | false    | group_membership | 23              | 2019-05-30 11:00:00 |
      | 23       | 102     | solution | solution_with_grant | answer_with_grant | all_with_grant | true                      | false    | group_membership | 25              | 2019-05-30 11:00:00 |
    When I send a PUT request to "/groups/25/permissions/23/102" with the following body:
    """
    {"can_view": "info"}
    """
    Then the response should be "updated"
    And the table "permissions_granted" should be:
      | group_id | item_id | source_group_id | origin           | can_view   | can_grant_view      | can_watch         | can_edit       | can_make_session_official   | is_owner   | can_enter_from      | can_enter_until     | TIMESTAMPDIFF(SECOND, latest_update_at, NOW()) < 3 |
      | 21       | 102     | 23              | group_membership | solution   | solution_with_grant | answer_with_grant | all_with_grant | true                        | true       | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 100     | 23              | group_membership | content    | none                | none              | none           | false                       | false      | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 102     | 23              | group_membership | none       | none                | none              | none           | false                       | false      | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 102     | 25              | group_membership | info       | enter               | none              | none           | true                        | false      | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 1                                                  |
    And the table "permissions_generated" should be:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 21       | 102     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 21       | 103     | content            | solution                 | answer              | all                | false              |
      | 23       | 100     | content            | none                     | none                | none               | false              |
      | 23       | 101     | info               | none                     | none                | none               | false              |
      | 23       | 102     | info               | enter                    | none                | none               | false              |
      | 23       | 103     | none               | enter                    | none                | none               | false              |
    And the table "attempts" should remain unchanged
    And the table "results" should be:
      | attempt_id | participant_id | item_id |
      | 0          | 21             | 102     |
      | 0          | 21             | 103     |
    And the table "results_propagate" should be empty

  Scenario Outline: There are no item's parents visible to the group, but the item is a root activity of one of the group's ancestors
    Given I am the user with id "21"
    And the database table "items" also has the following row:
      | id  | default_language_tag |
      | 104 | fr                   |
    And the database table "groups" also has the following rows:
      | id | name                 | type  | root_activity_id   | root_skill_id   |
      | 40 | Group with root item | Class | <root_activity_id> | <root_skill_id> |
    And the database table "groups_groups" also has the following row:
      | parent_group_id | child_group_id |
      | 40              | 23             |
    And the groups ancestors are computed
    And the database table "permissions_granted" also has the following rows:
      | group_id | item_id | can_view | can_grant_view      | can_watch         | can_edit       | can_make_session_official | is_owner | origin           | source_group_id | latest_update_at    |
      | 21       | 104     | solution | solution_with_grant | answer_with_grant | all_with_grant | true                      | true     | group_membership | 23              | 2019-05-30 11:00:00 |
    And the database table "permissions_generated" also has the following rows:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 21       | 104     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
    When I send a PUT request to "/groups/25/permissions/23/104" with the following body:
    """
    {
      "can_view": "solution"
    }
    """
    Then the response should be "updated"
    And the table "permissions_granted" should be:
      | group_id | item_id | source_group_id | origin           | can_view   | can_grant_view      | can_watch         | can_edit       | can_make_session_official   | is_owner   | can_enter_from      | can_enter_until     | TIMESTAMPDIFF(SECOND, latest_update_at, NOW()) < 3 |
      | 21       | 104     | 23              | group_membership | solution   | solution_with_grant | answer_with_grant | all_with_grant | true                        | true       | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 100     | 23              | group_membership | content    | none                | none              | none           | false                       | false      | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 0                                                  |
      | 23       | 104     | 25              | group_membership | solution   | none                | none              | none           | false                       | false      | 9999-12-31 23:59:59 | 9999-12-31 23:59:59 | 1                                                  |
    And the table "permissions_generated" should be:
      | group_id | item_id | can_view_generated | can_grant_view_generated | can_watch_generated | can_edit_generated | is_owner_generated |
      | 21       | 104     | solution           | solution_with_grant      | answer_with_grant   | all_with_grant     | true               |
      | 23       | 100     | content            | none                     | none                | none               | false              |
      | 23       | 101     | info               | none                     | none                | none               | false              |
      | 23       | 102     | none               | none                     | none                | none               | false              |
      | 23       | 103     | none               | none                     | none                | none               | false              |
      | 23       | 104     | solution           | none                     | none                | none               | false              |
  Examples:
    | root_activity_id | root_skill_id |
    | 104              | null          |
    | null             | 104           |
