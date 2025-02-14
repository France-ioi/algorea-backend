Feature: Create a user batch - robustness
  Background:
    Given the database has the following table "groups":
      | id | type    | name     | created_at          | require_personal_info_access_approval | require_lock_membership_approval_until | require_watch_approval |
      | 2  | Base    | AllUsers | 2015-08-10 12:34:55 | none                                  | null                                   | 0                      |
      | 3  | Club    | Club     | 2017-08-10 12:34:55 | view                                  | 3030-01-01 00:00:00                    | 1                      |
      | 4  | Friends | Friends  | 2018-08-10 12:34:55 | edit                                  | 2019-01-01 00:00:00                    | 0                      |
    And the database has the following user:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 3        | 21         | memberships           |
      | 4        | 21         | memberships_and_group |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 2               | 21             |
      | 3               | 4              |
      | 3               | 21             |
    And the groups ancestors are computed
    And the database has the following table "user_batch_prefixes":
      | group_prefix | group_id | allow_new | max_users |
      | test         | 3        | 1         | 2         |
      | test2        | 2        | 1         | 2         |
      | test3        | 3        | 0         | 2         |
    And the database has the following table "user_batches_v2":
      | group_prefix | custom_prefix | size |
      | test         | custom        | 1    |
    And the application config is:
      """
      auth:
        loginModuleURL: "https://login.algorea.org"
        clientID: "1"
        clientSecret: "tzxsLyFtJiGnmD6sjZMqSEidVpVsL3hEoSxIXCpI"
      domains:
        -
          domains: [127.0.0.1]
          allUsersGroup: 2
      """

  Scenario: Missing required fields
    Given I am the user with id "21"
    When I send a POST request to "/user-batches" with the following body:
    """
    {}
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Bad Request",
      "error_text": "Invalid input data",
      "errors": {
        "custom_prefix": ["missing field"],
        "group_prefix": ["missing field"],
        "password_length": ["missing field"],
        "postfix_length": ["missing field"],
        "subgroups": ["missing field"]
      }
    }
    """
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Wrong field values
    Given I am the user with id "21"
    When I send a POST request to "/user-batches" with the following body:
    """
    {
      "group_prefix": 123,
      "custom_prefix": "_wrong_",
      "subgroups": [],
      "postfix_length": 2,
      "password_length": 5
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Bad Request",
      "error_text": "Invalid input data",
      "errors": {
        "custom_prefix": ["The custom prefix should only consist of letters/digits/hyphens and be 2-14 characters long"],
        "group_prefix": ["expected type 'string', got unconvertible type 'float64'"],
        "password_length": ["password_length must be 6 or greater"],
        "postfix_length": ["postfix_length must be 3 or greater"],
        "subgroups": ["subgroups must contain at least 1 item"]
      }
    }
    """
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Wrong field values (another set)
    Given I am the user with id "21"
    When I send a POST request to "/user-batches" with the following body:
    """
    {
      "group_prefix": null,
      "custom_prefix": "1",
      "subgroups": [{}],
      "postfix_length": 30,
      "password_length": 51
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Bad Request",
      "error_text": "Invalid input data",
      "errors": {
        "custom_prefix": ["The custom prefix should only consist of letters/digits/hyphens and be 2-14 characters long"],
        "group_prefix": ["should not be null (expected type: string)"],
        "password_length": ["password_length must be 50 or less"],
        "postfix_length": ["postfix_length must be 29 or less"],
        "subgroups[0].count": ["missing field"],
        "subgroups[0].group_id": ["missing field"]
      }
    }
    """
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Wrong field values (one more set)
    Given I am the user with id "21"
    When I send a POST request to "/user-batches" with the following body:
    """
    {
      "group_prefix": "",
      "custom_prefix": "1234567890abcde",
      "subgroups": [{"count": 0, "group_id": null}],
      "postfix_length": null,
      "password_length": null
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "success": false,
      "message": "Bad Request",
      "error_text": "Invalid input data",
      "errors": {
        "custom_prefix": ["The custom prefix should only consist of letters/digits/hyphens and be 2-14 characters long"],
        "password_length": ["should not be null (expected type: int)"],
        "postfix_length": ["should not be null (expected type: int)"],
        "subgroups[0].count": ["count must be 1 or greater"],
        "subgroups[0].group_id": ["should not be null (expected type: int64)"]
      }
    }
    """
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario Outline: Wrong group prefix
    Given I am the user with id "21"
    When I send a POST request to "/user-batches" with the following body:
    """
    {
      "group_prefix": "<group_prefix>",
      "custom_prefix": "1234567890",
      "subgroups": [{"count": 1, "group_id": 4}],
      "postfix_length": 6,
      "password_length": 7
    }
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    Examples:
      | group_prefix |
      | 404          |
      | test2        |
      | test3        |

  Scenario: postfix_length is too small
    Given I am the user with id "21"
    When I send a POST request to "/user-batches" with the following body:
    """
    {
      "group_prefix": "test",
      "custom_prefix": "1234567890",
      "subgroups": [{"count": 16384, "group_id": 4}],
      "postfix_length": 3,
      "password_length": 7
    }
    """
    Then the response code should be 400
    And the response error message should contain "'postfix_length' is too small"
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: subgroups[...].group_id is not a descendant of the prefix group
    Given I am the user with id "21"
    When I send a POST request to "/user-batches" with the following body:
    """
    {
      "group_prefix": "test",
      "custom_prefix": "1234567890",
      "subgroups": [{"count": 1, "group_id": 2}],
      "postfix_length": 3,
      "password_length": 7
    }
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: subgroups[...].group_id is a user
    Given I am the user with id "21"
    When I send a POST request to "/user-batches" with the following body:
    """
    {
      "group_prefix": "test",
      "custom_prefix": "1234567890",
      "subgroups": [{"count": 1, "group_id": 21}],
      "postfix_length": 3,
      "password_length": 7
    }
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: user_batch_prefix.max_users exceeded
    Given I am the user with id "21"
    When I send a POST request to "/user-batches" with the following body:
    """
    {
      "group_prefix": "test",
      "custom_prefix": "1234567890",
      "subgroups": [{"count": 2, "group_id": 4}],
      "postfix_length": 3,
      "password_length": 7
    }
    """
    Then the response code should be 400
    And the response error message should contain "'user_batch_prefix.max_users' exceeded"
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: (group_prefix, custom_prefix) pair already exists
    Given I am the user with id "21"
    When I send a POST request to "/user-batches" with the following body:
    """
    {
      "group_prefix": "test",
      "custom_prefix": "custom",
      "subgroups": [{"count": 1, "group_id": 4}],
      "postfix_length": 3,
      "password_length": 7
    }
    """
    Then the response code should be 400
    And the response error message should contain "'custom_prefix' already exists for the given 'group_prefix'"
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged

  Scenario: Login module failed
    Given I am the user with id "21"
    And the login module "create" endpoint with params "amount=1&language=fr&login_fixed=1&password_length=7&postfix_length=3&prefix=test_12345_" returns 200 with encoded body:
      """
      {"success": false, "error": "some error"}
      """
    When I send a POST request to "/user-batches" with the following body:
    """
    {
      "group_prefix": "test",
      "custom_prefix": "12345",
      "subgroups": [{"count": 1, "group_id": 4}],
      "postfix_length": 3,
      "password_length": 7
    }
    """
    Then the response code should be 500
    And the response error message should contain "Login module failed"
    And the table "user_batches_v2" should remain unchanged
    And the table "users" should remain unchanged
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "groups_ancestors" should remain unchanged
    And logs should contain:
      """
      The login module returned an error for /platform_api/accounts_manager/create: some error
      """
