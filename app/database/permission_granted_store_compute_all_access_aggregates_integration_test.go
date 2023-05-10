//go:build !unit

package database_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/testhelpers"
)

type permissionsGeneratedResultRow struct {
	GroupID          int64
	ItemID           int64
	CanViewGenerated string
}

var expectedRow14 = permissionsGeneratedResultRow{
	GroupID:          1,
	ItemID:           4,
	CanViewGenerated: "solution",
}

func TestPermissionGrantedStore_ComputeAllAccess_AggregatesContentAccess(t *testing.T) {
	db := testhelpers.SetupDBWithFixture("permission_granted_store/compute_all_access/_common")
	defer func() { _ = db.Close() }()

	permissionGrantedStore := database.NewDataStore(db).PermissionsGranted()
	permissionGeneratedStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionGrantedStore.Where("group_id=1 AND item_id=1").
		UpdateColumn("can_view", "content").Error())
	assert.NoError(t, permissionGrantedStore.Where("group_id=1 AND item_id=2").
		UpdateColumn("can_view", "content").Error())
	assert.NoError(t, permissionGeneratedStore.Where("group_id=1 AND item_id=2").
		UpdateColumn("can_view_generated", "info").Error())
	assert.NoError(t, permissionGrantedStore.Where("group_id=1 AND item_id=3").
		UpdateColumn("can_view", "content").Error())
	assert.NoError(t, permissionGrantedStore.Where("group_id=2 AND item_id=1").
		UpdateColumn("can_view", "content").Error())
	assert.NoError(t, permissionGrantedStore.Where("group_id=2 AND item_id=11").
		UpdateColumn("can_view", "content").Error())
	assert.NoError(t, permissionGrantedStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))

	assertAllPermissionsGeneratedAreDone(t, permissionGeneratedStore)

	var result []permissionsGeneratedResultRow
	assert.NoError(t, permissionGeneratedStore.Order("group_id, item_id").Scan(&result).Error())
	assertPermissionsGeneratedResultRowsEqual(t, []permissionsGeneratedResultRow{
		{
			GroupID:          1,
			ItemID:           1,
			CanViewGenerated: "content",
		},
		{
			GroupID:          1,
			ItemID:           2,
			CanViewGenerated: "content",
		},
		{
			GroupID:          1,
			ItemID:           3,
			CanViewGenerated: "content",
		},
		expectedRow14,
		{
			GroupID:          1,
			ItemID:           11,
			CanViewGenerated: "content",
		},
		{
			GroupID:          1,
			ItemID:           12,
			CanViewGenerated: "content", // content_view_propagation = 'as_content' (from 4)
		},
		{
			GroupID:          2,
			ItemID:           1,
			CanViewGenerated: "content",
		},
		{
			GroupID:          2,
			ItemID:           11,
			CanViewGenerated: "content",
		},
		{
			GroupID:          2,
			ItemID:           12,
			CanViewGenerated: "none", // content_view_propagation = 'none' (from 11)
		},
	}, result)
}

func TestPermissionGrantedStore_ComputeAllAccess_AggregatesContentAccessAsInfo(t *testing.T) {
	db := testhelpers.SetupDBWithFixture("permission_granted_store/compute_all_access/_common")
	defer func() { _ = db.Close() }()

	permissionGrantedStore := database.NewDataStore(db).PermissionsGranted()
	permissionGeneratedStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionGrantedStore.Where("group_id=1 AND item_id=1").
		UpdateColumn("can_view", "content").Error())
	assert.NoError(t, permissionGrantedStore.Where("group_id=1 AND item_id=2").
		UpdateColumn("can_view", "content").Error())
	assert.NoError(t, permissionGrantedStore.Where("group_id=1 AND item_id=3").
		UpdateColumn("can_view", "content").Error())
	assert.NoError(t, permissionGrantedStore.Where("group_id=2 AND item_id=1").
		UpdateColumn("can_view", "content").Error())
	assert.NoError(t, permissionGrantedStore.Where("group_id=2 AND item_id=11").
		UpdateColumn("can_view", "content").Error())
	assert.NoError(t, permissionGrantedStore.ItemItems().Updates(map[string]interface{}{
		"content_view_propagation": "as_info",
	}).Error())
	assert.NoError(t, permissionGrantedStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))

	assertAllPermissionsGeneratedAreDone(t, permissionGeneratedStore)

	var result []permissionsGeneratedResultRow
	assert.NoError(t, permissionGeneratedStore.Order("group_id, item_id").Scan(&result).Error())
	assertPermissionsGeneratedResultRowsEqual(t, []permissionsGeneratedResultRow{
		{
			GroupID:          1,
			ItemID:           1,
			CanViewGenerated: "content",
		},
		{
			GroupID:          1,
			ItemID:           2,
			CanViewGenerated: "content",
		},
		{
			GroupID:          1,
			ItemID:           3,
			CanViewGenerated: "content",
		},
		expectedRow14,
		{
			GroupID:          1,
			ItemID:           11,
			CanViewGenerated: "info", // since content_view_propagation = "as_info"
		},
		{
			GroupID:          1,
			ItemID:           12,
			CanViewGenerated: "info", // since content_view_propagation = "as_info"
		},
		{
			GroupID:          2,
			ItemID:           1,
			CanViewGenerated: "content",
		},
		{
			GroupID:          2,
			ItemID:           11,
			CanViewGenerated: "content",
		},
		{
			GroupID:          2,
			ItemID:           12,
			CanViewGenerated: "info", // since content_view_propagation = "as_info"
		},
	}, result)
}

