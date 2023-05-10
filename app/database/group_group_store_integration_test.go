//go:build !unit

package database_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/testhelpers"
)

func TestGroupGroupStore_DeleteRelation(t *testing.T) {
	tests := []struct {
		name                         string
		fixture                      string
		shouldDeleteOrphans          bool
		remainingGroupIDs            []int64
		remainingGroupsGroups        []map[string]interface{}
		remainingGroupsAncestors     []map[string]interface{}
		expectedGrantedPermissions   []grantedPermission
		expectedGeneratedPermissions []permissionsGeneratedResultRow
	}{
		{
			name:                "can produce orphans",
			fixture:             "orphan",
			shouldDeleteOrphans: false,
			remainingGroupIDs:   []int64{1, 2, 3, 4, 5},
			remainingGroupsGroups: []map[string]interface{}{
				{"parent_group_id": "2", "child_group_id": "5"},
			},
			remainingGroupsAncestors: []map[string]interface{}{
				{"ancestor_group_id": "1", "child_group_id": "1"},
				{"ancestor_group_id": "2", "child_group_id": "2"},
				{"ancestor_group_id": "2", "child_group_id": "5"},
				{"ancestor_group_id": "3", "child_group_id": "3"},
				{"ancestor_group_id": "4", "child_group_id": "4"},
				{"ancestor_group_id": "5", "child_group_id": "5"},
			},
			expectedGrantedPermissions: []grantedPermission{
				{GroupID: 1, ItemID: 1, SourceGroupID: 1, Origin: "group_membership", CanView: "none"},
				{GroupID: 2, ItemID: 1, SourceGroupID: 2, Origin: "group_membership", CanView: "none"},
				{GroupID: 3, ItemID: 1, SourceGroupID: 3, Origin: "group_membership", CanView: "none"},
				{GroupID: 4, ItemID: 1, SourceGroupID: 4, Origin: "group_membership", CanView: "none"},
				{GroupID: 5, ItemID: 1, SourceGroupID: 5, Origin: "group_membership", CanView: "none"},
			},
			expectedGeneratedPermissions: []permissionsGeneratedResultRow{
				{GroupID: 1, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 2, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 3, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 4, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 5, ItemID: 1, CanViewGenerated: "none"},
			},
		},
		{
			name:                "no orphans",
			fixture:             "no_orphans",
			shouldDeleteOrphans: false,
			remainingGroupIDs:   []int64{1, 2, 3},
			remainingGroupsGroups: []map[string]interface{}{
				{"parent_group_id": "3", "child_group_id": "2"},
			},
			remainingGroupsAncestors: []map[string]interface{}{
				{"ancestor_group_id": "1", "child_group_id": "1"},
				{"ancestor_group_id": "2", "child_group_id": "2"},
				{"ancestor_group_id": "3", "child_group_id": "2"},
				{"ancestor_group_id": "3", "child_group_id": "3"},
			},
			expectedGrantedPermissions: []grantedPermission{
				{GroupID: 1, ItemID: 1, SourceGroupID: 1, Origin: "group_membership", CanView: "none"},
				{GroupID: 2, ItemID: 1, SourceGroupID: 1, Origin: "item_unlocking", CanView: "content"},
				{GroupID: 2, ItemID: 1, SourceGroupID: 2, Origin: "group_membership", CanView: "info"},
				{GroupID: 3, ItemID: 1, SourceGroupID: 1, Origin: "group_membership", CanView: "info"},
				{GroupID: 3, ItemID: 1, SourceGroupID: 3, Origin: "group_membership", CanView: "none"},
			},
			expectedGeneratedPermissions: []permissionsGeneratedResultRow{
				{GroupID: 1, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 2, ItemID: 1, CanViewGenerated: "content"},
				{GroupID: 3, ItemID: 1, CanViewGenerated: "info"},
			},
		},
		{
			name:                  "deletes orphans",
			fixture:               "orphan",
			shouldDeleteOrphans:   true,
			remainingGroupIDs:     []int64{1, 3, 4},
			remainingGroupsGroups: nil,
			remainingGroupsAncestors: []map[string]interface{}{
				{"ancestor_group_id": "1", "child_group_id": "1"},
				{"ancestor_group_id": "3", "child_group_id": "3"},
				{"ancestor_group_id": "4", "child_group_id": "4"},
			},
			expectedGrantedPermissions: []grantedPermission{
				{GroupID: 1, ItemID: 1, SourceGroupID: 1, Origin: "group_membership", CanView: "none"},
				{GroupID: 3, ItemID: 1, SourceGroupID: 3, Origin: "group_membership", CanView: "none"},
				{GroupID: 4, ItemID: 1, SourceGroupID: 4, Origin: "group_membership", CanView: "none"},
			},
			expectedGeneratedPermissions: []permissionsGeneratedResultRow{
				{GroupID: 1, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 3, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 4, ItemID: 1, CanViewGenerated: "none"},
			},
		},
		{
			name:                "deletes only orphans",
			fixture:             "deletes_only_orphans",
			shouldDeleteOrphans: true,
			remainingGroupIDs:   []int64{1, 3, 5, 10},
			remainingGroupsGroups: []map[string]interface{}{
				{"parent_group_id": "10", "child_group_id": "3"},
				{"parent_group_id": "10", "child_group_id": "5"},
			},
			remainingGroupsAncestors: []map[string]interface{}{
				{"ancestor_group_id": "1", "child_group_id": "1"},
				{"ancestor_group_id": "3", "child_group_id": "3"},
				{"ancestor_group_id": "5", "child_group_id": "5"},
				{"ancestor_group_id": "10", "child_group_id": "3"},
				{"ancestor_group_id": "10", "child_group_id": "5"},
				{"ancestor_group_id": "10", "child_group_id": "10"},
			},
			expectedGrantedPermissions: []grantedPermission{
				{GroupID: 1, ItemID: 1, SourceGroupID: 1, Origin: "group_membership", CanView: "none"},
				{GroupID: 3, ItemID: 1, SourceGroupID: 3, Origin: "group_membership", CanView: "none"},
				{GroupID: 5, ItemID: 1, SourceGroupID: 5, Origin: "group_membership", CanView: "none"},
				{GroupID: 10, ItemID: 1, SourceGroupID: 10, Origin: "group_membership", CanView: "none"},
			},
			expectedGeneratedPermissions: []permissionsGeneratedResultRow{
				{GroupID: 1, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 3, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 5, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 10, ItemID: 1, CanViewGenerated: "none"},
			},
		},
		{
			name:                  "deletes only orphans of accepted types",
			fixture:               "deletes_only_orphans_of_accepted_types",
			shouldDeleteOrphans:   true,
			remainingGroupIDs:     []int64{1, 8, 9, 11},
			remainingGroupsGroups: nil,
			remainingGroupsAncestors: []map[string]interface{}{
				{"ancestor_group_id": "1", "child_group_id": "1"},
				{"ancestor_group_id": "8", "child_group_id": "8"},
				{"ancestor_group_id": "9", "child_group_id": "9"},
				{"ancestor_group_id": "11", "child_group_id": "11"},
			},
			expectedGrantedPermissions: []grantedPermission{
				{GroupID: 1, ItemID: 1, SourceGroupID: 1, Origin: "group_membership", CanView: "none"},
				{GroupID: 8, ItemID: 1, SourceGroupID: 8, Origin: "group_membership", CanView: "none"},
				{GroupID: 9, ItemID: 1, SourceGroupID: 9, Origin: "group_membership", CanView: "none"},
				{GroupID: 11, ItemID: 1, SourceGroupID: 11, Origin: "group_membership", CanView: "none"},
			},
			expectedGeneratedPermissions: []permissionsGeneratedResultRow{
				{GroupID: 1, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 8, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 9, ItemID: 1, CanViewGenerated: "none"},
				{GroupID: 11, ItemID: 1, CanViewGenerated: "none"},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			db := testhelpers.SetupDBWithFixture("group_group_store/delete_relation/" + tt.fixture)
			defer func() { _ = db.Close() }()
			dataStore := database.NewDataStore(db)
			assert.NoError(t, dataStore.Table("groups_propagate").UpdateColumn("ancestors_computation_state", "done").Error())

			assert.NoError(t, dataStore.InTransaction(func(s *database.DataStore) error {
				s.PermissionsGranted().ComputeAllAccess()
				return nil
			}))
			err := dataStore.InTransaction(func(s *database.DataStore) error {
				return s.GroupGroups().DeleteRelation(1, 2, tt.shouldDeleteOrphans)
			})
			assert.Nil(t, err)
			assertGroupRelations(t, dataStore, tt.remainingGroupIDs, tt.remainingGroupsGroups, tt.remainingGroupsAncestors)
			assertGroupLinkedObjects(t, dataStore, tt.remainingGroupIDs, tt.expectedGrantedPermissions, tt.expectedGeneratedPermissions)
		})
	}
}

