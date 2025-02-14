Feature: Remove members from a group (groupRemoveMembers)
  Background:
    Given the database has the following table "groups":
      | id | type |
      | 13 | Club |
    And the database has the following users:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
      | 11       | user  | John        | Doe       |
      | 31       | jane  | Jane        | Doe       |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 13              | 11             |
      | 13              | 21             |
    And the groups ancestors are computed
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 13       | 31         | none                  |
      | 13       | 21         | memberships_and_group |
      | 31       | 31         | memberships_and_group |

  Scenario: Fails when the user is not a manager of the parent group
    Given I am the user with id "11"
    When I send a DELETE request to "/groups/13/members?user_ids=1,2"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the user is a manager of the parent group, but doesn't have enough permissions on it
    Given I am the user with id "31"
    When I send a DELETE request to "/groups/13/members?user_ids=1,2"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the user has enough permissions on the group, but the group is a user
    Given I am the user with id "31"
    When I send a DELETE request to "/groups/31/members?user_ids=1,2"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the user doesn't exist
    Given I am the user with id "404"
    When I send a DELETE request to "/groups/13/members?user_ids=1,2"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the parent group id is wrong
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/abc/members?user_ids=1,2"
    Then the response code should be 400
    And the response error message should contain "Wrong value for group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when user_ids is wrong
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/13/members?user_ids=1,abc,2"
    Then the response code should be 400
    And the response error message should contain "Unable to parse one of the integers given as query args (value: 'abc', param: 'user_ids')"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
