Feature: Update thread
  Background:
    Given the database has the following table "groups":
      | id | name   | type  |
      | 10 | Class  | Class |
      | 11 | School | Class |
      | 12 | Region | Class |
      | 20 | Group  | Class |
      | 30 | Group  | Class |
      | 50 | Group  | Class |
      | 51 | Group  | Class |
      | 60 | Group  | Class |
    And the database has the following users:
      | group_id | login   |
      | 1        | john    |
      | 2        | manager |
      | 3        | jack    |
      | 4        | jess    |
      | 5        | owner   |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 10              | 2              |
      | 10              | 3              |
      | 11              | 10             |
      | 12              | 11             |
      | 20              | 2              |
      | 20              | 3              |
      | 30              | 2              |
      | 30              | 3              |
      | 50              | 51             |
      | 51              | 2              |
      | 51              | 3              |
      | 51              | 4              |
      | 60              | 5              |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id   | default_language_tag | type    |
      | 2000 | en                   | Task    |
      | 2001 | en                   | Task    |
      | 2002 | en                   | Task    |
      | 2003 | en                   | Task    |
      | 2007 | en                   | Task    |
      | 2008 | en                   | Task    |
      | 2009 | en                   | Task    |
      | 2010 | en                   | Task    |
      | 3004 | en                   | Chapter |
      | 3005 | en                   | Chapter |
      | 3006 | en                   | Chapter |
      | 3010 | en                   | Task    |
    And the database has the following table "items_items":
      | parent_item_id | child_item_id | request_help_propagation | child_order |
      | 3004           | 3005          | 1                        | 1           |
      | 3004           | 2007          | 1                        | 2           |
      | 3005           | 3006          | 1                        | 1           |
      | 3005           | 2008          | 1                        | 2           |
      | 3006           | 2009          | 1                        | 1           |
      | 3006           | 2010          | 1                        | 2           |
    And the database has the following table "permissions_granted":
      | group_id | source_group_id | item_id | can_request_help_to | is_owner |
      | 3        | 11              | 2000    | 12                  | 0        |
      | 3        | 11              | 2001    | 12                  | 0        |
      | 5        | 5               | 3010    | null                | 1        |
      | 11       | 11              | 2002    | 12                  | 0        |
      | 11       | 11              | 2003    | 12                  | 0        |
      | 12       | 11              | 3004    | 12                  | 0        |
    And the database has the following table "threads":
      | item_id | participant_id | status | helper_group_id | latest_update_at |
    And the DB time now is "2022-01-01 00:00:00"

  Scenario: Create a thread if it doesn't exist
    Given I am the user with id "3"
    And there is no thread with "item_id=1000,participant_id=3"
    And I am a member of the group with id "100"
    And I can view content of the item 1000
    And I can request help to the group with id "100" on the item with id "1000"
    When I send a PUT request to "/items/1000/participant/3/thread" with the following body:
      """
      {
        "status": "waiting_for_trainer",
        "message_count": 1,
        "helper_group_id": 100
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "1000"
    And the table "threads" at item_id "1000" should be:
      | latest_update_at    | message_count | status              | helper_group_id |
      | 2022-01-01 00:00:00 | 1             | waiting_for_trainer | 100             |

  # To write on a thread, a user must fulfill either of those conditions:
  #  (1) be the participant of the thread
  #  (2) have can_watch>=answer permission on the item AND can_watch_members on the participant
  #  (3) be part of the group the participant has requested help to AND either have can_watch>=answer on the item
  #    OR (have can_watch>=answer on the item AND have a validated result on the item).
  Scenario: Can write to thread condition (1) when status is not set
    Given I am the user with id "1"
    And there is a thread with "item_id=10,participant_id=1"
    And I can view content of the item 10
    When I send a PUT request to "/items/10/participant/1/thread" with the following body:
      """
      {
        "message_count": 1
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "10"
    And the table "threads" at item_id "10" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 1             |

  Scenario: Can write to thread condition (2) when status is not set
    Given I am the user with id "2"
    And there is a thread with "item_id=20,participant_id=3"
    And I can view content of the item 20
    And I have the watch permission set to "answer" on the item 20
    And I am a manager of the group 3 and can watch for submissions from the group and its descendants
    When I send a PUT request to "/items/20/participant/3/thread" with the following body:
      """
      {
        "message_count": 2
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "20"
    And the table "threads" at item_id "20" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 2             |

  Scenario: Can write to thread condition (3) with can_watch>=answer on the item, when status is not set
    Given I am the user with id "2"
    And there is a thread with "item_id=30,participant_id=3"
    And I am part of the helper group of the thread
    And I can view content of the item 30
    And I have the watch permission set to "answer" on the item 30
    When I send a PUT request to "/items/30/participant/3/thread" with the following body:
      """
      {
        "message_count": 3
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "30"
    And the table "threads" at item_id "30" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 3             |

  Scenario: Can write to thread test condition (3) with validated item and can_watch=result permission on the item, when status is not set
    Given I am the user with id "4"
    And there is a thread with "item_id=40,participant_id=3"
    And I am part of the helper group of the thread
    And I have a validated result on the item 40
    And I can view content of the item 40
    And I have the watch permission set to "result" on the item 40
    When I send a PUT request to "/items/40/participant/3/thread" with the following body:
      """
      {
        "message_count": 4
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "40"
    And the table "threads" at item_id "40" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 4             |

  Scenario: Set message_count to 0
    Given I am the user with id "1"
    And there is a thread with "item_id=50,participant_id=1"
    And I can view content of the item 50
    When I send a PUT request to "/items/50/participant/1/thread" with the following body:
      """
      {
        "message_count": 0
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "50"
    And the table "threads" at item_id "50" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 0             |

  Scenario: Should set message_count to 0 if decrement to a negative value
    Given I am the user with id "1"
    And there is a thread with "item_id=60,participant_id=1,message_count=10"
    And I can view content of the item 60
    When I send a PUT request to "/items/60/participant/1/thread" with the following body:
      """
      {
        "message_count_increment": -11
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "60"
    And the table "threads" at item_id "60" should be:
      | latest_update_at    | message_count |
      | 2022-01-01 00:00:00 | 0             |

  Scenario Outline: Should increment message_count by message_count_increments
    Given I am the user with id "1"
    And there is a thread with "item_id=<item_id>,participant_id=1,message_count=10"
    And I can view content of the item <item_id>
    When I send a PUT request to "/items/<item_id>/participant/1/thread" with the following body:
      """
      {
        "message_count_increment": <message_count_increment>
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | message_count   |
      | 2022-01-01 00:00:00 | <message_count> |
    Examples:
      | item_id | message_count_increment | message_count |
      | 70      | 3                       | 13            |
      | 80      | -5                      | 5             |
      | 90      | 0                       | 10            |

  Scenario Outline: Participant of a thread can always switch the thread from open to any other status
    Given I am the user with id "3"
    And there is a thread with "item_id=<item_id>,participant_id=3,status=<old_status>"
    And I can view content of the item <item_id>
    And I have the watch permission set to "none" on the item <item_id>
    When I send a PUT request to "/items/<item_id>/participant/3/thread" with the following body:
      """
      {
        "status": "<status>"
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   |
      | 2022-01-01 00:00:00 | <status> |
    Examples:
      | item_id | old_status              | status                  |
      | 100     | waiting_for_trainer     | waiting_for_participant |
      | 100     | waiting_for_trainer     | closed                  |
      | 110     | waiting_for_participant | waiting_for_trainer     |
      | 110     | waiting_for_participant | closed                  |

  Scenario Outline: A user who has can_watch>=answer on the item AND can_watch_members on the participant can always switch to an open status when thread exists
    Given I am the user with id "2"
    And I can view content of the item <item_id>
    And I have the watch permission set to "answer" on the item <item_id>
    And I am a manager of the group 3 and can watch for submissions from the group and its descendants
    And there is a thread with "item_id=<item_id>,participant_id=3,status=closed"
    When I send a PUT request to "/items/<item_id>/participant/3/thread" with the following body:
      """
      {
        "status": "<status>",
        "helper_group_id": <helper_group_id>
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   |
      | 2022-01-01 00:00:00 | <status> |
    Examples:
      | item_id | status                  | helper_group_id |
      | 160     | waiting_for_participant | 30              |
      | 170     | waiting_for_trainer     | 30              |
      | 180     | waiting_for_participant | 30              |
      | 190     | waiting_for_trainer     | 30              |

  Scenario Outline: A user who has can_watch>=answer on the item AND can_watch_members on the participant can always switch to an open status when thread doesn't exists
    Given I am the user with id "2"
    And I can view content of the item <item_id>
    And I have the watch permission set to "answer" on the item <item_id>
    And I am a manager of the group 3 and can watch for submissions from the group and its descendants
    And there is no thread with "item_id=<item_id>,participant_id=3"
    When I send a PUT request to "/items/<item_id>/participant/3/thread" with the following body:
      """
      {
        "status": "<status>",
        "helper_group_id": <helper_group_id>
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   |
      | 2022-01-01 00:00:00 | <status> |
    Examples:
      | item_id | status                  | helper_group_id |
      | 200     | waiting_for_participant | 30              |
      | 210     | waiting_for_trainer     | 30              |

  Scenario Outline: Can switch to open if part of the group the participant has requested help to AND can_watch>=answer on the item
    Given I am the user with id "4"
    And I can view content of the item <item_id>
    And I have the watch permission set to "answer" on the item <item_id>
    And there is a thread with "item_id=<item_id>,participant_id=3,status=<old_status>,helper_group_id=50"
    When I send a PUT request to "/items/<item_id>/participant/3/thread" with the following body:
      """
      {
        "status": "<status>",
        "helper_group_id": 50
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   |
      | 2022-01-01 00:00:00 | <status> |
    Examples:
      | item_id | old_status              | status                  |
      | 220     | waiting_for_trainer     | waiting_for_participant |
      | 230     | waiting_for_participant | waiting_for_trainer     |

  Scenario Outline: Can switch to open if part of the group the participant has requested help to AND (have can_watch=result permission and a validated result on the item)
    Given I am the user with id "4"
    And I have a validated result on the item <item_id>
    And I can view content of the item <item_id>
    And I have the watch permission set to "result" on the item <item_id>
    And there is a thread with "item_id=<item_id>,participant_id=3,status=<old_status>,helper_group_id=50"
    When I send a PUT request to "/items/<item_id>/participant/3/thread" with the following body:
      """
      {
        "status": "<status>",
        "helper_group_id": 50
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   |
      | 2022-01-01 00:00:00 | <status> |
    Examples:
      | item_id | old_status              | status                  |
      | 240     | waiting_for_trainer     | waiting_for_participant |
      | 250     | waiting_for_participant | waiting_for_trainer     |

  Scenario: If status is open and not provided (no change): update helper_group_id
    Given I am the user with id "2"
    And I can view content of the item 260
    And I have the watch permission set to "answer" on the item 260
    And there is a thread with "item_id=260,participant_id=3,helper_group_id=10"
    When I send a PUT request to "/items/260/participant/3/thread" with the following body:
      """
      {
        "helper_group_id": 20
      }
      """
    Then the response should be "updated"
    And the table "threads" at item_id "260" should be:
      | latest_update_at    | helper_group_id |
      | 2022-01-01 00:00:00 | 20              |

  Scenario Outline: Participant of a thread can switch from non-open to open status when allowed to request help on the item
    Given I am the user with id "3"
    And I can view content of the item <item_id>
    And I have the watch permission set to "none" on the item <item_id>
    And there is a thread with "item_id=<item_id>,participant_id=3,status=closed,helper_group_id=<old_helper_group_id>"
    When I send a PUT request to "/items/<item_id>/participant/3/thread" with the following body:
      """
      {
        "status": "<status>",
        "helper_group_id": <helper_group_id>
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   | helper_group_id   |
      | 2022-01-01 00:00:00 | <status> | <helper_group_id> |
    Examples:
      | item_id | status                  | old_helper_group_id | helper_group_id | comment    |
      | 2000    | waiting_for_trainer     | 3                   | 11              |            |
      | 2001    | waiting_for_participant | 11                  | 12              |            |
      | 2007    | waiting_for_participant | 11                  | 12              | In chapter |
      | 2008    | waiting_for_participant | 11                  | 11              | In chapter |

  Scenario Outline: Participant of a thread can switch from non-open to open status when allowed to request help on the item when thread doesn't exists
    Given I am the user with id "3"
    And I can view content of the item <item_id>
    And I have the watch permission set to "none" on the item <item_id>
    And there is no thread with "item_id=<item_id>,participant_id=3"
    When I send a PUT request to "/items/<item_id>/participant/3/thread" with the following body:
      """
      {
        "status": "<status>",
        "helper_group_id": <helper_group_id>
      }
      """
    Then the response should be "updated"
    And the table "threads" should remain unchanged, regardless of the row with item_id "<item_id>"
    And the table "threads" at item_id "<item_id>" should be:
      | latest_update_at    | status   | helper_group_id   |
      | 2022-01-01 00:00:00 | <status> | <helper_group_id> |
    Examples:
      | item_id | status                  | helper_group_id | comment    |
      | 2002    | waiting_for_trainer     | 11              |            |
      | 2003    | waiting_for_participant | 12              |            |
      | 2009    | waiting_for_participant | 12              | In chapter |
      | 2010    | waiting_for_participant | 11              | In chapter |

  Scenario: Participant who can request help on region can request help on class
    Given I am the user with id "3"
    And there is no thread with "item_id=270,participant_id=3"
    And I can view content of the item 270
    And I can request help to the group with id "12" on the item with id "270"
    When I send a PUT request to "/items/270/participant/3/thread" with the following body:
      """
      {
        "status": "waiting_for_trainer",
        "helper_group_id": 10
      }
      """
    Then the response should be "updated"
    And the table "threads" at item_id "270" should be:
      | latest_update_at    | helper_group_id |
      | 2022-01-01 00:00:00 | 10              |

  Scenario: The owner of a thread can request help to himself
    Given I am the user with id "5"
    And there is no thread with "item_id=3010,participant_id=5"
    When I send a PUT request to "/items/3010/participant/5/thread" with the following body:
      """
      {
        "status": "waiting_for_trainer",
        "helper_group_id": 5
      }
      """
    Then the response should be "updated"
    And the table "threads" at item_id "3010" should be:
      | latest_update_at    | helper_group_id |
      | 2022-01-01 00:00:00 | 5               |

  Scenario: The owner of a thread can request help to a visible group
    Given I am the user with id "5"
    And there is no thread with "item_id=3010,participant_id=5"
    When I send a PUT request to "/items/3010/participant/5/thread" with the following body:
      """
      {
        "status": "waiting_for_trainer",
        "helper_group_id": 60
      }
      """
    Then the response should be "updated"
    And the table "threads" at item_id "3010" should be:
      | latest_update_at    | helper_group_id |
      | 2022-01-01 00:00:00 | 60              |