func assertGroupRelations(t *testing.T, dataStore *database.DataStore,
	remainingGroupIDs []int64, remainingGroupsGroups, remainingGroupsAncestors []map[string]interface{},
) {
	var rows []map[string]interface{}
	var ids []int64

	totalRemainingGroupIDs := make([]int64, 0, len(remainingGroupIDs)+1)
	totalRemainingGroupIDs = append(totalRemainingGroupIDs, remainingGroupIDs...)
	totalRemainingGroupIDs = append(totalRemainingGroupIDs, int64(111))
	assert.NoError(t, dataStore.Groups().Order("id").Pluck("id", &ids).Error())
	assert.Equal(t, totalRemainingGroupIDs, ids)
	assert.NoError(t, dataStore.ActiveGroupGroups().Select("parent_group_id, child_group_id").Order("parent_group_id, child_group_id").
		ScanIntoSliceOfMaps(&rows).Error())
	assert.Equal(t, remainingGroupsGroups, rows)
	assert.NoError(t, dataStore.ActiveGroupAncestors().Select("ancestor_group_id, child_group_id").Order("ancestor_group_id, child_group_id").
		ScanIntoSliceOfMaps(&rows).Error())
	assert.Equal(t, remainingGroupsAncestors, rows)
	var count int64
	assert.NoError(t, dataStore.Table("groups_propagate").Where("ancestors_computation_state != 'done'").Count(&count).Error())
	assert.Zero(t, count)
}

