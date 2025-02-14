Feature: Remove a direct parent-child relation between two groups - robustness
  Background:
    Given the database has the following table "groups":
      | id | name     | type  |
      | 11 | Group A  | Class |
      | 13 | Group B  | Class |
      | 14 | Group C  | Class |
      | 15 | Team     | Team  |
      | 16 | Group D  | Class |
      | 22 | Group    | Class |
      | 53 | AllUsers | Base  |
      | 55 | User     | User  |
    And the database has the following users:
      | group_id | login   | first_name  | last_name |
      | 21       | owner   | Jean-Michel | Blanquer  |
      | 23       | teacher | John        | Smith     |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 13       | 21         | memberships |
      | 14       | 21         | memberships |
      | 15       | 21         | memberships |
      | 16       | 21         | memberships |
      | 22       | 21         | memberships |
      | 53       | 21         | none        |
      | 55       | 21         | memberships |
      | 11       | 23         | none        |
      | 55       | 23         | none        |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | expires_at          |
      | 11              | 55             | 9999-12-31 23:59:59 |
      | 13              | 11             | 9999-12-31 23:59:59 |
      | 13              | 55             | 9999-12-31 23:59:59 |
      | 15              | 55             | 9999-12-31 23:59:59 |
      | 16              | 11             | 2019-05-30 11:00:00 |
      | 22              | 13             | 9999-12-31 23:59:59 |
      | 55              | 14             | 9999-12-31 23:59:59 |
    And the groups ancestors are computed

  Scenario: Parent group id is wrong
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/abc/relations/11"
    Then the response code should be 400
    And the response error message should contain "Wrong value for parent_group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Child group id is missing
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/13/relations/abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for child_group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: delete_orphans is wrong
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/13/relations/11?delete_orphans=abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for delete_orphans (should have a boolean value (0 or 1))"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: User is a manager of the child group, but is not a manager of the parent group
    Given I am the user with id "23"
    When I send a DELETE request to "/groups/13/relations/11"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: User is a manager of the two groups, but doesn't have enough rights on the parent group
    Given I am the user with id "23"
    When I send a DELETE request to "/groups/11/relations/11"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: User does not exist
    Given I am the user with id "404"
    When I send a DELETE request to "/groups/13/relations/11"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Child group is AllUsers group
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/13/relations/53"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Child group is User
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/13/relations/55"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Parent group is User
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/55/relations/14"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Parent group is Team
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/15/relations/55"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Parent and child are the same
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/13/relations/13"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Relation doesn't exist
    Given I am the user with id "21"
    When I send a DELETE request to "/groups/16/relations/11"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
