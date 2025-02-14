Feature: Update item - robustness
  Background:
    Given the database has the following users:
      | login | temp_user | group_id |
      | jdoe  | 0         | 11       |
    And the database has the following table "items":
      | id | default_language_tag | type    | requires_explicit_entry | duration | text_id |
      | 4  | fr                   | Chapter | 0                       | null     | id4     |
      | 20 | fr                   | Chapter | 0                       | null     | id20    |
      | 21 | fr                   | Chapter | 0                       | null     | id21    |
      | 22 | fr                   | Chapter | 0                       | null     | id22    |
      | 23 | fr                   | Skill   | 0                       | null     | id23    |
      | 24 | fr                   | Task    | 0                       | null     | id24    |
      | 25 | fr                   | Task    | 1                       | 00:00:01 | id25    |
      | 50 | fr                   | Chapter | 0                       | null     | id50    |
      | 60 | fr                   | Chapter | 0                       | null     | id60    |
      | 70 | fr                   | Skill   | 0                       | null     | id70    |
      | 80 | fr                   | Task    | 0                       | null     | id80    |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | child_order |
      | 4              | 21            | 0           |
      | 21             | 50            | 0           |
    And the database has the following table "items_ancestors":
      | ancestor_item_id | child_item_id |
      | 4                | 21            |
      | 21               | 50            |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated | can_edit_generated | is_owner_generated |
      | 11       | 4       | solution           | none               | false              |
      | 11       | 20      | info               | all                | false              |
      | 11       | 21      | solution           | none               | false              |
      | 11       | 22      | none               | children           | false              |
      | 11       | 23      | info               | all                | false              |
      | 11       | 24      | solution           | children           | false              |
      | 11       | 25      | solution           | all                | false              |
      | 11       | 50      | solution           | all                | false              |
      | 11       | 70      | solution           | all                | false              |
      | 11       | 80      | solution           | all                | false              |
    And the database has the following table "permissions_granted":
      | group_id | item_id | can_view | can_edit | is_owner | source_group_id |
      | 11       | 4       | solution | none     | false    | 11              |
      | 11       | 20      | info     | all      | false    | 11              |
      | 11       | 21      | solution | none     | false    | 11              |
      | 11       | 23      | info     | all      | false    | 11              |
      | 11       | 24      | solution | children | false    | 11              |
      | 11       | 25      | solution | all      | false    | 11              |
      | 11       | 50      | solution | all      | false    | 11              |
      | 11       | 70      | solution | all      | false    | 11              |
      | 11       | 80      | solution | all      | false    | 11              |
    And the groups ancestors are computed
    And the database has the following table "languages":
      | tag |
      | sl  |

  Scenario Outline: Wrong field value
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "<field>": <value>
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
          "<field>": ["<error>"]
        }
      }
      """
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged
  Examples:
    | field                            | value         | error                                                                              |
    | default_language_tag             | 1234          | expected type 'string', got unconvertible type 'float64'                           |
    | default_language_tag             | "unknown"     | default_language_tag must be a maximum of 6 characters in length                   |
    | default_language_tag             | ""            | default_language_tag must be at least 1 character in length                        |
    | default_language_tag             | "unknow"      | default language should exist and there should be item's strings in this language  |
    | default_language_tag             | "sl"          | default language should exist and there should be item's strings in this language  | # no strings for the tag
    | full_screen                      | "wrong value" | full_screen must be one of [forceYes forceNo default]                              |
    | full_screen                      | ""            | full_screen must be one of [forceYes forceNo default]                              |
    | validation_type                  | "Wrong"       | validation_type must be one of [None All AllButOne Categories One Manual]          |
    | entry_min_admitted_members_ratio | "Wrong"       | entry_min_admitted_members_ratio must be one of [All Half One None]                |
    | duration                         | ""            | invalid duration                                                                   |
    | duration                         | "12:34"       | invalid duration                                                                   |
    | duration                         | "-1:34:56"    | invalid duration                                                                   |
    | duration                         | "839:34:56"   | invalid duration                                                                   |
    | duration                         | "99:-1:56"    | invalid duration                                                                   |
    | duration                         | "99:60:56"    | invalid duration                                                                   |
    | duration                         | "99:59:-1"    | invalid duration                                                                   |
    | duration                         | "99:59:60"    | invalid duration                                                                   |
    | duration                         | "00:00:01"    | requires_explicit_entry should be true when the duration is not null               |
    | options                          | ""            | options should be a valid JSON or null                                             |

  Scenario: Invalid item_id
    And I am the user with id "11"
    When I send a PUT request to "/items/abc" with the following body:
      """
      {
        "type": "Task"
      }
      """
    Then the response code should be 400
    And the response error message should contain "Wrong value for item_id (should be int64)"
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: The user doesn't exist
    And I am the user with id "121"
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "type": "Task"
      }
      """
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: The user doesn't have rights to edit the item
    And I am the user with id "11"
    When I send a PUT request to "/items/60" with the following body:
      """
      {
        "type": "Task"
      }
      """
    Then the response code should be 403
    And the response error message should contain "No access rights to edit the item"
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: The user doesn't have rights to edit the item (can_edit = children)
    And I am the user with id "11"
    When I send a PUT request to "/items/24" with the following body:
      """
      {
        "url": "http://someurl.com"
      }
      """
    Then the response code should be 403
    And the response error message should contain "No access rights to edit the item's properties"
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: The user doesn't have rights to edit the item (can_view = info)
    And I am the user with id "11"
    When I send a PUT request to "/items/20" with the following body:
      """
      {
        "url": "http://someurl.com"
      }
      """
    Then the response code should be 403
    And the response error message should contain "No access rights to edit the item"
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: The user doesn't have rights to edit item's children
    And I am the user with id "11"
    When I send a PUT request to "/items/60" with the following body:
      """
      {
        "children": []
      }
      """
    Then the response code should be 403
    And the response error message should contain "No access rights to edit the item"
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: Non-unique children item IDs
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "children": [
          {"item_id": "21", "order": 1},
          {"item_id": "21", "order": 2}
        ]
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
          "children": ["children IDs should be unique and each should be visible to the user"]
        }
      }
      """
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: Child order is missing
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "children": [{"item_id": "21"}]
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
          "children[0].order": ["missing field"]
        }
      }
      """
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: Children items are not visible to the user
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "children": [
          {"item_id": "4", "order": 1},
          {"item_id": "22", "order": 2}
        ]
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
          "children": ["children IDs should be unique and each should be visible to the user"]
        }
      }
      """
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: The item is among child items
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "children": [
          {"item_id": "50", "order": 1}
        ]
      }
      """
    Then the response code should be 403
    And the response error message should contain "An item cannot become an ancestor of itself"
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario: The item is a descendant of a child item
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "children": [
          {"item_id": "21", "order": 1}
        ]
      }
      """
    Then the response code should be 403
    And the response error message should contain "An item cannot become an ancestor of itself"
    And the table "items" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "permissions_granted" should remain unchanged

  Scenario Outline: Not enough permissions for setting propagation in items_items
    Given I am the user with id "11"
    And the database table "items" also has the following row:
      | id | default_language_tag |
      | 90 | fr                   |
    And the database table "permissions_generated" also has the following row:
      | group_id | item_id | <permission_column> | can_view_generated |
      | 11       | 90      | <permission_value>  | info               |
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "children": [{
          "item_id": "90",
          "order": 1,
          "<field>": {{"<value>" != "true" && "<value>" != "false" ? "\"<value>\"" : <value>}}
        }]
      }
      """
    Then the response code should be 403
    And the response body should be, in JSON:
      """
      {
        "success": false,
        "message": "Forbidden",
        "error_text": "<error>"
      }
      """
    And the table "items" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged
    Examples:
      | field                         | value                       | permission_column        | permission_value         | error                                                            |
      | content_view_propagation      | as_content                  | can_grant_view_generated | none                     | Not enough permissions for setting content_view_propagation      |
      | content_view_propagation      | as_content                  | can_grant_view_generated | enter                    | Not enough permissions for setting content_view_propagation      |
      | content_view_propagation      | as_info                     | can_grant_view_generated | none                     | Not enough permissions for setting content_view_propagation      |
      | upper_view_levels_propagation | as_is                       | can_grant_view_generated | content_with_descendants | Not enough permissions for setting upper_view_levels_propagation |
      | upper_view_levels_propagation | as_content_with_descendants | can_grant_view_generated | content                  | Not enough permissions for setting upper_view_levels_propagation |
      | grant_view_propagation        | true                        | can_grant_view_generated | solution                 | Not enough permissions for setting grant_view_propagation        |
      | watch_propagation             | true                        | can_watch_generated      | answer                   | Not enough permissions for setting watch_propagation             |
      | edit_propagation              | true                        | can_edit_generated       | all                      | Not enough permissions for setting edit_propagation              |
      | request_help_propagation      | true                        | can_grant_view_generated | none                     | Not enough permissions for setting request_help_propagation      |
      | request_help_propagation      | true                        | can_grant_view_generated | enter                    | Not enough permissions for setting request_help_propagation      |

  Scenario: A child is a skill while the item is not a skill
    Given I am the user with id "11"
    When I send a PUT request to "/items/50" with the following body:
      """
      {
        "children": [
          {"item_id": "23", "order": 0}
        ]
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
          "children[0]": ["a skill cannot be a child of a non-skill item"]
        }
      }
      """
    And the table "items" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario Outline: The item cannot have children
    Given I am the user with id "11"
    When I send a PUT request to "/items/<item_id>" with the following body:
      """
      {
        "children": [
          {"item_id": "21", "order": 1}
        ]
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
          "children": ["a task cannot have children items"]
        }
      }
      """
    And the table "items" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged
    Examples:
      | item_id |
      | 24      |
      | 25      |

  Scenario: Should not be possible to set requires_explicit_entry to false when the duration is set
    Given I am the user with id "11"
    When I send a PUT request to "/items/25" with the following body:
      """
      {
        "requires_explicit_entry": false
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
          "requires_explicit_entry": ["requires_explicit_entry should be true when the duration is not null"]
        }
      }
      """
    And the table "items" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: A skill cannot have a duration or require an explicit entry
    Given I am the user with id "11"
    When I send a PUT request to "/items/70" with the following body:
      """
      {
        "duration": "00:00:01",
        "requires_explicit_entry": true
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
          "duration": ["cannot be set for skill items"],
          "requires_explicit_entry": ["cannot be set for skill items"]
        }
      }
      """
    And the table "items" should remain unchanged
    And the table "items_items" should remain unchanged
    And the table "items_ancestors" should remain unchanged
    And the table "items_strings" should remain unchanged
    And the table "permissions_granted" should remain unchanged
    And the table "permissions_generated" should remain unchanged

  Scenario: text_id should be unique
    Given I am the user with id "11"
    When I send a PUT request to "/items/80" with the following body:
      """
      {
        "text_id": "id70"
      }
      """
    Then the response code should be 403
    And the response body should be, in JSON:
      """
      {
        "success": false,
        "message": "Forbidden",
        "error_text": "Invalid input data",
        "errors":{
          "text_id": ["text_id must be unique"]
        }
      }
      """
    And the table "items" should remain unchanged