type grantedPermission struct {
	GroupID       int64
	ItemID        int64
	SourceGroupID int64
	Origin        string
	CanView       string
}

func assertGroupLinkedObjects(t *testing.T, dataStore *database.DataStore, remainingGroupIDs []int64,
	expectedGrantedPermissions []grantedPermission, expectedGeneratedPermissions []permissionsGeneratedResultRow,
) {
	var ids []int64
	assert.NoError(t, dataStore.Table("filters").Order("group_id").
		Pluck("group_id", &ids).Error())
	assert.Equal(t, remainingGroupIDs, ids)
	assert.NoError(t, dataStore.Attempts().Order("participant_id").
		Pluck("participant_id", &ids).Error())
	assert.Equal(t, remainingGroupIDs, ids)
	assert.NoError(t, dataStore.Results().Order("participant_id").
		Pluck("participant_id", &ids).Error())
	assert.Equal(t, remainingGroupIDs, ids)

	var grantedPermissions []grantedPermission
	assert.NoError(t, dataStore.PermissionsGranted().Order("group_id, item_id, source_group_id, origin").
		Select("group_id, item_id, source_group_id, origin, can_view").Scan(&grantedPermissions).Error())
	assert.Equal(t, expectedGrantedPermissions, grantedPermissions)

	var generatedPermissions []permissionsGeneratedResultRow
	assert.NoError(t, dataStore.Permissions().Order("group_id, item_id").
		Select("group_id, item_id, can_view_generated").Scan(&generatedPermissions).Error())
	assert.Equal(t, expectedGeneratedPermissions, generatedPermissions)

	assert.Equal(t, remainingGroupIDs, ids)
	assert.NoError(t, dataStore.Permissions().Order("group_id").
		Pluck("group_id", &ids).Error())
	assert.Equal(t, remainingGroupIDs, ids)

	totalRemainingGroupIDs := make([]int64, 0, len(remainingGroupIDs)+1)
	totalRemainingGroupIDs = append(totalRemainingGroupIDs, remainingGroupIDs...)
	totalRemainingGroupIDs = append(totalRemainingGroupIDs, int64(111))
	assert.NoError(t, dataStore.Table("groups_propagate").Order("id").
		Pluck("id", &ids).Error())
	assert.Equal(t, totalRemainingGroupIDs, ids)

	var cnt int64
	assert.NoError(t, dataStore.Table("permissions_propagate").Count(&cnt).Error())
	assert.Zero(t, cnt)
}

