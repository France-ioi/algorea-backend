Feature: Set users.notifications_read_at to NOW() for the current user
  Background:
    Given the database has the following users:
      | group_id | login | notifications_read_at |
      | 1        | user  | null                  |
      | 2        | admin | 2017-02-21 06:38:38   |

  Scenario: Successfully send a request
    Given I am the user with id "1"
    When I send a PUT request to "/current-user/notifications-read-at"
    Then the response should be "updated"
    And the table "users" should remain unchanged, regardless of the row with group_id "1"
    And the table "users" at group_id "1" should be:
      | group_id | login | ABS(TIMESTAMPDIFF(SECOND, notifications_read_at, NOW())) < 3 |
      | 1        | user  | 1                                                            |
