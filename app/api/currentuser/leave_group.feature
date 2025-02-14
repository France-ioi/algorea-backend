Feature: User leaves a group
  Background:
    Given the database has the following table "groups":
      | id | require_lock_membership_approval_until |
      | 11 | 2019-08-20 00:00:00                    |
      | 14 | 2019-08-20 00:00:00                    |
    And the database has the following user:
      | group_id |
      | 21       |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id | lock_membership_approved_at |
      | 11              | 21             | 2019-05-30 11:00:00         |
      | 14              | 21             | null                        |
    And the groups ancestors are computed

  Scenario: Successfully leave a group
    Given I am the user with id "21"
    When I send a DELETE request to "/current-user/group-memberships/11"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "deleted",
      "data": {"changed": true}
    }
    """
    And the table "groups_groups" should remain unchanged, regardless of the row with parent_group_id "11"
    And the table "groups_groups" should not contain parent_group_id "11"
    And the table "group_membership_changes" should be:
      | group_id | member_id | action | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 11       | 21        | left   | 21           | 1                                         |
    And the table "groups_ancestors" should remain unchanged, regardless of the row with ancestor_group_id "11"
    And the table "groups_ancestors" at ancestor_group_id "11" should be:
      | ancestor_group_id | child_group_id |
      | 11                | 11             |

  Scenario: Successfully leave a group (lock_user_deletion_until = NOW())
    Given I am the user with id "21"
    And the DB time now is "2019-08-20 00:00:00"
    When I send a DELETE request to "/current-user/group-memberships/11"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "deleted",
      "data": {"changed": true}
    }
    """
    And the table "groups_groups" should remain unchanged, regardless of the row with parent_group_id "11"
    And the table "groups_groups" should not contain parent_group_id "11"
    And the table "group_membership_changes" should be:
      | group_id | member_id | action | initiator_id | ABS(TIMESTAMPDIFF(SECOND, "2019-08-20 00:00:00", NOW())) < 3 |
      | 11       | 21        | left   | 21           | 1                                                            |
    And the table "groups_ancestors" should remain unchanged, regardless of the row with ancestor_group_id "11"
    And the table "groups_ancestors" at ancestor_group_id "11" should be:
      | ancestor_group_id | child_group_id |
      | 11                | 11             |

  Scenario: Successfully leave a group (lock_user_deletion_until > NOW(), but groups_groups.lock_membership_approved_at is NULL)
    Given I am the user with id "21"
    And the DB time now is "2019-08-20 00:00:00"
    When I send a DELETE request to "/current-user/group-memberships/14"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    {
      "success": true,
      "message": "deleted",
      "data": {"changed": true}
    }
    """
    And the table "groups_groups" should remain unchanged, regardless of the row with parent_group_id "14"
    And the table "groups_groups" should not contain parent_group_id "14"
    And the table "group_membership_changes" should be:
      | group_id | member_id | action | initiator_id | ABS(TIMESTAMPDIFF(SECOND, "2019-08-20 00:00:00", NOW())) < 3 |
      | 14       | 21        | left   | 21           | 1                                                            |
    And the table "groups_ancestors" should remain unchanged, regardless of the row with ancestor_group_id "14"
    And the table "groups_ancestors" at ancestor_group_id "14" should be:
      | ancestor_group_id | child_group_id |
      | 14                | 14             |
