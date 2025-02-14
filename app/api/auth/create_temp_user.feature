Feature: Create a temporary user
  Background:
    Given the application config is:
      """
      domains:
        -
          domains: [127.0.0.1]
          allUsersGroup: 2
          tempUsersGroup: 4
      """
    And the database has the following table "groups":
      | id | name      | type | text_id   |
      | 2  | AllUsers  | Base | AllUsers  |
      | 4  | TempUsers | User | TempUsers |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 2               | 4              |
    And the time now is "2020-07-16T22:02:28Z"

  Scenario Outline: Create a new temporary user
    Given the generated auth key is "ny93zqri9a2adn4v1ut6izd76xb3pccw"
    And the "Cookie" request header is "<cookie>"
    When I send a POST request to "/auth/temp-user<query>"
    Then the response code should be 201
    And the response body should be, in JSON:
      """
      {
        "success": true,
        "message": "created",
        "data": {<token_in_data> "expires_in": 7200}
      }
      """
    And the response header "Set-Cookie" should be "<expected_cookie>"
    And logs should contain:
      """
      Generated a session token expiring in 7200 seconds for a temporary user with group_id = 5577006791947779410
      """
    And the table "users" at group_id "5577006791947779410" should be:
      | group_id            | login_id | login        | temp_user | default_language            | ABS(TIMESTAMPDIFF(SECOND, registered_at, NOW())) < 3 | last_ip   |
      | 5577006791947779410 | 0        | tmp-49727887 | true      | <expected_default_language> | true                                                 | 127.0.0.1 |
    And the table "groups" should remain unchanged, regardless of the row with id "5577006791947779410"
    And the table "groups" at id "5577006791947779410" should be:
      | id                  | name         | type | description  | ABS(TIMESTAMPDIFF(SECOND, created_at, NOW())) < 3 | is_open | send_emails |
      | 5577006791947779410 | tmp-49727887 | User | tmp-49727887 | true                                              | false   | false       |
    And the table "groups_groups" should remain unchanged, regardless of the row with child_group_id "5577006791947779410"
    And the table "groups_groups" at child_group_id "5577006791947779410" should be:
      | parent_group_id | child_group_id      |
      | 4               | 5577006791947779410 |
    And the table "groups_ancestors" should be:
      | ancestor_group_id   | child_group_id      | is_self |
      | 2                   | 2                   | true    |
      | 2                   | 4                   | false   |
      | 2                   | 5577006791947779410 | false   |
      | 4                   | 4                   | true    |
      | 4                   | 5577006791947779410 | false   |
      | 5577006791947779410 | 5577006791947779410 | true    |
    And the table "sessions" should be:
      | session_id          | user_id             |
      | 6129484611666145821 | 5577006791947779410 |
    And the table "access_tokens" should be:
      | session_id          | token                            | ABS(TIMESTAMPDIFF(SECOND, NOW(), expires_at) - 7200) < 3 |
      | 6129484611666145821 | ny93zqri9a2adn4v1ut6izd76xb3pccw | true                                                     |
    And the table "attempts" should be:
      | participant_id      | id | creator_id          | ABS(TIMESTAMPDIFF(SECOND, NOW(), created_at)) < 3 | parent_attempt_id | root_item_id |
      | 5577006791947779410 | 0  | 5577006791947779410 | true                                              | null              | null         |
  Examples:
    | query                            | cookie                          | expected_default_language | expected_cookie                                                                                                                                                                                                                                                                                   | token_in_data                                      |
    |                                  | [Header not defined]            | fr                        | [Header not defined]                                                                                                                                                                                                                                                                              | "access_token":"ny93zqri9a2adn4v1ut6izd76xb3pccw", |
    | ?default_language=en             | [Header not defined]            | en                        | [Header not defined]                                                                                                                                                                                                                                                                              | "access_token":"ny93zqri9a2adn4v1ut6izd76xb3pccw", |
    | ?use_cookie=0                    | [Header not defined]            | fr                        | [Header not defined]                                                                                                                                                                                                                                                                              | "access_token":"ny93zqri9a2adn4v1ut6izd76xb3pccw", |
    | ?use_cookie=1&cookie_secure=1    | [Header not defined]            | fr                        | access_token=2!ny93zqri9a2adn4v1ut6izd76xb3pccw!127.0.0.1!/; Path=/; Domain=127.0.0.1; Expires=Fri, 17 Jul 2020 00:02:28 GMT; Max-Age=7200; HttpOnly; Secure; SameSite=None                                                                                                                       |                                                    |
    | ?use_cookie=1&cookie_same_site=1 | access_token=2!abcd!127.0.0.1!/ | fr                        | access_token=; Path=/; Domain=127.0.0.1; Expires=Thu, 16 Jul 2020 21:45:48 GMT; Max-Age=0; HttpOnly; Secure; SameSite=None\naccess_token=1!ny93zqri9a2adn4v1ut6izd76xb3pccw!127.0.0.1!/; Path=/; Domain=127.0.0.1; Expires=Fri, 17 Jul 2020 00:02:28 GMT; Max-Age=7200; HttpOnly; SameSite=Strict |                                                    |
