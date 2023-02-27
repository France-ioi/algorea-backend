Feature: Update thread
  Background:
    Given the database has the following table 'groups':
      | id | name    | type  |
      | 1  | john    | User  |
      | 2  | manager | User  |
      | 3  | jack    | User  |
      | 4  | jess    | User  |
      | 10 | Groupe  | Class |
    And the database has the following table 'users':
      | login   | group_id |
      | john    | 1        |
      | manager | 2        |
      | jack    | 3        |
      | jess    | 4        |
    And the database has the following table 'groups_groups':
      | parent_group_id | child_group_id |
      | 10              | 3              |
      | 10              | 4              |
    And the groups ancestors are computed
    And the database has the following table 'group_managers':
      | group_id | manager_id | can_watch_members |
      | 10       | 2          | true              |
    And the database has the following table 'items':
      | id   | default_language_tag |
      | 10   | en                   |
      | 20   | en                   |
      | 30   | en                   |
      | 40   | en                   |
      | 50   | en                   |
      | 60   | en                   |
      | 70   | en                   |
      | 80   | en                   |
      | 90   | en                   |
      | 100  | en                   |
      | 110  | en                   |
      | 120  | en                   |
      | 130  | en                   |
      | 140  | en                   |
      | 150  | en                   |
      | 160  | en                   |
      | 170  | en                   |
      | 180  | en                   |
      | 190  | en                   |
      | 200  | en                   |
      | 210  | en                   |
      | 220  | en                   |
      | 230  | en                   |
      | 240  | en                   |
      | 250  | en                   |
      | 1000 | en                   |
    And the database has the following table 'permissions_generated':
      | group_id | item_id | can_watch_generated |
      | 2        | 20      | answer              |
      | 4        | 30      | answer              |
      | 1        | 100     | none                |
      | 1        | 110     | none                |
      | 1        | 120     | none                |
      | 1        | 130     | none                |
      | 2        | 160     | answer              |
      | 2        | 170     | answer_with_grant   |
      | 2        | 180     | answer              |
      | 2        | 190     | answer              |
      | 2        | 200     | answer_with_grant   |
      | 2        | 210     | answer              |
      | 4        | 220     | answer              |
      | 4        | 230     | answer_with_grant   |
    And the database has the following table 'results':
      | attempt_id | participant_id | item_id | validated_at        |
      | 0          | 4              | 40      | 2020-01-01 00:00:00 |
      | 0          | 4              | 240     | 2020-01-01 00:00:00 |
      | 0          | 4              | 250     | 2020-01-01 00:00:00 |
    And the database has the following table 'threads':
      | item_id | participant_id | status                  | helper_group_id | latest_update_at    | message_count |
      | 10      | 1              | waiting_for_trainer     | 1               | 2020-01-01 00:00:00 | 1             |
      | 20      | 3              | waiting_for_trainer     | 3               | 2020-01-01 00:00:00 | 1             |
      | 30      | 3              | waiting_for_trainer     | 10              | 2020-01-01 00:00:00 | 1             |
      | 40      | 3              | waiting_for_trainer     | 10              | 2020-01-01 00:00:00 | 1             |
      | 50      | 1              | waiting_for_trainer     | 1               | 2020-01-01 00:00:00 | 1             |
      | 60      | 1              | waiting_for_trainer     | 1               | 2020-01-01 00:00:00 | 10            |
      | 70      | 1              | waiting_for_trainer     | 1               | 2020-01-01 00:00:00 | 10            |
      | 80      | 3              | waiting_for_trainer     | 3               | 2020-01-01 00:00:00 | 10            |
      | 90      | 3              | waiting_for_trainer     | 3               | 2020-01-01 00:00:00 | 10            |
      | 100     | 3              | waiting_for_trainer     | 3               | 2020-01-01 00:00:00 | 10            |
      | 110     | 3              | waiting_for_participant | 3               | 2020-01-01 00:00:00 | 10            |
      | 120     | 3              | closed                  | 3               | 2020-01-01 00:00:00 | 10            |
      | 130     | 3              | closed                  | 3               | 2020-01-01 00:00:00 | 10            |
      | 220     | 3              | waiting_for_trainer     | 10              | 2020-01-01 00:00:00 | 10            |
      | 230     | 3              | waiting_for_participant | 10              | 2020-01-01 00:00:00 | 10            |
      | 240     | 3              | waiting_for_trainer     | 10              | 2020-01-01 00:00:00 | 10            |
      | 250     | 3              | waiting_for_participant | 10              | 2020-01-01 00:00:00 | 10            |
    And the time now is "2022-01-01T00:00:00Z"

  Scenario: Create a thread if it doesn't exist
    Given I am the user with id "1"
    When I send a PUT request to "/items/1000/participant/1/thread" with the following body:
      """
      {
        "message_count": 1
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "1000"
    And the table "threads" at item_id "1000" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 1             |

  # To write on a thread, a user must fulfill either of those conditions:
  #  (1) be the participant of the thread
  #  (2) have can_watch>=answer permission on the item AND can_watch_members on the participant
  #  (3) be part of the group the participant has requested help to AND either have can_watch>=answer on the item
  #    OR have validated the item.
  Scenario: Can write to thread condition (1) when status is not set
    Given I am the user with id "1"
    When I send a PUT request to "/items/10/participant/1/thread" with the following body:
      """
      {
        "message_count": 1
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "10"
    And the table "threads" at item_id "10" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 1             |

  Scenario: Can write to thread condition (2) when status is not set
    Given I am the user with id "2"
    When I send a PUT request to "/items/20/participant/3/thread" with the following body:
      """
      {
        "message_count": 2
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "20"
    And the table "threads" at item_id "20" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 2             |

  Scenario: Can write to thread condition (3) with can_watch>=answer on the item, when status is not set
    Given I am the user with id "2"
    When I send a PUT request to "/items/20/participant/3/thread" with the following body:
      """
      {
        "message_count": 3
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "20"
    And the table "threads" at item_id "20" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 3             |

  Scenario: Can write to thread test condition (3) with validated item, when status is not set
    Given I am the user with id "4"
    When I send a PUT request to "/items/40/participant/3/thread" with the following body:
      """
      {
        "message_count": 4
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "40"
    And the table "threads" at item_id "40" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 4             |

  Scenario: Set message_count to 0
    Given I am the user with id "1"
    When I send a PUT request to "/items/50/participant/1/thread" with the following body:
      """
      {
        "message_count": 0
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "50"
    And the table "threads" at item_id "50" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 0             |

  Scenario: Should set message_count to 0 if decrement to a negative value
    Given I am the user with id "1"
    When I send a PUT request to "/items/60/participant/1/thread" with the following body:
      """
      {
        "message_count_increment": -11
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "60"
    And the table "threads" at item_id "60" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 0             |

  Scenario Outline: Should increment message_count by message_count_increments
    Given I am the user with id "1"
    When I send a PUT request to "/items/<item_id>/participant/1/thread" with the following body:
      """
      {
        "message_count_increment": <message_count_increment>
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | message_count   |
      | 2022-01-01 00:00:00 | <message_count> |
    Examples:
      | item_id | message_count_increment | message_count |
      | 70      | 3                       | 13            |
      | 80      | -5                      | 5             |
      | 90      | 0                       | 10            |

  Scenario Outline: Participant of a thread can always switch the thread from open to any other status
    Given I am the user with id "1"
    When I send a PUT request to "/items/<item_id>/participant/1/thread" with the following body:
      """
      {
        "status": "<status>"
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   |
      | 2022-01-01 00:00:00 | <status> |
    Examples:
      | item_id | status                  |
      | 100     | waiting_for_participant |
      | 100     | closed                  |
      | 110     | waiting_for_trainer     |
      | 110     | closed                  |

    # TODO: Should have request_help on the item once forum permissions are merged
  Scenario Outline: Participant of a thread can switch from non-open status when allowed to request help on the item
    Given I am the user with id "1"
    When I send a PUT request to "/items/<item_id>/participant/1/thread" with the following body:
      """
      {
        "status": "<status>"
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   |
      | 2022-01-01 00:00:00 | <status> |
    Examples:
      | item_id | status                  | comment               |
      | 120     | waiting_for_trainer     |                       |
      | 130     | waiting_for_participant |                       |
      | 140     | waiting_for_trainer     | Doesn't exist: Create |
      | 150     | waiting_for_participant | Doesn't exist: Create |

  Scenario Outline: A user who has can_watch>=answer on the item AND can_watch_members on the participant can always switch to an open status
    Given I am the user with id "2"
    When I send a PUT request to "/items/<item_id>/participant/1/thread" with the following body:
      """
      {
        "status": "<status>"
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   |
      | 2022-01-01 00:00:00 | <status> |
    Examples:
      | item_id | status                  | comment                 |
      | 160     | waiting_for_participant |                         |
      | 170     | waiting_for_trainer     |                         |
      | 180     | waiting_for_participant |                         |
      | 190     | waiting_for_trainer     |                         |
      | 200     | waiting_for_participant | # Doesn't exist: Create |
      | 210     | waiting_for_trainer     | # Doesn't exist: Create |

  Scenario Outline: Can switch to open if part of the group the participant has requested help to AND can_watch>=answer on the item
    Given I am the user with id "4"
    When I send a PUT request to "/items/<item_id>/participant/3/thread" with the following body:
      """
      {
        "status": "<status>"
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   |
      | 2022-01-01 00:00:00 | <status> |
    Examples:
      | item_id | status                  |
      | 220     | waiting_for_participant |
      | 230     | waiting_for_trainer     |

  Scenario Outline: >
      Can switch to open if:
        - part of the group the participant has requested help to AND
        - have validated the item AND
        - thread open
    Given I am the user with id "4"
    When I send a PUT request to "/items/<item_id>/participant/3/thread" with the following body:
      """
      {
        "status": "<status>"
      }
      """
    Then the response should be "updated"
    And the table "threads" should stay unchanged but the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   |
      | 2022-01-01 00:00:00 | <status> |
    Examples:
      | item_id | status                  |
      | 240     | waiting_for_participant |
      | 250     | waiting_for_trainer     |

