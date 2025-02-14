Feature: Remove user batch (userBatchRemove)
  Background:
    Given the database has the following table "groups":
      | id | name                      | type  | require_lock_membership_approval_until |
      | 13 | class                     | Class | 9999-12-31 23:59:59                    |
      | 21 | user                      | User  | null                                   |
      | 23 | test_custom_another_user  | User  | null                                   |
      | 22 | test_custom_user          | User  | null                                   |
      | 24 | test_custom1_user         | User  | null                                   |
      | 25 | test_custom1_another_user | User  | null                                   |
      | 26 | test1_custom_user         | User  | null                                   |
      | 27 | test1_custom_another_user | User  | null                                   |
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
      | 13       | 21         | memberships |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | lock_membership_approved_at |
      | 13              | 21             | null                        |
      | 13              | 22             | 2019-05-30 11:00:00         |
    And the groups ancestors are computed
    And the database has the following table "user_batch_prefixes":
      | group_prefix | group_id | allow_new |
      | test         | 13       | 1         |
      | test1        | 13       | 1         |
      | test2        | 13       | 0         |
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

  Scenario: Remove a user batch
    Given I am the user with id "21"
    And the login module "delete" endpoint with params "prefix=test_custom_" returns 200 with encoded body:
      """
      {"success": true}
      """
    When I send a DELETE request to "/user-batches/test/custom"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {"success": true, "message": "deleted"}
    """
    And the table "user_batches_v2" should remain unchanged, regardless of the rows with group_prefix "test"
    And the table "user_batches_v2" at group_prefix "test" should be:
      | group_prefix | custom_prefix | size | creator_id |
      | test         | custom1       | 200  | 13         |
    And the table "users" should remain unchanged, regardless of the rows with login "test_custom_user,test_custom_another_user"
    And the table "users" should not contain login "test_custom_user,test_custom_another_user"
    And the table "groups" should remain unchanged, regardless of the rows with name "test_custom_user,test_custom_another_user"
    And the table "groups" should not contain name "test_custom_user,test_custom_another_user"
