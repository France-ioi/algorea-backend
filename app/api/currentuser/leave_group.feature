Feature: User leaves a group
  Background:
    Given the database has the following table 'users':
      | ID | idGroupSelf | idGroupOwned |
      | 1  | 21          | 22           |
    And the database has the following table 'groups':
      | ID |
      | 11 |
      | 14 |
      | 21 |
      | 22 |
    And the database has the following table 'groups_ancestors':
      | ID | idGroupAncestor | idGroupChild | bIsSelf |
      | 1  | 11              | 11           | 1       |
      | 2  | 11              | 21           | 0       |
      | 3  | 14              | 14           | 1       |
      | 4  | 21              | 21           | 1       |
      | 5  | 22              | 22           | 1       |
    And the database has the following table 'groups_groups':
      | ID | idGroupParent | idGroupChild | sType              | sStatusDate          |
      | 1  | 11            | 21           | invitationAccepted | 2017-04-29T06:38:38Z |
      | 7  | 14            | 21           | left               | 2017-02-21T06:38:38Z |

  Scenario: Successfully leave a group
    Given I am the user with ID "1"
    When I send a DELETE request to "/current-user/memberships/11"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "deleted"
    }
    """
    And the table "groups_groups" should stay unchanged but the row with ID "1"
    And the table "groups_groups" at ID "1" should be:
      | ID | idGroupParent | idGroupChild | sType | (sStatusDate IS NOT NULL) AND (ABS(NOW() - sStatusDate) < 3) |
      | 1  | 11            | 21           | left  | 1                                                            |
    And the table "groups_ancestors" should stay unchanged but the row with ID "2"
    And the table "groups_ancestors" should not contain ID "2"

  Scenario: Leave a group that already have been left
    Given I am the user with ID "1"
    When I send a DELETE request to "/current-user/memberships/14"
    Then the response code should be 205
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "not changed"
    }
    """
    And the table "groups_groups" should stay unchanged
    And the table "groups_ancestors" should stay unchanged

