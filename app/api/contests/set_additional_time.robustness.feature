Feature: Set additional time in the contest for the group (contestSetAdditionalTime) - robustness
  Background:
    Given the database has the following table "groups":
      | id | name    | type  |
      | 12 | Group A | Class |
      | 13 | Group B | Other |
      | 14 | Group C | Other |
    And the database has the following users:
      | group_id | login |
      | 21       | owner |
      | 31       | john  |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_grant_group_access | can_watch_members |
      | 12       | 21         | false                  | true              |
      | 13       | 21         | true                   | true              |
      | 14       | 21         | true                   | false             |
      | 31       | 21         | true                   | true              |
    And the database has the following table "items":
      | id | duration | entry_participant_type | default_language_tag |
      | 50 | 00:00:00 | User                   | fr                   |
      | 60 | null     | User                   | fr                   |
      | 10 | 00:00:02 | User                   | fr                   |
      | 70 | 00:00:03 | User                   | fr                   |
      | 80 | 00:00:04 | Team                   | fr                   |
      | 90 | 00:00:04 | Team                   | fr                   |
      | 95 | 00:00:04 | Team                   | fr                   |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated       | can_grant_view_generated | can_watch_generated |
      | 13       | 50      | content                  | enter                    | result              |
      | 13       | 60      | content_with_descendants | enter                    | result              |
      | 13       | 70      | content                  | enter                    | result              |
      | 21       | 50      | content                  | none                     | result              |
      | 21       | 60      | content                  | enter                    | result              |
      | 21       | 70      | content                  | enter                    | result              |
      | 21       | 80      | content                  | enter                    | result              |
      | 21       | 90      | content                  | enter                    | none                |
      | 21       | 95      | info                     | enter                    | result              |
    And the database has the following table "groups_contest_items":
      | group_id | item_id | additional_time |
      | 13       | 50      | 01:00:00        |
      | 13       | 60      | 01:01:00        |

  Scenario: Wrong item_id
    Given I am the user with id "21"
    When I send a PUT request to "/contests/abc/groups/13/additional-times?seconds=0"
    Then the response code should be 400
    And the response error message should contain "Wrong value for item_id (should be int64)"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: Wrong group_id
    Given I am the user with id "21"
    When I send a PUT request to "/contests/50/groups/abc/additional-times?seconds=0"
    Then the response code should be 400
    And the response error message should contain "Wrong value for group_id (should be int64)"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: Wrong 'seconds'
    Given I am the user with id "21"
    When I send a PUT request to "/contests/50/groups/13/additional-times?seconds=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for seconds (should be int64)"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: 'seconds' is too big
    Given I am the user with id "21"
    When I send a PUT request to "/contests/50/groups/13/additional-times?seconds=3020400"
    Then the response code should be 400
    And the response error message should contain "'seconds' should be between -3020399 and 3020399"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: 'seconds' is too small
    Given I am the user with id "21"
    When I send a PUT request to "/contests/50/groups/13/additional-times?seconds=-3020400"
    Then the response code should be 400
    And the response error message should contain "'seconds' should be between -3020399 and 3020399"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: No such item
    Given I am the user with id "21"
    When I send a PUT request to "/contests/404/groups/13/additional-times?seconds=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: No access to the item
    Given I am the user with id "21"
    When I send a PUT request to "/contests/10/groups/13/additional-times?seconds=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: The item is not a timed contest
    Given I am the user with id "21"
    When I send a PUT request to "/contests/60/groups/13/additional-times?seconds=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: The user is not a contest admin (can_view = info)
    Given I am the user with id "21"
    When I send a PUT request to "/contests/95/groups/13/additional-times?seconds=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: The user is not a contest admin (can_grant_view = none)
    Given I am the user with id "21"
    When I send a PUT request to "/contests/50/groups/13/additional-times?seconds=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: The user is not a contest admin (can_watch = none)
    Given I am the user with id "21"
    When I send a PUT request to "/contests/90/groups/13/additional-times?seconds=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: The user cannot grant access to the group
    Given I am the user with id "21"
    When I send a PUT request to "/contests/70/groups/12/additional-times?seconds=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: The user cannot watch group members
    Given I am the user with id "21"
    When I send a PUT request to "/contests/70/groups/14/additional-times?seconds=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: No such group
    Given I am the user with id "21"
    When I send a PUT request to "/contests/70/groups/404/additional-times?seconds=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged

  Scenario: Team contest and a user
    Given I am the user with id "21"
    When I send a PUT request to "/contests/80/groups/31/additional-times?seconds=0"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "permissions_generated" should remain unchanged
    And the table "groups_contest_items" should remain unchanged
