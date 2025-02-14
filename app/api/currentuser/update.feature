Feature: Update user data
  Scenario: Update an existing user
    Given I am the user with id "11"
    And the database has the following users:
      | group_id | login    | latest_login_at     | latest_activity_at  | registered_at       | default_language |
      | 11       | mohammed | 2019-06-16 21:01:25 | 2019-06-16 22:05:44 | 2019-05-10 10:42:11 | en               |
      | 13       | john     | 2018-06-16 21:01:25 | 2018-06-16 22:05:44 | 2018-05-10 10:42:11 | en               |
    When I send a PUT request to "/current-user" with the following body:
      """
      {"default_language": "sl"}
      """
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "success": true,
        "message": "updated"
      }
      """
    And the table "users" should remain unchanged, regardless of the row with group_id "11"
    And the table "users" at group_id "11" should be:
      | group_id | latest_login_at     | latest_activity_at  | registered_at       | default_language   |
      | 11       | 2019-06-16 21:01:25 | 2019-06-16 22:05:44 | 2019-05-10 10:42:11 | sl                 |
