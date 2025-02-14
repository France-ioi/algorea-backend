Feature: Reject group requests - robustness
  Background:
    Given the database has the following table "groups":
      | id  | type  |
      | 13  | Club  |
      | 14  | Class |
      | 31  | Class |
      | 111 | User  |
      | 121 | Team  |
      | 122 | User  |
      | 123 | User  |
      | 131 | User  |
      | 141 | User  |
    And the database has the following users:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
      | 11       | user  | John        | Doe       |
      | 12       | jane  | Jane        | Doe       |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 12       | 12         | memberships |
      | 13       | 21         | memberships |
      | 13       | 12         | none        |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 13              | 111            |
      | 13              | 121            |
      | 13              | 123            |
    And the groups ancestors are computed

  Scenario: Fails when the user is not a manager of the parent group
    Given I am the user with id "11"
    When I send a POST request to "/groups/13/join-requests/reject?group_ids=31,141,21,11,13"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the user is a manager of the parent group, but doesn't have enough rights to manage memberships
    Given I am the user with id "12"
    When I send a POST request to "/groups/13/join-requests/reject?group_ids=31,141,21,11,13"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the user has enough rights to manage memberships, but the group is a user
    Given I am the user with id "12"
    When I send a POST request to "/groups/12/join-requests/reject?group_ids=31,141,21,11,13"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the user doesn't exist
    Given I am the user with id "404"
    When I send a POST request to "/groups/13/join-requests/reject?group_ids=31,141,21,11,13"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the parent group id is wrong
    Given I am the user with id "21"
    When I send a POST request to "/groups/abc/join-requests/reject?group_ids=31,141,21,11,13"
    Then the response code should be 400
    And the response error message should contain "Wrong value for parent_group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when group_ids is wrong
    Given I am the user with id "21"
    When I send a POST request to "/groups/13/join-requests/reject?group_ids=31,abc,11,13"
    Then the response code should be 400
    And the response error message should contain "Unable to parse one of the integers given as query args (value: 'abc', param: 'group_ids')"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
