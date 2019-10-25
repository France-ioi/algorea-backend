Feature: Get group invitations for the current user
  Background:
    Given the database has the following table 'groups':
      | id | type      | name               | description            |
      | 1  | Class     | Our Class          | Our class group        |
      | 2  | Team      | Our Team           | Our team group         |
      | 3  | Club      | Our Club           | Our club group         |
      | 4  | Friends   | Our Friends        | Group for our friends  |
      | 5  | Other     | Other people       | Group for other people |
      | 6  | Class     | Another Class      | Another class group    |
      | 7  | Team      | Another Team       | Another team group     |
      | 8  | Club      | Another Club       | Another club group     |
      | 9  | Friends   | Some other friends | Another friends group  |
      | 10 | Other     | Secret group       | Our secret group       |
      | 11 | UserSelf  | user self          |                        |
      | 12 | UserAdmin | user admin         |                        |
      | 21 | UserSelf  | owner self         |                        |
      | 22 | UserAdmin | owner admin        |                        |
    And the database has the following table 'users':
      | login | temp_user | group_id | owned_group_id | first_name  | last_name | grade |
      | owner | 0         | 21       | 22             | Jean-Michel | Blanquer  | 3     |
      | user  | 0         | 11       | 12             | John        | Doe       | 1     |
    And the database has the following table 'groups_groups':
      | id | parent_group_id | child_group_id | type               | type_changed_at           | inviting_user_group_id |
      | 2  | 1               | 21             | invitationSent     | {{relativeTime("-169h")}} | null                   |
      | 3  | 2               | 21             | invitationRefused  | {{relativeTime("-168h")}} | 21                     |
      | 4  | 3               | 21             | requestSent        | {{relativeTime("-167h")}} | 21                     |
      | 5  | 4               | 21             | requestRefused     | {{relativeTime("-166h")}} | 11                     |
      | 6  | 5               | 21             | invitationAccepted | {{relativeTime("-165h")}} | 11                     |
      | 7  | 6               | 21             | requestAccepted    | {{relativeTime("-164h")}} | 11                     |
      | 8  | 7               | 21             | removed            | {{relativeTime("-163h")}} | 21                     |
      | 9  | 8               | 21             | left               | {{relativeTime("-162h")}} | 21                     |
      | 10 | 9               | 21             | direct             | {{relativeTime("-161h")}} | 11                     |
      | 11 | 1               | 22             | invitationSent     | {{relativeTime("-170h")}} | 11                     |
      | 12 | 10              | 21             | joinedByCode       | {{relativeTime("-180h")}} | null                   |

  Scenario: Show all invitations
    Given I am the user with group_id "21"
    When I send a GET request to "/current-user/group-invitations"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    [
      {
        "id": "5",
        "inviting_user": {
          "group_id": "11",
          "first_name": "John",
          "last_name": "Doe",
          "login": "user"
        },
        "group": {
          "id": "4",
          "name": "Our Friends",
          "description": "Group for our friends",
          "type": "Friends"
        },
        "type_changed_at": "{{timeToRFC(db("groups_groups[4][type_changed_at]"))}}",
        "type": "requestRefused"
      },
      {
        "id": "4",
        "inviting_user": {
          "group_id": "21",
          "first_name": "Jean-Michel",
          "last_name": "Blanquer",
          "login": "owner"
        },
        "group": {
          "id": "3",
          "name": "Our Club",
          "description": "Our club group",
          "type": "Club"
        },
        "type_changed_at": "{{timeToRFC(db("groups_groups[3][type_changed_at]"))}}",
        "type": "requestSent"
      },
      {
        "id": "2",
        "inviting_user": null,
        "group": {
          "id": "1",
          "name": "Our Class",
          "description": "Our class group",
          "type": "Class"
        },
        "type_changed_at": "{{timeToRFC(db("groups_groups[1][type_changed_at]"))}}",
        "type": "invitationSent"
      }
    ]
    """

  Scenario: Request the first row
    Given I am the user with group_id "21"
    When I send a GET request to "/current-user/group-invitations?limit=1"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    [
      {
        "id": "5",
        "inviting_user": {
          "group_id": "11",
          "first_name": "John",
          "last_name": "Doe",
          "login": "user"
        },
        "group": {
          "id": "4",
          "name": "Our Friends",
          "description": "Group for our friends",
          "type": "Friends"
        },
        "type_changed_at": "{{timeToRFC(db("groups_groups[4][type_changed_at]"))}}",
        "type": "requestRefused"
      }
    ]
    """

  Scenario: Filter out old invitations
    Given I am the user with group_id "21"
    When I send a GET request to "/current-user/group-invitations?within_weeks=1"
    Then the response code should be 200
    And the response body should be, in JSON:
    """
    [
      {
        "id": "5",
        "inviting_user": {
          "group_id": "11",
          "first_name": "John",
          "last_name": "Doe",
          "login": "user"
        },
        "group": {
          "id": "4",
          "name": "Our Friends",
          "description": "Group for our friends",
          "type": "Friends"
        },
        "type_changed_at": "{{timeToRFC(db("groups_groups[4][type_changed_at]"))}}",
        "type": "requestRefused"
      },
      {
        "id": "4",
        "inviting_user": {
          "group_id": "21",
          "first_name": "Jean-Michel",
          "last_name": "Blanquer",
          "login": "owner"
        },
        "group": {
          "id": "3",
          "name": "Our Club",
          "description": "Our club group",
          "type": "Club"
        },
        "type_changed_at": "{{timeToRFC(db("groups_groups[3][type_changed_at]"))}}",
        "type": "requestSent"
      }
    ]
    """
