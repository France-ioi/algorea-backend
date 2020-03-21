// +build !unit

package database_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/testhelpers"
)

type existingResultsRow struct {
	ParticipantID          int64
	AttemptID              int64
	ItemID                 int64
	LatestActivityAt       string
	ResultPropagationState string
}

func testAttemptStoreComputeAllAttemptsCreatesNew(t *testing.T, fixtures []string,
	expectedNewResults []existingResultsRow) {
	mergedFixtures := make([]string, 0, len(fixtures)+1)
	mergedFixtures = append(mergedFixtures, `
		groups: [{id: 1}, {id: 2}, {id: 3}, {id: 4}]
		groups_ancestors:
			- {ancestor_group_id: 1, child_group_id: 1}
			- {ancestor_group_id: 2, child_group_id: 2}
			- {ancestor_group_id: 3, child_group_id: 3}
			- {ancestor_group_id: 1, child_group_id: 2}
			- {ancestor_group_id: 1, child_group_id: 3}
			- {ancestor_group_id: 2, child_group_id: 3}
			- {ancestor_group_id: 4, child_group_id: 4}
			- {ancestor_group_id: 4, child_group_id: 3, expires_at: 2019-05-30 11:00:00}
		items:
			- {id: 111, default_language_tag: fr}
			- {id: 222, default_language_tag: fr}
			- {id: 333, default_language_tag: fr}
			- {id: 444, default_language_tag: fr, requires_explicit_entry: 1}
			- {id: 555, default_language_tag: fr}
		items_items:
			- {parent_item_id: 111, child_item_id: 222, child_order: 1}
			- {parent_item_id: 222, child_item_id: 333, child_order: 1}
			- {parent_item_id: 444, child_item_id: 333, child_order: 1}
			- {parent_item_id: 555, child_item_id: 444, child_order: 1}
		items_ancestors:
			- {ancestor_item_id: 111, child_item_id: 222}
			- {ancestor_item_id: 111, child_item_id: 333}
			- {ancestor_item_id: 222, child_item_id: 333}
			- {ancestor_item_id: 444, child_item_id: 333}
			- {ancestor_item_id: 555, child_item_id: 333}
			- {ancestor_item_id: 555, child_item_id: 444}
		attempts:
			- {participant_id: 3, id: 1}
		results:
			- {participant_id: 3, attempt_id: 1, item_id: 333, latest_activity_at: "2019-05-30 11:00:00", result_propagation_state: to_be_propagated}
	`)
	mergedFixtures = append(mergedFixtures, fixtures...)
	db := testhelpers.SetupDBWithFixtureString(mergedFixtures...)
	defer func() { _ = db.Close() }()

	resultStore := database.NewDataStore(db).Results()
	err := resultStore.InTransaction(func(s *database.DataStore) error {
		return s.Attempts().ComputeAllAttempts()
	})
	assert.NoError(t, err)

	const expectedDate = "2019-05-30 11:00:00"
	for i := range expectedNewResults {
		expectedNewResults[i].ResultPropagationState = "done"
		expectedNewResults[i].LatestActivityAt = expectedDate
	}
	expectedNewResults = append(expectedNewResults,
		existingResultsRow{ParticipantID: 3, AttemptID: 1, ItemID: 333, LatestActivityAt: expectedDate, ResultPropagationState: "done"})
	var result []existingResultsRow
	assert.NoError(t, resultStore.Select("participant_id, attempt_id, item_id, latest_activity_at, result_propagation_state").
		Order("participant_id, attempt_id, item_id").Scan(&result).Error())
	assert.Equal(t, expectedNewResults, result)
}

func TestAttemptStore_ComputeAllAttempts_CreatesNew(t *testing.T) {
	for _, test := range []struct {
		name               string
		fixtures           []string
		expectedNewResults []existingResultsRow
	}{
		{name: "should not create new results if no permissions for parent items"},
		{
			name:     "should not create new results if can_view_generated = none for ancestor items",
			fixtures: []string{`permissions_generated: [{group_id: 3, item_id: 111, can_view_generated: none}]`},
		},
		{
			name:     "should not create new results if can_view_generated > none only for the item (not for its ancestor)",
			fixtures: []string{`permissions_generated: [{group_id: 3, item_id: 333, can_view_generated: info}]`},
		},
		{
			name:     "should not create new results if can_view_generated > none for an ancestor items and the group's expired ancestor",
			fixtures: []string{`permissions_generated: [{group_id: 4, item_id: 111, can_view_generated: info}]`},
		},
		{
			name:     "creates new results if can_view_generated > none for an ancestor items and the group itself",
			fixtures: []string{`permissions_generated: [{group_id: 3, item_id: 111, can_view_generated: info}]`},
			expectedNewResults: []existingResultsRow{
				{ParticipantID: 3, AttemptID: 1, ItemID: 111}, {ParticipantID: 3, AttemptID: 1, ItemID: 222},
			},
		},
		{
			name:     "creates new results if can_view_generated > none for an ancestor items and the group's ancestor",
			fixtures: []string{`permissions_generated: [{group_id: 1, item_id: 111, can_view_generated: info}]`},
			expectedNewResults: []existingResultsRow{
				{ParticipantID: 3, AttemptID: 1, ItemID: 111}, {ParticipantID: 3, AttemptID: 1, ItemID: 222},
			},
		},
		{
			name: "creates new results if can_view_generated > none for an ancestor items and the group itself, " +
				"but only for visible items's descendants",
			fixtures:           []string{`permissions_generated: [{group_id: 3, item_id: 222, can_view_generated: info}]`},
			expectedNewResults: []existingResultsRow{{ParticipantID: 3, AttemptID: 1, ItemID: 222}},
		},
		{
			name: "creates new results if can_view_generated > none for an ancestor items and the group's ancestor, " +
				"but only for visible items's descendants",
			fixtures:           []string{`permissions_generated: [{group_id: 1, item_id: 222, can_view_generated: info}]`},
			expectedNewResults: []existingResultsRow{{ParticipantID: 3, AttemptID: 1, ItemID: 222}},
		},
		{
			name:     "should not create new results for items requiring explicit entry",
			fixtures: []string{`permissions_generated: [{group_id: 1, item_id: 555, can_view_generated: info}]`},
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			testAttemptStoreComputeAllAttemptsCreatesNew(t, test.fixtures, test.expectedNewResults)
		})
	}
}
