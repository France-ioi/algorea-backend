package database

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var accessTestCases = []struct {
	desc              string
	itemIDs           []int64
	itemAccessDetails []ItemAccessDetailsWithID
	err               error
}{
	{
		desc:              "empty IDs",
		itemIDs:           nil,
		itemAccessDetails: nil,
		err:               nil,
	},
	{
		desc:              "empty access results",
		itemIDs:           []int64{21, 22, 23},
		itemAccessDetails: nil,
		err:               fmt.Errorf("not visible item_id 21"),
	},
	{
		desc:    "missing access result on one of the items",
		itemIDs: []int64{21, 22, 23},
		itemAccessDetails: []ItemAccessDetailsWithID{
			{ItemID: 21, ItemAccessDetails: ItemAccessDetails{CanView: "content_with_descendants"}},
			{ItemID: 23, ItemAccessDetails: ItemAccessDetails{CanView: "content_with_descendants"}},
		},
		err: fmt.Errorf("not visible item_id 22"),
	},
	{
		desc:    "no access on one of the items",
		itemIDs: []int64{21, 22, 23},
		itemAccessDetails: []ItemAccessDetailsWithID{
			{ItemID: 21, ItemAccessDetails: ItemAccessDetails{CanView: "content_with_descendants"}},
			{ItemID: 22, ItemAccessDetails: ItemAccessDetails{CanView: "none"}},
			{ItemID: 23, ItemAccessDetails: ItemAccessDetails{CanView: "content_with_descendants"}},
		},
		err: fmt.Errorf("not enough perm on item_id 22"),
	},
	{
		desc:    "full access on all items",
		itemIDs: []int64{21, 22, 23},
		itemAccessDetails: []ItemAccessDetailsWithID{
			{ItemID: 21, ItemAccessDetails: ItemAccessDetails{CanView: "content_with_descendants"}},
			{ItemID: 22, ItemAccessDetails: ItemAccessDetails{CanView: "content_with_descendants"}},
			{ItemID: 23, ItemAccessDetails: ItemAccessDetails{CanView: "content_with_descendants"}},
		},
		err: nil,
	},
	{
		desc:    "content access on all but last, info access to the last",
		itemIDs: []int64{21, 22, 23},
		itemAccessDetails: []ItemAccessDetailsWithID{
			{ItemID: 21, ItemAccessDetails: ItemAccessDetails{CanView: "content"}},
			{ItemID: 22, ItemAccessDetails: ItemAccessDetails{CanView: "content"}},
			{ItemID: 23, ItemAccessDetails: ItemAccessDetails{CanView: "info"}},
		},
		err: nil,
	},
	{
		desc:    "content access on all but last, no access to the last",
		itemIDs: []int64{21, 22, 23},
		itemAccessDetails: []ItemAccessDetailsWithID{
			{ItemID: 21, ItemAccessDetails: ItemAccessDetails{CanView: "content"}},
			{ItemID: 22, ItemAccessDetails: ItemAccessDetails{CanView: "content"}},
			{ItemID: 23, ItemAccessDetails: ItemAccessDetails{CanView: "none"}},
		},
		err: errors.New("not enough perm on item_id 23"),
	},
	{
		desc:    "content access on all but last, no access details for the last",
		itemIDs: []int64{21, 22, 23},
		itemAccessDetails: []ItemAccessDetailsWithID{
			{ItemID: 21, ItemAccessDetails: ItemAccessDetails{CanView: "content"}},
			{ItemID: 22, ItemAccessDetails: ItemAccessDetails{CanView: "content"}},
		},
		err: errors.New("not visible item_id 23"),
	},
}

