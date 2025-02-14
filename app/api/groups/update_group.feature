Feature: Update a group (groupEdit)
  Background:
    Given the database has the following table "groups":
      | id | name    | grade | description     | created_at          | type    | root_activity_id    | root_skill_id | is_open | is_public | code       | code_lifetime | code_expires_at     | open_activity_when_joining | is_official_session | require_members_to_join_parent | organizer | address_line1 | address_line2 | address_postcode | address_city | address_country | expected_start      | frozen_membership | require_personal_info_access_approval | require_lock_membership_approval_until | require_watch_approval | max_participants | enforce_max_participants |
      | 11 | Group A | -3    | Group A is here | 2019-02-06 09:26:40 | Class   | 1672978871462145361 | null          | true    | true      | ybqybxnlyo | 3600          | 2017-10-13 05:39:48 | true                       | false               | false                          | null      | null          | null          | null             | null         | null            | null                | false             | none                                  | null                                   | false                  | null             | false                    |
      | 13 | Group B | -2    | Group B is here | 2019-03-06 09:26:40 | Class   | 1672978871462145461 | null          | true    | true      | ybabbxnlyo | 3600          | 2017-10-14 05:39:48 | true                       | false               | false                          | null      | null          | null          | null             | null         | null            | null                | false             | edit                                  | 2020-01-01 00:00:00                    | true                   | 5                | true                     |
      | 14 | Group C | -4    | Group C         | 2019-04-06 09:26:40 | Club    | null                | null          | true    | false     | null       | null          | null                | false                      | false               | false                          | null      | null          | null          | null             | null         | null            | null                | false             | none                                  | null                                   | false                  | null             | false                    |
      | 15 | Group D | -4    | Group D         | 2019-04-06 09:26:40 | Session | null                | null          | true    | false     | null       | null          | null                | false                      | true                | false                          | null      | null          | null          | null             | null         | null            | null                | false             | none                                  | null                                   | false                  | null             | false                    |
      | 16 | Group E | -3    | Group E is here | 2018-04-06 09:26:40 | Session | 1672978871462145461 | 4567          | true    | false     | babbxnlyoy | 7384          | 2018-10-14 05:39:48 | true                       | true                | false                          | Organizer | Address1      | Address2      | Postcode         | City         | Country         | 2019-05-30 11:00:00 | true              | edit                                  | 2019-05-30 11:00:00                    | true                   | 10               | true                     |
      | 17 | Group F | -2    | null            | 2017-04-06 09:26:40 | Session | null                | null          | false   | true      | null       | null          | null                | false                      | false               | true                           | null      | null          | null          | null             | null         | null            | null                | false             | none                                  | null                                   | false                  | null             | false                    |
      | 18 | Group G | -2    | null            | 2017-04-06 09:26:40 | Session | null                | null          | false   | true      | null       | null          | null                | false                      | false               | true                           | null      | null          | null          | null             | null         | null            | null                | false             | none                                  | null                                   | false                  | null             | false                    |
      | 21 | owner   | -4    | owner           | 2019-04-06 09:26:40 | User    | null                | null          | false   | false     | null       | null          | null                | false                      | false               | false                          | null      | null          | null          | null             | null         | null            | null                | false             | none                                  | null                                   | false                  | null             | false                    |
      | 24 | other   | -4    | other           | 2019-04-06 09:26:40 | User    | null                | null          | false   | false     | null       | null          | null                | false                      | false               | false                          | null      | null          | null          | null             | null         | null            | null                | false             | none                                  | null                                   | false                  | null             | false                    |
      | 25 | jane    | -4    | jane            | 2019-04-06 09:26:40 | User    | null                | null          | false   | false     | null       | null          | null                | false                      | false               | false                          | null      | null          | null          | null             | null         | null            | null                | false             | none                                  | null                                   | false                  | null             | false                    |
      | 31 | user    | -4    | owner           | 2019-04-06 09:26:40 | User    | null                | null          | false   | false     | null       | null          | null                | false                      | false               | false                          | null      | null          | null          | null             | null         | null            | null                | false             | none                                  | null                                   | false                  | null             | false                    |
      | 41 | john    | -4    | john            | 2019-04-06 09:26:40 | User    | null                | null          | false   | false     | null       | null          | null                | false                      | false               | false                          | null      | null          | null          | null             | null         | null            | null                | false             | none                                  | null                                   | false                  | null             | false                    |
      | 50 | Admins  | -4    | Admins          | 2019-04-06 09:26:40 | Club    | null                | null          | false   | false     | null       | null          | null                | false                      | false               | false                          | null      | null          | null          | null             | null         | null            | null                | false             | none                                  | null                                   | false                  | null             | false                    |
    And the database has the following users:
      | group_id | login | first_name  | last_name | default_language |
      | 21       | owner | Jean-Michel | Blanquer  | fr               |
      | 24       | other | John        | Doe       | en               |
      | 25       | jane  | Jane        | Doe       | en               |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 13       | 21         | memberships_and_group |
      | 14       | 21         | none                  |
      | 14       | 50         | memberships_and_group |
      | 15       | 24         | memberships_and_group |
      | 16       | 25         | none                  |
      | 17       | 25         | none                  |
      | 18       | 25         | memberships           |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 13              | 41             |
      | 50              | 21             |
    And the groups ancestors are computed
    And the database has the following table "group_pending_requests":
      | group_id | member_id | type          |
      | 13       | 21        | invitation    |
      | 13       | 24        | join_request  |
      | 13       | 31        | join_request  |
      | 13       | 41        | leave_request |
      | 14       | 31        | join_request  |
    And the database has the following table "items":
      | id   | default_language_tag | type    |
      | 123  | fr                   | Task    |
      | 4567 | fr                   | Skill   |
      | 5678 | fr                   | Chapter |
      | 6789 | fr                   | Task    |
      | 7890 | fr                   | Task    |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 21       | 4567    | info               |
      | 21       | 5678    | info               |
      | 21       | 6789    | info               |
      | 21       | 7890    | info               |
      | 24       | 123     | info               |
      | 50       | 123     | info               |
    And the database has the following table "permissions_granted":
      | group_id | item_id | can_make_session_official | is_owner | source_group_id |
      | 24       | 123     | true                      | false    | 13              |
      | 50       | 123     | false                     | true     | 13              |

  Scenario Outline: User is a manager of the group, all fields are not nulls, updates group_pending_requests
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "is_public": false,
      "name": "Team B",
      "grade": 10,
      "description": "Team B is here",
      "is_open": false,
      "code_lifetime": 2147483647,
      "code_expires_at": "2019-12-31T23:59:59Z",
      "open_activity_when_joining": false,
      "root_activity_id": "<root_activity_id>",
      "root_skill_id": "4567",

      "require_members_to_join_parent": true,
      "frozen_membership": false,

      "organizer": "Association France-ioi",
      "address_line1": "Chez Jacques-Henri Jourdan,",
      "address_line2": "42, rue de Cronstadt",
      "address_postcode": "75015",
      "address_city": "Paris",
      "address_country": "France",
      "expected_start": "2019-05-03T12:00:00+01:00",
      "require_personal_info_access_approval": "view",
      "require_lock_membership_approval_until": "2018-05-30T11:00:00Z",
      "require_watch_approval": true,
      "max_participants": 8,
      "enforce_max_participants": true
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged, regardless of the row with id "13"
    And the table "groups" at id "13" should be:
      | id | name   | grade | description    | created_at          | type  | root_activity_id   | root_skill_id | is_open | is_public | code       | code_lifetime | code_expires_at     | open_activity_when_joining | require_members_to_join_parent | organizer              | address_line1               | address_line2        | address_postcode | address_city | address_country | expected_start      | require_personal_info_access_approval | require_lock_membership_approval_until | require_watch_approval | max_participants | enforce_max_participants |
      | 13 | Team B | 10    | Team B is here | 2019-03-06 09:26:40 | Class | <root_activity_id> | 4567          | false   | false     | ybabbxnlyo | 2147483647    | 2019-12-31 23:59:59 | false                      | true                           | Association France-ioi | Chez Jacques-Henri Jourdan, | 42, rue de Cronstadt | 75015            | Paris        | France          | 2019-05-03 11:00:00 | view                                  | 2018-05-30 11:00:00                    | true                   | 8                | true                     |
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should be:
      | group_id | member_id | type          |
      | 13       | 21        | invitation    |
      | 13       | 41        | leave_request |
      | 14       | 31        | join_request  |
    And the table "group_membership_changes" should be:
      | group_id | member_id | action               | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 13       | 24        | join_request_refused | 21           | 1                                         |
      | 13       | 31        | join_request_refused | 21           | 1                                         |
  Examples:
    | root_activity_id |
    | 5678             |
    | 6789             |
    | 7890             |

  Scenario: User is a manager of the group, nullable fields are nulls
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "is_public": false,
      "name": "Club B",
      "description": null,
      "is_open": false,
      "open_activity_when_joining": false,
      "root_activity_id": null,
      "grade": 0,
      "code_expires_at": null,
      "code_lifetime": null,
      "require_lock_membership_approval_until": null,
      "max_participants": null,
      "enforce_max_participants": false
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged, regardless of the row with id "13"
    And the table "groups" at id "13" should be:
      | id | name   | grade | description | created_at          | type  | root_activity_id | is_open | is_public | code       | code_lifetime | code_expires_at | open_activity_when_joining | require_lock_membership_approval_until | max_participants |
      | 13 | Club B | 0     | null        | 2019-03-06 09:26:40 | Class | null             | false   | false     | ybabbxnlyo | null          | null            | false                      | null                                   | null             |

  Scenario: User is a manager of the group, does not update group_pending_requests (is_public is still true)
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "is_public": true,
      "name": "Club B",
      "description": null,
      "is_open": false,
      "open_activity_when_joining": false,
      "root_activity_id": null,
      "grade": 0,
      "code_expires_at": null,
      "code_lifetime": null
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged, regardless of the row with id "13"
    And the table "groups" at id "13" should be:
      | id | name   | grade | description | created_at          | type  | root_activity_id | is_open | is_public | code       | code_lifetime | code_expires_at | open_activity_when_joining |
      | 13 | Club B | 0     | null        | 2019-03-06 09:26:40 | Class | null             | false   | true      | ybabbxnlyo | null          | null            | false                      |
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should remain unchanged

  Scenario: User is a manager of the group, does not update group_pending_requests (is_public is not changed)
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "name": "Club B",
      "description": null,
      "is_open": false,
      "open_activity_when_joining": false,
      "root_activity_id": null,
      "grade": 0,
      "code_expires_at": null,
      "code_lifetime": null
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged, regardless of the row with id "13"
    And the table "groups" at id "13" should be:
      | id | name   | grade | description | created_at          | type  | root_activity_id | is_open | is_public | code       | code_lifetime | code_expires_at | open_activity_when_joining |
      | 13 | Club B | 0     | null        | 2019-03-06 09:26:40 | Class | null             | false   | true      | ybabbxnlyo | null          | null            | false                      |
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should remain unchanged

  Scenario: User is a manager of the group, does not update group_pending_requests (is_public changes from false to true)
    Given I am the user with id "21"
    When I send a PUT request to "/groups/14" with the following body:
    """
    {
      "is_public": true
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged, regardless of the row with id "14"
    And the table "groups" at id "14" should be:
      | id | name    | grade | description | created_at          | type | root_activity_id | is_official_session | is_open | is_public | code | code_lifetime | code_expires_at | open_activity_when_joining |
      | 14 | Group C | -4    | Group C     | 2019-04-06 09:26:40 | Club | null             | false               | true    | true      | null | null          | null            | false                      |
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should remain unchanged

  Scenario: User is a manager of the group, but no fields provided
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: User attaches the group to the activity as an official session
    Given I am the user with id "21"
    When I send a PUT request to "/groups/14" with the following body:
    """
    {
      "root_activity_id": "123",
      "is_official_session": true
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged, regardless of the row with id "14"
    And the table "groups" at id "14" should be:
      | id | name    | grade | description | created_at          | type | root_activity_id | is_official_session | is_open | is_public | code | code_lifetime | code_expires_at | open_activity_when_joining |
      | 14 | Group C | -4    | Group C     | 2019-04-06 09:26:40 | Club | 123              | true                | true    | false     | null | null          | null            | false                      |
    And the table "groups_groups" should remain unchanged

  Scenario: User replaces the activity of the official session
    Given I am the user with id "24"
    When I send a PUT request to "/groups/15" with the following body:
    """
    {
      "root_activity_id": "123"
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged, regardless of the row with id "15"
    And the table "groups" at id "15" should be:
      | id | name    | grade | description | created_at          | type    | root_activity_id | is_official_session | is_open | is_public | code | code_lifetime | code_expires_at | open_activity_when_joining |
      | 15 | Group D | -4    | Group D     | 2019-04-06 09:26:40 | Session | 123              | true                | true    | false     | null | null          | null            | false                      |
    And the table "groups_groups" should remain unchanged

  Scenario: Pending requests stay unchanged when 'frozen_membership' is not changed
    Given I am the user with id "21"
    When I send a PUT request to "/groups/14" with the following body:
    """
    {
      "frozen_membership": false
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged, regardless of the row with id "14"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should remain unchanged

  Scenario: Removes pending requests and invitations when 'frozen_membership' becomes true
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "frozen_membership": true
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged, regardless of the row with id "13"
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should be:
      | group_id | member_id | type          |
      | 14       | 31        | join_request  |
    And the table "group_membership_changes" should be:
      | group_id | member_id | action                | initiator_id | ABS(TIMESTAMPDIFF(SECOND, at, NOW())) < 3 |
      | 13       | 21        | invitation_withdrawn  | 21           | 1                                         |
      | 13       | 24        | join_request_refused  | 21           | 1                                         |
      | 13       | 31        | join_request_refused  | 21           | 1                                         |
      | 13       | 41        | leave_request_refused | 21           | 1                                         |

  Scenario: User is a manager of the group with can_manage=none: allows setting all the fields to the previous values
    Given I am the user with id "25"
    When I send a PUT request to "/groups/16" with the following body:
    """
    {
      "is_public": false,
      "name": "Group E",
      "grade": -3,
      "description": "Group E is here",
      "is_open": true,
      "code_lifetime": 7384,
      "code_expires_at": "2018-10-14T05:39:48Z",
      "open_activity_when_joining": true,
      "root_activity_id": "1672978871462145461",
      "root_skill_id": "4567",
      "is_official_session": true,

      "require_members_to_join_parent": false,
      "require_personal_info_access_approval": "edit",
      "require_lock_membership_approval_until": "2019-05-30T11:00:00Z",
      "require_watch_approval": true,
      "max_participants": 10,
      "enforce_max_participants": true,
      "frozen_membership": true,

      "organizer": "Organizer",
      "address_line1": "Address1",
      "address_line2": "Address2",
      "address_postcode": "Postcode",
      "address_city": "City",
      "address_country": "Country",
      "expected_start": "2019-05-30T12:00:00+01:00"
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should remain unchanged

  Scenario: Should be able to update the description, root_activity_id and root_skill_id from a value to null
    Given I am the user with id "100"
    And the database has the following users:
      | group_id | login   |
      | 100      | manager |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 101      | 100        | memberships_and_group |
    And the database table "groups" also has the following row:
      | id  | name    | description       | root_activity_id | root_skill_id |
      | 101 | Group   | Group description | 200              | 100           |
    And the database has the following table "groups_groups":
      | parent_group_id | child_group_id |
      | 101             | 100            |
    And the groups ancestors are computed
    And the database has the following table "items":
      | id  | default_language_tag | type  |
      | 100 | fr                   | Skill |
    When I send a PUT request to "/groups/101" with the following body:
      """
      {
        "description": null,
        "root_activity_id": null,
        "root_skill_id": null
      }
      """
    Then the response should be "updated"
    And the table "groups" at id "101" should be:
      | description | root_activity_id | root_skill_id |
      | null        | null             | null          |

  Scenario: User is a manager of the group with can_manage=none: allows setting all the fields to the previous values (mostly nulls)
    Given I am the user with id "25"
    When I send a PUT request to "/groups/17" with the following body:
    """
    {
      "is_public": true,
      "name": "Group F",
      "grade": -2,
      "description": null,
      "is_open": false,
      "code_lifetime": null,
      "code_expires_at": null,
      "open_activity_when_joining": false,
      "root_activity_id": null,
      "root_skill_id": null,
      "is_official_session": false,

      "require_members_to_join_parent": true,
      "require_personal_info_access_approval": "none",
      "require_watch_approval": false,
      "require_lock_membership_approval_until": null,
      "max_participants": null,
      "enforce_max_participants": false,
      "frozen_membership": false,

      "organizer": null,
      "address_line1": null,
      "address_line2": null,
      "address_postcode": null,
      "address_city": null,
      "address_country": null,
      "expected_start": null
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should remain unchanged

  Scenario: User is a manager of the group with can_manage=memberships: allows changing all the membership-related fields
    Given I am the user with id "25"
    When I send a PUT request to "/groups/18" with the following body:
    """
    {
      "code_lifetime": 3723,
      "code_expires_at": "2030-05-30T11:00:00Z",
      "frozen_membership": true,
      "max_participants": 15,
      "enforce_max_participants": true
    }
    """
    Then the response should be "updated"
    And the table "groups" should remain unchanged, regardless of the row with id "18"
    And the table "groups" at id "18" should be:
      | code_lifetime | code_expires_at     | frozen_membership | max_participants | enforce_max_participants |
      | 3723          | 2030-05-30 11:00:00 | true              | 15               | true                     |
    And the table "groups_groups" should remain unchanged
    And the table "group_pending_requests" should remain unchanged
    And the table "group_membership_changes" should remain unchanged
