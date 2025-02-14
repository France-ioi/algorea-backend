Feature: Delete the current user - robustness
  Background:
    Given the DB time now is "2019-08-09 23:59:59"
    And the database has the following table "groups":
      | id | type  | name      | require_lock_membership_approval_until |
      | 2  | Base  | AllUsers  | null                                   |
      | 4  | Base  | TempUsers | null                                   |
      | 21 | User  | user      | null                                   |
      | 50 | Class | Our class | 2019-08-10 00:00:00                    |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | lock_membership_approved_at |
      | 2               | 4              | null                        |
      | 2               | 21             | null                        |
      | 50              | 21             | 2019-05-30 11:00:00         |
    And the groups ancestors are computed
    And the database has the following user:
      | group_id | temp_user | login | login_id |
      | 21       | 0         | user  | 1234567  |
    And the application config is:
      """
      auth:
        loginModuleURL: "https://login.algorea.org"
        clientID: "1"
        clientSecret: "tzxsLyFtJiGnmD6sjZMqSEidVpVsL3hEoSxIXCpI"
      """

  Scenario: User cannot delete himself right now
    Given I am the user with id "21"
    When I send a DELETE request to "/current-user"
    Then the response code should be 403
    And the response error message should contain "You cannot delete yourself right now"
    And logs should contain:
      """
      A user with group_id = 21 tried to delete himself, but he is a member of a group with lock_user_deletion_until >= NOW()
      """
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Login module fails
    Given I am the user with id "21"
    And the DB time now is "2019-08-10 00:00:00"
    And the login module "unlink_client" endpoint for user id "1234567" returns 200 with encoded body:
      """
      {"success": false, "error": "some error"}
      """
    When I send a DELETE request to "/current-user"
    Then the response code should be 500
    And the response error message should contain "Login module failed"
    And the table "users" should be empty
    And the table "groups" should be:
      | id | type  | name      | require_lock_membership_approval_until |
      | 2  | Base  | AllUsers  | null                                   |
      | 4  | Base  | TempUsers | null                                   |
      | 50 | Class | Our class | 2019-08-10 00:00:00                    |
    And the table "groups_groups" should be:
      | parent_group_id | child_group_id |
      | 2               | 4              |
    And the table "groups_ancestors" should be:
      | ancestor_group_id | child_group_id | is_self |
      | 2                 | 2              | true    |
      | 2                 | 4              | false   |
      | 4                 | 4              | true    |
      | 50                | 50             | true    |
    And logs should contain:
      """
      The login module returned an error for /platform_api/accounts_manager/unlink_client: some error
      """
