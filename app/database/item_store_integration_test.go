//go:build !unit

package database_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/testhelpers"
)

func setupDB() *database.DB {
	return testhelpers.SetupDBWithFixture("visibility")
}

func TestItemStore_VisibleMethods(t *testing.T) {
	tests := []struct {
		methodToCall string
		args         []interface{}
		column       string
		expected     []int64
	}{
		{methodToCall: "Visible", column: "id", expected: []int64{190, 191, 192, 1900, 1901, 1902, 19000, 19001, 19002}},
		{methodToCall: "VisibleByID", args: []interface{}{int64(191)}, column: "id", expected: []int64{191}},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.methodToCall, func(t *testing.T) {
			db := setupDB()
			defer func() { _ = db.Close() }()

			groupID := int64(11)
			dataStore := database.NewDataStore(db)
			itemStore := dataStore.Items()

			var result []int64
			parameters := make([]reflect.Value, 0, len(testCase.args)+1)
			parameters = append(parameters, reflect.ValueOf(groupID))
			for _, arg := range testCase.args {
				parameters = append(parameters, reflect.ValueOf(arg))
			}
			db = reflect.ValueOf(itemStore).MethodByName(testCase.methodToCall).
				Call(parameters)[0].Interface().(*database.DB).Pluck(testCase.column, &result)
			assert.NoError(t, db.Error())

			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestItemStore_CheckSubmissionRights(t *testing.T) {
	db := testhelpers.SetupDBWithFixture("item_store/check_submission_rights")
	defer func() { _ = db.Close() }()

	tests := []struct {
		name          string
		participantID int64
		attemptID     int64
		itemID        int64
		wantHasAccess bool
		wantReason    error
		wantError     error
	}{
		{name: "normal", participantID: 10, attemptID: 1, itemID: 13, wantHasAccess: true, wantReason: nil, wantError: nil},
		{
			name: "read-only", participantID: 10, attemptID: 2, itemID: 12, wantHasAccess: false,
			wantReason: errors.New("item is read-only"), wantError: nil,
		},
		{
			name: "no access", participantID: 11, attemptID: 1, itemID: 10, wantHasAccess: false,
			wantReason: errors.New("no access to the task item"), wantError: nil,
		},
		{
			name: "info access", participantID: 11, attemptID: 2, itemID: 10, wantHasAccess: false,
			wantReason: errors.New("no access to the task item"), wantError: nil,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			assert.NoError(t, database.NewDataStore(db).InTransaction(func(store *database.DataStore) error {
				hasAccess, reason, err := store.Items().CheckSubmissionRights(test.participantID, test.itemID)
				assert.Equal(t, test.wantHasAccess, hasAccess)
				assert.Equal(t, test.wantReason, reason)
				assert.Equal(t, test.wantError, err)
				assert.NoError(t, err)
				return nil
			}))
		})
	}
}

func TestItemStore_GetItemIDFromTextID(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [
			{id: 11, text_id: "id11", default_language_tag: fr},
			{id: 12, text_id: "id12", default_language_tag: fr},
			{id: 13, text_id: "id13", default_language_tag: fr},
			{id: 14, default_language_tag: fr}
		]
	`)
	defer func() { _ = db.Close() }()

	tests := []struct {
		name       string
		textID     string
		wantItemID int64
		wantError  error
	}{
		{name: "Should retrieve the corresponding item", textID: "id12", wantItemID: 12, wantError: nil},
		{
			name: "Should return an error if textID is empty", textID: "", wantItemID: 0,
			wantError: errors.New("record not found"),
		},
		{
			name: "Should return an error if no corresponding item", textID: "doesn't exist", wantItemID: 0,
			wantError: errors.New("record not found"),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			assert.NoError(t, database.NewDataStore(db).InTransaction(func(store *database.DataStore) error {
				itemID, err := store.Items().GetItemIDFromTextID(test.textID)
				assert.Equal(t, test.wantItemID, itemID)
				assert.Equal(t, test.wantError, err)
				return nil
			}))
		})
	}
}

func TestItemStore_IsValidParticipationHierarchyForParentAttempt_And_BreadcrumbsHierarchyForParentAttempt(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items:
			- {id: 1, default_language_tag: fr, allows_multiple_attempts: 1}
			- {id: 2, default_language_tag: fr}
			- {id: 3, default_language_tag: fr, allows_multiple_attempts: 1}
			- {id: 4, default_language_tag: fr}
			- {id: 5, default_language_tag: fr, allows_multiple_attempts: 1}
			- {id: 6, default_language_tag: fr}
			- {id: 7, default_language_tag: fr, allows_multiple_attempts: 1}
			- {id: 8, default_language_tag: fr, allows_multiple_attempts: 1}
			- {id: 9, default_language_tag: fr}
		items_items:
			- {parent_item_id: 1, child_item_id: 3, child_order: 1}
			- {parent_item_id: 3, child_item_id: 5, child_order: 1}
			- {parent_item_id: 5, child_item_id: 7, child_order: 1}

			- {parent_item_id: 2, child_item_id: 4, child_order: 1}
			- {parent_item_id: 4, child_item_id: 6, child_order: 1}
			- {parent_item_id: 6, child_item_id: 8, child_order: 1}
			- {parent_item_id: 9, child_item_id: 6, child_order: 1}
		groups:
			- {id: 50, root_activity_id: 4}
			- {id: 100, root_activity_id: 2}
			- {id: 101, root_activity_id: 4}
			- {id: 102}
			- {id: 103, root_activity_id: 2}
			- {id: 104, root_activity_id: 2}
			- {id: 105, root_activity_id: 2}
			- {id: 106, root_activity_id: 2}
			- {id: 107, root_activity_id: 2}
			- {id: 108, root_activity_id: 2}
			- {id: 109, root_activity_id: 2}
			- {id: 110, root_activity_id: 2}
			- {id: 111, root_activity_id: 2}
			- {id: 112, root_activity_id: 2}
			- {id: 113, root_activity_id: 2}
			- {id: 114, root_activity_id: 2}
			- {id: 115, root_activity_id: 2}
			- {id: 116, root_activity_id: 2}
			- {id: 117, root_activity_id: 2}
			- {id: 118, root_activity_id: 2}
			- {id: 119, root_activity_id: 1}
			- {id: 120, root_skill_id: 4}
			- {id: 121}
			- {id: 122, root_activity_id: 9}
			- {id: 123}
			- {id: 124}
			- {id: 125, root_skill_id: 9}
			- {id: 126}
			- {id: 127}
		groups_groups:
			- {parent_group_id: 50, child_group_id: 102}
			- {parent_group_id: 120, child_group_id: 121}
			- {parent_group_id: 123, child_group_id: 102}
			- {parent_group_id: 124, child_group_id: 122}
			- {parent_group_id: 126, child_group_id: 121}
			- {parent_group_id: 127, child_group_id: 125}
		group_managers:
			- {manager_id: 101, group_id: 122}
			- {manager_id: 123, group_id: 124}
			- {manager_id: 120, group_id: 125}
			- {manager_id: 126, group_id: 127}
		permissions_generated:
			- {group_id: 50, item_id: 4, can_view_generated: content}
			- {group_id: 100, item_id: 2, can_view_generated: content}
			- {group_id: 100, item_id: 4, can_view_generated: content}
			- {group_id: 101, item_id: 4, can_view_generated: content}
			- {group_id: 101, item_id: 6, can_view_generated: content}
			- {group_id: 101, item_id: 8, can_view_generated: content}
			- {group_id: 101, item_id: 9, can_view_generated: content}
			- {group_id: 102, item_id: 6, can_view_generated: content}
			- {group_id: 102, item_id: 9, can_view_generated: content}
			- {group_id: 103, item_id: 2, can_view_generated: content}
			- {group_id: 103, item_id: 4, can_view_generated: info}
			- {group_id: 104, item_id: 2, can_view_generated: content}
			- {group_id: 104, item_id: 4, can_view_generated: none}
			- {group_id: 105, item_id: 2, can_view_generated: content}
			- {group_id: 105, item_id: 4, can_view_generated: info}
			- {group_id: 105, item_id: 6, can_view_generated: content}
			- {group_id: 106, item_id: 2, can_view_generated: info}
			- {group_id: 106, item_id: 4, can_view_generated: content}
			- {group_id: 106, item_id: 6, can_view_generated: content}
			- {group_id: 107, item_id: 2, can_view_generated: content}
			- {group_id: 107, item_id: 4, can_view_generated: content}
			- {group_id: 107, item_id: 6, can_view_generated: content}
			- {group_id: 108, item_id: 2, can_view_generated: content}
			- {group_id: 108, item_id: 4, can_view_generated: content}
			- {group_id: 108, item_id: 6, can_view_generated: content}
			- {group_id: 109, item_id: 2, can_view_generated: content}
			- {group_id: 109, item_id: 4, can_view_generated: content}
			- {group_id: 109, item_id: 6, can_view_generated: content}
			- {group_id: 110, item_id: 2, can_view_generated: content}
			- {group_id: 110, item_id: 4, can_view_generated: content}
			- {group_id: 110, item_id: 6, can_view_generated: content}
			- {group_id: 111, item_id: 2, can_view_generated: content}
			- {group_id: 111, item_id: 4, can_view_generated: content}
			- {group_id: 111, item_id: 6, can_view_generated: content}
			- {group_id: 112, item_id: 2, can_view_generated: content}
			- {group_id: 112, item_id: 4, can_view_generated: content}
			- {group_id: 112, item_id: 6, can_view_generated: content}
			- {group_id: 113, item_id: 2, can_view_generated: content}
			- {group_id: 113, item_id: 4, can_view_generated: content}
			- {group_id: 113, item_id: 6, can_view_generated: content}
			- {group_id: 114, item_id: 2, can_view_generated: content}
			- {group_id: 114, item_id: 4, can_view_generated: content}
			- {group_id: 114, item_id: 6, can_view_generated: content}
			- {group_id: 114, item_id: 8, can_view_generated: content}
			- {group_id: 115, item_id: 2, can_view_generated: content}
			- {group_id: 115, item_id: 4, can_view_generated: content}
			- {group_id: 115, item_id: 6, can_view_generated: content}
			- {group_id: 115, item_id: 8, can_view_generated: content}
			- {group_id: 116, item_id: 2, can_view_generated: content}
			- {group_id: 116, item_id: 4, can_view_generated: content}
			- {group_id: 116, item_id: 6, can_view_generated: content}
			- {group_id: 116, item_id: 8, can_view_generated: content}
			- {group_id: 117, item_id: 2, can_view_generated: content}
			- {group_id: 117, item_id: 4, can_view_generated: content}
			- {group_id: 117, item_id: 6, can_view_generated: content}
			- {group_id: 117, item_id: 8, can_view_generated: content}
			- {group_id: 118, item_id: 2, can_view_generated: content}
			- {group_id: 118, item_id: 4, can_view_generated: content}
			- {group_id: 118, item_id: 6, can_view_generated: content}
			- {group_id: 118, item_id: 8, can_view_generated: content}
			- {group_id: 119, item_id: 1, can_view_generated: content}
			- {group_id: 119, item_id: 3, can_view_generated: content}
			- {group_id: 119, item_id: 5, can_view_generated: content}
			- {group_id: 119, item_id: 7, can_view_generated: content}
			- {group_id: 120, item_id: 4, can_view_generated: content}
			- {group_id: 120, item_id: 6, can_view_generated: content}
			- {group_id: 120, item_id: 9, can_view_generated: content}
			- {group_id: 121, item_id: 4, can_view_generated: content}
			- {group_id: 121, item_id: 6, can_view_generated: content}
			- {group_id: 121, item_id: 9, can_view_generated: content}
		attempts:
			- {participant_id: 100, id: 0}
			- {participant_id: 100, id: 200, root_item_id: 2, parent_attempt_id: 0}
			- {participant_id: 101, id: 200, root_item_id: 4, allows_submissions_until: 3019-06-30 11:00:00}
			- {participant_id: 101, id: 201, root_item_id: 6, parent_attempt_id: 200, allows_submissions_until: 3019-06-30 11:00:00}
			- {participant_id: 101, id: 202, root_item_id: 9, allows_submissions_until: 3019-06-30 11:00:00}
			- {participant_id: 101, id: 203, root_item_id: 6, parent_attempt_id: 202, allows_submissions_until: 3019-06-30 11:00:00}
			- {participant_id: 102, id: 200, root_item_id: 4}
			- {participant_id: 102, id: 201, root_item_id: 9}
			- {participant_id: 103, id: 200, root_item_id: 2}
			- {participant_id: 104, id: 200, root_item_id: 2}
			- {participant_id: 105, id: 200, root_item_id: 2}
			- {participant_id: 105, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 106, id: 200, root_item_id: 2}
			- {participant_id: 106, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 107, id: 200, root_item_id: 2}
			- {participant_id: 107, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 108, id: 200, root_item_id: 2}
			- {participant_id: 108, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 109, id: 200, root_item_id: 2, allows_submissions_until: 2019-06-30 11:00:00}
			- {participant_id: 109, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 110, id: 200, root_item_id: 2}
			- {participant_id: 110, id: 201, root_item_id: 4, parent_attempt_id: 200, allows_submissions_until: 2019-06-30 11:00:00}
			- {participant_id: 111, id: 200, root_item_id: 2, ended_at: 2019-06-30 11:00:00}
			- {participant_id: 111, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 112, id: 200, root_item_id: 2}
			- {participant_id: 112, id: 201, root_item_id: 4, parent_attempt_id: 200, ended_at: 2019-06-30 11:00:00}
			- {participant_id: 113, id: 200}
			- {participant_id: 114, id: 0}
			- {participant_id: 114, id: 200, root_item_id: 2}
			- {participant_id: 114, id: 201, root_item_id: 4, parent_attempt_id: 0}
			- {participant_id: 115, id: 0}
			- {participant_id: 115, id: 200, parent_attempt_id: 0}
			- {participant_id: 116, id: 0}
			- {participant_id: 116, id: 200}
			- {participant_id: 116, id: 201, root_item_id: 6, parent_attempt_id: 0}
			- {participant_id: 117, id: 0}
			- {participant_id: 117, id: 200, parent_attempt_id: 0}
			- {participant_id: 118, id: 0}
			- {participant_id: 118, id: 200, root_item_id: 6, parent_attempt_id: 0}
			- {participant_id: 119, id: 0}
			- {participant_id: 119, id: 150, parent_attempt_id: 0}
			- {participant_id: 119, id: 200, root_item_id: 5, parent_attempt_id: 0}
			- {participant_id: 119, id: 250, parent_attempt_id: 0}
			- {participant_id: 120, id: 200, root_item_id: 4}
			- {participant_id: 120, id: 201, root_item_id: 9}
			- {participant_id: 121, id: 200, root_item_id: 4}
			- {participant_id: 121, id: 201, root_item_id: 9}
		results:
			- {participant_id: 100, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 101, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 101, attempt_id: 201, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 101, attempt_id: 202, item_id: 9, started_at: 2019-05-30 11:00:00}
			- {participant_id: 101, attempt_id: 203, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 102, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 102, attempt_id: 201, item_id: 9, started_at: 2019-05-30 11:00:00}
			- {participant_id: 103, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 104, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 105, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 105, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 106, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 106, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 107, attempt_id: 200, item_id: 2, started_at: null}
			- {participant_id: 107, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 108, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 108, attempt_id: 201, item_id: 4, started_at: null}
			- {participant_id: 109, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 109, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 110, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 110, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 111, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 111, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 112, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 112, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 113, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 113, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 114, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 114, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 114, attempt_id: 201, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 115, attempt_id: 0, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 115, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 115, attempt_id: 200, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 116, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 116, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 116, attempt_id: 201, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 117, attempt_id: 0, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 117, attempt_id: 0, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 117, attempt_id: 200, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 118, attempt_id: 0, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 118, attempt_id: 0, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 118, attempt_id: 200, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 119, attempt_id: 0, item_id: 1, started_at: 2019-05-30 11:00:00}
			- {participant_id: 119, attempt_id: 100, item_id: 1, started_at: 2019-05-29 11:00:00}
			- {participant_id: 119, attempt_id: 0, item_id: 3, started_at: 2019-05-29 11:00:00}
			- {participant_id: 119, attempt_id: 150, item_id: 3, started_at: 2019-05-30 11:00:00}
			- {participant_id: 119, attempt_id: 200, item_id: 5, started_at: 2019-05-30 11:00:00}
			- {participant_id: 119, attempt_id: 250, item_id: 5, started_at: 2019-05-29 11:00:00}
			- {participant_id: 120, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 120, attempt_id: 201, item_id: 9, started_at: 2019-05-30 11:00:00}
			- {participant_id: 121, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 121, attempt_id: 201, item_id: 9, started_at: 2019-05-30 11:00:00}
	`)
	defer func() { _ = db.Close() }()

	assert.NoError(t, database.NewDataStore(db).InTransaction(func(store *database.DataStore) error {
		store.GroupGroups().CreateNewAncestors()
		return nil
	}))

	type args struct {
		ids                               []int64
		groupID                           int64
		parentAttemptID                   int64
		requireContentAccessToTheLastItem bool
	}
	tests := []struct {
		name                 string
		args                 args
		want                 bool
		wantAttemptIDMap     map[int64]int64
		wantAttemptNumberMap map[int64]int
	}{
		{name: "empty list of ids", args: args{ids: []int64{}, groupID: 100, parentAttemptID: 0}},
		{name: "one item, but parentAttemptID != 0", args: args{ids: []int64{2}, groupID: 100, parentAttemptID: 200}},
		{name: "wrong parentAttemptID", args: args{ids: []int64{2, 4}, groupID: 100, parentAttemptID: 0}},
		{
			name:                 "first item is the group's activity",
			args:                 args{ids: []int64{4, 6}, groupID: 101, parentAttemptID: 200},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{4: 200},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "first item is an activity of the group's ancestor",
			args:                 args{ids: []int64{4, 6}, groupID: 102, parentAttemptID: 200},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{4: 200},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "first item is the group's skill",
			args:                 args{ids: []int64{4, 6}, groupID: 120, parentAttemptID: 200},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{4: 200},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "first item is a skill of the group's ancestor",
			args:                 args{ids: []int64{4, 6}, groupID: 121, parentAttemptID: 200},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{4: 200},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "first item is an activity of a group managed by the given group",
			args:                 args{ids: []int64{9, 6}, groupID: 101, parentAttemptID: 202},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{9: 202},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "first item is an activity of a descendant of a group managed by the given group",
			args:                 args{ids: []int64{9, 6}, groupID: 102, parentAttemptID: 201},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{9: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "first item is a skill of a group managed by the given group",
			args:                 args{ids: []int64{9, 6}, groupID: 120, parentAttemptID: 201},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{9: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "first item is a skill of a descendant of a group managed by the given group",
			args:                 args{ids: []int64{9, 6}, groupID: 121, parentAttemptID: 201},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{9: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name: "first item is not the group's activity/skill nor an activity/skill of a group managed by the group",
			args: args{ids: []int64{6, 8}, groupID: 101, parentAttemptID: 201},
		},
		{
			name:                 "no content access to the last item when requireContentAccessToTheLastItem = true",
			args:                 args{ids: []int64{2, 4}, groupID: 103, parentAttemptID: 200, requireContentAccessToTheLastItem: true},
			wantAttemptIDMap:     map[int64]int64{2: 200},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name: "no access to the last item when requireContentAccessToTheLastItem = false",
			args: args{ids: []int64{2, 4}, groupID: 104, parentAttemptID: 200, requireContentAccessToTheLastItem: false},
		},
		{
			name:                 "content access to the last item when requireContentAccessToTheLastItem = true",
			args:                 args{ids: []int64{4, 6}, groupID: 101, parentAttemptID: 200, requireContentAccessToTheLastItem: true},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{4: 200},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "info access to the last item when requireContentAccessToTheLastItem = false",
			args:                 args{ids: []int64{2, 4}, groupID: 103, parentAttemptID: 200, requireContentAccessToTheLastItem: false},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{2: 200},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name: "no access to the last item when requireContentAccessToTheLastItem = true",
			args: args{ids: []int64{2, 4}, groupID: 104, parentAttemptID: 200, requireContentAccessToTheLastItem: true},
		},
		{name: "no content access to the second to the last item", args: args{ids: []int64{2, 4, 6}, groupID: 105, parentAttemptID: 201}},
		{name: "no content access to the first item", args: args{ids: []int64{2, 4, 6}, groupID: 106, parentAttemptID: 201}},
		{name: "result of the first item is not started", args: args{ids: []int64{2, 4, 6}, groupID: 107, parentAttemptID: 201}},
		{name: "result of the second to the last item is not started", args: args{ids: []int64{2, 4, 6}, groupID: 108, parentAttemptID: 201}},
		{
			name:                 "attempt of the first item is expired",
			args:                 args{ids: []int64{2, 4, 6}, groupID: 109, parentAttemptID: 201},
			wantAttemptIDMap:     map[int64]int64{2: 200, 4: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "attempt of the second to the last item is expired",
			args:                 args{ids: []int64{2, 4, 6}, groupID: 110, parentAttemptID: 201},
			wantAttemptIDMap:     map[int64]int64{2: 200, 4: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "attempt of the first item is ended",
			args:                 args{ids: []int64{2, 4, 6}, groupID: 111, parentAttemptID: 201},
			wantAttemptIDMap:     map[int64]int64{2: 200, 4: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "attempt of the second to the last item is ended",
			args:                 args{ids: []int64{2, 4, 6}, groupID: 112, parentAttemptID: 201},
			wantAttemptIDMap:     map[int64]int64{2: 200, 4: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{name: "the first item is not a parent of the second item", args: args{ids: []int64{4, 4, 6}, groupID: 113, parentAttemptID: 200}},
		{
			name: "the second to the last item is not a parent of the last item",
			args: args{ids: []int64{2, 4, 4}, groupID: 113, parentAttemptID: 200},
		},
		{
			name: "the first item's attempt is not a parent for the second items's attempt while the second item's attempt root_item_id is set",
			args: args{ids: []int64{2, 4, 6, 8}, groupID: 114, parentAttemptID: 201},
		},
		{
			name: "the first item's attempt is not the same as the the second items's attempt " +
				"while the second item's attempt root_item_id is not set",
			args: args{ids: []int64{2, 4, 6, 8}, groupID: 115, parentAttemptID: 200},
		},
		{
			name: "the third from the end item's attempt is not a parent for the second to the last items's attempt " +
				"while the second to the last item's attempt root_item_id is set",
			args: args{ids: []int64{2, 4, 6, 8}, groupID: 116, parentAttemptID: 201},
		},
		{
			name: "the third from the end item's attempt is not the same as the second to the last items's attempt " +
				"while the second to the last item's attempt root_item_id is not set",
			args: args{ids: []int64{2, 4, 6, 8}, groupID: 117, parentAttemptID: 200},
		},
		{
			name:                 "everything is okay (1 item)",
			args:                 args{ids: []int64{4}, groupID: 101, parentAttemptID: 0},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "everything is okay (4 items)",
			args:                 args{ids: []int64{2, 4, 6, 8}, groupID: 118, parentAttemptID: 200},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{2: 0, 4: 0, 6: 200},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "everything is okay (4 items allowing multiple attempts)",
			args:                 args{ids: []int64{1, 3, 5, 7}, groupID: 119, parentAttemptID: 200},
			want:                 true,
			wantAttemptIDMap:     map[int64]int64{1: 0, 3: 0, 5: 200},
			wantAttemptNumberMap: map[int64]int{1: 1, 3: 1, 5: 2},
		},
	}
	for _, tt := range tests {
		tt := tt
		testEachWriteLockMode(t, tt.name+": is valid", func(writeLock bool) func(*testing.T) {
			return func(t *testing.T) {
				assert.NoError(t, database.NewDataStore(db).InTransaction(func(store *database.DataStore) error {
					got, err := store.Items().IsValidParticipationHierarchyForParentAttempt(
						tt.args.ids, tt.args.groupID, tt.args.parentAttemptID, tt.args.requireContentAccessToTheLastItem, writeLock)
					assert.Equal(t, tt.want, got)
					assert.NoError(t, err)
					return nil
				}))
			}
		})
		testEachWriteLockMode(t, tt.name+": breadcrumbs hierarchy", func(writeLock bool) func(*testing.T) {
			return func(t *testing.T) {
				assert.NoError(t, database.NewDataStore(db).InTransaction(func(store *database.DataStore) error {
					gotIDs, gotNumbers, err := store.Items().BreadcrumbsHierarchyForParentAttempt(
						tt.args.ids, tt.args.groupID, tt.args.parentAttemptID, writeLock)
					assertBreadcrumbsHierarchy(t, tt.wantAttemptIDMap, gotIDs, tt.wantAttemptNumberMap, gotNumbers, err)
					return nil
				}))
			}
		})
	}
}

func assertBreadcrumbsHierarchy(t *testing.T,
	wantAttemptIDMap, gotIDs map[int64]int64,
	wantAttemptNumberMap, gotNumbers map[int64]int, err error,
) {
	assert.Equal(t, wantAttemptIDMap, gotIDs)
	assert.Equal(t, wantAttemptNumberMap, gotNumbers)
	assert.NoError(t, err)
}

func TestItemStore_BreadcrumbsHierarchyForAttempt(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items:
			- {id: 1, default_language_tag: fr, allows_multiple_attempts: 1}
			- {id: 2, default_language_tag: fr}
			- {id: 3, default_language_tag: fr, allows_multiple_attempts: 1}
			- {id: 4, default_language_tag: fr}
			- {id: 5, default_language_tag: fr, allows_multiple_attempts: 1}
			- {id: 6, default_language_tag: fr}
			- {id: 7, default_language_tag: fr, allows_multiple_attempts: 1}
			- {id: 8, default_language_tag: fr}
		items_items:
			- {parent_item_id: 1, child_item_id: 3, child_order: 1}
			- {parent_item_id: 3, child_item_id: 5, child_order: 1}
			- {parent_item_id: 5, child_item_id: 7, child_order: 1}

			- {parent_item_id: 2, child_item_id: 4, child_order: 1}
			- {parent_item_id: 4, child_item_id: 6, child_order: 1}
			- {parent_item_id: 6, child_item_id: 8, child_order: 1}
		groups:
			- {id: 50, root_activity_id: 4}
			- {id: 100, root_activity_id: 2}
			- {id: 101, root_activity_id: 4}
			- {id: 102}
			- {id: 103, root_activity_id: 2}
			- {id: 104, root_activity_id: 2}
			- {id: 105, root_activity_id: 2}
			- {id: 106, root_activity_id: 2}
			- {id: 107, root_activity_id: 2}
			- {id: 108, root_activity_id: 2}
			- {id: 109, root_activity_id: 2}
			- {id: 110, root_activity_id: 2}
			- {id: 111, root_activity_id: 2}
			- {id: 112, root_activity_id: 2}
			- {id: 113, root_activity_id: 2}
			- {id: 114, root_activity_id: 2}
			- {id: 115, root_activity_id: 2}
			- {id: 116, root_activity_id: 2}
			- {id: 117, root_activity_id: 2}
			- {id: 118, root_activity_id: 2}
			- {id: 119, root_activity_id: 1}
			- {id: 120, root_skill_id: 4}
			- {id: 121}
		groups_groups:
			- {parent_group_id: 50, child_group_id: 102}
			- {parent_group_id: 120, child_group_id: 121}
		permissions_generated:
			- {group_id: 50, item_id: 4, can_view_generated: content}
			- {group_id: 100, item_id: 2, can_view_generated: content}
			- {group_id: 100, item_id: 4, can_view_generated: content}
			- {group_id: 101, item_id: 4, can_view_generated: content}
			- {group_id: 101, item_id: 6, can_view_generated: content}
			- {group_id: 101, item_id: 8, can_view_generated: content}
			- {group_id: 102, item_id: 6, can_view_generated: content}
			- {group_id: 103, item_id: 2, can_view_generated: content}
			- {group_id: 103, item_id: 4, can_view_generated: info}
			- {group_id: 104, item_id: 2, can_view_generated: content}
			- {group_id: 104, item_id: 4, can_view_generated: none}
			- {group_id: 105, item_id: 2, can_view_generated: content}
			- {group_id: 105, item_id: 4, can_view_generated: info}
			- {group_id: 105, item_id: 6, can_view_generated: content}
			- {group_id: 106, item_id: 2, can_view_generated: info}
			- {group_id: 106, item_id: 4, can_view_generated: content}
			- {group_id: 106, item_id: 6, can_view_generated: content}
			- {group_id: 107, item_id: 2, can_view_generated: content}
			- {group_id: 107, item_id: 4, can_view_generated: content}
			- {group_id: 107, item_id: 6, can_view_generated: content}
			- {group_id: 108, item_id: 2, can_view_generated: content}
			- {group_id: 108, item_id: 4, can_view_generated: content}
			- {group_id: 108, item_id: 6, can_view_generated: content}
			- {group_id: 109, item_id: 2, can_view_generated: content}
			- {group_id: 109, item_id: 4, can_view_generated: content}
			- {group_id: 109, item_id: 6, can_view_generated: content}
			- {group_id: 110, item_id: 2, can_view_generated: content}
			- {group_id: 110, item_id: 4, can_view_generated: content}
			- {group_id: 110, item_id: 6, can_view_generated: content}
			- {group_id: 111, item_id: 2, can_view_generated: content}
			- {group_id: 111, item_id: 4, can_view_generated: content}
			- {group_id: 111, item_id: 6, can_view_generated: content}
			- {group_id: 112, item_id: 2, can_view_generated: content}
			- {group_id: 112, item_id: 4, can_view_generated: content}
			- {group_id: 112, item_id: 6, can_view_generated: content}
			- {group_id: 113, item_id: 2, can_view_generated: content}
			- {group_id: 113, item_id: 4, can_view_generated: content}
			- {group_id: 113, item_id: 6, can_view_generated: content}
			- {group_id: 114, item_id: 2, can_view_generated: content}
			- {group_id: 114, item_id: 4, can_view_generated: content}
			- {group_id: 114, item_id: 6, can_view_generated: content}
			- {group_id: 114, item_id: 8, can_view_generated: content}
			- {group_id: 115, item_id: 2, can_view_generated: content}
			- {group_id: 115, item_id: 4, can_view_generated: content}
			- {group_id: 115, item_id: 6, can_view_generated: content}
			- {group_id: 115, item_id: 8, can_view_generated: content}
			- {group_id: 116, item_id: 2, can_view_generated: content}
			- {group_id: 116, item_id: 4, can_view_generated: content}
			- {group_id: 116, item_id: 6, can_view_generated: content}
			- {group_id: 116, item_id: 8, can_view_generated: content}
			- {group_id: 117, item_id: 2, can_view_generated: content}
			- {group_id: 117, item_id: 4, can_view_generated: content}
			- {group_id: 117, item_id: 6, can_view_generated: content}
			- {group_id: 117, item_id: 8, can_view_generated: content}
			- {group_id: 118, item_id: 2, can_view_generated: content}
			- {group_id: 118, item_id: 4, can_view_generated: content}
			- {group_id: 118, item_id: 6, can_view_generated: content}
			- {group_id: 118, item_id: 8, can_view_generated: content}
			- {group_id: 119, item_id: 1, can_view_generated: content}
			- {group_id: 119, item_id: 3, can_view_generated: content}
			- {group_id: 119, item_id: 5, can_view_generated: content}
			- {group_id: 119, item_id: 7, can_view_generated: content}
			- {group_id: 120, item_id: 4, can_view_generated: content}
			- {group_id: 120, item_id: 6, can_view_generated: content}
			- {group_id: 121, item_id: 4, can_view_generated: content}
			- {group_id: 121, item_id: 6, can_view_generated: content}
		attempts:
			- {participant_id: 100, id: 0}
			- {participant_id: 100, id: 200, root_item_id: 2, parent_attempt_id: 0}
			- {participant_id: 100, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 101, id: 200, root_item_id: 4, allows_submissions_until: 3019-06-30 11:00:00}
			- {participant_id: 101, id: 201, root_item_id: 6, parent_attempt_id: 200, allows_submissions_until: 3019-06-30 11:00:00}
			- {participant_id: 101, id: 202, root_item_id: 8, parent_attempt_id: 201}
			- {participant_id: 102, id: 200, root_item_id: 4}
			- {participant_id: 102, id: 201, root_item_id: 6, parent_attempt_id: 200}
			- {participant_id: 103, id: 200, root_item_id: 2}
			- {participant_id: 103, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 104, id: 200, root_item_id: 2}
			- {participant_id: 104, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 105, id: 200, root_item_id: 2}
			- {participant_id: 105, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 105, id: 202, root_item_id: 6, parent_attempt_id: 201}
			- {participant_id: 106, id: 200, root_item_id: 2}
			- {participant_id: 106, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 106, id: 202, root_item_id: 6, parent_attempt_id: 201}
			- {participant_id: 107, id: 200, root_item_id: 2}
			- {participant_id: 107, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 107, id: 202, root_item_id: 6, parent_attempt_id: 201}
			- {participant_id: 108, id: 200, root_item_id: 2}
			- {participant_id: 108, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 108, id: 202, root_item_id: 6, parent_attempt_id: 201}
			- {participant_id: 109, id: 200, root_item_id: 2, allows_submissions_until: 2019-06-30 11:00:00}
			- {participant_id: 109, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 109, id: 202, root_item_id: 6, parent_attempt_id: 201}
			- {participant_id: 110, id: 200, root_item_id: 2}
			- {participant_id: 110, id: 201, root_item_id: 4, parent_attempt_id: 200, allows_submissions_until: 2019-06-30 11:00:00}
			- {participant_id: 110, id: 202, root_item_id: 6, parent_attempt_id: 201}
			- {participant_id: 111, id: 200, root_item_id: 2, ended_at: 2019-06-30 11:00:00}
			- {participant_id: 111, id: 201, root_item_id: 4, parent_attempt_id: 200}
			- {participant_id: 111, id: 202, root_item_id: 6, parent_attempt_id: 201}
			- {participant_id: 112, id: 200, root_item_id: 2}
			- {participant_id: 112, id: 201, root_item_id: 4, parent_attempt_id: 200, ended_at: 2019-06-30 11:00:00}
			- {participant_id: 112, id: 202, root_item_id: 6, parent_attempt_id: 201}
			- {participant_id: 113, id: 200}
			- {participant_id: 114, id: 0}
			- {participant_id: 114, id: 200, root_item_id: 2}
			- {participant_id: 114, id: 201, root_item_id: 4, parent_attempt_id: 0}
			- {participant_id: 114, id: 202, root_item_id: 8, parent_attempt_id: 201}
			- {participant_id: 115, id: 0}
			- {participant_id: 115, id: 200, parent_attempt_id: 0}
			- {participant_id: 116, id: 0}
			- {participant_id: 116, id: 200}
			- {participant_id: 116, id: 201, root_item_id: 6, parent_attempt_id: 0}
			- {participant_id: 117, id: 0}
			- {participant_id: 117, id: 200, parent_attempt_id: 0}
			- {participant_id: 118, id: 0}
			- {participant_id: 118, id: 200, root_item_id: 6, parent_attempt_id: 0}
			- {participant_id: 118, id: 201, root_item_id: 8, parent_attempt_id: 200}
			- {participant_id: 119, id: 0}
			- {participant_id: 119, id: 50}
			- {participant_id: 119, id: 200, root_item_id: 5, parent_attempt_id: 0}
			- {participant_id: 119, id: 250, root_item_id: 5, parent_attempt_id: 0}
			- {participant_id: 119, id: 201, root_item_id: 7, parent_attempt_id: 200}
			- {participant_id: 119, id: 251, root_item_id: 7, parent_attempt_id: 200}
			- {participant_id: 120, id: 200, root_item_id: 4}
			- {participant_id: 120, id: 201, root_item_id: 6, parent_attempt_id: 200}
			- {participant_id: 121, id: 200, root_item_id: 4}
			- {participant_id: 121, id: 201, root_item_id: 6, parent_attempt_id: 200}
		results:
			- {participant_id: 100, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 100, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 101, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 101, attempt_id: 201, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 101, attempt_id: 202, item_id: 8, started_at: 2019-05-30 11:00:00}
			- {participant_id: 102, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 102, attempt_id: 201, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 103, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 103, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 104, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 104, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 105, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 105, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 105, attempt_id: 202, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 106, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 106, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 106, attempt_id: 202, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 107, attempt_id: 200, item_id: 2, started_at: null}
			- {participant_id: 107, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 107, attempt_id: 202, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 108, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 108, attempt_id: 201, item_id: 4, started_at: null}
			- {participant_id: 108, attempt_id: 202, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 109, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 109, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 109, attempt_id: 202, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 110, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 110, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 110, attempt_id: 202, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 111, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 111, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 111, attempt_id: 202, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 112, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 112, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 112, attempt_id: 202, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 113, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 113, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 113, attempt_id: 200, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 114, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 114, attempt_id: 201, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 114, attempt_id: 201, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 114, attempt_id: 202, item_id: 8, started_at: 2019-05-30 11:00:00}
			- {participant_id: 115, attempt_id: 0, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 115, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 115, attempt_id: 200, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 115, attempt_id: 200, item_id: 8, started_at: 2019-05-30 11:00:00}
			- {participant_id: 116, attempt_id: 200, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 116, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 116, attempt_id: 201, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 116, attempt_id: 201, item_id: 8, started_at: 2019-05-30 11:00:00}
			- {participant_id: 117, attempt_id: 0, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 117, attempt_id: 0, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 117, attempt_id: 200, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 117, attempt_id: 200, item_id: 8, started_at: 2019-05-30 11:00:00}
			- {participant_id: 118, attempt_id: 0, item_id: 2, started_at: 2019-05-30 11:00:00}
			- {participant_id: 118, attempt_id: 0, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 118, attempt_id: 200, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 118, attempt_id: 201, item_id: 8, started_at: 2019-05-30 11:00:00}
			- {participant_id: 119, attempt_id: 0, item_id: 1, started_at: 2019-05-30 11:00:00}
			- {participant_id: 119, attempt_id: 50, item_id: 1, started_at: 2019-05-29 11:00:00}
			- {participant_id: 119, attempt_id: 0, item_id: 3, started_at: 2019-05-30 11:00:00}
			- {participant_id: 119, attempt_id: 100, item_id: 3, started_at: 2019-05-29 11:00:00}
			- {participant_id: 119, attempt_id: 200, item_id: 5, started_at: 2019-05-29 11:00:00}
			- {participant_id: 119, attempt_id: 250, item_id: 5, started_at: 2019-05-30 11:00:00}
			- {participant_id: 119, attempt_id: 201, item_id: 7, started_at: 2019-05-30 11:00:00}
			- {participant_id: 119, attempt_id: 251, item_id: 7, started_at: 2019-05-29 11:00:00}
			- {participant_id: 120, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 120, attempt_id: 201, item_id: 6, started_at: 2019-05-30 11:00:00}
			- {participant_id: 121, attempt_id: 200, item_id: 4, started_at: 2019-05-30 11:00:00}
			- {participant_id: 121, attempt_id: 201, item_id: 6, started_at: 2019-05-30 11:00:00}
	`)
	defer func() { _ = db.Close() }()

	assert.NoError(t, database.NewDataStore(db).InTransaction(func(store *database.DataStore) error {
		store.GroupGroups().CreateNewAncestors()
		return nil
	}))

	type args struct {
		ids       []int64
		groupID   int64
		attemptID int64
	}
	tests := []struct {
		name                 string
		args                 args
		wantAttemptIDMap     map[int64]int64
		wantAttemptNumberMap map[int64]int
	}{
		{name: "empty list of ids", args: args{ids: []int64{}, groupID: 100, attemptID: 0}},
		{
			name:                 "one item",
			args:                 args{ids: []int64{2}, groupID: 100, attemptID: 200},
			wantAttemptIDMap:     map[int64]int64{2: 200},
			wantAttemptNumberMap: map[int64]int{},
		},
		{name: "wrong attemptID", args: args{ids: []int64{2, 4}, groupID: 100, attemptID: 0}},
		{
			name:                 "first item is the group's activity",
			args:                 args{ids: []int64{4, 6}, groupID: 101, attemptID: 201},
			wantAttemptIDMap:     map[int64]int64{4: 200, 6: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "first item is an activity of the group's ancestor",
			args:                 args{ids: []int64{4, 6}, groupID: 102, attemptID: 201},
			wantAttemptIDMap:     map[int64]int64{4: 200, 6: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "first item is the group's skill",
			args:                 args{ids: []int64{4, 6}, groupID: 120, attemptID: 201},
			wantAttemptIDMap:     map[int64]int64{4: 200, 6: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "first item is a skill of the group's ancestor",
			args:                 args{ids: []int64{4, 6}, groupID: 121, attemptID: 201},
			wantAttemptIDMap:     map[int64]int64{4: 200, 6: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name: "first item is not the group's activity/skill",
			args: args{ids: []int64{6, 8}, groupID: 101, attemptID: 202},
		},
		{
			name: "no access to the last item",
			args: args{ids: []int64{2, 4}, groupID: 104, attemptID: 201},
		},
		{
			name:                 "content access to the last item",
			args:                 args{ids: []int64{4, 6}, groupID: 101, attemptID: 201},
			wantAttemptIDMap:     map[int64]int64{4: 200, 6: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "info access to the last item",
			args:                 args{ids: []int64{2, 4}, groupID: 103, attemptID: 201},
			wantAttemptIDMap:     map[int64]int64{2: 200, 4: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{name: "no content access to the second to the last item", args: args{ids: []int64{2, 4, 6}, groupID: 105, attemptID: 202}},
		{name: "no content access to the first item", args: args{ids: []int64{2, 4, 6}, groupID: 106, attemptID: 202}},
		{name: "result of the first item is not started", args: args{ids: []int64{2, 4, 6}, groupID: 107, attemptID: 202}},
		{name: "result of the second to the last item is not started", args: args{ids: []int64{2, 4, 6}, groupID: 108, attemptID: 202}},
		{name: "result of the last item is not started", args: args{ids: []int64{2, 4}, groupID: 108, attemptID: 201}},
		{
			name:                 "attempt of the first item is expired",
			args:                 args{ids: []int64{2, 4, 6}, groupID: 109, attemptID: 202},
			wantAttemptIDMap:     map[int64]int64{2: 200, 4: 201, 6: 202},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "attempt of the second to the last item is expired",
			args:                 args{ids: []int64{2, 4, 6}, groupID: 110, attemptID: 202},
			wantAttemptIDMap:     map[int64]int64{2: 200, 4: 201, 6: 202},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "attempt of the last item is expired",
			args:                 args{ids: []int64{2, 4}, groupID: 110, attemptID: 201},
			wantAttemptIDMap:     map[int64]int64{2: 200, 4: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "attempt of the first item is ended",
			args:                 args{ids: []int64{2, 4, 6}, groupID: 111, attemptID: 202},
			wantAttemptIDMap:     map[int64]int64{2: 200, 4: 201, 6: 202},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "attempt of the second to the last item is ended",
			args:                 args{ids: []int64{2, 4, 6}, groupID: 112, attemptID: 202},
			wantAttemptIDMap:     map[int64]int64{2: 200, 4: 201, 6: 202},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "attempt of the last item is ended",
			args:                 args{ids: []int64{2, 4}, groupID: 112, attemptID: 201},
			wantAttemptIDMap:     map[int64]int64{2: 200, 4: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{name: "the first item is not a parent of the second item", args: args{ids: []int64{4, 4, 6}, groupID: 113, attemptID: 200}},
		{name: "the second to the last item is not a parent of the last item", args: args{ids: []int64{2, 4, 4}, groupID: 113, attemptID: 200}},
		{
			name: "the first item's attempt is not a parent for the second items's attempt while the second item's attempt root_item_id is set",
			args: args{ids: []int64{2, 4, 6, 8}, groupID: 114, attemptID: 202},
		},
		{
			name: "the first item's attempt is not the same as the the second items's attempt " +
				"while the second item's attempt root_item_id is not set",
			args: args{ids: []int64{2, 4, 6, 8}, groupID: 115, attemptID: 200},
		},
		{
			name: "the third to the end item's attempt is not a parent for the second to the last items's attempt " +
				"while the second to the last item's attempt root_item_id is set",
			args: args{ids: []int64{2, 4, 6, 8}, groupID: 116, attemptID: 201},
		},
		{
			name: "the second to the last item's attempt is not a parent for the last items's attempt " +
				"while the last item's attempt root_item_id is set",
			args: args{ids: []int64{2, 4, 6}, groupID: 116, attemptID: 201},
		},
		{
			name: "the third from the end item's attempt is not the same as the second to the last items's attempt " +
				"while the second to the last item's attempt root_item_id is not set",
			args: args{ids: []int64{2, 4, 6, 8}, groupID: 117, attemptID: 200},
		},
		{
			name: "the second the last item's attempt is not the same as the last items's attempt " +
				"while the last item's attempt root_item_id is not set",
			args: args{ids: []int64{2, 4, 6}, groupID: 117, attemptID: 200},
		},
		{
			name:                 "everything is okay (1 item)",
			args:                 args{ids: []int64{4}, groupID: 101, attemptID: 200},
			wantAttemptIDMap:     map[int64]int64{4: 200},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "everything is okay (4 items)",
			args:                 args{ids: []int64{2, 4, 6, 8}, groupID: 118, attemptID: 201},
			wantAttemptIDMap:     map[int64]int64{2: 0, 4: 0, 6: 200, 8: 201},
			wantAttemptNumberMap: map[int64]int{},
		},
		{
			name:                 "everything is okay (4 items allowing multiple attempts)",
			args:                 args{ids: []int64{1, 3, 5, 7}, groupID: 119, attemptID: 201},
			wantAttemptIDMap:     map[int64]int64{1: 0, 3: 0, 5: 200, 7: 201},
			wantAttemptNumberMap: map[int64]int{1: 2, 3: 1, 5: 1, 7: 2},
		},
	}
	for _, tt := range tests {
		tt := tt
		testEachWriteLockMode(t, tt.name, func(writeLock bool) func(*testing.T) {
			return func(t *testing.T) {
				assert.NoError(t, database.NewDataStore(db).InTransaction(func(store *database.DataStore) error {
					gotIDs, gotNumbers, err := store.Items().BreadcrumbsHierarchyForAttempt(
						tt.args.ids, tt.args.groupID, tt.args.attemptID, writeLock)
					assertBreadcrumbsHierarchy(t, tt.wantAttemptIDMap, gotIDs, tt.wantAttemptNumberMap, gotNumbers, err)
					return nil
				}))
			}
		})
	}
}

func testEachWriteLockMode(t *testing.T, testName string, testGenFunc func(writeLock bool) func(*testing.T)) {
	for _, writeLock := range []bool{false, true} {
		writeLock := writeLock
		var lockName string
		if writeLock {
			lockName = "(with write lock)"
		} else {
			lockName = "(without write lock)"
		}
		t.Run(testName+" "+lockName, testGenFunc(writeLock))
	}
}

func TestItemStore_TriggerBeforeInsert_SetsPlatformID(t *testing.T) {
	tests := []struct {
		name           string
		url            *string
		wantPlatformID *int64
	}{
		{name: "url is null", url: nil, wantPlatformID: nil},
		{name: "chooses a platform with higher priority", url: ptrString("1234"), wantPlatformID: ptrInt64(2)},
		{name: "url doesn't match any regexp", url: ptrString("34"), wantPlatformID: nil},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			db := testhelpers.SetupDBWithFixtureString(`
				platforms:
					- {id: 3, regexp: "^1.*", priority: 1}
					- {id: 4, regexp: "^2.*", priority: 2}
					- {id: 2, regexp: "^1.*", priority: 3}
					- {id: 1, regexp: "^4.*", priority: 4}
				languages: [{tag: fr}]`)
			defer func() { _ = db.Close() }()

			itemStore := database.NewDataStore(db).Items()
			assert.NoError(t, itemStore.WithForeignKeyChecksDisabled(func(store *database.DataStore) error {
				return store.Items().InsertMap(map[string]interface{}{
					"url":                  test.url,
					"default_language_tag": "fr",
				})
			}))
			var platformID *int64
			t.Skip()
			assert.NoError(t, itemStore.PluckFirst("platform_id", &platformID).Error())
			if test.wantPlatformID == nil {
				assert.Nil(t, platformID)
			} else {
				assert.NotNil(t, platformID)
				if platformID != nil {
					assert.Equal(t, *test.wantPlatformID, *platformID)
				}
			}
		})
	}
}

func TestItemStore_TriggerBeforeUpdate_SetsPlatformID(t *testing.T) {
	tests := []struct {
		name           string
		updateMap      map[string]interface{}
		wantPlatformID *int64
	}{
		{name: "url is unchanged", updateMap: map[string]interface{}{"type": "Chapter"}, wantPlatformID: ptrInt64(1)},
		{name: "new url is null", updateMap: map[string]interface{}{"url": nil}, wantPlatformID: nil},
		{
			name: "chooses a platform with higher priority", updateMap: map[string]interface{}{"url": ptrString("12345")},
			wantPlatformID: ptrInt64(2),
		},
		{name: "new url doesn't match any regexp", updateMap: map[string]interface{}{"url": ptrString("34")}, wantPlatformID: nil},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			db := testhelpers.SetupDBWithFixtureString(`
				platforms:
					- {id: 1, regexp: "^4.*", priority: 4}
					- {id: 2, regexp: "^1.*", priority: 3}
					- {id: 4, regexp: "^2.*", priority: 2}
					- {id: 3, regexp: "^1.*", priority: 1}
				languages: [{tag: fr}]
				items:
					- {id: 1, platform_id: 1, url: 444, default_language_tag: fr}`)
			defer func() { _ = db.Close() }()

			itemStore := database.NewDataStore(db).Items()
			assert.NoError(t, itemStore.UpdateColumns(test.updateMap).Error())
			var platformID *int64
			t.Skip()
			assert.NoError(t, itemStore.ByID(1).PluckFirst("platform_id", &platformID).Error())
			if test.wantPlatformID == nil {
				assert.Nil(t, platformID)
			} else {
				assert.NotNil(t, platformID)
				if platformID != nil {
					assert.Equal(t, *test.wantPlatformID, *platformID)
				}
			}
		})
	}
}

func TestItemStore_PlatformsTriggerAfterInsert_SetsPlatformID(t *testing.T) {
	tests := []struct {
		name            string
		regexp          string
		priority        int
		wantPlatformIDs []*int64
	}{
		{
			name:   "recalculates items linked to platforms with lower priority or no platform",
			regexp: "1", priority: 3,
			wantPlatformIDs: []*int64{ptrInt64(2), ptrInt64(1), ptrInt64(2), ptrInt64(2)},
		},
		{
			name:   "recalculates items linked to platforms with lower priority or no platform (higher priority)",
			regexp: "1", priority: 6,
			wantPlatformIDs: []*int64{ptrInt64(5), ptrInt64(4), ptrInt64(5), nil},
		},
		{
			name:   "recalculates only item without a platform when the new platform has the lowest priority",
			regexp: "1", priority: -1,
			wantPlatformIDs: []*int64{ptrInt64(4), ptrInt64(1), ptrInt64(2), ptrInt64(2)},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			db := testhelpers.SetupDBWithFixtureString(`
				platforms:
					- {id: 3, regexp: "^1.*", priority: 1}
					- {id: 4, regexp: "^2.*", priority: 2}
					- {id: 2, regexp: "^1.*", priority: 4}
					- {id: 1, regexp: "^4.*", priority: 5}
				languages: [{tag: fr}]
				items:
					- {id: 1, platform_id: 4, url: 1234, default_language_tag: fr}
					- {id: 2, platform_id: 1, url: 234, default_language_tag: fr}
					- {id: 3, platform_id: null, url: 123, default_language_tag: fr}
					- {id: 4, platform_id: 2, url: "987", default_language_tag: fr}`)
			defer func() { _ = db.Close() }()

			itemStore := database.NewDataStore(db).Items()
			assert.NoError(t, itemStore.ByID(1).UpdateColumn("platform_id", 4).Error())
			assert.NoError(t, itemStore.ByID(2).UpdateColumn("platform_id", 1).Error())
			assert.NoError(t, itemStore.ByID(3).UpdateColumn("platform_id", nil).Error())
			assert.NoError(t, itemStore.ByID(4).UpdateColumn("platform_id", 2).Error())
			assert.NoError(t, itemStore.
				Exec("INSERT platforms (id, `regexp`, priority) VALUES (5, ?, ?)", test.regexp, test.priority).Error())
			t.Skip()
			var platformIDs []*int64
			assert.NoError(t, itemStore.Order("id").Pluck("platform_id", &platformIDs).Error())
			assert.Equal(t, test.wantPlatformIDs, platformIDs)
		})
	}
}

func TestItemStore_PlatformsTriggerAfterUpdate_SetsPlatformID(t *testing.T) {
	tests := []struct {
		name            string
		regexp          string
		priority        int
		wantPlatformIDs []*int64
	}{
		{
			name:   "recalculates items linked to platforms with lower priority or no platform or the modified platform (only priority is changed)",
			regexp: "^1.*", priority: 3,
			wantPlatformIDs: []*int64{ptrInt64(2), ptrInt64(1), ptrInt64(2), nil},
		},
		{
			name:   "recalculates items linked to platforms with lower priority or no platform or the modified platform (only regexp is changed)",
			regexp: "1", priority: 4,
			wantPlatformIDs: []*int64{ptrInt64(2), ptrInt64(1), ptrInt64(2), nil},
		},
		{
			name:   "recalculates items linked to platforms with lower priority or no platform or the modified platform (higher priority)",
			regexp: "1", priority: 6,
			wantPlatformIDs: []*int64{ptrInt64(2), ptrInt64(4), ptrInt64(2), nil},
		},
		{
			name:   "recalculates only item without a platform when the new platform has the lowest priority",
			regexp: "1", priority: -1,
			wantPlatformIDs: []*int64{ptrInt64(4), ptrInt64(1), ptrInt64(3), nil},
		},
		{
			name:   "doesn't recalculate anything when regexp & priority stays unchanged",
			regexp: "^1.*", priority: 4,
			wantPlatformIDs: []*int64{ptrInt64(4), ptrInt64(1), nil, ptrInt64(2)},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			db := testhelpers.SetupDBWithFixtureString(`
				platforms:
					- {id: 3, regexp: "^1.*", priority: 1}
					- {id: 4, regexp: "^2.*", priority: 2}
					- {id: 2, regexp: "^1.*", priority: 4}
					- {id: 1, regexp: "^4.*", priority: 5}
				languages: [{tag: fr}]
				items:
					- {id: 1, platform_id: 4, url: 1234, default_language_tag: fr}
					- {id: 2, platform_id: 1, url: 234, default_language_tag: fr}
					- {id: 3, platform_id: null, url: 123, default_language_tag: fr}
					- {id: 4, platform_id: 2, url: "987", default_language_tag: fr}`)
			defer func() { _ = db.Close() }()

			itemStore := database.NewDataStore(db).Items()
			assert.NoError(t, itemStore.ByID(1).UpdateColumn("platform_id", 4).Error())
			assert.NoError(t, itemStore.ByID(2).UpdateColumn("platform_id", 1).Error())
			assert.NoError(t, itemStore.ByID(3).UpdateColumn("platform_id", nil).Error())
			assert.NoError(t, itemStore.ByID(4).UpdateColumn("platform_id", 2).Error())
			assert.NoError(t, itemStore.Table("platforms").Where("id = ?", 2).
				UpdateColumns(map[string]interface{}{"regexp": test.regexp, "priority": test.priority}).Error())
			var platformIDs []*int64
			t.Skip()
			assert.NoError(t, itemStore.Order("id").Pluck("platform_id", &platformIDs).Error())
			assert.Equal(t, test.wantPlatformIDs, platformIDs)
		})
	}
}

func Test_ItemStore_DeleteItem(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		languages: [{tag: fr}]
		items: [{id: 1234, default_language_tag: fr},{id: 1235, default_language_tag: fr}]
		items_items: [{parent_item_id: 1234, child_item_id: 1235, child_order: 1}]
		items_strings: [{item_id: 1234, language_tag: fr}, {item_id: 1235, language_tag: fr}]
	`)
	defer func() { _ = db.Close() }()
	store := database.NewDataStore(db)
	assert.NoError(t, store.InTransaction(func(store *database.DataStore) error {
		return store.Items().DeleteItem(1235)
	}))
	var ids []int64
	assert.NoError(t, store.Items().Pluck("id", &ids).Error())
	assert.Equal(t, []int64{1234}, ids)
	assert.NoError(t, store.ItemStrings().Pluck("item_id", &ids).Error())
	assert.Equal(t, []int64{1234}, ids)
	assert.NoError(t, store.Table("items_propagate").
		Where("ancestors_computation_state != 'done'").Pluck("id", &ids).Error())
	assert.Empty(t, ids)
	t.Skip()
	assert.NoError(t, store.Table("permissions_propagate").Pluck("item_id", &ids).Error())
	assert.Empty(t, ids)
}
