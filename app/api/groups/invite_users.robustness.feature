Feature: Invite users - robustness
  Background:
    Given the database has the following table 'groups':
      | id  |
      | 11  |
      | 12  |
      | 13  |
      | 21  |
      | 22  |
    And the database has the following table 'users':
      | login | group_id | owned_group_id | first_name  | last_name |
      | owner | 21       | 22             | Jean-Michel | Blanquer  |
      | user  | 11       | 12             | John        | Doe       |
    And the database has the following table 'group_managers':
      | group_id | manager_id |
      | 13       | 21         |
    And the database has the following table 'groups_ancestors':
      | ancestor_group_id | child_group_id | is_self |
      | 11                | 11             | 1       |
      | 12                | 12             | 1       |
      | 13                | 13             | 1       |
      | 21                | 21             | 1       |
      | 22                | 22             | 1       |

  Scenario: Fails when the user is not a manager of the parent group
    Given I am the user with id "11"
    When I send a POST request to "/groups/13/invitations" with the following body:
      """
      {
        "logins": ["john", "jane", "owner", "barack"]
      }
      """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should stay unchanged
    And the table "group_pending_requests" should be empty
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should stay unchanged

  Scenario: Fails when the user doesn't exist
    Given I am the user with id "404"
    When I send a POST request to "/groups/13/invitations" with the following body:
      """
      {
        "logins": ["john", "jane", "owner", "barack"]
      }
      """
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "groups_groups" should stay unchanged
    And the table "group_pending_requests" should be empty
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should stay unchanged

  Scenario: Fails when the parent group id is wrong
    Given I am the user with id "21"
    When I send a POST request to "/groups/abc/invitations" with the following body:
      """
      {
        "logins": ["john", "jane", "owner", "barack"]
      }
      """
    Then the response code should be 400
    And the response error message should contain "Wrong value for parent_group_id (should be int64)"
    And the table "groups_groups" should stay unchanged
    And the table "group_pending_requests" should be empty
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should stay unchanged

  Scenario: Fails when logins are wrong
    Given I am the user with id "21"
    When I send a POST request to "/groups/13/invitations" with the following body:
      """
      {
        "logins": [1, 2, 3]
      }
      """
    Then the response code should be 400
    And the response error message should contain "Json: cannot unmarshal number into Go struct field .logins of type string"
    And the table "groups_groups" should stay unchanged
    And the table "group_pending_requests" should be empty
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should stay unchanged

  Scenario: Fails when logins are not present
    Given I am the user with id "21"
    When I send a POST request to "/groups/13/invitations" with the following body:
      """
      {
      }
      """
    Then the response code should be 400
    And the response error message should contain "There should be at least one login listed"
    And the table "groups_groups" should stay unchanged
    And the table "group_pending_requests" should be empty
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should stay unchanged

  Scenario: Fails when logins are empty
    Given I am the user with id "21"
    When I send a POST request to "/groups/13/invitations" with the following body:
      """
      {
        "logins": []
      }
      """
    Then the response code should be 400
    And the response error message should contain "There should be at least one login listed"
    And the table "groups_groups" should stay unchanged
    And the table "group_pending_requests" should be empty
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should stay unchanged

  Scenario: Fails when too many logins
    Given I am the user with id "21"
    When I send a POST request to "/groups/13/invitations" with the following body:
      """
      {
        "logins": [
          "1","2","3","4","5","6","7","8","9","10",
          "1","2","3","4","5","6","7","8","9","10",
          "1","2","3","4","5","6","7","8","9","10",
          "1","2","3","4","5","6","7","8","9","10",
          "1","2","3","4","5","6","7","8","9","10",
          "1","2","3","4","5","6","7","8","9","10",
          "1","2","3","4","5","6","7","8","9","10",
          "1","2","3","4","5","6","7","8","9","10",
          "1","2","3","4","5","6","7","8","9","10",
          "1","2","3","4","5","6","7","8","9","10",
          "1"
        ]
      }
      """
    Then the response code should be 400
    And the response error message should contain "There should be no more than 100 logins"
    And the table "groups_groups" should stay unchanged
    And the table "group_pending_requests" should be empty
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should stay unchanged
