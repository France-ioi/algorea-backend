Feature: Delete an item dependency - robustness
  Background:
    Given the database has the following table "groups":
      | id | name    | type  |
      | 13 | Group B | Team  |
      | 15 | Group C | Class |
      | 26 | team    | Team  |
    And the database has the following users:
      | group_id | login      | default_language |
      | 11       | jdoe       |                  |
      | 14       | nosolution |                  |
      | 17       | fr         | fr               |
      | 22       | info       |                  |
      | 23       | jane       |                  |
    And the database has the following table "items":
      | id  | type    | default_language_tag | requires_explicit_entry |
      | 100 | Task    | en                   | true                    |
      | 200 | Task    | en                   | true                    |
      | 210 | Chapter | en                   | false                   |
      | 220 | Chapter | en                   | false                   |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 13              | 11             |
      | 13              | 17             |
      | 15              | 11             |
      | 15              | 14             |
      | 15              | 17             |
      | 26              | 11             |
      | 26              | 22             |
    And the groups ancestors are computed
    And the database has the following table "item_dependencies":
      | item_id | dependent_item_id | score | grant_content_view |
      | 100     | 210               | 22    | true               |
      | 100     | 220               | 10    | true               |
      | 200     | 210               | 20    | true               |
      | 200     | 220               | 30    | false              |
    And the database has the following table "permissions_granted":
      | group_id | item_id | source_group_id | origin         | latest_update_at    | can_view                 | can_enter_from      | can_enter_until     | can_grant_view | can_watch | can_edit | can_make_session_official | is_owner |
      | 22       | 200     | 22              | item_unlocking | 2019-05-30 11:00:00 | info                     | 3019-12-31 23:59:59 | 2020-01-31 23:59:59 | none           | none      | none     | false                     | false    |
      | 22       | 210     | 22              | item_unlocking | 2019-05-30 11:00:00 | info                     | 2019-12-31 23:59:59 | 2020-01-31 23:59:59 | none           | none      | none     | false                     | false    |
      | 26       | 210     | 26              | item_unlocking | 2019-05-30 11:00:00 | content_with_descendants | 2019-12-31 23:59:59 | 2020-01-31 23:59:59 | none           | none      | none     | false                     | false    |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       | can_grant_view_generated | can_edit_generated | can_watch_generated | is_owner_generated |
      | 11       | 200     | solution                 | content                  | children           | result              | true               |
      | 11       | 210     | solution                 | none                     | children           | none                | true               |
      | 11       | 220     | solution                 | none                     | none               | none                | false              |
      | 13       | 200     | solution                 | none                     | none               | none                | false              |
      | 13       | 210     | solution                 | none                     | none               | none                | false              |
      | 13       | 220     | solution                 | none                     | none               | none                | false              |
      | 15       | 200     | none                     | none                     | all                | none                | false              |
      | 15       | 210     | content_with_descendants | none                     | none               | none                | false              |
      | 17       | 200     | solution                 | none                     | none               | none                | false              |
      | 17       | 210     | solution                 | none                     | none               | none                | false              |
      | 17       | 220     | solution                 | none                     | none               | none                | false              |
      | 22       | 200     | solution                 | none                     | none               | none                | false              |
      | 22       | 210     | info                     | none                     | none               | result              | false              |
      | 22       | 220     | info                     | none                     | none               | none                | false              |
      | 23       | 200     | info                     | none                     | none               | none                | false              |
      | 26       | 200     | solution                 | none                     | none               | none                | false              |
      | 26       | 210     | content_with_descendants | none                     | none               | none                | false              |
      | 26       | 220     | info                     | none                     | none               | none                | false              |
    And the database has the following table "languages":
      | tag |
      | fr  |

  Scenario: Invalid dependent_item_id
    Given I am the user with id "11"
    When I send a DELETE request to "/items/aaaaa/prerequisites/200"
    Then the response code should be 400
    And the response error message should contain "Wrong value for dependent_item_id (should be int64)"
    And the table "item_dependencies" should remain unchanged

  Scenario: Invalid prerequisite_item_id
    Given I am the user with id "11"
    When I send a DELETE request to "/items/210/prerequisites/aaa"
    Then the response code should be 400
    And the response error message should contain "Wrong value for prerequisite_item_id (should be int64)"
    And the table "item_dependencies" should remain unchanged

  Scenario: No can_edit=all on the dependent item
    Given I am the user with id "11"
    When I send a DELETE request to "/items/210/prerequisites/200"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "item_dependencies" should remain unchanged