const (
	done                                   = "done"
	groupGroupMarksResultsAsChangedFixture = `
	items:
		- {id: 1, default_language_tag: fr}
		- {id: 2, default_language_tag: fr}
		- {id: 3, default_language_tag: fr}
	items_ancestors:
		- {ancestor_item_id: 1, child_item_id: 2}
		- {ancestor_item_id: 1, child_item_id: 3}
		- {ancestor_item_id: 2, child_item_id: 3}
	groups: [{id: 101}, {id: 102}, {id: 103}, {id: 104}, {id: 105}, {id: 106}, {id: 107}, {id: 108}]
	groups_groups:
		- {parent_group_id: 101, child_group_id: 102}
		- {parent_group_id: 101, child_group_id: 103}
		- {parent_group_id: 101, child_group_id: 104}
		- {parent_group_id: 101, child_group_id: 105}
		- {parent_group_id: 101, child_group_id: 106}
		- {parent_group_id: 102, child_group_id: 103}
		- {parent_group_id: 104, child_group_id: 105}
		- {parent_group_id: 107, child_group_id: 105}
		- {parent_group_id: 108, child_group_id: 104, expires_at: 2019-05-30 11:00:00}
	permissions_generated:
		- {group_id: 102, item_id: 1, can_view_generated: info}
		- {group_id: 101, item_id: 1, can_view_generated: none}
		- {group_id: 105, item_id: 3, can_view_generated: content}
		- {group_id: 107, item_id: 1, can_view_generated: none}
		- {group_id: 108, item_id: 1, can_view_generated: solution}
	attempts:
		- {id: 1, participant_id: 101}
		- {id: 1, participant_id: 102}
		- {id: 1, participant_id: 103}
		- {id: 1, participant_id: 104}
		- {id: 1, participant_id: 105}
		- {id: 1, participant_id: 106}
		- {id: 1, participant_id: 107}
		- {id: 1, participant_id: 108}
	results:
		- {attempt_id: 1, participant_id: 101, item_id: 1}
		- {attempt_id: 1, participant_id: 102, item_id: 1}
		- {attempt_id: 1, participant_id: 103, item_id: 1}
		- {attempt_id: 1, participant_id: 104, item_id: 1}
		- {attempt_id: 1, participant_id: 105, item_id: 1}
		- {attempt_id: 1, participant_id: 106, item_id: 1}
		- {attempt_id: 1, participant_id: 107, item_id: 1}
		- {attempt_id: 1, participant_id: 108, item_id: 1}
		- {attempt_id: 1, participant_id: 101, item_id: 2}
		- {attempt_id: 1, participant_id: 102, item_id: 2}
		- {attempt_id: 1, participant_id: 103, item_id: 2}
		- {attempt_id: 1, participant_id: 104, item_id: 2}
		- {attempt_id: 1, participant_id: 105, item_id: 2}
		- {attempt_id: 1, participant_id: 106, item_id: 2}
		- {attempt_id: 1, participant_id: 107, item_id: 2}
		- {attempt_id: 1, participant_id: 108, item_id: 2}
		- {attempt_id: 1, participant_id: 101, item_id: 3}
		- {attempt_id: 1, participant_id: 102, item_id: 3}
		- {attempt_id: 1, participant_id: 103, item_id: 3}
		- {attempt_id: 1, participant_id: 104, item_id: 3}
		- {attempt_id: 1, participant_id: 105, item_id: 3}
		- {attempt_id: 1, participant_id: 106, item_id: 3}
		- {attempt_id: 1, participant_id: 107, item_id: 3}
		- {attempt_id: 1, participant_id: 108, item_id: 3}`
)

