Feature: Update an item string entry - robustness

  Background:
    Given the database has the following user:
      | group_id | login |
      | 11       | jdoe  |
    And the database has the following table "items":
      | id | default_language_tag |
      | 21 | en                   |
      | 22 | en                   |
      | 50 | en                   |
      | 60 | sl                   |
    And the database has the following table "items_strings":
      | item_id | language_tag | title  | image_url                  | subtitle        | description        |
      | 50      | en           | Item 2 | http://myurl.com/item2.jpg | Item 2 Subtitle | Item 2 Description |
      | 50      | sl           | Item 3 | http://myurl.com/item3.jpg | Item 3 Subtitle | Item 3 Description |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated | can_edit_generated | is_owner_generated |
      | 11       | 21      | solution           | children           | false              |
      | 11       | 22      | info               | all                | false              |
      | 11       | 50      | solution           | all                | true               |
    And the database has the following table "languages":
      | tag |
      | en  |
      | sl  |

  Scenario: User not found
    Given I am the user with id "404"
    When I send a PUT request to "/items/50/strings/default" with the following body:
      """
      {
        "title": "The title"
      }
      """
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "items_strings" should remain unchanged

  Scenario: Invalid item_id
    Given I am the user with id "11"
    When I send a PUT request to "/items/abc/strings/default" with the following body:
      """
      {
        "title": "The title"
      }
      """
    Then the response code should be 400
    And the response error message should contain "Wrong value for item_id (should be int64)"
    And the table "items_strings" should remain unchanged

  Scenario: The title is too long
    Given I am the user with id "11"
    When I send a PUT request to "/items/50/strings/default" with the following body:
      """
      {
        "title": "123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901"
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
          "title": ["title must be a maximum of 200 characters in length"]
        }
      }
      """
    And the table "items_strings" should remain unchanged

  Scenario: Image URL is too long
    Given I am the user with id "11"
    When I send a PUT request to "/items/50/strings/default" with the following body:
      """
      {
        "image_url": "123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
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
          "image_url": ["image_url must be a maximum of 2,048 characters in length"]
        }
      }
      """
    And the table "items_strings" should remain unchanged

  Scenario: The subtitle is too long
    Given I am the user with id "11"
    When I send a PUT request to "/items/50/strings/default" with the following body:
      """
      {
        "subtitle": "123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901"
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
          "subtitle": ["subtitle must be a maximum of 200 characters in length"]
        }
      }
      """
    And the table "items_strings" should remain unchanged

  Scenario: Wrong language
    Given I am the user with id "11"
    When I send a PUT request to "/items/50/strings/unknown" with the following body:
      """
      {
      }
      """
    Then the response code should be 400
    And the response error message should contain "No such language"
    And the table "items_strings" should remain unchanged

  Scenario: The user doesn't have rights to manage the item
    And I am the user with id "11"
    When I send a PUT request to "/items/60/strings/default" with the following body:
      """
      {
        "title": "The title"
      }
      """
    Then the response code should be 403
    And the response error message should contain "No access rights to edit the item"

  Scenario: The user doesn't have rights to manage the item (can_edit = children)
    And I am the user with id "11"
    When I send a PUT request to "/items/21/strings/default" with the following body:
      """
      {
        "title": "The title"
      }
      """
    Then the response code should be 403
    And the response error message should contain "No access rights to edit the item"

  Scenario: The user doesn't have rights to manage the item (can_view = info)
    And I am the user with id "11"
    When I send a PUT request to "/items/22/strings/default" with the following body:
      """
      {
        "title": "The title"
      }
      """
    Then the response code should be 403
    And the response error message should contain "No access rights to edit the item"
