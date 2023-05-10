package database

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_JoinsUserAndDefaultItemStrings(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mockUser := &User{GroupID: 2, DefaultLanguage: "sl"}

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `items` LEFT JOIN items_strings default_strings " +
			"ON default_strings.item_id = items.id AND default_strings.language_tag = items.default_language_tag " +
			"LEFT JOIN items_strings user_strings ON user_strings.item_id=items.id AND user_strings.language_tag = ?")).
		WithArgs("sl").
		WillReturnRows(mock.NewRows([]string{"id"}))

	var result []interface{}
	err := db.Table("items").JoinsUserAndDefaultItemStrings(mockUser).Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_WhereUserHasViewPermissionOnItems(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()
	ClearAllDBEnums()
	MockDBEnumQueries(mock)

	mock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT * FROM `items` WHERE EXISTS(SELECT 1 FROM permissions_generated AS permissions "+
			"JOIN groups_ancestors_active AS ancestors ON ancestors.child_group_id = ? AND ancestors.ancestor_group_id = permissions.group_id "+
			"WHERE permissions.item_id = items.id AND can_view_generated_value >= ? LIMIT 1)")+"$").
		WithArgs(123, NewDataStore(db).PermissionsGranted().ViewIndexByName("content")).
		WillReturnRows(mock.NewRows([]string{"id"}))

	var result []interface{}
	err := db.Table("items").WhereUserHasViewPermissionOnItems(&User{GroupID: 123}, "content").
		Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_WhereItemsAreVisible(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()
	ClearAllDBEnums()
	MockDBEnumQueries(mock)

	mock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT * FROM `items` WHERE EXISTS(SELECT 1 FROM permissions_generated AS permissions "+
			"JOIN groups_ancestors_active AS ancestors ON ancestors.child_group_id = ? AND ancestors.ancestor_group_id = permissions.group_id "+
			"WHERE permissions.item_id = items.id AND can_view_generated_value >= ? LIMIT 1)")+"$").
		WithArgs(2, NewDataStore(db).PermissionsGranted().ViewIndexByName("info")).
		WillReturnRows(mock.NewRows([]string{"id"}))

	var result []interface{}
	err := db.Table("items").WhereItemsAreVisible(2).Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_WhereUserHaveStartedResultOnItem(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	user := User{
		GroupID: 1,
	}

	mock.ExpectQuery("^" + regexp.QuoteMeta(
		"SELECT `items`.* FROM `items` "+
			"JOIN results AS current_user_results ON current_user_results.item_id = items.id AND current_user_results.participant_id = ? "+
			"WHERE (current_user_results.started = 1)") + "$").
		WithArgs(user.GroupID).
		WillReturnRows(mock.NewRows([]string{"id"}))

	var result []interface{}
	err := db.Table("items").WhereUserHaveStartedResultOnItem(&user).Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