func TestGroupGroupStore_TriggerAfterInsert_MarksResultsAsChanged(t *testing.T) {
	for _, test := range []struct {
		name            string
		parentGroupID   int64
		childGroupID    int64
		expiresAt       string
		expectedChanged []resultPrimaryKey
	}{
		{
			name:          "group joins another group",
			parentGroupID: 103,
			childGroupID:  104,
			expiresAt:     "9999-12-31 23:59:59",
			expectedChanged: []resultPrimaryKey{
				{104, 1, 2},
				{104, 1, 3},
				{105, 1, 2},
				{105, 1, 3},
			},
		},
		{
			name:            "group joins a group, but the relation is expired",
			parentGroupID:   103,
			childGroupID:    104,
			expiresAt:       "2019-05-30 11:00:00",
			expectedChanged: []resultPrimaryKey{},
		},
		{
			name:            "group having no results joins another group",
			parentGroupID:   105,
			childGroupID:    103,
			expiresAt:       "9999-12-31 23:59:59",
			expectedChanged: []resultPrimaryKey{},
		},
		{
			name:            "parent group has no permissions on ancestor items",
			parentGroupID:   106,
			childGroupID:    104,
			expiresAt:       "9999-12-31 23:59:59",
			expectedChanged: []resultPrimaryKey{},
		},
		{
			name:            "no new visible item ancestors after joining a group",
			parentGroupID:   105,
			childGroupID:    102,
			expiresAt:       "9999-12-31 23:59:59",
			expectedChanged: []resultPrimaryKey{},
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			db := testhelpers.SetupDBWithFixtureString(groupGroupMarksResultsAsChangedFixture)
			defer func() { _ = db.Close() }()

			dataStore := database.NewDataStore(db)
			assert.NoError(t, dataStore.InTransaction(func(store *database.DataStore) error {
				store.GroupGroups().CreateNewAncestors()
				return nil
			}))
			assert.NoError(t, dataStore.GroupGroups().InsertMap(map[string]interface{}{
				"parent_group_id": test.parentGroupID, "child_group_id": test.childGroupID, "expires_at": test.expiresAt,
			}))

			assertResultsMarkedAsChanged(t, dataStore, test.expectedChanged)
		})
	}
}

func TestGroupGroupStore_TriggerAfterUpdate_MarksResultsAsChanged(t *testing.T) {
	for _, test := range []struct {
		name            string
		parentGroupID   int64
		childGroupID    int64
		expiresAt       string
		expectedChanged []resultPrimaryKey
		doNotSetExpired bool
		noChanges       bool
	}{
		{
			name:          "restore an expired relation",
			parentGroupID: 103,
			childGroupID:  104,
			expiresAt:     "9999-12-31 23:59:59",
			expectedChanged: []resultPrimaryKey{
				{104, 1, 2},
				{104, 1, 3},
				{105, 1, 2},
				{105, 1, 3},
			},
		},
		{
			name:            "expire the relation",
			parentGroupID:   103,
			childGroupID:    104,
			doNotSetExpired: true,
			expiresAt:       "2019-05-30 11:00:00",
			expectedChanged: []resultPrimaryKey{},
		},
		{
			name:            "group having no results joins another group",
			parentGroupID:   105,
			childGroupID:    103,
			expiresAt:       "9999-12-31 23:59:59",
			expectedChanged: []resultPrimaryKey{},
		},
		{
			name:            "parent group has no permissions on ancestor items",
			parentGroupID:   106,
			childGroupID:    104,
			expiresAt:       "9999-12-31 23:59:59",
			expectedChanged: []resultPrimaryKey{},
		},
		{
			name:            "no new visible item ancestors after joining a group",
			parentGroupID:   105,
			childGroupID:    102,
			expiresAt:       "9999-12-31 23:59:59",
			expectedChanged: []resultPrimaryKey{},
		},
		{
			name:            "no changes",
			parentGroupID:   105,
			childGroupID:    102,
			doNotSetExpired: true,
			expiresAt:       "9999-12-31 23:59:59",
			expectedChanged: []resultPrimaryKey{},
			noChanges:       true,
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			expiresAt := "2019-05-30 11:00:00"
			if test.doNotSetExpired {
				expiresAt = maxDateTime
			}
			db := testhelpers.SetupDBWithFixtureString(
				fmt.Sprintf("groups_groups: [{parent_group_id: %d, child_group_id: %d, expires_at: %s}]",
					test.parentGroupID, test.childGroupID, expiresAt),
				groupGroupMarksResultsAsChangedFixture)
			defer func() { _ = db.Close() }()

			dataStore := database.NewDataStore(db)
			assert.NoError(t, dataStore.InTransaction(func(store *database.DataStore) error {
				store.GroupGroups().CreateNewAncestors()
				return nil
			}))
			groupGroupStore := dataStore.GroupGroups()
			result := groupGroupStore.Where("parent_group_id = ?", test.parentGroupID).
				Where("child_group_id = ?", test.childGroupID).
				UpdateColumn("expires_at", test.expiresAt)
			assert.NoError(t, result.Error())
			if test.noChanges {
				assert.Zero(t, result.RowsAffected())
			} else {
				assert.Equal(t, int64(1), result.RowsAffected())
			}
			assertResultsMarkedAsChanged(t, dataStore, test.expectedChanged)
		})
	}
}

