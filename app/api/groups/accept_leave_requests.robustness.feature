Feature: Accept requests to leave a group - robustness
  Background:
    Given the database has the following table "groups":
      | id  | type    | frozen_membership |
      | 11  | Class   | false             |
      | 13  | Team    | false             |
      | 14  | Friends | true              |
      | 31  | User    | false             |
      | 111 | User    | false             |
      | 121 | User    | false             |
      | 122 | User    | false             |
      | 123 | User    | false             |
      | 131 | User    | false             |
      | 141 | User    | false             |
      | 151 | User    | false             |
      | 161 | User    | false             |
      | 444 | Team    | false             |
    And the database has the following user:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 13              | 31             |
      | 13              | 111            |
      | 13              | 121            |
      | 13              | 123            |
      | 13              | 141            |
      | 13              | 151            |
      | 14              | 151            |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type          |
      | 13       | 21        | invitation    |
      | 13       | 31        | leave_request |
      | 13       | 141       | leave_request |
      | 13       | 161       | join_request  |
      | 14       | 11        | invitation    |
      | 14       | 21        | join_request  |
      | 14       | 151       | leave_request |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 14       | 21         | memberships |
      | 444      | 21         | memberships |

  Scenario: Fails when the user is not a manager of the parent group
    Given I am the user with id "21"
    When I send a POST request to "/groups/13/leave-requests/accept?group_ids=31,141,21,11,13"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the user is a manager of the parent group, but doesn't have enough rights to manage memberships
    Given I am the user with id "21"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage |
      | 13       | 21         | none       |
    When I send a POST request to "/groups/13/leave-requests/accept?group_ids=31,141,21,11,13"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the user has enough rights to manage memberships, but the group is a user
    Given I am the user with id "21"
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage  |
      | 21       | 21         | memberships |
    When I send a POST request to "/groups/21/leave-requests/accept?group_ids=31,141,21,11,13"
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the parent group id is wrong
    Given I am the user with id "21"
    When I send a POST request to "/groups/abc/leave-requests/accept?group_ids=31,141,21,11,13"
    Then the response code should be 400
    And the response error message should contain "Wrong value for parent_group_id (should be int64)"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when group_ids is wrong
    Given I am the user with id "21"
    When I send a POST request to "/groups/13/leave-requests/accept?group_ids=31,abc,11,13"
    Then the response code should be 400
    And the response error message should contain "Unable to parse one of the integers given as query args (value: 'abc', param: 'group_ids')"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when the group membership is frozen
    Given I am the user with id "21"
    When I send a POST request to "/groups/14/leave-requests/accept?group_ids=151"
    Then the response code should be 403
    And the response error message should contain "Group membership is frozen"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged

  Scenario: Fails when several group_ids are given for a team
    Given I am the user with id "21"
    When I send a POST request to "/groups/444/leave-requests/accept?group_ids=11,151"
    Then the response code should be 400
    And the response error message should contain "There should be no more than one id in group_ids when the parent group is a team"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should be empty
    And the table "groups_ancestors" should remain unchanged
