package database

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestItemStore_ContestManagedByUser(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()

	ClearAllDBEnums()
	MockDBEnumQueries(dbMock)

	dbMock.ExpectQuery("^"+regexp.QuoteMeta("SELECT `items`.`id` FROM `items` "+
		"JOIN LATERAL ("+
		"SELECT permissions.item_id, MAX(can_view_generated_value) AS can_view_generated_value, "+
		"MAX(can_grant_view_generated_value) AS can_grant_view_generated_value, "+
		"MAX(can_watch_generated_value) AS can_watch_generated_value, "+
		"MAX(can_edit_generated_value) AS can_edit_generated_value, "+
		"MAX(is_owner_generated) AS is_owner_generated "+
		"FROM permissions_generated AS permissions "+
		"JOIN groups_ancestors_active AS ancestors ON ancestors.ancestor_group_id = permissions.group_id "+
		"WHERE (ancestors.child_group_id = ?) AND permissions.item_id = items.id "+
		"GROUP BY `permissions`.`item_id` "+
		"HAVING MAX(can_view_generated_value) >= ?"+
		") AS permissions ON permissions.item_id = items.id "+
		"WHERE (items.id = ?) AND (items.duration IS NOT NULL) AND (IFNULL(can_grant_view_generated_value, 1) >= ?) AND "+
		"(IFNULL(can_watch_generated_value, 1) >= ?) LIMIT 1")+"$").
		WithArgs(int64(2), 3, int64(123), 2, 2).WillReturnRows(dbMock.NewRows([]string{"id"}).AddRow(123))
	var id int64
	err := NewDataStore(db).Items().ContestManagedByUser(123, &User{GroupID: 2}).
		PluckFirst("items.id", &id).Error()
	assert.NoError(t, err)
	assert.Equal(t, int64(123), id)
}

func TestItemStore_DeleteItem_MustBeRunInTransaction(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	itemStore := NewDataStore(db).Items()
	assert.PanicsWithValue(t, ErrNoTransaction, func() {
		_ = itemStore.DeleteItem(1)
	})
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemStore_DeleteItem_ShouldUseNamedLock(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("^"+regexp.QuoteMeta("SELECT GET_LOCK(?, ?)")+"$").
		WithArgs("items_items", 3).
		WillReturnRows(sqlmock.NewRows([]string{"SELECT GET_LOCK(?, ?)"}).AddRow(int64(0)))
	mock.ExpectRollback()

	store := NewDataStore(db)
	_ = store.InTransaction(func(store *DataStore) error {
		return store.Items().DeleteItem(1)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}
