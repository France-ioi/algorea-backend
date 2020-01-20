package database

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPermissionGeneratedStore_AccessRightsForItemsVisibleToUser(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mockUser := &User{GroupID: 2, DefaultLanguage: "sl"}

	clearAllPermissionEnums()
	mockPermissionEnumQueries(mock)
	defer clearAllPermissionEnums()

	mock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT item_id, MAX(can_view_generated_value) AS can_view_generated_value, "+
			"MAX(can_grant_view_generated_value) AS can_grant_view_generated_value, "+
			"MAX(can_watch_generated_value) AS can_watch_generated_value, "+
			"MAX(can_edit_generated_value) AS can_edit_generated_value, "+
			"MAX(is_owner_generated) AS is_owner_generated "+
			"FROM permissions_generated AS permissions JOIN "+
			"( SELECT * FROM groups_ancestors_active "+
			"WHERE groups_ancestors_active.child_group_id = ? ) AS ancestors "+
			"ON ancestors.ancestor_group_id = permissions.group_id "+
			"WHERE (can_view_generated_value >= ?) GROUP BY permissions.item_id")+"$").
		WithArgs(2, NewDataStore(db).PermissionsGranted().ViewIndexByName("info")).
		WillReturnRows(mock.NewRows([]string{"id"}))

	var result []interface{}
	err := NewDataStore(db).Permissions().VisibleToUser(mockUser).Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPermissionGeneratedStore_WithViewPermissionForUser(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mockUser := &User{GroupID: 2, DefaultLanguage: "sl"}

	mockPermissionEnumQueries(mock)
	defer clearAllPermissionEnums()

	mock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT item_id, MAX(can_view_generated_value) AS can_view_generated_value, "+
			"MAX(can_grant_view_generated_value) AS can_grant_view_generated_value, "+
			"MAX(can_watch_generated_value) AS can_watch_generated_value, "+
			"MAX(can_edit_generated_value) AS can_edit_generated_value, "+
			"MAX(is_owner_generated) AS is_owner_generated "+
			"FROM permissions_generated AS permissions JOIN "+
			"( SELECT * FROM groups_ancestors_active "+
			"WHERE groups_ancestors_active.child_group_id = ? ) AS ancestors "+
			"ON ancestors.ancestor_group_id = permissions.group_id "+
			"WHERE (can_view_generated_value >= ?) GROUP BY permissions.item_id")+"$").
		WithArgs(2, NewDataStore(db).PermissionsGranted().ViewIndexByName("content")).
		WillReturnRows(mock.NewRows([]string{"id"}))

	var result []interface{}
	err := NewDataStore(db).Permissions().WithViewPermissionForUser(mockUser, "content").Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