func TestPermissionGrantedStore_ComputeAllAccess_AggregatesAccess(t *testing.T) {
	for _, access := range []string{"solution", "content_with_descendants"} {
		access := access
		t.Run(access, func(t *testing.T) {
			db := testhelpers.SetupDBWithFixture("permission_granted_store/compute_all_access/_common")
			defer func() { _ = db.Close() }()

			permissionGrantedStore := database.NewDataStore(db).PermissionsGranted()
			permissionGeneratedStore := database.NewDataStore(db).Permissions()
			assert.NoError(t, permissionGrantedStore.Where("group_id=1 AND item_id=1").
				UpdateColumn("can_view", access).Error())
			assert.NoError(t, permissionGrantedStore.Where("group_id=1 AND item_id=2").
				UpdateColumn("can_view", access).Error())
			assert.NoError(t, permissionGrantedStore.Where("group_id=1 AND item_id=3").
				UpdateColumn("can_view", access).Error())
			assert.NoError(t, permissionGrantedStore.Where("group_id=2 AND item_id=1").
				UpdateColumn("can_view", access).Error())
			assert.NoError(t, permissionGrantedStore.Where("group_id=2 AND item_id=11").
				UpdateColumn("can_view", access).Error())
			assert.NoError(t, permissionGrantedStore.InTransaction(func(ds *database.DataStore) error {
				ds.PermissionsGranted().ComputeAllAccess()
				return nil
			}))

			assertAllPermissionsGeneratedAreDone(t, permissionGeneratedStore)

			var result []permissionsGeneratedResultRow
			assert.NoError(t, permissionGeneratedStore.Order("group_id, item_id").Scan(&result).Error())
			assertPermissionsGeneratedResultRowsEqual(t, []permissionsGeneratedResultRow{
				{
					GroupID:          1,
					ItemID:           1,
					CanViewGenerated: access,
				},
				{
					GroupID:          1,
					ItemID:           2,
					CanViewGenerated: access,
				},
				{
					GroupID:          1,
					ItemID:           3,
					CanViewGenerated: access,
				},
				expectedRow14,
				{
					GroupID:          1,
					ItemID:           11,
					CanViewGenerated: "content", // since content_view_propagation = "as_content"
				},
				{
					GroupID:          1,
					ItemID:           12,
					CanViewGenerated: "content", // since content_view_propagation = "as_content" (from 4)
				},
				{
					GroupID:          2,
					ItemID:           1,
					CanViewGenerated: access,
				},
				{
					GroupID:          2,
					ItemID:           11,
					CanViewGenerated: access,
				},
				{
					GroupID:          2,
					ItemID:           12,
					CanViewGenerated: "none", // since content_view_propagation = "none" (from 11)
				},
			}, result)
		})
	}
}

