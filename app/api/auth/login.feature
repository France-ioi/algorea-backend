Feature: Generate a login state, set the cookie, and redirect to the auth url
  Scenario: Successful redirect
    Given the generated auth keys are "o5yuy6wmpe607bknrmvrrduy5xe60zd7","ny93zqri9a2adn4v1ut6izd76xb3pccw"
    And the time now is "2019-07-16T22:02:29Z"
    And the DB time now is "2019-07-16 22:02:29"
    And the application config is:
    """
    auth:
      loginModuleURL: "https://login.algorea.org"
      clientID: "1"
      callbackURL: "https://backend.algorea.org/auth/login-callback"
    """
    When I send a POST request to "/auth/login"
    Then the response code should be 302
    And the response header "Location" should be "https://login.algorea.org/oauth/authorize?approval_prompt=auto&client_id=1&redirect_uri=https%3A%2F%2Fbackend.algorea.org%2Fauth%2Flogin-callback&response_type=code&scope=account&state=o5yuy6wmpe607bknrmvrrduy5xe60zd7"
    And the response header "Set-Cookie" should be "login_csrf=ny93zqri9a2adn4v1ut6izd76xb3pccw; Path=/; Domain=127.0.0.1; Expires=Wed, 17 Jul 2019 00:02:29 GMT; Max-Age=7200; HttpOnly; Secure"
    And the table "login_states" should be:
      | cookie                           | state                            | ABS(TIMESTAMPDIFF(SECOND, NOW(), expiration_date) - 7200) < 3 |
      | ny93zqri9a2adn4v1ut6izd76xb3pccw | o5yuy6wmpe607bknrmvrrduy5xe60zd7 | true                                                          |

  Scenario: Sets insecure cookies for HTTP
    Given the generated auth keys are "jajdsfpaisdjf029j22ijfeljlsdfdsa","12i3rjidjsf98i2jlksdjflsdjfldskf"
    And the time now is "2019-07-16T22:03:29Z"
    And the application config is:
    """
    auth:
      loginModuleURL: "https://login.algorea.org"
      clientID: "2"
      callbackURL: "http://backend.algorea.org/auth/login-callback"
    """
    When I send a POST request to "/auth/login"
    Then the response code should be 302
    And the response header "Location" should be "https://login.algorea.org/oauth/authorize?approval_prompt=auto&client_id=2&redirect_uri=http%3A%2F%2Fbackend.algorea.org%2Fauth%2Flogin-callback&response_type=code&scope=account&state=jajdsfpaisdjf029j22ijfeljlsdfdsa"
    And the response header "Set-Cookie" should be "login_csrf=12i3rjidjsf98i2jlksdjflsdjfldskf; Path=/; Domain=127.0.0.1; Expires=Wed, 17 Jul 2019 00:03:29 GMT; Max-Age=7200; HttpOnly"