func TestGroupGroupStore_TriggerBeforeUpdate_RefusesToModifyParentGroupIDOrChildGroupID(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		groups_groups: [{parent_group_id: 1, child_group_id: 2}]
	`)
	defer func() { _ = db.Close() }()

	const expectedErrorMessage = "Error 1644: Unable to change immutable " +
		"groups_groups.parent_group_id and/or groups_groups.child_group_id"

	groupGroupStore := database.NewDataStore(db).GroupGroups()
	result := groupGroupStore.Where("parent_group_id = 1 AND child_group_id = 2").
		UpdateColumn("parent_group_id", 3)
	assert.EqualError(t, result.Error(), expectedErrorMessage)
	result = groupGroupStore.Where("parent_group_id = 1 AND child_group_id = 2").
		UpdateColumn("child_group_id", 3)
	assert.EqualError(t, result.Error(), expectedErrorMessage)
}

type resultPrimaryKey struct {
	ParticipantID int64
	AttemptID     int64
	ItemID        int64
}

func assertResultsMarkedAsChanged(t *testing.T, dataStore *database.DataStore, expectedChanged []resultPrimaryKey) {
	type resultRow struct {
		ParticipantID int64
		AttemptID     int64
		ItemID        int64
		State         string
	}
	var results []resultRow
	queryResultsAndStatesForTests(t, dataStore.Results(), &results, "")

	expectedChangedResultsMap := make(map[resultPrimaryKey]bool, len(expectedChanged))
	for _, result := range expectedChanged {
		expectedChangedResultsMap[result] = true
	}
	for _, dbResult := range results {
		expectedState := done
		if expectedChangedResultsMap[resultPrimaryKey{
			ParticipantID: dbResult.ParticipantID, AttemptID: dbResult.AttemptID, ItemID: dbResult.ItemID,
		}] {
			expectedState = "to_be_propagated"
		}
		assert.Equal(t, expectedState, dbResult.State,
			"Wrong result propagation state for result with participant_id=%d, attempt_id=%d, item_id=%d",
			dbResult.ParticipantID, dbResult.AttemptID, dbResult.ItemID)
		delete(expectedChangedResultsMap,
			resultPrimaryKey{ParticipantID: dbResult.ParticipantID, AttemptID: dbResult.AttemptID, ItemID: dbResult.ItemID})
	}
	assert.Empty(t, expectedChangedResultsMap,
		"Cannot find results that should be marked as 'to_be_propagated': %#v", expectedChangedResultsMap)
}

func queryResultsAndStatesForTests(t *testing.T, s *database.ResultStore, result interface{}, customColumns string) {
	columns := "participant_id, attempt_id, item_id, IFNULL(state, 'done') AS state"
	if customColumns != "" {
		columns += "," + customColumns
	}
	resultStore := s.Results()
	assert.NoError(t,
		resultStore.Select(columns).
			Joins("LEFT JOIN results_propagate USING(participant_id, attempt_id, item_id)").
			Union(resultStore.Select(columns).
				Joins("RIGHT JOIN results_propagate USING(participant_id, attempt_id, item_id)").SubQuery()).
			Order("participant_id, attempt_id, item_id").Scan(result).Error())
}
