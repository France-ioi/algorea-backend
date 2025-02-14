Feature: Remove user batch (userBatchRemove) - robustness
  Background:
    Given the database has the following table "groups":
      | id | name   | type  | require_lock_membership_approval_until |
      | 13 | class  | Class | null                                   |
      | 14 | class2 | Class | 9999-12-31 23:59:59                    |
    And the database has the following users:
      | group_id | login                     |
      | 21       | owner                     |
      | 22       | test_custom_user          |
      | 23       | test_custom_another_user  |
      | 24       | test_custom1_user         |
      | 25       | test_custom1_another_user |
      | 26       | test1_custom_user         |
      | 27       | test1_custom_another_user |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 13       | 13         | memberships |
      | 13       | 22         | none        |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | lock_membership_approved_at |
      | 13              | 21             | null                        |
      | 14              | 22             | 2019-05-30 11:00:00         |
    And the groups ancestors are computed
    And the database has the following table "user_batch_prefixes":
      | group_prefix | group_id | allow_new |
      | test         | 13       | 1         |
      | test1        | 13       | 1         |
      | test2        | 13       | 0         |
      | test3        | 21       | 1         |
      | test4        | 14       | 1         |
    And the database has the following table "user_batches_v2":
      | group_prefix | custom_prefix | size | creator_id |
      | test         | custom        | 100  | null       |
      | test         | custom1       | 200  | 13         |
      | test1        | custom        | 300  | 21         |
    And the application config is:
      """
      auth:
        loginModuleURL: "https://login.algorea.org"
        clientID: "1"
        clientSecret: "tzxsLyFtJiGnmD6sjZMqSEidVpVsL3hEoSxIXCpI"
      """

  Scenario Outline: User batch doesn't exist
    Given I am the user with id "21"
    When I send a DELETE request to "/user-batches/<group_prefix>/<custom_prefix>"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
  Examples:
    | group_prefix | custom_prefix |
    | unknown      | custom        |
    | test         | unknown       |

  Scenario: The current user cannot manage the group linked to the batch prefix
    Given I am the user with id "22"
    When I send a DELETE request to "/user-batches/test/custom"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged

  Scenario: There are some users with locked membership that the current user cannot manage
    Given I am the user with id "21"
    When I send a DELETE request to "/user-batches/test/custom"
    Then the response code should be 422
    And the response error message should contain "There are users with locked membership"
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And logs should contain:
      """
      User with group_id = 21 failed to delete a user batch because of locked membership (group_prefix = 'test', custom_prefix = 'custom')
      """

  Scenario: Login moudule fails
    Given I am the user with id "21"
    And the login module "delete" endpoint with params "prefix=test1_custom_" returns 200 with encoded body:
      """
      {"success": false, "error": "some error"}
      """
    When I send a DELETE request to "/user-batches/test1/custom"
    Then the response code should be 500
    And the response error message should contain "Login module failed"
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And logs should contain:
      """
      The login module returned an error for /platform_api/accounts_manager/delete: some error
      """
