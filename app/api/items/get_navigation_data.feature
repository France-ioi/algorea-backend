Feature: Get item for tree navigation
  Background:
    Given the database has the following table 'groups':
      | id | name            | text_id | grade | type      |
      | 11 | jdoe            |         | -2    | UserAdmin |
      | 12 | jdoe-admin      |         | -2    | UserAdmin |
      | 13 | Group B         |         | -2    | Class     |
      | 14 | gray_root       |         | -2    | UserAdmin |
      | 15 | gray_root-admin |         | -2    | UserAdmin |
      | 16 | gray_mid        |         | -2    | UserAdmin |
      | 17 | gray_mid-admin  |         | -2    | UserAdmin |
      | 18 | french          |         | -2    | UserAdmin |
      | 19 | french-admin    |         | -2    | UserAdmin |
    And the database has the following table 'languages':
      | id | code |
      | 2  | fr   |
    And the database has the following table 'users':
      | login     | temp_user | group_id | owned_group_id | default_language |
      | jdoe      | 0         | 11       | 12             |                  |
      | gray_root | 0         | 14       | 15             |                  |
      | gray_mid  | 0         | 16       | 17             |                  |
      | fr_user   | 0         | 18       | 19             | fr               |
    And the database has the following table 'groups_groups':
      | id | parent_group_id | child_group_id |
      | 61 | 13              | 11             |
    And the database has the following table 'groups_ancestors':
      | id | ancestor_group_id | child_group_id | is_self |
      | 71 | 11                | 11             | 1       |
      | 72 | 12                | 12             | 1       |
      | 73 | 13                | 13             | 1       |
      | 74 | 13                | 11             | 0       |
      | 75 | 14                | 14             | 1       |
      | 76 | 16                | 16             | 1       |
      | 77 | 18                | 18             | 1       |
    And the database has the following table 'items':
      | id  | type     | teams_editable | no_score | unlocked_item_ids | transparent_folder |
      | 200 | Category | false          | false    | 1234,2345         | true               |
      | 210 | Chapter  | false          | false    | 1234,2345         | true               |
      | 220 | Chapter  | false          | false    | 1234,2345         | true               |
      | 230 | Chapter  | false          | false    | 1234,2345         | true               |
      | 211 | Task     | false          | false    | 1234,2345         | true               |
      | 231 | Task     | false          | false    | 1234,2345         | true               |
      | 232 | Task     | false          | false    | 1234,2345         | true               |
      | 250 | Task     | false          | false    | 1234,2345         | true               |
    And the database has the following table 'groups_items':
      | id | group_id | item_id | cached_full_access_since | cached_partial_access_since | cached_grayed_access_since |
      | 43 | 13       | 200     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 44 | 13       | 210     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 45 | 13       | 220     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 46 | 13       | 230     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 47 | 13       | 211     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 48 | 13       | 231     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 49 | 13       | 232     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 42 | 13       | 250     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 50 | 14       | 200     | 3019-03-22 08:00:00      | 3019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 51 | 14       | 210     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 52 | 14       | 220     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 53 | 14       | 230     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 54 | 14       | 211     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 55 | 14       | 231     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 56 | 14       | 232     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 60 | 16       | 200     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 61 | 16       | 210     | 3019-03-22 08:00:00      | 3019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 62 | 16       | 220     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 63 | 16       | 230     | 3019-03-22 08:00:00      | 3019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 64 | 16       | 211     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 65 | 16       | 231     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 66 | 16       | 232     | 2019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 70 | 18       | 200     | 3019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 71 | 18       | 210     | 3019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 72 | 18       | 220     | 3019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 73 | 18       | 230     | 3019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 74 | 18       | 211     | 3019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 75 | 18       | 231     | 3019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
      | 76 | 18       | 232     | 3019-03-22 08:00:00      | 2019-03-22 08:00:00         | 2019-03-22 08:00:00        |
    And the database has the following table 'items_items':
      | id | parent_item_id | child_item_id | child_order | partial_access_propagation | difficulty |
      | 54 | 200            | 210           | 3           | None                       | 0          |
      | 55 | 200            | 220           | 2           | AsGrayed                   | 0          |
      | 56 | 200            | 230           | 1           | AsPartial                  | 0          |
      | 57 | 210            | 211           | 1           | None                       | 0          |
      | 58 | 230            | 231           | 2           | None                       | 0          |
      | 59 | 230            | 232           | 1           | None                       | 0          |
    And the database has the following table 'items_strings':
      | id | item_id | language_id | title       |
      | 53 | 200     | 1           | Category 1  |
      | 54 | 210     | 1           | Chapter A   |
      | 55 | 220     | 1           | Chapter B   |
      | 56 | 230     | 1           | Chapter C   |
      | 57 | 211     | 1           | Task 1      |
      | 58 | 231     | 1           | Task 2      |
      | 59 | 232     | 1           | Task 3      |
      | 63 | 200     | 2           | Catégorie 1 |
      | 64 | 210     | 2           | Chapitre A  |
      | 66 | 230     | 2           | Chapitre C  |
      | 67 | 211     | 2           | Tâche 1     |
    And the database has the following table 'groups_attempts':
      | id  | group_id | item_id | order | score | submissions_attempts | validated | finished | key_obtained | started_at          | finished_at         | validated_at        |
      | 101 | 11       | 200     | 1     | 12341 | 11                   | true      | true     | true         | 2019-01-30 09:26:41 | 2019-02-01 09:26:41 | 2019-01-31 09:26:41 |
      | 102 | 11       | 210     | 1     | 12342 | 12                   | true      | true     | true         | 2019-01-30 09:26:42 | 2019-02-01 09:26:42 | 2019-01-31 09:26:42 |
      | 105 | 11       | 211     | 1     | 12343 | 13                   | true      | true     | true         | 2019-01-30 09:26:43 | 2019-02-01 09:26:43 | 2019-01-31 09:26:43 |
      | 103 | 11       | 220     | 1     | 12344 | 14                   | true      | true     | true         | 2019-01-30 09:26:44 | 2019-02-01 09:26:44 | 2019-01-31 09:26:44 |
      | 104 | 11       | 230     | 1     | 12345 | 15                   | true      | true     | true         | 2019-01-30 09:26:45 | 2019-02-01 09:26:45 | 2019-01-31 09:26:45 |
      | 106 | 11       | 231     | 1     | 12346 | 16                   | true      | true     | true         | 2019-01-30 09:26:46 | 2019-02-01 09:26:46 | 2019-01-31 09:26:46 |
      | 107 | 11       | 232     | 1     | 12347 | 17                   | true      | true     | true         | 2019-01-30 09:26:47 | 2019-02-01 09:26:47 | 2019-01-31 09:26:47 |
    And the database has the following table 'users_items':
      | user_group_id | item_id | active_attempt_id |
      | 11            | 200     | 101               |
      | 11            | 210     | 102               |
      | 11            | 211     | 105               |
      | 11            | 220     | 103               |
      | 11            | 230     | 104               |
      | 11            | 231     | 106               |
      | 11            | 232     | 107               |

  Scenario: Get tree structure
    Given I am the user with group_id "11"
    When I send a GET request to "/items/200/as-nav-tree"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "id": "200",
        "type": "Category",
        "transparent_folder": true,
        "has_unlocked_items": true,
        "string": {
          "title": "Category 1"
        },
        "user_active_attempt": {
          "score": 12341,
          "validated": true,
          "finished": true,
          "key_obtained": true,
          "submissions_attempts": 11,
          "started_at": "2019-01-30T09:26:41Z",
          "validated_at": "2019-01-31T09:26:41Z",
          "finished_at": "2019-02-01T09:26:41Z"
        },
        "access_rights": {
          "partial_access": true,
          "full_access": true,
          "gray_access": true
        },
        "children": [
          {
            "id": "230",
            "order": 1,
            "partial_access_propagation": "AsPartial",
            "type": "Chapter",
            "transparent_folder": true,
            "has_unlocked_items": true,
            "string": {
              "title": "Chapter C"
            },
            "user_active_attempt": {
              "score": 12345,
              "validated": true,
              "finished": true,
              "key_obtained": true,
              "submissions_attempts": 15,
              "started_at": "2019-01-30T09:26:45Z",
              "validated_at": "2019-01-31T09:26:45Z",
              "finished_at": "2019-02-01T09:26:45Z"
            },
            "access_rights": {
              "partial_access": true,
              "full_access": true,
              "gray_access": true
            },
            "children": [
              {
                "id": "232",
                "order": 1,
                "partial_access_propagation": "None",
                "type": "Task",
                "transparent_folder": true,
                "has_unlocked_items": true,
                "string": {
                  "title": "Task 3"
                },
                "user_active_attempt": {
                  "score": 12347,
                  "validated": true,
                  "finished": true,
                  "key_obtained": true,
                  "submissions_attempts": 17,
                  "started_at": "2019-01-30T09:26:47Z",
                  "validated_at": "2019-01-31T09:26:47Z",
                  "finished_at": "2019-02-01T09:26:47Z"
                },
                "access_rights": {
                  "partial_access": true,
                  "full_access": true,
                  "gray_access": true
                },
                "children": null
              },
              {
                "id": "231",
                "order": 2,
                "partial_access_propagation": "None",
                "type": "Task",
                "transparent_folder": true,
                "has_unlocked_items": true,
                "string": {
                  "title": "Task 2"
                },
                "user_active_attempt": {
                  "score": 12346,
                  "validated": true,
                  "finished": true,
                  "key_obtained": true,
                  "submissions_attempts": 16,
                  "started_at": "2019-01-30T09:26:46Z",
                  "validated_at": "2019-01-31T09:26:46Z",
                  "finished_at": "2019-02-01T09:26:46Z"
                },
                "access_rights": {
                  "partial_access": true,
                  "full_access": true,
                  "gray_access": true
                },
                "children": null
              }
            ]
          },
          {
            "id": "220",
            "order": 2,
            "partial_access_propagation": "AsGrayed",
            "type": "Chapter",
            "transparent_folder": true,
            "has_unlocked_items": true,
            "string": {
              "title": "Chapter B"
            },
            "user_active_attempt": {
              "score": 12344,
              "validated": true,
              "finished": true,
              "key_obtained": true,
              "submissions_attempts": 14,
              "started_at": "2019-01-30T09:26:44Z",
              "validated_at": "2019-01-31T09:26:44Z",
              "finished_at": "2019-02-01T09:26:44Z"
            },
            "access_rights": {
              "partial_access": true,
              "full_access": true,
              "gray_access": true
            },
            "children": []
          },
          {
            "id": "210",
            "order": 3,
            "partial_access_propagation": "None",
            "type": "Chapter",
            "transparent_folder": true,
            "has_unlocked_items": true,
            "string": {
              "title": "Chapter A"
            },
            "user_active_attempt": {
              "score": 12342,
              "validated": true,
              "finished": true,
              "key_obtained": true,
              "submissions_attempts": 12,
              "started_at": "2019-01-30T09:26:42Z",
              "validated_at": "2019-01-31T09:26:42Z",
              "finished_at": "2019-02-01T09:26:42Z"
            },
            "access_rights": {
              "partial_access": true,
              "full_access": true,
              "gray_access": true
            },
            "children": [
              {
                "id": "211",
                "order": 1,
                "partial_access_propagation": "None",
                "type": "Task",
                "transparent_folder": true,
                "has_unlocked_items": true,
                "string": {
                  "title": "Task 1"
                },
                "user_active_attempt": {
                  "score": 12343,
                  "validated": true,
                  "finished": true,
                  "key_obtained": true,
                  "submissions_attempts": 13,
                  "started_at": "2019-01-30T09:26:43Z",
                  "validated_at": "2019-01-31T09:26:43Z",
                  "finished_at": "2019-02-01T09:26:43Z"
                },
                "access_rights": {
                  "partial_access": true,
                  "full_access": true,
                  "gray_access": true
                },
                "children": null
              }
            ]
          }
        ]
      }
      """

  Scenario: Should return only one node if the root item doesn't have children
    Given I am the user with group_id "11"
    When I send a GET request to "/items/232/as-nav-tree"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "id": "232",
        "type": "Task",
        "transparent_folder": true,
        "has_unlocked_items": true,
        "string": {
          "title": "Task 3"
        },
        "user_active_attempt": {
          "score": 12347,
          "validated": true,
          "finished": true,
          "key_obtained": true,
          "submissions_attempts": 17,
          "started_at": "2019-01-30T09:26:47Z",
          "validated_at": "2019-01-31T09:26:47Z",
          "finished_at": "2019-02-01T09:26:47Z"
        },
        "access_rights": {
          "partial_access": true,
          "full_access": true,
          "gray_access": true
        },
        "children": []
      }
      """

  Scenario: Should return a subtree having two levels if the root item doesn't have grandchildren
    Given I am the user with group_id "11"
    When I send a GET request to "/items/230/as-nav-tree"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "id": "230",
        "type": "Chapter",
        "transparent_folder": true,
        "has_unlocked_items": true,
        "string": {
          "title": "Chapter C"
        },
        "user_active_attempt": {
          "score": 12345,
          "validated": true,
          "finished": true,
          "key_obtained": true,
          "submissions_attempts": 15,
          "started_at": "2019-01-30T09:26:45Z",
          "validated_at": "2019-01-31T09:26:45Z",
          "finished_at": "2019-02-01T09:26:45Z"
        },
        "access_rights": {
          "partial_access": true,
          "full_access": true,
          "gray_access": true
        },
        "children": [
          {
            "id": "232",
            "order": 1,
            "partial_access_propagation": "None",
            "type": "Task",
            "transparent_folder": true,
            "has_unlocked_items": true,
            "string": {
              "title": "Task 3"
            },
            "user_active_attempt": {
              "score": 12347,
              "validated": true,
              "finished": true,
              "key_obtained": true,
              "submissions_attempts": 17,
              "started_at": "2019-01-30T09:26:47Z",
              "validated_at": "2019-01-31T09:26:47Z",
              "finished_at": "2019-02-01T09:26:47Z"
            },
            "access_rights": {
              "partial_access": true,
              "full_access": true,
              "gray_access": true
            },
            "children": []
          },
          {
            "id": "231",
            "order": 2,
            "partial_access_propagation": "None",
            "type": "Task",
            "transparent_folder": true,
            "has_unlocked_items": true,
            "string": {
              "title": "Task 2"
            },
            "user_active_attempt": {
              "score": 12346,
              "validated": true,
              "finished": true,
              "key_obtained": true,
              "submissions_attempts": 16,
              "started_at": "2019-01-30T09:26:46Z",
              "validated_at": "2019-01-31T09:26:46Z",
              "finished_at": "2019-02-01T09:26:46Z"
            },
            "access_rights": {
              "partial_access": true,
              "full_access": true,
              "gray_access": true
            },
            "children": []
          }
        ]
      }
      """

  Scenario: Should return only one node if the user has only grayed access to the root item
    Given I am the user with group_id "14"
    When I send a GET request to "/items/200/as-nav-tree"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "id": "200",
        "type": "Category",
        "transparent_folder": true,
        "has_unlocked_items": true,
        "string": {
          "title": "Category 1"
        },
        "user_active_attempt": null,
        "access_rights": {
          "partial_access": false,
          "full_access": false,
          "gray_access": true
        },
        "children": []
      }
      """

  Scenario: Should skip children of grayed nodes
    Given I am the user with group_id "16"
    When I send a GET request to "/items/200/as-nav-tree"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "id": "200",
        "type": "Category",
        "transparent_folder": true,
        "has_unlocked_items": true,
        "string": {
          "title": "Category 1"
        },
        "user_active_attempt": null,
        "access_rights": {
          "partial_access": true,
          "full_access": true,
          "gray_access": true
        },
        "children": [
          {
            "id": "230",
            "order": 1,
            "partial_access_propagation": "AsPartial",
            "type": "Chapter",
            "transparent_folder": true,
            "has_unlocked_items": true,
            "string": {
              "title": "Chapter C"
            },
            "user_active_attempt": null,
            "access_rights": {
              "partial_access": false,
              "full_access": false,
              "gray_access": true
            },
            "children": []
          },
          {
            "id": "220",
            "order": 2,
            "partial_access_propagation": "AsGrayed",
            "type": "Chapter",
            "transparent_folder": true,
            "has_unlocked_items": true,
            "string": {
              "title": "Chapter B"
            },
            "user_active_attempt": null,
            "access_rights": {
              "partial_access": true,
              "full_access": true,
              "gray_access": true
            },
            "children": []
          },
          {
            "id": "210",
            "order": 3,
            "partial_access_propagation": "None",
            "type": "Chapter",
            "transparent_folder": true,
            "has_unlocked_items": true,
            "string": {
              "title": "Chapter A"
            },
            "user_active_attempt": null,
            "access_rights": {
              "partial_access": false,
              "full_access": false,
              "gray_access": true
            },
            "children": []
          }
        ]
      }
      """

  Scenario: Should prefer the user's default language for titles
    Given I am the user with group_id "18"
    When I send a GET request to "/items/200/as-nav-tree"
    Then the response code should be 200
    And the response body should be, in JSON:
      """
      {
        "id": "200",
        "type": "Category",
        "transparent_folder": true,
        "has_unlocked_items": true,
        "string": {
          "title": "Catégorie 1"
        },
        "user_active_attempt": null,
        "access_rights": {
          "partial_access": true,
          "full_access": false,
          "gray_access": true
        },
        "children": [
          {
            "id": "230",
            "order": 1,
            "partial_access_propagation": "AsPartial",
            "type": "Chapter",
            "transparent_folder": true,
            "has_unlocked_items": true,
            "string": {
              "title": "Chapitre C"
            },
            "user_active_attempt": null,
            "access_rights": {
              "partial_access": true,
              "full_access": false,
              "gray_access": true
            },
            "children": [
              {
                "id": "232",
                "order": 1,
                "partial_access_propagation": "None",
                "type": "Task",
                "transparent_folder": true,
                "has_unlocked_items": true,
                "string": {
                  "title": "Task 3"
                },
                "user_active_attempt": null,
                "access_rights": {
                  "partial_access": true,
                  "full_access": false,
                  "gray_access": true
                },
                "children": null
              },
              {
                "id": "231",
                "order": 2,
                "partial_access_propagation": "None",
                "type": "Task",
                "transparent_folder": true,
                "has_unlocked_items": true,
                "string": {
                  "title": "Task 2"
                },
                "user_active_attempt": null,
                "access_rights": {
                  "partial_access": true,
                  "full_access": false,
                  "gray_access": true
                },
                "children": null
              }
            ]
          },
          {
            "id": "220",
            "order": 2,
            "partial_access_propagation": "AsGrayed",
            "type": "Chapter",
            "transparent_folder": true,
            "has_unlocked_items": true,
            "string": {
              "title": "Chapter B"
            },
            "user_active_attempt": null,
            "access_rights": {
              "partial_access": true,
              "full_access": false,
              "gray_access": true
            },
            "children": []
          },
          {
            "id": "210",
            "order": 3,
            "partial_access_propagation": "None",
            "type": "Chapter",
            "transparent_folder": true,
            "has_unlocked_items": true,
            "string": {
              "title": "Chapitre A"
            },
            "user_active_attempt": null,
            "access_rights": {
              "partial_access": true,
              "full_access": false,
              "gray_access": true
            },
            "children": [
              {
                "id": "211",
                "order": 1,
                "partial_access_propagation": "None",
                "type": "Task",
                "transparent_folder": true,
                "has_unlocked_items": true,
                "string": {
                  "title": "Tâche 1"
                },
                "user_active_attempt": null,
                "access_rights": {
                  "partial_access": true,
                  "full_access": false,
                  "gray_access": true
                },
                "children": null
              }
            ]
          }
        ]
      }
      """
