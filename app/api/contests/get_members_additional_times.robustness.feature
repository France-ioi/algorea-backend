Feature: Get additional times for a group of users/teams on a contest (contestListMembersAdditionalTime) - robustness
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
      | ID | sDuration |
      | 50 | 00:00:00  |
      | 60 | null      |
      | 10 | 00:00:02  |
      | 70 | 00:00:03  |
    And the database has the following table 'groups_items':
      | idGroup | idItem | sCachedPartialAccessDate | sCachedGrayedAccessDate | sCachedFullAccessDate | sCachedAccessSolutionsDate |
      | 13      | 50     | 2017-05-29T06:38:38Z     | null                    | null                  | null                       |
      | 13      | 60     | null                     | 2017-05-29T06:38:38Z    | null                  | null                       |
      | 13      | 70     | null                     | null                    | 2017-05-29T06:38:38Z  | null                       |
      | 21      | 50     | null                     | null                    | null                  | null                       |
      | 21      | 60     | null                     | null                    | 2018-05-29T06:38:38Z  | null                       |
      | 21      | 70     | null                     | null                    | 2018-05-29T06:38:38Z  | null                       |

  Scenario: Wrong item_id
    Given I am the user with ID "1"
    When I send a GET request to "/contests/abc/groups/13/members/additional-times"
    Then the response code should be 400
    And the response error message should contain "Wrong value for item_id (should be int64)"

  Scenario: No such item
    Given I am the user with ID "1"
    When I send a GET request to "/contests/90/groups/13/members/additional-times"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: No access to the item
    Given I am the user with ID "1"
    When I send a GET request to "/contests/10/groups/13/members/additional-times"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: The item is not a timed contest
    Given I am the user with ID "1"
    When I send a GET request to "/contests/60/groups/13/members/additional-times"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: The user is not a contest admin
    Given I am the user with ID "1"
    When I send a GET request to "/contests/50/groups/13/members/additional-times"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: Wrong group_id
    Given I am the user with ID "1"
    When I send a GET request to "/contests/70/groups/abc/members/additional-times"
    Then the response code should be 400
    And the response error message should contain "Wrong value for group_id (should be int64)"

  Scenario: The group is not owned by the user
    Given I am the user with ID "1"
    When I send a GET request to "/contests/70/groups/12/members/additional-times"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: No such group
    Given I am the user with ID "1"
    When I send a GET request to "/contests/70/groups/404/members/additional-times"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"

  Scenario: Wrong sort
    Given I am the user with ID "1"
    When I send a GET request to "/contests/70/groups/13/members/additional-times?sort=title"
    Then the response code should be 400
    And the response error message should contain "Unallowed field in sorting parameters: "title""

