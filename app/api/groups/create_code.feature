Feature: Create a new code for the given group

  Background:
    Given the database has the following table "groups":
      | id | name    | description     | created_at          | type  | code       | code_lifetime | code_expires_at     |
      | 11 | Group A | Group A is here | 2019-02-06 09:26:40 | Class | ybqybxnlyo | 3600          | 2017-10-13 05:39:48 |
      | 13 | Group B | Group B is here | 2019-03-06 09:26:40 | Class | 3456789abc | 3600          | 2017-10-14 05:39:48 |
    And the database has the following user:
      | group_id | login | first_name  | last_name | default_language |
      | 21       | owner | Jean-Michel | Blanquer  | fr               |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 13       | 21         | memberships |

  Scenario: User is a manager of the group
    Given I am the user with id "21"
    And the generated group code is "newpassword"
    When I send a POST request to "/groups/13/code"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {"code":"newpassword"}
    """
    And the table "groups" should remain unchanged, regardless of the row with id "13"
    And the table "groups" at id "13" should be:
      | id | name    | description     | created_at          | type  | code        | code_lifetime | code_expires_at     |
      | 13 | Group B | Group B is here | 2019-03-06 09:26:40 | Class | newpassword | 3600          | 2017-10-14 05:39:48 |
