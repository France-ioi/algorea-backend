//go:build !unit

package database_test

import (
	"net/http/httptest"
	"testing"

	"github.com/France-ioi/AlgoreaBackend/v2/app/database"
	"github.com/France-ioi/AlgoreaBackend/v2/golang"
	"github.com/France-ioi/AlgoreaBackend/v2/testhelpers"
)

// This test checks that running the item ancestors propagation in a separate transaction doesn't introduce a breach
// allowing to create a cycle in the item relations graph. The test performs the following scenario:
//  1. There are two users: john and jane. There are two skills: 1 and 2.
//  2. Both users have permissions to view content and edit children of both skills.
//  3. John creates a relation between the skills: 1 is a parent of 2.
//  4. Right after John created the relation, but before the item ancestors propagation is started,
//     Jane tries to create a relation between the skills: 2 is a parent of 1 .
//  5. Jane's request should return 403 because it's not allowed to create a cycle.
func TestPropagation_RunningItemAncestorsPropagationInSeparateTransactionDoesntIntroduceBreachAllowingToCreateCycleInItemGraph(
	t *testing.T,
) {
	testhelpers.NewPropagationVerifier(golang.NewSet(database.PropagationStepItemAncestorsInit)).
		WithFixture(`
			groups:
				- {id: 3, type: User}
				- {id: 4, type: User}

			users:
				- {group_id: 3, login: john}
				- {group_id: 4, login: jane}

			items:
				- {id: 1, type: Skill, default_language_tag: 'en'}
				- {id: 2, type: Skill, default_language_tag: 'en'}

			permissions_granted:
				- {group_id: 3, item_id: 1, can_view: 'content', can_edit: 'children', source_group_id: 1}
				- {group_id: 3, item_id: 2, can_view: 'content', can_edit: 'children', source_group_id: 1}
				- {group_id: 4, item_id: 1, can_view: 'content', can_edit: 'children', source_group_id: 1}
				- {group_id: 4, item_id: 2, can_view: 'content', can_edit: 'children', source_group_id: 1}

			sessions:
				- {session_id: 3, user_id: 3}
				- {session_id: 4, user_id: 4}

			access_tokens:
				- {token: 'token_john', session_id: 3, expires_at: '9999-12-31 23:59:59'}
				- {token: 'token_jane', session_id: 4, expires_at: '9999-12-31 23:59:59'}
		`).
		WithHook(func(step database.PropagationStep, _, _ int, _ *database.DataStore, appServer *httptest.Server) {
			if step == database.PropagationStepItemAncestorsInit {
				// Should return 403 because it's not allowed to create a cycle
				testhelpers.VerifyTestHTTPRequestWithToken(t, appServer, "token_jane", 403,
					"PUT", "/items/2", nil, map[string]interface{}{
						"children": []map[string]interface{}{
							{"item_id": 1, "order": 1},
						},
					})
			}
		}).
		Run(t, func(dataStore *database.DataStore, appServer *httptest.Server) {
			testhelpers.VerifyTestHTTPRequestWithToken(t, appServer, "token_john", 200,
				"PUT", "/items/1", nil, map[string]interface{}{
					"children": []map[string]interface{}{
						{"item_id": 2, "order": 1},
					},
				})
		})
}
