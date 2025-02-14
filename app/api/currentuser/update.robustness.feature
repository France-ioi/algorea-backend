Feature: Update user data - robustness
  Background:
    Given the database has the following users:
      | group_id | login    | latest_login_at     | latest_activity_at  | registered_at       | default_language |
      | 11       | mohammed | 2019-06-16 21:01:25 | 2019-06-16 22:05:44 | 2019-05-10 10:42:11 | en               |
      | 13       | john     | 2018-06-16 21:01:25 | 2018-06-16 22:05:44 | 2018-05-10 10:42:11 | en               |

  Scenario: invalid default_language
    Given I am the user with id "11"
    When I send a PUT request to "/current-user" with the following body:
      """
      {"default_language": null}
      """
    Then the response code should be 400
    And the response body should be, in JSON:
      """
      {
        "success": false,
        "message": "Bad Request",
        "error_text": "Invalid input data",
        "errors":{
          "default_language":["should not be null (expected type: string)"]
         }
      }
      """
    And the table "users" should remain unchanged
