Feature: Get group by name (contestGetGroupByName) - robustness
  Background:
    Given the database has the following table 'users':
      | ID | sLogin | idGroupSelf | idGroupOwned |
      | 1  | owner  | 21          | 22           |
    And the database has the following table 'groups':
      | ID | sName      |
      | 12 | Group A    |
      | 13 | Group B    |
    And the database has the following table 'groups_ancestors':
      | idGroupAncestor | idGroupChild | bIsSelf |
      | 12              | 12           | 1       |
      | 13              | 13           | 1       |
      | 21              | 21           | 1       |
      | 22              | 13           | 0       |
      | 22              | 22           | 1       |
    And the database has the following table 'items':
      | ID |
      | 50 |
      | 60 |
      | 10 |
      | 70 |
    And the database has the following table 'groups_items':
      | idGroup | idItem | sCachedPartialAccessDate | sCachedGrayedAccessDate | sCachedFullAccessDate |
      | 13      | 50     | 2017-05-29T06:38:38Z     | null                    | null                  |
      | 13      | 60     | null                     | 2017-05-29T06:38:38Z    | null                  |
      | 13      | 70     | null                     | null                    | 2017-05-29T06:38:38Z  |

  Scenario: Wrong item_id
    Given I am the user with ID "1"
    When I send a GET request to "/contests/abc/group-by-name?name=Group%20B"
    Then the response code should be 400
    And the response error message should contain "Wrong value for item_id (should be int64)"

  Scenario: name is missing
    Given I am the user with ID "1"
    When I send a GET request to "/contests/50/group-by-name"
    Then the response code should be 400
    And the response error message should contain "Missing name"

  Scenario: No such item
    Given I am the user with ID "1"
    When I send a GET request to "/contests/90/group-by-name?name=Group%20B"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: No access to the item
    Given I am the user with ID "1"
    When I send a GET request to "/contests/10/group-by-name?name=Group%20B"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: The group is not owned by the user
    Given I am the user with ID "1"
    When I send a GET request to "/contests/70/group-by-name?name=Group%20A"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: No such group (case)
    Given I am the user with ID "1"
    When I send a GET request to "/contests/70/group-by-name?name=Group%20b"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: No such group (space)
    Given I am the user with ID "1"
    When I send a GET request to "/contests/70/group-by-name?name=Group%20B%20"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
