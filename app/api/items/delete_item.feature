Feature: Delete an item
  Background:
    Given the database has the following table "groups":
      | id | name    | type    |
      | 10 | Friends | Friends |
    And the database has the following user:
      | group_id | login |
      | 11       | jdoe  |
    And the database has the following table "items_propagate":
      | id | ancestors_computation_state |
      | 20 | done                        |
      | 21 | done                        |
      | 22 | done                        |
    And the database has the following table "items":
      | id | default_language_tag |
      | 20 | fr                   |
      | 21 | fr                   |
      | 22 | fr                   |
    And the database has the following table "permissions_propagate":
      | group_id | item_id |
      | 10       | 22      |
      | 11       | 21      |
      | 11       | 22      |
    And the database has the following table "permissions_generated":
      | group_id | item_id | is_owner_generated |
      | 10       | 22      | true               |
      | 11       | 21      | false              |
    And the database has the following table "permissions_granted":
      | group_id | item_id | is_owner | source_group_id |
      | 10       | 22      | true     | 10              |
      | 11       | 21      | false    | 10              |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 10              | 11             |
    And the groups ancestors are computed
    And the database has the following table "attempts":
      | id | participant_id | root_item_id |
      | 0  | 10             | null         |
      | 1  | 10             | 22           |
    And the database has the following table "answers":
      | participant_id | attempt_id | item_id | author_id | created_at          |
      | 10             | 0          | 21      | 10        | 2019-05-30 11:00:00 |
      | 10             | 0          | 22      | 10        | 2019-05-30 11:00:00 |
      | 10             | 1          | 21      | 10        | 2019-05-30 11:00:00 |
    And the database has the following table "filters":
      | id | user_id | item_id |
      | 1  | 10      | 21      |
      | 2  | 10      | 22      |
      | 3  | 11      | null    |
    And the database has the following table "groups_contest_items":
      | group_id | item_id |
      | 10       | 21      |
      | 11       | 22      |
    And the database has the following table "item_dependencies":
      | item_id | dependent_item_id |
      | 21      | 21                |
      | 21      | 22                |
      | 22      | 21                |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 20               | 21            |
      | 20               | 22            |
      | 21               | 22            |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 20             | 21            | 1           |
      | 21             | 22            | 1           |
    And the database has the following table "items_strings":
      | item_id | language_tag |
      | 20      | fr           |
      | 21      | fr           |
      | 22      | en           |
      | 22      | fr           |
    And the database has the following table "results":
      | attempt_id | participant_id | item_id |
      | 0          | 10             | 21      |
      | 0          | 10             | 22      |
      | 1          | 10             | 21      |
    And the database has the following table "languages":
      | tag |
      | fr  |
      | en  |
      | sl  |

  Scenario: Delete an item
    Given I am the user with id "11"
    When I send a DELETE request to "/items/22"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "success": true,
        "message": "deleted"
      }
      """
    And the table "items" should remain unchanged, except that the rows with id "22" should be deleted

    And the table "items_strings" should remain unchanged, except that the rows with item_id "22" should be deleted
    And the table "permissions_propagate" should be empty
    And the table "items_items" should remain unchanged, except that the rows with parent_item_id,child_item_id "22" should be deleted
    And the table "items_propagate" should remain unchanged, except that the rows with id "22" should be deleted
    And the table "items_ancestors" should remain unchanged, except that the rows with ancestor_item_id,child_item_id "22" should be deleted
    And the table "permissions_granted" should remain unchanged, except that the rows with item_id "22" should be deleted
    And the table "permissions_generated" should remain unchanged, except that the rows with item_id "22" should be deleted
    And the table "item_dependencies" should remain unchanged, except that the rows with item_id,dependent_item_id "22" should be deleted
    And the table "groups_contest_items" should remain unchanged, except that the rows with item_id "22" should be deleted
    And the table "attempts" should remain unchanged, regardless of the rows with id "1"
    And the table "attempts" at id "1" should be:
      | id | participant_id | root_item_id |
      | 1  | 10             | null         |
    And the table "results" should remain unchanged, except that the rows with item_id "22" should be deleted
    And the table "results_propagate" should be empty
    And the table "answers" should remain unchanged, except that the rows with item_id "22" should be deleted
    And the table "filters" should remain unchanged
