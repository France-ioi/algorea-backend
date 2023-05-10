package database

import (
	"errors"
	"reflect"
	"regexp"
	"sync"
	"testing"

	"bou.ke/monkey"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestItemItemStore_ChildrenOf(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	const parentItemID = 123
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `items_items` WHERE items_items.parent_item_id=?")).
		WithArgs(parentItemID).
		WillReturnRows(mock.NewRows([]string{"id"}))

	store := NewDataStore(db).ItemItems()
	newStore := store.ChildrenOf(parentItemID)
	assert.NotEqual(t, store, newStore)
	assert.Equal(t, "items_items", newStore.DataStore.tableName)

	var result []interface{}
	err := newStore.Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemItemStore_After_MustBeInTransaction(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()

	assert.PanicsWithValue(t, ErrNoTransaction, func() {
		_ = NewDataStore(db).ItemItems().After()
	})

	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestItemItemStore_After_HandlesErrorOfCreateNewAncestors(t *testing.T) {
	expectedError := errors.New("some error")

	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()
	dbMock.ExpectBegin()
	dbMock.ExpectExec("^INSERT INTO  items_propagate").WillReturnError(expectedError)
	dbMock.ExpectRollback()

	assert.Equal(t, expectedError, db.inTransaction(func(trDB *DB) error {
		return NewDataStore(trDB).ItemItems().After()
	}))

	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestItemItemStore_After_HandlesErrorOfComputeAllAccess(t *testing.T) {
	expectedError := errors.New("some error")

	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()
	dbMock.ExpectBegin()
	dbMock.ExpectExec("^INSERT INTO  items_propagate").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectPrepare("UPDATE items_propagate")
	dbMock.ExpectPrepare("DELETE items_ancestors")
	dbMock.ExpectPrepare("INSERT IGNORE INTO items_ancestors")
	dbMock.ExpectPrepare("INSERT IGNORE INTO items_ancestors")
	dbMock.ExpectPrepare("UPDATE items_propagate")
	dbMock.ExpectExec("UPDATE items_propagate").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectExec("DELETE items_ancestors").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectExec("INSERT IGNORE INTO items_ancestors").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectExec("INSERT IGNORE INTO items_ancestors").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectExec("UPDATE items_propagate").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectPrepare("^INSERT INTO permissions_propagate").WillReturnError(expectedError)
	dbMock.ExpectRollback()

	assert.Equal(t, expectedError, db.inTransaction(func(trDB *DB) error {
		return NewDataStore(trDB).ItemItems().After()
	}))

	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestItemItemStore_After_HandlesErrorOfPropagate(t *testing.T) {
	expectedError := errors.New("some error")

	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()
	dbMock.ExpectBegin()
	dbMock.ExpectExec("^INSERT INTO  items_propagate").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectPrepare("UPDATE items_propagate")
	dbMock.ExpectPrepare("DELETE items_ancestors")
	dbMock.ExpectPrepare("INSERT IGNORE INTO items_ancestors")
	dbMock.ExpectPrepare("INSERT IGNORE INTO items_ancestors")
	dbMock.ExpectPrepare("UPDATE items_propagate")
	dbMock.ExpectExec("UPDATE items_propagate").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectExec("DELETE items_ancestors").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectExec("INSERT IGNORE INTO items_ancestors").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectExec("INSERT IGNORE INTO items_ancestors").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectExec("UPDATE items_propagate").WillReturnResult(sqlmock.NewResult(0, 0))
	dbMock.ExpectPrepare("^INSERT INTO permissions_propagate")
	dbMock.ExpectPrepare("^DELETE FROM permissions_propagate")
	dbMock.ExpectPrepare("^INSERT INTO permissions_generated")
	dbMock.ExpectPrepare("^UPDATE permissions_propagate")
	dbMock.ExpectExec("^INSERT INTO permissions_propagate").WillReturnError(expectedError)
	dbMock.ExpectRollback()

	assert.Equal(t, expectedError, db.inTransaction(func(trDB *DB) error {
		return NewDataStore(trDB).ItemItems().After()
	}))

	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestItemItemStore_PropagationEnums(t *testing.T) {
	tests := []struct {
		name             string
		columnName       string
		name2index       map[string]int
		index2name       map[int]string
		name2indexMethod string
		index2nameMethod string
	}{
		{
			name:             "UpperViewLevelsPropagationEnum",
			columnName:       "items_items.upper_view_levels_propagation",
			name2index:       map[string]int{"use_content_view_propagation": 1, "as_content_with_descendants": 2, "as_is": 3},
			index2name:       map[int]string{1: "use_content_view_propagation", 2: "as_content_with_descendants", 3: "as_is"},
			name2indexMethod: "UpperViewLevelsPropagationIndexByName",
			index2nameMethod: "UpperViewLevelsPropagationNameByIndex",
		},
		{
			name:             "ContentViewPropagationEnum",
			columnName:       "items_items.content_view_propagation",
			name2index:       map[string]int{"none": 1, "as_info": 2, "as_content": 3},
			index2name:       map[int]string{1: "none", 2: "as_info", 3: "as_content"},
			name2indexMethod: "ContentViewPropagationIndexByName",
			index2nameMethod: "ContentViewPropagationNameByIndex",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			db, _ := NewDBMock()
			defer func() { _ = db.Close() }()
			itemItemStore := NewDataStore(db).ItemItems()

			ClearAllDBEnums()
			var oldLock *monkey.PatchGuard
			oldLock = monkey.PatchInstanceMethod(reflect.TypeOf(&sync.RWMutex{}), "Lock", func(mutex *sync.RWMutex) {
				oldLock.Unpatch()
				mutex.Lock()
				oldLock.Restore()
				fakeDBEnums(test.columnName, test.name2index, test.index2name)
			})
			defer monkey.UnpatchAll()
			defer ClearAllDBEnums()

			name2indexMethod := reflect.ValueOf(itemItemStore).MethodByName(test.name2indexMethod)
			for name, index := range test.name2index {
				result := name2indexMethod.Call([]reflect.Value{reflect.ValueOf(name)})
				assert.Len(t, result, 1)
				assert.Equal(t, index, result[0].Interface())
			}
			assert.Panics(t, func() { name2indexMethod.Call([]reflect.Value{reflect.ValueOf("unknown")}) })
			ClearAllDBEnums()
			index2NameMethod := reflect.ValueOf(itemItemStore).MethodByName(test.index2nameMethod)
			for name, index := range test.name2index {
				result := index2NameMethod.Call([]reflect.Value{reflect.ValueOf(index)})
				assert.Len(t, result, 1)
				assert.Equal(t, name, result[0].Interface())
			}
			assert.Panics(t, func() { index2NameMethod.Call([]reflect.Value{reflect.ValueOf(0)}) })
		})
	}
}

func TestItemItemStore_UpperViewLevelsPropagationIndexByName_Load(t *testing.T) {
	db, sqlMock := NewDBMock()
	defer func() { _ = db.Close() }()

	MockDBEnumQueries(sqlMock)
	itemItemStore := NewDataStore(db).ItemItems()

	ClearAllDBEnums()
	defer ClearAllDBEnums()

	assert.Equal(t, 3, itemItemStore.UpperViewLevelsPropagationIndexByName("as_is"))
}

func TestItemItemStore_ContentViewPropagationIndexByName_Load(t *testing.T) {
	db, sqlMock := NewDBMock()
	defer func() { _ = db.Close() }()

	MockDBEnumQueries(sqlMock)
	itemItemStore := NewDataStore(db).ItemItems()

	ClearAllDBEnums()
	defer ClearAllDBEnums()

	assert.Equal(t, 3, itemItemStore.ContentViewPropagationIndexByName("as_content"))
}

func TestItemItemStore_WithItemsRelationsLock(t *testing.T) {
	assertNamedLockMethod(t, "items_items", int(itemsRelationsLockTimeout.Seconds()), "items_items",
		func(store *DataStore) func(func(store *DataStore) error) error {
			return func(txFunc func(store *DataStore) error) error {
				return store.ItemItems().WithItemsRelationsLock(txFunc)
			}
		})
}
