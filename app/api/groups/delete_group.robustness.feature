Feature: Delete a group - robustness
  Background:
    Given the database has the following table "groups":
      | id | name    | type  |
      | 11 | Group A | Class |
      | 55 | User    | User  |
    And the database has the following users:
      | group_id | login   | first_name  | last_name |
      | 21       | owner   | Jean-Michel | Blanquer  |
      | 23       | teacher | John        | Smith     |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 11       | 21         | memberships_and_group |
      | 11       | 23         | memberships           |
      | 55       | 23         | memberships_and_group |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | expires_at          |
      | 11              | 55             | 9999-12-31 23:59:59 |
    And the groups ancestors are computed

  Scenario: Group id is invalid
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: User doesn't have enough rights on the group
    Given I am the user with id "23"
    When I send a DELETE request to "/groups/11"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: User does not exist
    Given I am the user with id "404"
    When I send a DELETE request to "/groups/11"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: The group is User
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/55"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: The group doesn't exist
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/404"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: The group has a child
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/11"
    Then the response code should be 404
    And the response error message should contain "The group must be empty"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
