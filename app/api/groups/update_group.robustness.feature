Feature: Update a group (groupEdit) - robustness
  Background:
    Given the database has the following table "groups":
      | id | name    | grade | description     | created_at          | type  | root_activity_id    | is_official_session | is_open | is_public | code       | code_lifetime | code_expires_at     | open_activity_when_joining | frozen_membership | require_personal_info_access_approval | require_lock_membership_approval_until | require_watch_approval | max_participants | enforce_max_participants |
      | 11 | Group A | -3    | Group A is here | 2019-02-06 09:26:40 | Class | 1672978871462145361 | false               | true    | true      | ybqybxnlyo | 3600          | 2017-10-13 05:39:48 | true                       | 0                 | none                                  | null                                   | false                  | null             | false                    |
      | 13 | Group B | -2    | Group B is here | 2019-03-06 09:26:40 | Class | 1672978871462145461 | false               | true    | true      | ybabbxnlyo | 3600          | 2017-10-14 05:39:48 | true                       | 1                 | none                                  | null                                   | false                  | 5                | true                     |
      | 14 | Group C | -2    | Group C is here | 2019-03-06 09:26:40 | Class | null                | false               | true    | true      | null       | null          | 2017-10-14 05:39:48 | true                       | 0                 | none                                  | null                                   | false                  | null             | false                    |
      | 15 | Group D | -2    | Group D is here | 2019-03-06 09:26:40 | Class | null                | true                | true    | true      | null       | null          | 2017-10-14 05:39:48 | true                       | 0                 | none                                  | null                                   | false                  | null             | false                    |
      | 16 | Group E | -2    | Group E is here | 2019-03-06 09:26:40 | Class | null                | true                | true    | true      | null       | null          | 2017-10-14 05:39:48 | true                       | 0                 | edit                                  | 2019-05-30 11:00:00                    | true                   | 10               | true                     |
      | 17 | Group F | -2    | Group F is here | 2019-03-06 09:26:40 | Class | null                | true                | true    | true      | null       | null          | 2017-10-14 05:39:48 | true                       | 0                 | none                                  | null                                   | false                  | 1                | false                    |
    And the database has the following users:
      | group_id | login | first_name  | last_name |
      | 21       | owner | Jean-Michel | Blanquer  |
      | 31       | user  | John        | Doe       |
      | 41       | jane  | Jane        | Doe       |
    And the database has the following table "group_managers":
      | group_id | manager_id | can_manage            |
      | 13       | 21         | memberships_and_group |
      | 14       | 21         | memberships_and_group |
      | 15       | 21         | memberships_and_group |
      | 16       | 21         | memberships_and_group |
      | 17       | 21         | memberships           |
      | 17       | 31         | none                  |
      | 17       | 41         | memberships_and_group |
    And the database table "groups_ancestors" also has the following rows:
      | ancestor_group_id | child_group_id | expires_at          |
      | 17                | 21             | 2019-05-30 11:00:00 |
    And the database has the following table "items":
      | id   | default_language_tag | type   |
      | 123  | fr                   | Task   |
      | 124  | fr                   | Task   |
      | 5678 | fr                   | Task   |
      | 6789 | fr                   | Skill  |
      | 7890 | fr                   | Skill  |
    And the database has the following table "permissions_generated":
      | group_id | item_id | can_view_generated |
      | 21       | 123     | info               |
      | 21       | 124     | info               |
      | 21       | 5678    | none               |
      | 21       | 6789    | info               |
    And the database has the following table "permissions_granted":
      | group_id | item_id | can_make_session_official | source_group_id |
      | 21       | 123     | false                     | 13              |
      | 17       | 124     | true                      | 13              |

  Scenario: Should fail if the user is not a manager of the group
    Given I am the user with id "31"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {}
    """
    Then the response code should be 403
    And the response error message should contain "Insufficient access rights"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: Should fail if the user is not found
    Given I am the user with id "404"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {}
    """
    Then the response code should be 401
    And the response error message should contain "Invalid access token"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: User is a manager of the group, but required fields are not filled in correctly
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "is_public": 15,
      "name": 123,
      "grade": "grade",
      "description": 14.5,
      "is_open": "true",
      "code_lifetime": -1,
      "code_expires_at": "the end",
      "open_activity_when_joining": 12,

      "root_activity_id": "abc",
      "is_official_session": "abc",
      "require_members_to_join_parent": "abc",
      "require_personal_info_access_approval": "unknown",
      "require_watch_approval": "abc",
      "require_lock_membership_approval_until": "abc",
      "max_participants": "abc",
      "enforce_max_participants": "abc",
      "organizer": 123,
      "address_line1": 123,
      "address_line2": 123,
      "address_postcode": 123,
      "address_city": 123,
      "address_country": 123,
      "expected_start": "abc"
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "error_text": "Invalid input data",
      "errors": {
        "description": ["expected type 'string', got unconvertible type 'float64'"],
        "is_public": ["expected type 'bool', got unconvertible type 'float64'"],
        "grade": ["expected type 'int32', got unconvertible type 'string'"],
        "name": ["expected type 'string', got unconvertible type 'float64'"],
        "open_activity_when_joining": ["expected type 'bool', got unconvertible type 'float64'"],
        "is_open": ["expected type 'bool', got unconvertible type 'string'"],
        "code_expires_at": ["decoding error: parsing time \"the end\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"the end\" as \"2006\""],
        "code_lifetime": ["can be null or an integer between 0 and 2147483647 inclusively"],
        "root_activity_id": ["decoding error: strconv.ParseInt: parsing \"abc\": invalid syntax"],
        "is_official_session": ["expected type 'bool', got unconvertible type 'string'"],
        "require_members_to_join_parent": ["expected type 'bool', got unconvertible type 'string'"],
        "require_personal_info_access_approval": ["require_personal_info_access_approval must be one of [none view edit]"],
        "require_lock_membership_approval_until": ["decoding error: parsing time \"abc\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"abc\" as \"2006\""],
        "require_watch_approval": ["expected type 'bool', got unconvertible type 'string'"],
        "max_participants": ["expected type 'int', got unconvertible type 'string'"],
        "enforce_max_participants": ["expected type 'bool', got unconvertible type 'string'"],
        "organizer": ["expected type 'string', got unconvertible type 'float64'"],
        "address_line1": ["expected type 'string', got unconvertible type 'float64'"],
        "address_line2": ["expected type 'string', got unconvertible type 'float64'"],
        "address_postcode": ["expected type 'string', got unconvertible type 'float64'"],
        "address_city": ["expected type 'string', got unconvertible type 'float64'"],
        "address_country": ["expected type 'string', got unconvertible type 'float64'"],
        "expected_start": ["decoding error: parsing time \"abc\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"abc\" as \"2006\""]
      },
      "message": "Bad Request",
      "success": false
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: The group id is not a number
    Given I am the user with id "21"
    When I send a PUT request to "/groups/1_3" with the following body:
    """
    {
    }
    """
    Then the response code should be 400
    And the response error message should contain "Wrong value for group_id (should be int64)"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: The root activity does not exist
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "root_activity_id": "404"
    }
    """
    Then the response code should be 403
    And the response error message should contain "No access to the root activity or it is a skill"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: The user cannot view the root activity
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "root_activity_id": "5678"
    }
    """
    Then the response code should be 403
    And the response error message should contain "No access to the root activity or it is a skill"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: The root activity is visible, but it is a skill
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "root_activity_id": "6789"
    }
    """
    Then the response code should be 403
    And the response error message should contain "No access to the root activity or it is a skill"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: The root skill does not exist
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "root_skill_id": "404"
    }
    """
    Then the response code should be 403
    And the response error message should contain "No access to the root skill or it is not a skill"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: The user cannot view the root skill
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "root_skill_id": "7890"
    }
    """
    Then the response code should be 403
    And the response error message should contain "No access to the root skill or it is not a skill"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: The root skill is visible, but it is not a skill
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "root_skill_id": "123"
    }
    """
    Then the response code should be 403
    And the response error message should contain "No access to the root skill or it is not a skill"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: is_official_session becomes true & root_activity_id becomes not null while the user doesn't have the permission
    Given I am the user with id "21"
    When I send a PUT request to "/groups/14" with the following body:
    """
    {
      "root_activity_id": "123",
      "is_official_session": true
    }
    """
    Then the response code should be 403
    And the response error message should contain "Not enough permissions for attaching the group to the activity as an official session"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: is_official_session becomes true & root_activity_id is set in the db while the user doesn't have the permission
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "is_official_session": true
    }
    """
    Then the response code should be 403
    And the response error message should contain "Not enough permissions for attaching the group to the activity as an official session"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: is_official_session is true in the db & root_activity_id becomes not null while the user doesn't have the permission
    Given I am the user with id "21"
    When I send a PUT request to "/groups/15" with the following body:
    """
    {
      "root_activity_id": "123"
    }
    """
    Then the response code should be 403
    And the response error message should contain "Not enough permissions for attaching the group to the activity as an official session"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: is_official_session is true in the db & root_activity_id becomes not null while the user doesn't have the permission because of expired membership
    Given I am the user with id "21"
    When I send a PUT request to "/groups/16" with the following body:
    """
    {
      "root_activity_id": "124"
    }
    """
    Then the response code should be 403
    And the response error message should contain "Not enough permissions for attaching the group to the activity as an official session"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: is_official_session becomes true, but root_activity_id is null in the db
    Given I am the user with id "21"
    When I send a PUT request to "/groups/14" with the following body:
    """
    {
      "is_official_session": true
    }
    """
    Then the response code should be 400
    And the response error message should contain "The root_activity_id should be set for official sessions"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: is_official_session becomes true, but the new root_activity_id is null
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "root_activity_id": null,
      "is_official_session": true
    }
    """
    Then the response code should be 400
    And the response error message should contain "The root_activity_id should be set for official sessions"
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: frozen_membership changes from true to false
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "frozen_membership": false
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "error_text": "Invalid input data",
      "errors": {
        "frozen_membership": ["can only be changed from false to true"]
      },
      "message": "Bad Request",
      "success": false
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario Outline: Should return an error if a require_* field is strengthened, and there is at least one user in the group, but approval_change_action is not given
    Given I am the user with id "21"
    And the server time now is "2020-01-01T01:00:00Z"
    And the database table "groups" also has the following rows:
      | id  | name  | grade | description | created_at          | type  | root_activity_id | is_official_session | is_open | is_public | code | code_lifetime | code_expires_at     | open_activity_when_joining | frozen_membership | require_personal_info_access_approval       | require_lock_membership_approval_until       | require_watch_approval       | max_participants | enforce_max_participants |
      | 101 | Group | 1     | Group       | 2020-01-01 00:00:00 | Class | null             | true                | true    | true      | null | null          | 2020-01-01 00:00:00 | true                       | 0                 | <require_personal_info_access_approval_old> | <require_lock_membership_approval_until_old> | <require_watch_approval_old> | 1                | false                    |
      | 110 | Team  | 1     | Team        | 2020-01-01 00:00:00 | Team  | null             | true                | true    | true      | null | null          | 2020-01-01 00:00:00 | true                       | 0                 | <require_personal_info_access_approval_old> | <require_lock_membership_approval_until_old> | <require_watch_approval_old> | 1                | false                    |
    And the database table "group_managers" also has the following rows:
      | group_id | manager_id | can_manage            |
      | 101      | 21         | memberships_and_group |
    # There must be at least one user in the group. Otherwise it's not considered a strengthening.
    And the database table "groups_groups" also has the following row:
      | parent_group_id | child_group_id |
      | 101             | 110            |
    And the groups ancestors are computed
    # There is at least one user in the group
    And the database table "groups_ancestors" also has the following rows:
      | ancestor_group_id | child_group_id | expires_at          |
      | 101               | 21             | 2021-01-01 00:00:00 |
    When I send a PUT request to "/groups/101" with the following body:
      """
      {
        "require_watch_approval": <require_watch_approval_new>,
        "require_personal_info_access_approval": "<require_personal_info_access_approval_new>",
        "require_lock_membership_approval_until": <require_lock_membership_approval_until_new>
      }
      """
    Then the response body should be, in JSON:
      """
      {
        "error_text": "Invalid input data",
        "errors": {
          "<error_field>": ["Strengthening requires parameter approval_change_action"]
        },
        "message": "Bad Request",
        "success": false
      }
      """
    And the response code should be 400
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    Examples:
      | error_field                            | require_watch_approval_old | require_watch_approval_new | require_personal_info_access_approval_old | require_personal_info_access_approval_new | require_lock_membership_approval_until_old | require_lock_membership_approval_until_new |
      | require_watch_approval                 | false                      | true                       | none                                      | none                                      | null                                       | null                                       |
      | require_personal_info_access_approval  | false                      | false                      | none                                      | view                                      | null                                       | null                                       |
      | require_personal_info_access_approval  | false                      | false                      | none                                      | edit                                      | null                                       | null                                       |
      | require_personal_info_access_approval  | false                      | false                      | view                                      | edit                                      | null                                       | null                                       |
      | require_lock_membership_approval_until | false                      | false                      | none                                      | none                                      | null                                       | "2020-01-01T01:00:01Z"                     |
      | require_lock_membership_approval_until | false                      | false                      | none                                      | none                                      | 2020-01-01 12:00:00                        | "2020-01-01T12:00:01Z"                     |

  Scenario: Should return an error if approval_change_action has an invalid value
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "require_personal_info_access_approval": "edit",
      "approval_change_action": "invalid"
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "error_text": "Invalid input data",
      "errors": {
        "approval_change_action": ["approval_change_action must be one of [empty reinvite]"]
      },
      "message": "Bad Request",
      "success": false
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario Outline: Should return an error if no field is strengthened, and approval_change_action is given
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "approval_change_action": "<approval_change_action>"
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "error_text": "Invalid input data",
      "errors": {
        "approval_change_action": ["must be present only if a 'require_*' field is strengthened"]
      },
      "message": "Bad Request",
      "success": false
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
    Examples:
      | approval_change_action |
      | empty                  |
      | reinvite               |

  Scenario: Doesn't allow setting max_participants to null when enforce_max_participant is true
    Given I am the user with id "21"
    When I send a PUT request to "/groups/13" with the following body:
    """
    {
      "max_participants": null
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "error_text": "Invalid input data",
      "errors": {
        "max_participants": ["cannot be set to null when 'enforce_max_participants' is true"]
      },
      "message": "Bad Request",
      "success": false
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: Doesn't allow setting enforce_max_participants to true when max_participant is null
    Given I am the user with id "21"
    When I send a PUT request to "/groups/14" with the following body:
    """
    {
      "enforce_max_participants": true
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "error_text": "Invalid input data",
      "errors": {
        "enforce_max_participants": ["cannot be set to true when 'max_participants' is null"]
      },
      "message": "Bad Request",
      "success": false
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: Doesn't allow setting enforce_max_participants to true and max_participant to null
    Given I am the user with id "41"
    When I send a PUT request to "/groups/17" with the following body:
    """
    {
      "max_participants": null,
      "enforce_max_participants": true
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "error_text": "Invalid input data",
      "errors": {
        "max_participants": ["cannot be set to null when 'enforce_max_participants' is true"],
        "enforce_max_participants": ["cannot be set to true when 'max_participants' is null"]
      },
      "message": "Bad Request",
      "success": false
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: Doesn't allow changing fields requiring can_manage>=memberships_and_group for managers with can_manage=memberships
    Given I am the user with id "21"
    When I send a PUT request to "/groups/17" with the following body:
    """
    {
      "is_public": false,
      "name": "Team B",
      "grade": 10,
      "description": "Team B is here",
      "is_open": false,
      "open_activity_when_joining": false,
      "root_activity_id": "5678",
      "root_skill_id": "6789",
      "is_official_session": false,
      "require_members_to_join_parent": true,
      "expected_start": "2019-05-03T12:00:00+01:00",
      "require_personal_info_access_approval": "view",
      "require_lock_membership_approval_until": "2018-05-30T11:00:00Z",
      "require_watch_approval": true,
      "organizer": "Association France-ioi",
      "address_line1": "Chez Jacques-Henri Jourdan,",
      "address_line2": "42, rue de Cronstadt",
      "address_postcode": "75015",
      "address_city": "Paris",
      "address_country": "France",

      "code_lifetime": 359999,
      "code_expires_at": "2019-12-31T23:59:59Z",
      "frozen_membership": true,
      "max_participants": 8,
      "enforce_max_participants": true
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "error_text": "Invalid input data",
      "errors": {
        "name": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "grade": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "description": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "is_open": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "is_public": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "open_activity_when_joining": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "root_activity_id": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "root_skill_id": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "is_official_session": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "require_members_to_join_parent": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "address_city": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "address_line1": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "address_line2": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "address_postcode": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "address_country": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "organizer": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "expected_start": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "require_personal_info_access_approval": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "require_lock_membership_approval_until": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "require_watch_approval": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"]
      },
      "message": "Bad Request",
      "success": false
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged

  Scenario: Doesn't allow changing fields requiring can_manage>=memberships_and_group for managers with can_manage=none
    Given I am the user with id "31"
    When I send a PUT request to "/groups/17" with the following body:
    """
    {
      "is_public": false,
      "name": "Team B",
      "grade": 10,
      "description": "Team B is here",
      "is_open": false,
      "open_activity_when_joining": false,
      "root_activity_id": "5678",
      "root_skill_id": "6789",
      "is_official_session": false,
      "require_members_to_join_parent": true,
      "expected_start": "2019-05-03T12:00:00+01:00",
      "require_personal_info_access_approval": "view",
      "require_lock_membership_approval_until": "2018-05-30T11:00:00Z",
      "require_watch_approval": true,
      "organizer": "Association France-ioi",
      "address_line1": "Chez Jacques-Henri Jourdan,",
      "address_line2": "42, rue de Cronstadt",
      "address_postcode": "75015",
      "address_city": "Paris",
      "address_country": "France",

      "code_lifetime": 359999,
      "code_expires_at": "2019-12-31T23:59:59Z",
      "frozen_membership": true,
      "max_participants": 8,
      "enforce_max_participants": true
    }
    """
    Then the response code should be 400
    And the response body should be, in JSON:
    """
    {
      "error_text": "Invalid input data",
      "errors": {
        "name": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "grade": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "description": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "is_open": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "is_public": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "open_activity_when_joining": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "root_activity_id": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "root_skill_id": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "is_official_session": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "require_members_to_join_parent": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "address_city": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "address_line1": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "address_line2": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "address_postcode": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "address_country": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "organizer": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "expected_start": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "require_personal_info_access_approval": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "require_lock_membership_approval_until": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],
        "require_watch_approval": ["only managers with 'can_manage' \u003e= 'memberships_and_group' can modify this field"],

        "code_expires_at": ["only managers with 'can_manage' \u003e= 'memberships' can modify this field"],
        "code_lifetime": ["only managers with 'can_manage' \u003e= 'memberships' can modify this field"],
        "frozen_membership": ["only managers with 'can_manage' \u003e= 'memberships' can modify this field"],
        "max_participants": ["only managers with 'can_manage' \u003e= 'memberships' can modify this field"],
        "enforce_max_participants": ["only managers with 'can_manage' \u003e= 'memberships' can modify this field"]
      },
      "message": "Bad Request",
      "success": false
    }
    """
    And the table "groups" should remain unchanged
    And the table "groups_groups" should remain unchanged