func TestPermissionGrantedStore_ComputeAllAccess_PropagatesCanView(t *testing.T) {
	for _, testcase := range []struct {
		canView                    string
		contentViewPropagation     string
		upperViewLevelsPropagation string
		expectedCanView            string
	}{
		{
			canView: "none", contentViewPropagation: "as_content",
			upperViewLevelsPropagation: "as_is",
			expectedCanView:            "none",
		},
		{
			canView: "info", contentViewPropagation: "as_content",
			upperViewLevelsPropagation: "as_is",
			expectedCanView:            "none",
		},
		{
			canView: "content", contentViewPropagation: "none",
			upperViewLevelsPropagation: "as_is",
			expectedCanView:            "none",
		},
		{
			canView: "content", contentViewPropagation: "as_info",
			upperViewLevelsPropagation: "as_is",
			expectedCanView:            "info",
		},
		{
			canView: "content", contentViewPropagation: "as_content",
			upperViewLevelsPropagation: "as_is",
			expectedCanView:            "content",
		},
		{
			canView: "content_with_descendants", contentViewPropagation: "none",
			upperViewLevelsPropagation: "use_content_view_propagation",
			expectedCanView:            "none",
		},
		{
			canView: "content_with_descendants", contentViewPropagation: "as_info",
			upperViewLevelsPropagation: "use_content_view_propagation",
			expectedCanView:            "info",
		},
		{
			canView: "content_with_descendants", contentViewPropagation: "as_content",
			upperViewLevelsPropagation: "use_content_view_propagation",
			expectedCanView:            "content",
		},
		{
			canView: "content_with_descendants", contentViewPropagation: "none",
			upperViewLevelsPropagation: "as_content_with_descendants",
			expectedCanView:            "content_with_descendants",
		},
		{
			canView: "content_with_descendants", contentViewPropagation: "none",
			upperViewLevelsPropagation: "as_is",
			expectedCanView:            "content_with_descendants",
		},
		{
			canView: "solution", contentViewPropagation: "none",
			upperViewLevelsPropagation: "use_content_view_propagation", expectedCanView: "none",
		},
		{
			canView: "solution", contentViewPropagation: "as_info",
			upperViewLevelsPropagation: "use_content_view_propagation", expectedCanView: "info",
		},
		{
			canView: "solution", contentViewPropagation: "as_content",
			upperViewLevelsPropagation: "use_content_view_propagation", expectedCanView: "content",
		},
		{
			canView: "solution", contentViewPropagation: "none",
			upperViewLevelsPropagation: "as_content_with_descendants", expectedCanView: "content_with_descendants",
		},
		{
			canView: "solution", contentViewPropagation: "none",
			upperViewLevelsPropagation: "as_is", expectedCanView: "solution",
		},
	} {
		testcase := testcase
		t.Run(testcase.canView+" as "+testcase.expectedCanView, func(t *testing.T) {
			db := testhelpers.SetupDBWithFixtureString(`
				items: [{id: 1, default_language_tag: fr}, {id: 2, default_language_tag: fr}]
				groups: [{id: 1}]
				items_items:
					- {parent_item_id: 1, child_item_id: 2, child_order: 1,
						content_view_propagation: ` + testcase.contentViewPropagation + `,
						upper_view_levels_propagation: ` + testcase.upperViewLevelsPropagation + `}
				permissions_granted: [{group_id: 1, item_id: 1, source_group_id: 1, can_view: ` + testcase.canView + `}]`)
			permissionStore := database.NewDataStore(db).Permissions()
			assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
				ds.PermissionsGranted().ComputeAllAccess()
				return nil
			}))
			var result string
			assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 2").
				PluckFirst("can_view_generated", &result).Error())
			assert.Equal(t, testcase.expectedCanView, result)
		})
	}
}

