Feature: Update an item string entry

  Background:
    Given the database has the following user:
      | group_id | login |
      | 11       | jdoe  |
    And the database has the following table "items":
      | id | default_language_tag |
      | 21 | en                   |
      | 50 | en                   |
      | 60 | sl                   |
    And the database has the following table "items_strings":
      | item_id | language_tag | title  | image_url                  | subtitle        | description        |
      | 50      | en           | Item 2 | http://myurl.com/item2.jpg | Item 2 Subtitle | Item 2 Description |
      | 50      | sl           | Item 3 | http://myurl.com/item3.jpg | Item 3 Subtitle | Item 3 Description |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated | can_edit_generated | is_owner_generated |
      | 11       | 21      | content            | none               | false              |
      | 11       | 50      | solution           | all                | true               |
      | 11       | 60      | content            | all                | true               |
    And the database has the following table "languages":
      | tag |
      | en  |
      | sl  |

  Scenario: Update the default language string
    Given I am the user with id "11"
    When I send a PUT request to "/items/50/strings/default" with the following body:
      """
      {
        "title": "The title",
        "image_url": "http://mysite.com/image.jpg",
        "subtitle": "The subtitle",
        "description": "The description"
      }
      """
    Then the response should be "updated"
    And the table "items_strings" should remain unchanged, regardless of the row with language_tag "en"
    And the table "items_strings" at language_tag "en" should be:
      | item_id | language_tag | title     | image_url                   | subtitle     | description     |
      | 50      | en           | The title | http://mysite.com/image.jpg | The subtitle | The description |

  Scenario: Update the default language string with an image_url > 100 and < 2048 characters.
    Given I am the user with id "11"
    When I send a PUT request to "/items/50/strings/default" with the following body:
      """
      {
        "title": "The title",
        "image_url": "http://mysite.com/image-1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890.jpg",
        "subtitle": "The subtitle",
        "description": "The description"
      }
      """
    Then the response should be "updated"
    And the table "items_strings" should remain unchanged, regardless of the row with language_tag "en"
    And the table "items_strings" at language_tag "en" should be:
      | item_id | language_tag | title     | image_url                                                                                                                        | subtitle     | description     |
      | 50      | en           | The title | http://mysite.com/image-1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890.jpg | The subtitle | The description |

  Scenario: Update the specified language string
    Given I am the user with id "11"
    When I send a PUT request to "/items/50/strings/sl" with the following body:
      """
      {
        "title": "The title",
        "image_url": "http://mysite.com/image.jpg",
        "subtitle": "The subtitle",
        "description": "The description"
      }
      """
    Then the response should be "updated"
    And the table "items_strings" should remain unchanged, regardless of the row with language_tag "sl"
    And the table "items_strings" at language_tag "sl" should be:
      | item_id | language_tag | title     | image_url                   | subtitle     | description     |
      | 50      | sl           | The title | http://mysite.com/image.jpg | The subtitle | The description |

  Scenario: Insert the default language string
    Given I am the user with id "11"
    When I send a PUT request to "/items/60/strings/default" with the following body:
      """
      {
        "title": "The title",
        "image_url": "http://mysite.com/image.jpg",
        "subtitle": "The subtitle",
        "description": "The description"
      }
      """
    Then the response should be "updated"
    And the table "items_strings" should remain unchanged, regardless of the row with item_id "60"
    And the table "items_strings" at item_id "60" should be:
      | item_id | language_tag | title     | image_url                   | subtitle     | description     |
      | 60      | sl           | The title | http://mysite.com/image.jpg | The subtitle | The description |

  Scenario: Insert the specified language string
    Given I am the user with id "11"
    When I send a PUT request to "/items/60/strings/en" with the following body:
      """
      {
        "title": "The title",
        "image_url": "http://mysite.com/image.jpg",
        "subtitle": "The subtitle",
        "description": "The description"
      }
      """
    Then the response should be "updated"
    And the table "items_strings" should remain unchanged, regardless of the row with item_id "60"
    And the table "items_strings" at item_id "60" should be:
      | item_id | language_tag | title     | image_url                   | subtitle     | description     |
      | 60      | en           | The title | http://mysite.com/image.jpg | The subtitle | The description |

  Scenario: Insert the specified language string with nulls
    Given I am the user with id "11"
    When I send a PUT request to "/items/60/strings/en" with the following body:
      """
      {
        "title": "The title",
        "image_url": null,
        "subtitle": null,
        "description": null
      }
      """
    Then the response should be "updated"
    And the table "items_strings" should remain unchanged, regardless of the row with item_id "60"
    And the table "items_strings" at item_id "60" should be:
      | item_id | language_tag | title     | image_url | subtitle | description |
      | 60      | en           | The title | null      | null     | null        |

  Scenario: Insert the specified language string with empty strings
    Given I am the user with id "11"
    When I send a PUT request to "/items/60/strings/en" with the following body:
      """
      {
        "title": "",
        "image_url": "",
        "subtitle": "",
        "description": ""
      }
      """
    Then the response should be "updated"
    And the table "items_strings" should remain unchanged, regardless of the row with item_id "60"
    And the table "items_strings" at item_id "60" should be:
      | item_id | language_tag | title | image_url | subtitle | description |
      | 60      | en           |       |           |          |             |

  Scenario: Update the specified language string with nulls
    Given I am the user with id "11"
    When I send a PUT request to "/items/50/strings/sl" with the following body:
      """
      {
        "title": "The title",
        "image_url": null,
        "subtitle": null,
        "description": null
      }
      """
    Then the response should be "updated"
    And the table "items_strings" should remain unchanged, regardless of the row with language_tag "sl"
    And the table "items_strings" at language_tag "sl" should be:
      | item_id | language_tag | title     | image_url | subtitle | description |
      | 50      | sl           | The title | null      | null     | null        |

  Scenario: Valid without any fields
    Given I am the user with id "11"
    When I send a PUT request to "/items/50/strings/default" with the following body:
      """
      {
      }
      """
    Then the response should be "updated"
    And the table "items_strings" should remain unchanged