func TestItemStore_CheckAccess(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()

	clearAllPermissionEnums()
	mockPermissionEnumQueries(dbMock)
	NewDataStore(db).PermissionsGranted().loadAllPermissionEnums()

	for _, tC := range accessTestCases {
		tC := tC
		t.Run(tC.desc, func(t *testing.T) {
			err := NewDataStore(db).Items().checkAccess(tC.itemIDs, tC.itemAccessDetails)
			if err != nil {
				if tC.err != nil {
					if want, got := tC.err.Error(), err.Error(); want != got {
						t.Fatalf("Expected error to be %v, got: %v", want, got)
					}
					return
				}
				t.Fatalf("Unexpected error: %v", err)
			}
			if tC.err != nil {
				t.Fatalf("Expected error %v", tC.err)
			}
		})
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestItemStore_ValidateUserAccess(t *testing.T) {
	for _, tC := range accessTestCases {
		tC := tC
		t.Run(tC.desc, func(t *testing.T) {
			db, dbMock := NewDBMock()
			defer func() { _ = db.Close() }()
			clearAllPermissionEnums()
			mockPermissionEnumQueries(dbMock)
			permissionStore := NewDataStore(db).PermissionsGranted()
			permissionStore.loadAllPermissionEnums()

			dbRows := dbMock.NewRows([]string{"item_id", "can_view_generated_value"})
			for _, row := range tC.itemAccessDetails {
				dbRows = dbRows.AddRow(row.ItemID, permissionStore.ViewIndexByName(row.CanView))
			}
			args := make([]driver.Value, 0, len(tC.itemIDs)+2)
			args = append(args, 123, permissionStore.ViewIndexByName("info"))
			for _, id := range tC.itemIDs {
				args = append(args, id)
			}
			questionMarks := "NULL"
			if len(tC.itemIDs) > 0 {
				questionMarks = "?"
				if len(tC.itemIDs) > 1 {
					questionMarks += strings.Repeat(",?", len(tC.itemIDs)-1)
				}
			}
			dbMock.ExpectQuery("^" + regexp.QuoteMeta(
				"SELECT item_id, MAX(can_view_generated_value) AS can_view_generated_value, "+
					"MAX(can_grant_view_generated_value) AS can_grant_view_generated_value, "+
					"MAX(can_watch_generated_value) AS can_watch_generated_value, "+
					"MAX(can_edit_generated_value) AS can_edit_generated_value, "+
					"MAX(is_owner_generated) AS is_owner_generated "+
					"FROM permissions_generated AS permissions "+
					"JOIN ( "+
					"SELECT * FROM groups_ancestors_active WHERE groups_ancestors_active.child_group_id = ? "+
					") AS ancestors "+
					"ON ancestors.ancestor_group_id = permissions.group_id "+
					"WHERE (can_view_generated_value >= ?) AND (item_id IN ("+questionMarks+")) "+
					"GROUP BY permissions.item_id") + "$").
				WithArgs(args...).
				WillReturnRows(dbRows)
			result, err := permissionStore.Items().ValidateUserAccess(&User{GroupID: 123}, tC.itemIDs)
			assert.NoError(t, err)
			assert.Equal(t, tC.err == nil, result)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

func TestItemStore_ValidateUserAccess_FailsOnDBError(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()
	clearAllPermissionEnums()
	mockPermissionEnumQueries(dbMock)

	expectedError := errors.New("some error")
	dbMock.ExpectQuery("SELECT item_id").WillReturnError(expectedError)
	result, err := NewDataStore(db).Items().ValidateUserAccess(&User{GroupID: 123}, []int64{1, 2, 3})
	assert.Equal(t, expectedError, err)
	assert.False(t, result)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestItemStore_CheckSubmissionRights_MustBeInTransaction(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()

	assert.PanicsWithValue(t, ErrNoTransaction, func() {
		_, _, _ = NewDataStore(db).Items().CheckSubmissionRights(12, &User{GroupID: 14})
	})

	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestItemStore_ContestManagedByUser(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()

	clearAllPermissionEnums()
	mockPermissionEnumQueries(dbMock)

	dbMock.ExpectQuery(regexp.QuoteMeta("SELECT items.id FROM `items` " +
		"JOIN permissions_generated ON permissions_generated.item_id = items.id " +
		"JOIN groups_ancestors_active ON groups_ancestors_active.ancestor_group_id = permissions_generated.group_id AND " +
		"groups_ancestors_active.child_group_id = ? " +
		"WHERE (items.id = ?) AND (items.duration IS NOT NULL) " +
		"GROUP BY items.id " +
		"HAVING (MAX(can_view_generated_value) >= ?) " +
		"LIMIT 1")).WillReturnRows(dbMock.NewRows([]string{"id"}).AddRow(123))
	var id int64
	err := NewDataStore(db).Items().ContestManagedByUser(123, &User{GroupID: 2}).
		PluckFirst("items.id", &id).Error()
	assert.NoError(t, err)
	assert.Equal(t, int64(123), id)
}

func TestItemStore_CanGrantViewContentOnAll_MustBeInTransaction(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()

	assert.PanicsWithValue(t, ErrNoTransaction, func() {
		_, _ = NewDataStore(db).Items().CanGrantViewContentOnAll(&User{GroupID: 14}, 20)
	})

	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestItemStore_CanGrantViewContentOnAll_HandlesDBErrors(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")
	grantViewNames = map[string]int{"content": 3}
	defer clearAllPermissionEnums()
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("").WillReturnError(expectedError)
	dbMock.ExpectRollback()

	assert.Equal(t, expectedError, NewDataStore(db).InTransaction(func(store *DataStore) error {
		result, err := store.Items().CanGrantViewContentOnAll(&User{GroupID: 14}, 20)
		assert.False(t, result)
		return err
	}))

	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestItemStore_AreAllVisible_MustBeInTransaction(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()

	assert.PanicsWithValue(t, ErrNoTransaction, func() {
		_, _ = NewDataStore(db).Items().AreAllVisible(&User{GroupID: 14}, 20)
	})

	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestItemStore_AreAllVisible_HandlesDBErrors(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("").WillReturnError(expectedError)
	dbMock.ExpectRollback()

	assert.Equal(t, expectedError, NewDataStore(db).InTransaction(func(store *DataStore) error {
		result, err := store.Items().AreAllVisible(&User{GroupID: 14}, 20)
		assert.False(t, result)
		return err
	}))

	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestItemStore_GetAccessDetailsForIDs_HandlesDBErrors(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()
	clearAllPermissionEnums()
	mockPermissionEnumQueries(dbMock)

	expectedError := errors.New("some error")
	dbMock.ExpectQuery("SELECT item_id").WillReturnError(expectedError)

	result, err := NewDataStore(db).Items().GetAccessDetailsForIDs(&User{GroupID: 14}, []int64{20})
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)

	assert.NoError(t, dbMock.ExpectationsWereMet())
}
