package database

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPermissionGeneratedStore_AggregatedPermissionsForItemsVisibleToGroup(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	ClearAllDBEnums()
	MockDBEnumQueries(mock)
	defer ClearAllDBEnums()

	mock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT permissions.item_id, MAX(can_view_generated_value) AS can_view_generated_value, "+
			"MAX(can_grant_view_generated_value) AS can_grant_view_generated_value, "+
			"MAX(can_watch_generated_value) AS can_watch_generated_value, "+
			"MAX(can_edit_generated_value) AS can_edit_generated_value, "+
			"MAX(is_owner_generated) AS is_owner_generated "+
			"FROM permissions_generated AS permissions "+
			"JOIN groups_ancestors_active AS ancestors ON ancestors.ancestor_group_id = permissions.group_id "+
			"WHERE ancestors.child_group_id = ? GROUP BY `permissions`.`item_id` "+
			"HAVING MAX(can_view_generated_value) >= ?")+"$").
		WithArgs(3, NewDataStore(db).PermissionsGranted().ViewIndexByName("info")).
		WillReturnRows(mock.NewRows([]string{"id"}))

	var result []interface{}
	err := NewDataStore(db).Permissions().AggregatedPermissionsForItemsVisibleToGroup(3).Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPermissionGeneratedStore_AggregatedPermissionsForItemsOnWhichGroupHasViewPermission(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	MockDBEnumQueries(mock)
	defer ClearAllDBEnums()

	mock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT permissions.item_id, MAX(can_view_generated_value) AS can_view_generated_value, "+
			"MAX(can_grant_view_generated_value) AS can_grant_view_generated_value, "+
			"MAX(can_watch_generated_value) AS can_watch_generated_value, "+
			"MAX(can_edit_generated_value) AS can_edit_generated_value, "+
			"MAX(is_owner_generated) AS is_owner_generated "+
			"FROM permissions_generated AS permissions "+
			"JOIN groups_ancestors_active AS ancestors ON ancestors.ancestor_group_id = permissions.group_id "+
			"WHERE ancestors.child_group_id = ? GROUP BY `permissions`.`item_id` "+
			"HAVING MAX(can_view_generated_value) >= ?")+"$").
		WithArgs(2, NewDataStore(db).PermissionsGranted().ViewIndexByName("content")).
		WillReturnRows(mock.NewRows([]string{"id"}))

	var result []interface{}
	err := NewDataStore(db).Permissions().AggregatedPermissionsForItemsOnWhichGroupHasViewPermission(2, "content").Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
