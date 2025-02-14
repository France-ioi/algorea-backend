Feature: Add a parent-child relation between two groups - robustness

  Background:
    Given the database has the following table "groups":
      | id | name     | type  |
      | 11 | Group A  | Class |
      | 13 | Group B  | Class |
      | 16 | AllUsers | Base  |
      | 18 | User     | User  |
      | 19 | Team     | Team  |
      | 77 | Group C  | Class |
      | 78 | Group D  | Class |
      | 79 | Group E  | Class |
    And the database has the following users:
      | group_id | login   | first_name  | last_name |
      | 21       | owner   | Jean-Michel | Blanquer  |
      | 25       | student | Jane        | Doe       |
      | 27       | admin   | John        | Doe       |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 11       | 21         | memberships_and_group |
      | 13       | 21         | memberships           |
      | 78       | 21         | memberships           |
      | 79       | 21         | none                  |
      | 11       | 25         | memberships           |
      | 11       | 27         | memberships_and_group |
      | 13       | 27         | memberships_and_group |
      | 16       | 27         | memberships_and_group |
      | 18       | 27         | memberships_and_group |
      | 19       | 27         | memberships_and_group |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 13              | 11             |
    And the groups ancestors are computed

  Scenario: Parent group id is wrong
    Given I am the user with id "21"
    When I send a POST request to "/groups/abc/relations/11"
    Then the response code should be 400
    And the response error message should contain "Wrong value for parent_group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Child group id is missing
    Given I am the user with id "21"
    When I send a POST request to "/groups/13/relations/abc"
    Then the response code should be 400
    And the response error message should contain "Wrong value for child_group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: User is a manager of the parent group, but is not a manager of the child group
    Given I am the user with id "21"
    When I send a POST request to "/groups/13/relations/77"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: User is a manager of the child group, but is not a manager of the parent group
    Given I am the user with id "25"
    When I send a POST request to "/groups/13/relations/11"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: User is a manager of the two groups, but doesn't have enough rights on the child group
    Given I am the user with id "21"
    When I send a POST request to "/groups/13/relations/78"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: User is a manager of the two groups, but doesn't have enough rights on the parent group
    Given I am the user with id "21"
    When I send a POST request to "/groups/79/relations/11"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: User does not exist
    Given I am the user with id "404"
    When I send a POST request to "/groups/13/relations/11"
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Child group is AllUsers groups
    Given I am the user with id "27"
    When I send a POST request to "/groups/13/relations/16"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Child group is User
    Given I am the user with id "27"
    When I send a POST request to "/groups/13/relations/18"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Parent group is User
    Given I am the user with id "27"
    When I send a POST request to "/groups/18/relations/11"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Parent group is Team
    Given I am the user with id "27"
    When I send a POST request to "/groups/19/relations/11"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: A group cannot become an ancestor of itself
    Given I am the user with id "27"
    When I send a POST request to "/groups/11/relations/13"
    Then the response code should be 403
    And the response error message should contain "A group cannot become an ancestor of itself"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Parent and child are the same
    Given I am the user with id "27"
    When I send a POST request to "/groups/13/relations/13"
    Then the response code should be 400
    And the response error message should contain "A group cannot become its own parent"
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