func TestPermissionGrantedStore_ComputeAllAccess_PropagatesMaxOfParentsCanView(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items:
			- {id: 1, default_language_tag: fr}
			- {id: 2, default_language_tag: fr}
			- {id: 3, default_language_tag: fr}
			- {id: 4, default_language_tag: fr}
		groups: [{id: 1}]
		items_items:
			- {parent_item_id: 1, child_item_id: 4, child_order: 1, content_view_propagation: as_content,
				upper_view_levels_propagation: use_content_view_propagation}
			- {parent_item_id: 2, child_item_id: 4, child_order: 2, content_view_propagation: as_info,
				upper_view_levels_propagation: as_is}
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 1, can_view: info}
			- {group_id: 1, item_id: 2, source_group_id: 1, can_view: content_with_descendants}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 4").
		PluckFirst("can_view_generated", &result).Error())
	assert.Equal(t, "content_with_descendants", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_PropagatesMaxOfParentsAndGrantedCanView(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}, {id: 2, default_language_tag: fr}]
		groups: [{id: 1}, {id: 2}]
		items_items:
			- {parent_item_id: 1, child_item_id: 2, child_order: 1, content_view_propagation: as_content,
				upper_view_levels_propagation: as_is}
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 1, can_view: content}
			- {group_id: 1, item_id: 2, source_group_id: 1, can_view: content_with_descendants}
			- {group_id: 2, item_id: 2, source_group_id: 1, can_view: solution}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 2").
		PluckFirst("can_view_generated", &result).Error())
	assert.Equal(t, "content_with_descendants", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_AggregatesMaxOfGrantedCanView(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}]
		groups: [{id: 1}, {id: 2}]
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 2, can_view: content}
			- {group_id: 1, item_id: 1, source_group_id: 1, can_view: content_with_descendants}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 1").
		PluckFirst("can_view_generated", &result).Error())
	assert.Equal(t, "content_with_descendants", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_AggregatesCanViewAsSolutionForOwners(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}]
		groups: [{id: 1}, {id: 2}]
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 2, can_view: content}
			- {group_id: 1, item_id: 1, source_group_id: 1, can_view: content_with_descendants, is_owner: 1}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 1").
		PluckFirst("can_view_generated", &result).Error())
	assert.Equal(t, "solution", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_PropagatesMaxOfParentsCanGrantView(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items:
			- {id: 1, default_language_tag: fr}
			- {id: 2, default_language_tag: fr}
			- {id: 3, default_language_tag: fr}
			- {id: 4, default_language_tag: fr}
			- {id: 5, default_language_tag: fr}
		groups: [{id: 1}]
		items_items:
			- {parent_item_id: 1, child_item_id: 5, child_order: 1, grant_view_propagation: 1}
			- {parent_item_id: 2, child_item_id: 5, child_order: 2, grant_view_propagation: 1}
			- {parent_item_id: 3, child_item_id: 5, child_order: 2, grant_view_propagation: 1}
			- {parent_item_id: 4, child_item_id: 5, child_order: 2, grant_view_propagation: 0}
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 1, can_grant_view: content}
			- {group_id: 1, item_id: 2, source_group_id: 1, can_grant_view: content_with_descendants}
			- {group_id: 1, item_id: 3, source_group_id: 1, can_grant_view: solution_with_grant}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 5").
		PluckFirst("can_grant_view_generated", &result).Error())
	assert.Equal(t, "solution", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_PropagatesMaxOfParentsAndGrantedCanGrantView(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}, {id: 2, default_language_tag: fr}]
		groups: [{id: 1}, {id: 2}]
		items_items:
			- {parent_item_id: 1, child_item_id: 2, child_order: 1, grant_view_propagation: 1}
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 1, can_grant_view: content}
			- {group_id: 1, item_id: 2, source_group_id: 1, can_grant_view: solution_with_grant}
			- {group_id: 2, item_id: 2, source_group_id: 1, can_grant_view: solution}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 2").
		PluckFirst("can_grant_view_generated", &result).Error())
	assert.Equal(t, "solution_with_grant", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_AggregatesMaxOfGrantedCanGrantView(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}]
		groups: [{id: 1}]
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 1, origin: self, can_grant_view: content}
			- {group_id: 1, item_id: 1, source_group_id: 1, origin: group_membership, can_grant_view: content_with_descendants}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 1").
		PluckFirst("can_grant_view_generated", &result).Error())
	assert.Equal(t, "content_with_descendants", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_AggregatesCanGrantViewAsSolutionWithGrantForOwners(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}]
		groups: [{id: 1}, {id: 2}]
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 2, can_grant_view: content}
			- {group_id: 1, item_id: 1, source_group_id: 1, can_grant_view: content_with_descendants, is_owner: 1}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 1").
		PluckFirst("can_grant_view_generated", &result).Error())
	assert.Equal(t, "solution_with_grant", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_PropagatesMaxOfParentsCanWatch(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items:
			- {id: 1, default_language_tag: fr}
			- {id: 2, default_language_tag: fr}
			- {id: 3, default_language_tag: fr}
			- {id: 4, default_language_tag: fr}
			- {id: 5, default_language_tag: fr}
		groups: [{id: 1}]
		items_items:
			- {parent_item_id: 1, child_item_id: 5, child_order: 1, watch_propagation: 1}
			- {parent_item_id: 2, child_item_id: 5, child_order: 2, watch_propagation: 1}
			- {parent_item_id: 3, child_item_id: 5, child_order: 2, watch_propagation: 1}
			- {parent_item_id: 4, child_item_id: 5, child_order: 2, watch_propagation: 0}
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 1, can_watch: result}
			- {group_id: 1, item_id: 2, source_group_id: 1, can_watch: answer}
			- {group_id: 1, item_id: 3, source_group_id: 1, can_watch: answer_with_grant}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 5").
		PluckFirst("can_watch_generated", &result).Error())
	assert.Equal(t, "answer", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_PropagatesMaxOfParentsAndGrantedCanWatch(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}, {id: 2, default_language_tag: fr}]
		groups: [{id: 1}, {id: 2}]
		items_items:
			- {parent_item_id: 1, child_item_id: 2, child_order: 1, watch_propagation: 1}
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 1, can_watch: result}
			- {group_id: 1, item_id: 2, source_group_id: 1, can_watch: answer_with_grant}
			- {group_id: 2, item_id: 2, source_group_id: 1, can_watch: answer}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 2").
		PluckFirst("can_watch_generated", &result).Error())
	assert.Equal(t, "answer_with_grant", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_AggregatesMaxOfGrantedCanWatch(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}]
		groups: [{id: 1}, {id: 2}]
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 2, can_watch: result}
			- {group_id: 1, item_id: 1, source_group_id: 1, can_watch: answer}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 1").
		PluckFirst("can_watch_generated", &result).Error())
	assert.Equal(t, "answer", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_AggregatesCanWatchAsAnswerWithGrantForOwners(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}]
		groups: [{id: 1}, {id: 2}]
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 2, can_watch: result}
			- {group_id: 1, item_id: 1, source_group_id: 1, can_watch: answer, is_owner: 1}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 1").
		PluckFirst("can_watch_generated", &result).Error())
	assert.Equal(t, "answer_with_grant", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_PropagatesMaxOfParentsCanEdit(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items:
			- {id: 1, default_language_tag: fr}
			- {id: 2, default_language_tag: fr}
			- {id: 3, default_language_tag: fr}
			- {id: 4, default_language_tag: fr}
			- {id: 5, default_language_tag: fr}
		groups: [{id: 1}]
		items_items:
			- {parent_item_id: 1, child_item_id: 5, child_order: 1, edit_propagation: 1}
			- {parent_item_id: 2, child_item_id: 5, child_order: 2, edit_propagation: 1}
			- {parent_item_id: 3, child_item_id: 5, child_order: 2, edit_propagation: 1}
			- {parent_item_id: 4, child_item_id: 5, child_order: 2, edit_propagation: 0}
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 1, can_edit: children}
			- {group_id: 1, item_id: 2, source_group_id: 1, can_edit: all}
			- {group_id: 1, item_id: 3, source_group_id: 1, can_edit: all_with_grant}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 5").
		PluckFirst("can_edit_generated", &result).Error())
	assert.Equal(t, "all", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_PropagatesMaxOfParentsAndGrantedCanEdit(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}, {id: 2, default_language_tag: fr}]
		groups: [{id: 1}, {id: 2}]
		items_items:
			- {parent_item_id: 1, child_item_id: 2, child_order: 1, edit_propagation: 1}
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 1, can_edit: children}
			- {group_id: 1, item_id: 2, source_group_id: 1, can_edit: all_with_grant}
			- {group_id: 2, item_id: 2, source_group_id: 1, can_edit: all}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 2").
		PluckFirst("can_edit_generated", &result).Error())
	assert.Equal(t, "all_with_grant", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_AggregatesMaxOfGrantedCanEdit(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}]
		groups: [{id: 1}, {id: 2}]
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 2, can_edit: children}
			- {group_id: 1, item_id: 1, source_group_id: 1, can_edit: all}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 1").
		PluckFirst("can_edit_generated", &result).Error())
	assert.Equal(t, "all", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_AggregatesCanEditAsAllWithGrantForOwners(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		items: [{id: 1, default_language_tag: fr}]
		groups: [{id: 1}, {id: 2}]
		permissions_granted:
			- {group_id: 1, item_id: 1, source_group_id: 2, can_edit: children}
			- {group_id: 1, item_id: 1, source_group_id: 1, can_edit: all, is_owner: 1}`)
	permissionStore := database.NewDataStore(db).Permissions()
	assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
		ds.PermissionsGranted().ComputeAllAccess()
		return nil
	}))
	var result string
	assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 1").
		PluckFirst("can_edit_generated", &result).Error())
	assert.Equal(t, "all_with_grant", result)
}

func TestPermissionGrantedStore_ComputeAllAccess_Propagates(t *testing.T) {
	type testStruct struct {
		parentValue     string
		propagationMode bool
		expectedValue   string
	}
	for _, testsuite := range []struct {
		column            string
		propagationColumn string
		tests             []testStruct
	}{
		{
			column: "can_grant_view", propagationColumn: "grant_view_propagation",
			tests: []testStruct{
				{parentValue: "none", propagationMode: true, expectedValue: "none"},
				{parentValue: "enter", propagationMode: true, expectedValue: "enter"},
				{parentValue: "enter", propagationMode: false, expectedValue: "none"},
				{parentValue: "content", propagationMode: true, expectedValue: "content"},
				{parentValue: "content", propagationMode: false, expectedValue: "none"},
				{parentValue: "content_with_descendants", propagationMode: true, expectedValue: "content_with_descendants"},
				{parentValue: "content_with_descendants", propagationMode: false, expectedValue: "none"},
				{parentValue: "solution", propagationMode: true, expectedValue: "solution"},
				{parentValue: "solution", propagationMode: false, expectedValue: "none"},
				{parentValue: "solution_with_grant", propagationMode: true, expectedValue: "solution"},
				{parentValue: "solution_with_grant", propagationMode: false, expectedValue: "none"},
			},
		},
		{
			column: "can_watch", propagationColumn: "watch_propagation",
			tests: []testStruct{
				{parentValue: "none", propagationMode: true, expectedValue: "none"},
				{parentValue: "result", propagationMode: true, expectedValue: "result"},
				{parentValue: "result", propagationMode: false, expectedValue: "none"},
				{parentValue: "answer", propagationMode: true, expectedValue: "answer"},
				{parentValue: "answer", propagationMode: false, expectedValue: "none"},
				{parentValue: "answer_with_grant", propagationMode: true, expectedValue: "answer"},
				{parentValue: "answer_with_grant", propagationMode: false, expectedValue: "none"},
			},
		},
		{
			column: "can_edit", propagationColumn: "edit_propagation",
			tests: []testStruct{
				{parentValue: "none", propagationMode: true, expectedValue: "none"},
				{parentValue: "children", propagationMode: true, expectedValue: "children"},
				{parentValue: "children", propagationMode: false, expectedValue: "none"},
				{parentValue: "all", propagationMode: true, expectedValue: "all"},
				{parentValue: "all", propagationMode: false, expectedValue: "none"},
				{parentValue: "all_with_grant", propagationMode: true, expectedValue: "all"},
				{parentValue: "all_with_grant", propagationMode: false, expectedValue: "none"},
			},
		},
	} {
		testsuite := testsuite
		t.Run(testsuite.column, func(t *testing.T) {
			for _, testcase := range testsuite.tests {
				testcase := testcase
				testPropagates(t, testsuite.column, testsuite.propagationColumn, testcase.parentValue,
					testcase.propagationMode, testcase.expectedValue)
			}
		})
	}
}

func testPropagates(t *testing.T, column, propagationColumn, valueForParent string, propagationMode bool, expectedValue string) {
	t.Run(valueForParent+" as "+expectedValue, func(t *testing.T) {
		grantViewPropagationString := fmt.Sprint(propagationMode)
		db := testhelpers.SetupDBWithFixtureString(`
				items: [{id: 1, default_language_tag: fr}, {id: 2, default_language_tag: fr}]
				groups: [{id: 1}]
				items_items:
					- {parent_item_id: 1, child_item_id: 2, child_order: 1,
						` + propagationColumn + `: ` + grantViewPropagationString + `}
				permissions_granted: [{group_id: 1, item_id: 1, source_group_id: 1, ` + column + `: ` + valueForParent + `}]`)
		permissionStore := database.NewDataStore(db).Permissions()
		assert.NoError(t, permissionStore.InTransaction(func(ds *database.DataStore) error {
			ds.PermissionsGranted().ComputeAllAccess()
			return nil
		}))
		var result string
		assert.NoError(t, permissionStore.Where("group_id = 1 AND item_id = 2").
			PluckFirst(column+"_generated", &result).Error())
		assert.Equal(t, expectedValue, result)
	})
}

func ptrTime(t time.Time) *time.Time { return &t }

func assertPermissionsGeneratedResultRowsEqual(t *testing.T, expected, got []permissionsGeneratedResultRow) {
	if len(got) != len(expected) {
		assert.ElementsMatch(t, expected, got)
		return
	}

	for i := 0; i < len(expected); i++ {
		assert.Equal(t, expected[i], got[i])
	}
}

func assertAllPermissionsGeneratedAreDone(t *testing.T, permissionGeneratedStore *database.PermissionGeneratedStore) {
	var cnt int64
	t.Skip()
	assert.NoError(t, permissionGeneratedStore.Table("permissions_propagate").Count(&cnt).Error())
	assert.Zero(t, cnt, "found not done group-item pairs")
}
