package database

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

const someName = "some name"

func TestDB_inTransaction_NoErrors(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1 AS id").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectCommit()

	type resultStruct struct {
		ID int64
	}
	var result []resultStruct
	err := db.inTransaction(func(db *DB) error {
		return db.Raw("SELECT 1 AS id").Scan(&result).Error()
	})

	assert.NoError(t, err)
	assert.Equal(t, []resultStruct{{1}}, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_DBErrorOnBegin(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")

	mock.ExpectBegin().WillReturnError(expectedError)

	gotError := db.inTransaction(func(db *DB) error {
		var result []interface{}
		return db.Raw("SELECT 1").Scan(&result).Error()
	})
	assert.Equal(t, expectedError, gotError)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_DBError(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").WillReturnError(expectedError)
	mock.ExpectRollback()

	gotError := db.inTransaction(func(db *DB) error {
		var result []interface{}
		return db.Raw("SELECT 1").Scan(&result).Error()
	})
	assert.Contains(t, gotError.Error(), expectedError.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_Panic(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").WillReturnError(expectedError)
	mock.ExpectRollback()

	assert.PanicsWithValue(t, expectedError.(interface{}), func() {
		_ = db.inTransaction(func(db *DB) error {
			var result []interface{}
			db.Raw("SELECT 1").Scan(&result)
			panic(expectedError)
		})
	})
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_ErrorOnRollback(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").WillReturnError(expectedError)
	mock.ExpectRollback().WillReturnError(errors.New("rollback error"))

	t.Skip()
	assert.PanicsWithValue(t, expectedError, func() {
		_ = db.inTransaction(func(db *DB) error {
			var result []interface{}
			err := db.Raw("SELECT 1").Scan(&result).Error()
			assert.Equal(t, expectedError, err)
			return err
		})
	})
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_ErrorOnCommit(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("commit error")

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").
		WillReturnRows(mock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectCommit().WillReturnError(expectedError)

	assert.Equal(t, expectedError, db.inTransaction(func(db *DB) error {
		var result []interface{}
		return db.Raw("SELECT 1").Scan(&result).Error()
	}))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_RetriesOnDeadLockError(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	var duration time.Duration
	monkey.Patch(time.Sleep, func(d time.Duration) { duration += d })
	defer monkey.UnpatchAll()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").
		WillReturnError(&mysql.MySQLError{Number: 1213})
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").
		WillReturnRows(mock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectCommit()

	t.Skip()
	assert.NoError(t, db.inTransaction(func(db *DB) error {
		var result []interface{}
		return db.Raw("SELECT 1").Scan(&result).Error()
	}))
	assert.InEpsilon(t, transactionDelayBetweenRetries, duration, 0.05)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_RetriesOnDeadLockErrorAndPanicsOnRollbackError(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	var duration time.Duration
	monkey.Patch(time.Sleep, func(d time.Duration) { duration += d })
	defer monkey.UnpatchAll()

	expectedError := errors.New("rollback error")
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").
		WillReturnError(&mysql.MySQLError{Number: 1213})
	mock.ExpectRollback().WillReturnError(expectedError)

	t.Skip()
	assert.PanicsWithValue(t, expectedError, func() {
		_ = db.inTransaction(func(db *DB) error {
			var result []interface{}
			return db.Raw("SELECT 1").Scan(&result).Error()
		})
	})
	assert.Zero(t, duration)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_RetriesOnDeadLockPanic(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	var duration time.Duration
	monkey.Patch(time.Sleep, func(d time.Duration) { duration += d })
	defer monkey.UnpatchAll()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").
		WillReturnError(&mysql.MySQLError{Number: 1213})
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").
		WillReturnRows(mock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectCommit()

	assert.NoError(t, db.inTransaction(func(db *DB) error {
		var result []interface{}
		mustNotBeError(db.Raw("SELECT 1").Scan(&result).Error())
		return nil
	}))
	assert.InEpsilon(t, transactionDelayBetweenRetries, duration, 0.05)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_RetriesOnDeadLockPanicAndPanicsOnRollbackError(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	var duration time.Duration
	monkey.Patch(time.Sleep, func(d time.Duration) { duration += d })
	defer monkey.UnpatchAll()

	expectedError := errors.New("rollback error")
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").
		WillReturnError(&mysql.MySQLError{Number: 1213})
	mock.ExpectRollback().WillReturnError(expectedError)

	assert.PanicsWithValue(t, expectedError, func() {
		_ = db.inTransaction(func(db *DB) error {
			var result []interface{}
			mustNotBeError(db.Raw("SELECT 1").Scan(&result).Error())
			return nil
		})
	})
	assert.Zero(t, duration)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_RetriesAllowedUpToTheLimit_Panic(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	var duration time.Duration
	monkey.Patch(time.Sleep, func(d time.Duration) { duration += d })
	defer monkey.UnpatchAll()

	for i := 0; i < transactionRetriesLimit; i++ {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT 1").
			WillReturnError(&mysql.MySQLError{Number: 1213})
		mock.ExpectRollback()
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").
		WillReturnRows(mock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectCommit()

	assert.NoError(t, db.inTransaction(func(db *DB) error {
		var result []interface{}
		mustNotBeError(db.Raw("SELECT 1").Scan(&result).Error())
		return nil
	}))
	assert.InEpsilon(t, transactionRetriesLimit*transactionDelayBetweenRetries, duration, transactionRetriesLimit*0.05)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_RetriesAllowedUpToTheLimit_Error(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	var duration time.Duration
	monkey.Patch(time.Sleep, func(d time.Duration) { duration += d })
	defer monkey.UnpatchAll()

	for i := 0; i < transactionRetriesLimit; i++ {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT 1").
			WillReturnError(&mysql.MySQLError{Number: 1213})
		mock.ExpectRollback()
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1").
		WillReturnRows(mock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectCommit()

	assert.NoError(t, db.inTransaction(func(db *DB) error {
		var result []interface{}
		return db.Raw("SELECT 1").Scan(&result).Error()
	}))
	assert.InEpsilon(t, transactionRetriesLimit*transactionDelayBetweenRetries, duration, transactionRetriesLimit*0.05)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_RetriesAboveTheLimitAreDisallowed_Panic(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	var duration time.Duration
	monkey.Patch(time.Sleep, func(d time.Duration) { duration += d })
	defer monkey.UnpatchAll()

	for i := 0; i < transactionRetriesLimit+1; i++ {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT 1").
			WillReturnError(&mysql.MySQLError{Number: 1213})
		mock.ExpectRollback()
	}

	assert.Equal(t, errors.New("transaction retries limit exceeded"),
		db.inTransaction(func(db *DB) error {
			var result []interface{}
			mustNotBeError(db.Raw("SELECT 1").Scan(&result).Error())
			return nil
		}))
	assert.InEpsilon(t, transactionRetriesLimit*transactionDelayBetweenRetries, duration, transactionRetriesLimit*0.05)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_inTransaction_RetriesAboveTheLimitAreDisallowed_Error(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	var duration time.Duration
	monkey.Patch(time.Sleep, func(d time.Duration) { duration += d })
	defer monkey.UnpatchAll()

	for i := 0; i < transactionRetriesLimit+1; i++ {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT 1").
			WillReturnError(&mysql.MySQLError{Number: 1213})
		mock.ExpectRollback()
	}

	assert.Equal(t, errors.New("transaction retries limit exceeded"),
		db.inTransaction(func(db *DB) error {
			var result []interface{}
			return db.Raw("SELECT 1").Scan(&result).Error()
		}))
	assert.InEpsilon(t, transactionRetriesLimit*transactionDelayBetweenRetries, duration, transactionRetriesLimit*0.05)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_QueryConstructors(t *testing.T) {
	tests := []struct {
		name              string
		funcToPrepare     func(*DB) *DB
		funcToCall        func(*DB) (*DB, []*DB)
		expectedQuery     string
		expectedQueryArgs []driver.Value
	}{
		{
			name:          "Limit",
			funcToCall:    func(db *DB) (*DB, []*DB) { return db.Limit(1), nil },
			expectedQuery: "SELECT * FROM `myTable` LIMIT 1",
		},
		{
			name:              "Or",
			funcToPrepare:     func(db *DB) *DB { return db.Where("id = ?", 1) },
			funcToCall:        func(db *DB) (*DB, []*DB) { return db.Or("otherID = ?", 2), nil },
			expectedQuery:     "SELECT * FROM `myTable` WHERE id = ? OR otherID = ?",
			expectedQueryArgs: []driver.Value{1, 2},
		},
		{
			name:          "Order",
			funcToCall:    func(db *DB) (*DB, []*DB) { return db.Order("id"), nil },
			expectedQuery: "SELECT * FROM `myTable` ORDER BY id",
		},
		{
			name:          "Having",
			funcToCall:    func(db *DB) (*DB, []*DB) { return db.Having("id > 0"), nil },
			expectedQuery: "SELECT * FROM `myTable` HAVING id > 0",
		},
		{
			name:          "Raw",
			funcToCall:    func(db *DB) (*DB, []*DB) { return db.Raw("SELECT 1"), nil },
			expectedQuery: "SELECT 1",
		},
		{
			name: "Union",
			funcToCall: func(db *DB) (newDB *DB, dbs []*DB) {
				dbTwo := db.Table("otherTable")
				dbs = append(dbs, dbTwo)
				return db.Union(dbTwo.QueryExpr()), dbs
			},
			expectedQuery: "(SELECT * FROM `myTable`) UNION (SELECT * FROM `otherTable`)",
		},
		{
			name: "UnionAll",
			funcToCall: func(db *DB) (newDB *DB, dbs []*DB) {
				dbTwo := db.Table("otherTable")
				dbs = append(dbs, dbTwo)
				return db.UnionAll(dbTwo.QueryExpr()), dbs
			},
			expectedQuery: "(SELECT * FROM `myTable`) UNION ALL (SELECT * FROM `otherTable`)",
		},
		{
			name: "With",
			funcToCall: func(db *DB) (newDB *DB, dbs []*DB) {
				return db.With("myTable AS (?)", db.Table("users").QueryExpr()), dbs
			},
			expectedQuery: "WITH myTable AS (SELECT * FROM `users`) SELECT * FROM `myTable`",
		},
		{
			name: "With + Select",
			funcToCall: func(db *DB) (newDB *DB, dbs []*DB) {
				return db.With("myTable AS (?)", db.Table("users").QueryExpr()).
					With("myTable2 AS (?)", db.Table("sessions").QueryExpr()).
					Select("id"), dbs
			},
			expectedQuery: "WITH myTable AS (SELECT * FROM `users`), " +
				"myTable2 AS (SELECT * FROM `sessions`) " +
				"SELECT id FROM `myTable`",
		},
		{
			name: "With with empty query",
			funcToCall: func(db *DB) (newDB *DB, dbs []*DB) {
				return db.With("").Select("id"), dbs
			},
			expectedQuery: "SELECT id FROM `myTable`",
		},
		{
			name: "Select + AddSelect",
			funcToCall: func(db *DB) (newDB *DB, dbs []*DB) {
				db1 := db.Select("id")
				dbs = append(dbs, db1)
				return db1.AddSelect("name"), dbs
			},
			expectedQuery: "SELECT id,name FROM `myTable`",
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			db, mock := NewDBMock()
			defer func() { _ = db.Close() }()

			mock.ExpectQuery("^" + regexp.QuoteMeta(testCase.expectedQuery) + "$").
				WithArgs(testCase.expectedQueryArgs...).
				WillReturnRows(mock.NewRows([]string{"1"}).AddRow(1))

			db = db.Table("myTable")
			if testCase.funcToPrepare != nil {
				db = testCase.funcToPrepare(db)
			}
			resultDB, oldDBObjects := testCase.funcToCall(db)
			assert.NotEqual(t, resultDB, db)
			for _, oldDBObject := range oldDBObjects {
				assert.NotEqual(t, oldDBObject, db)
			}
			assert.NoError(t, resultDB.Error())

			var result []interface{}
			assert.NoError(t, resultDB.Scan(&result).Error())
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDB_Count(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM `myTable`")).
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(1))

	db = db.Table("myTable")

	var result int64
	countDB := db.Count(&result)

	assert.NotEqual(t, countDB, db)
	assert.NoError(t, countDB.Error())
	assert.Equal(t, int64(1), result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Count_DoesNothingIfDBContainsError(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")
	db = db.Table("myTable")
	_ = db.db.AddError(expectedError)

	var result int64
	countDB := db.Count(&result)

	assert.Equal(t, countDB, db)
	assert.Equal(t, expectedError, countDB.Error())
	assert.Zero(t, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Take(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable` WHERE id = 1 LIMIT 1")).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))

	db = db.Table("myTable")

	type resultType struct{ ID int }
	var result resultType
	takeDB := db.Take(&result, "id = 1")

	assert.NotEqual(t, takeDB, db)
	assert.NoError(t, takeDB.Error())
	assert.Equal(t, resultType{1}, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_HasRows(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM `myTable` WHERE id = 1 LIMIT 1")).
		WillReturnRows(mock.NewRows([]string{"1"}).AddRow(1))

	db = db.Table("myTable")

	found, err := db.Where("id = 1").HasRows()

	assert.NoError(t, err)
	assert.True(t, found)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_HasRows_NoTable(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 LIMIT 1")).
		WillReturnRows(mock.NewRows([]string{"1"}).AddRow("1"))

	found, err := db.Select("NOW()").HasRows()

	assert.NoError(t, err)
	assert.True(t, found)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_HasRows_NoRows(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM `myTable` WHERE id = 1 LIMIT 1")).
		WillReturnRows(mock.NewRows([]string{"1"}))

	db = db.Table("myTable")

	found, err := db.Where("id = 1").HasRows()

	assert.NoError(t, err)
	assert.False(t, found)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_HasRows_Error(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM `myTable` WHERE id = 1 LIMIT 1")).
		WillReturnError(expectedError)

	db = db.Table("myTable")

	found, err := db.Where("id = 1").HasRows()

	assert.Equal(t, expectedError, err)
	assert.False(t, found)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Pluck(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT `id` FROM `myTable`")).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))

	db = db.Table("myTable")

	var result []int64
	pluckDB := db.Pluck("id", &result)

	assert.NotEqual(t, pluckDB, db)
	assert.NoError(t, pluckDB.Error())
	assert.Equal(t, []int64{1}, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Pluck_DoesNothingIfErrorIsSet(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	db = db.Table("myTable")
	expectedError := errors.New("some error")
	db.db.Error = expectedError
	var result []int64
	pluckDB := db.Pluck("id", &result)
	assert.Equal(t, expectedError, pluckDB.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Pluck_WipesOldData(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT `id` FROM `myTable`")).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))

	db = db.Table("myTable")

	result := []int64{1, 2, 3}
	db.Pluck("id", &result)

	assert.Equal(t, []int64{1}, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Pluck_NonSlicePointer(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	db = db.Table("myTable")

	result := 1
	assert.PanicsWithValue(t, "values should be a pointer to a slice, not a pointer to int", func() {
		db.Pluck("id", &result)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Pluck_NonPointer(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	db = db.Table("myTable")

	result := 1
	assert.PanicsWithValue(t, "values should be a pointer to a slice, not int", func() {
		db.Pluck("id", result)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_PluckFirst(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT `id` FROM `myTable` LIMIT 1")).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))

	db = db.Table("myTable")

	var result int64
	pluckFirstDB := db.PluckFirst("id", &result)

	assert.NotEqual(t, pluckFirstDB, db)
	assert.NoError(t, pluckFirstDB.Error())
	assert.Equal(t, int64(1), result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_PluckFirst_NotFound(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT `id` FROM `myTable` LIMIT 1")).
		WillReturnRows(mock.NewRows([]string{"id"}))

	db = db.Table("myTable")

	var result int64
	pluckFirstDB := db.PluckFirst("id", &result)

	assert.NotEqual(t, pluckFirstDB, db)
	assert.Equal(t, gorm.ErrRecordNotFound, pluckFirstDB.Error())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_PluckFirst_Error(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT `id` FROM `myTable` LIMIT 1")).
		WillReturnError(expectedError)

	db = db.Table("myTable")

	var result int64
	pluckFirstDB := db.PluckFirst("id", &result)

	assert.NotEqual(t, pluckFirstDB, db)
	assert.Equal(t, expectedError, pluckFirstDB.Error())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Scan(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).
		WillReturnRows(mock.NewRows([]string{"id", "value"}).AddRow(int64(1), "value"))

	db = db.Table("myTable")

	type resultType struct {
		ID    int64
		Value string
	}
	var result []resultType
	scanDB := db.Scan(&result)

	assert.NotEqual(t, scanDB, db)
	assert.NoError(t, scanDB.Error())
	assert.Equal(t, []resultType{{ID: 1, Value: "value"}}, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Scan_WipesOldData(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).
		WillReturnRows(mock.NewRows([]string{"id", "value"}).AddRow(int64(1), "value"))

	db = db.Table("myTable")

	type resultType struct {
		ID    int64
		Value string
	}
	result := []resultType{{ID: 2, Value: "another value"}, {ID: 3, Value: "third value"}}
	scanDB := db.Scan(&result)

	assert.NoError(t, scanDB.Error())
	assert.Equal(t, []resultType{{ID: 1, Value: "value"}}, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Scan_NonSlicePointer(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).
		WillReturnRows(mock.NewRows([]string{"id", "value"}).AddRow(int64(1), "value"))

	db = db.Table("myTable")

	result := 1
	scanDB := db.Scan(&result)

	assert.EqualError(t, scanDB.Error(), "sql: expected 2 destination arguments in Scan, not 1")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Scan_NonPointer(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).
		WillReturnRows(mock.NewRows([]string{"id", "value"}).AddRow(int64(1), "value"))

	db = db.Table("myTable")

	type resultType struct {
		ID    int64
		Value string
	}
	var result []resultType
	assert.Panics(t, func() {
		db.Scan(result)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_RowsAffected(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec("^" + regexp.QuoteMeta("UPDATE `myTable` SET `myColumn`=?") + "$").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(-1, 123))
	mock.ExpectCommit()

	rowsAffected := db.Table("myTable").UpdateColumn("myColumn", 1).RowsAffected()

	assert.Equal(t, int64(123), rowsAffected)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Delete(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `myTable`") + `\s+` +
		regexp.QuoteMeta("WHERE id = 1")).
		WillReturnResult(sqlmock.NewResult(-1, 1))
	mock.ExpectCommit()

	db = db.Table("myTable")

	deleteDB := db.Delete("id = 1")

	assert.NotEqual(t, deleteDB, db)
	assert.NoError(t, deleteDB.Error())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Exec(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	query := "UPDATE users set id = ? WHERE id = ?"
	mock.ExpectExec("^"+regexp.QuoteMeta(query)+"$").
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(-1, 1))

	execDB := db.Exec(query, 1, 2)

	assert.NotEqual(t, execDB, db)
	assert.NoError(t, execDB.Error())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_insertMaps(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	dataRows := []map[string]interface{}{{"id": int64(1), "sField": "some value", "sNullField": nil}}

	expectedError := errors.New("some error")
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `myTable` (`id`, `sField`, `sNullField`) VALUES (?, ?, ?)")).
		WithArgs(int64(1), "some value", nil).
		WillReturnError(expectedError)

	assert.Equal(t, expectedError, db.insertMaps("myTable", dataRows))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_insertMaps_MultipleRows(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	dataRows := []map[string]interface{}{
		{"id": int64(1), "sField": "some value", "sNullField": "value"},
		{"id": int64(2), "sField": "another value", "sNullField": nil},
	}

	expectedError := errors.New("some error")
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `myTable` (`id`, `sField`, `sNullField`) VALUES (?, ?, ?), (?, ?, ?)")).
		WithArgs(int64(1), "some value", "value", int64(2), "another value", nil).
		WillReturnError(expectedError)

	assert.Equal(t, expectedError, db.insertMaps("myTable", dataRows))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_insertMaps_WithEmptyArray(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()
	var dataRows []map[string]interface{}
	assert.NoError(t, db.insertMaps("myTable", dataRows))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_insertOrUpdateMaps(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	dataRows := []map[string]interface{}{{"id": int64(1), "sField": "some value", "sNullField": nil}}

	expectedError := errors.New("some error")
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `myTable` (`id`, `sField`, `sNullField`) VALUES (?, ?, ?)"+
		" ON DUPLICATE KEY UPDATE `sField` = VALUES(`sField`), `sNullField` = VALUES(`sNullField`)")).
		WithArgs(int64(1), "some value", nil).
		WillReturnError(expectedError)

	assert.Equal(t, expectedError, db.insertOrUpdateMaps("myTable", dataRows, []string{"sField", "sNullField"}))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_insertOrUpdateMaps_WithEmptyArray(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()
	var dataRows []map[string]interface{}
	assert.NoError(t, db.insertOrUpdateMaps("myTable", dataRows, []string{"sField", "sNullField"}))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanIntoSlices(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, Field FROM `myTable`")).
		WillReturnRows(
			mock.NewRows([]string{"id", "Field"}).
				AddRow(1, "value").AddRow(2, "another value").AddRow([]byte("3"), nil))

	db = db.Table("myTable").Select("id, Field")

	ids := make([]int64, 0, 3)
	fields := make([]*string, 0, 3)

	dbScan := db.ScanIntoSlices(&ids, &fields)
	assert.Equal(t, dbScan, db)
	assert.NoError(t, dbScan.Error())

	assert.Equal(t, []int64{1, 2, 3}, ids)
	assert.Equal(t, []*string{ptrString("value"), ptrString("another value"), nil}, fields)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanIntoSlices_DoesNothingIfErrorIsSet(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")
	_ = db.db.AddError(expectedError)
	var result []int64
	dbScan := db.ScanIntoSlices(&result)
	assert.Equal(t, dbScan, db)
	assert.Equal(t, expectedError, dbScan.Error())

	assert.Equal(t, []int64(nil), result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanIntoSlices_WipesOldData(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).
		WillReturnRows(
			mock.NewRows([]string{"id", "Field"}).
				AddRow(1, "value").AddRow(2, "another value").AddRow([]byte("3"), nil))

	db = db.Table("myTable")

	ids := []int64{10, 20, 30}
	fields := []*string{ptrString("old value1"), ptrString("old value2"), ptrString("old value3")}

	db.ScanIntoSlices(&ids, &fields)
	assert.Equal(t, []int64{1, 2, 3}, ids)
	assert.Equal(t, []*string{ptrString("value"), ptrString("another value"), nil}, fields)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanIntoSlices_RowsError(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).
		WillReturnError(expectedError)
	db = db.Table("myTable")

	var result []int64
	dbScan := db.ScanIntoSlices(&result)
	assert.Equal(t, dbScan, db)
	assert.Equal(t, expectedError, dbScan.Error())

	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanIntoSlices_ErrorOnScan(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.Rows{}), "Scan", func(*sql.Rows, ...interface{}) error { return expectedError })
	defer monkey.UnpatchAll()
	db = db.Table("myTable")

	var result []int64
	dbScan := db.ScanIntoSlices(&result)
	assert.Equal(t, dbScan, db)
	assert.Equal(t, expectedError, dbScan.Error())

	assert.Equal(t, []int64(nil), result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanIntoSliceOfMaps(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).
		WillReturnRows(
			mock.NewRows([]string{"id", "Field"}).
				AddRow(1, "value").AddRow(2, "another value").AddRow([]byte("3"), nil))

	db = db.Table("myTable")

	var result []map[string]interface{}
	dbScan := db.ScanIntoSliceOfMaps(&result)
	assert.Equal(t, dbScan, db)
	assert.NoError(t, dbScan.Error())

	assert.Equal(t, []map[string]interface{}{
		{"id": int64(1), "Field": "value"},
		{"id": int64(2), "Field": "another value"},
		{"id": "3", "Field": nil},
	}, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanIntoSliceOfMaps_DoesNothingIfErrorIsSet(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")
	_ = db.db.AddError(expectedError)
	var result []map[string]interface{}
	dbScan := db.ScanIntoSliceOfMaps(&result)
	assert.Equal(t, dbScan, db)
	assert.Equal(t, expectedError, dbScan.Error())

	assert.Equal(t, []map[string]interface{}(nil), result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanIntoSliceOfMaps_WipesOldData(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).
		WillReturnRows(
			mock.NewRows([]string{"id", "Field"}).
				AddRow(1, "value").AddRow(2, "another value").AddRow([]byte("3"), nil))

	db = db.Table("myTable")

	result := []map[string]interface{}{
		{"column": "value"},
	}
	db.ScanIntoSliceOfMaps(&result)
	assert.Equal(t, []map[string]interface{}{
		{"id": int64(1), "Field": "value"},
		{"id": int64(2), "Field": "another value"},
		{"id": "3", "Field": nil},
	}, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanIntoSliceOfMaps_RowsError(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).
		WillReturnError(expectedError)
	db = db.Table("myTable")

	var result []map[string]interface{}
	dbScan := db.ScanIntoSliceOfMaps(&result)
	assert.Equal(t, dbScan, db)
	assert.Equal(t, expectedError, dbScan.Error())

	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanIntoSliceOfMaps_ErrorOnGettingColumns(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.Rows{}), "Columns", func(*sql.Rows) ([]string, error) { return nil, expectedError })
	defer monkey.UnpatchAll()
	db = db.Table("myTable")

	var result []map[string]interface{}
	dbScan := db.ScanIntoSliceOfMaps(&result)
	assert.Equal(t, dbScan, db)
	assert.Equal(t, expectedError, dbScan.Error())

	assert.Equal(t, []map[string]interface{}(nil), result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanIntoSliceOfMaps_ErrorOnScan(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.Rows{}), "Scan", func(*sql.Rows, ...interface{}) error { return expectedError })
	defer monkey.UnpatchAll()
	db = db.Table("myTable")

	var result []map[string]interface{}
	dbScan := db.ScanIntoSliceOfMaps(&result)
	assert.Equal(t, dbScan, db)
	assert.Equal(t, expectedError, dbScan.Error())

	assert.Equal(t, []map[string]interface{}(nil), result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ScanAndHandleMaps_FailsIfHandlerReturnsError(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `myTable`")).WillReturnRows(mock.NewRows([]string{"id"}).
		AddRow(1).AddRow(2))
	db = db.Table("myTable")

	dbScan := db.ScanAndHandleMaps(func(map[string]interface{}) error {
		return expectedError
	})
	assert.Equal(t, dbScan, db)
	assert.Equal(t, expectedError, dbScan.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Updates(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `myTable` SET `id`=?,`name`=?")).
		WithArgs(1, someName).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	db = db.Table("myTable")
	updateDB := db.Updates(map[string]interface{}{"id": 1, "name": someName})
	assert.NotEqual(t, updateDB, db)
	assert.NoError(t, updateDB.Error())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_UpdateColumn(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `myTable` SET `name`=?")).
		WithArgs(someName).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	db = db.Table("myTable")
	updateDB := db.UpdateColumn("name", someName)
	assert.NotEqual(t, updateDB, db)
	assert.NoError(t, updateDB.Error())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOpenRawDBConnection(t *testing.T) {
	db, err := OpenRawDBConnection("/db")
	assert.NoError(t, err)
	assert.Contains(t, sql.Drivers(), "instrumented-mysql")
	assertRawDBIsOK(t, db)
}

type dummyConn struct{}

func (c *dummyConn) Begin() (driver.Tx, error)           { return nil, nil }
func (c *dummyConn) Close() error                        { return nil }
func (c *dummyConn) Prepare(string) (driver.Stmt, error) { return nil, nil }

var _ driver.Conn = &dummyConn{}

func TestOpen_DSN(t *testing.T) {
	dbMock, _, err := sqlmock.NewWithDSN("/mysql")
	assert.NoError(t, err)
	defer func() { _ = dbMock.Close() }()
	connector, _ := mysql.NewConnector(mysql.NewConfig())
	reflConnector := reflect.TypeOf(connector)
	reflMethod, _ := reflConnector.MethodByName("Connect")
	patch := monkey.PatchInstanceMethod(reflConnector, "Connect",
		reflect.MakeFunc(reflMethod.Type, func(args []reflect.Value) []reflect.Value {
			return []reflect.Value{reflect.ValueOf(&dummyConn{}), reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
		}).Interface())
	defer patch.Unpatch()
	db, err := Open("/mysql")
	assert.Nil(t, err)
	assert.NotNil(t, db)
	assert.NotNil(t, db.db)
	assert.Contains(t, sql.Drivers(), "instrumented-mysql")
	rawDB, err := db.db.DB()
	assert.NoError(t, err)
	assertRawDBIsOK(t, rawDB)
}

func TestOpen_DSN_Error(t *testing.T) {
	expectedError := errors.New("some error")
	patch := monkey.Patch(gorm.Open, func(gorm.Dialector, ...gorm.Option) (*gorm.DB, error) {
		return nil, expectedError
	})
	defer patch.Unpatch()
	db, err := Open("/mysql")
	assert.Nil(t, db)
	assert.Equal(t, err, expectedError)
}

func TestOpen_WrongSourceType(t *testing.T) {
	db, err := Open(1234)
	assert.Equal(t, errors.New("unknown database source type: int (1234)"), err)
	assert.Nil(t, db)
}

func TestOpen_OpenRawDBConnectionError(t *testing.T) {
	expectedError := errors.New("some error")
	monkey.Patch(OpenRawDBConnection, func(string) (*sql.DB, error) { return &sql.DB{}, expectedError })
	defer monkey.UnpatchAll()

	db, err := Open("mydsn")
	assert.Equal(t, expectedError, err)
	assert.Nil(t, db)
}

func assertRawDBIsOK(t *testing.T, rawDB *sql.DB) {
	assert.Equal(t, "*instrumentedsql.WrappedDriver", fmt.Sprintf("%T", rawDB.Driver()))
	assert.Contains(t, fmt.Sprintf("%#v", rawDB), "parent:(*mysql.connector)")
}

func TestDB_isInTransaction_ReturnsTrue(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectCommit()

	assert.NoError(t, db.inTransaction(func(db *DB) error {
		assert.True(t, db.isInTransaction())
		return nil
	}))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_isInTransaction_ReturnsFalse(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	assert.False(t, db.isInTransaction())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_mustBeInTransaction_DoesNothingInTransaction(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectCommit()

	assert.NoError(t, db.inTransaction(func(db *DB) error {
		assert.NotPanics(t, func() { db.mustBeInTransaction() })
		return nil
	}))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_mustBeInTransaction_PanicWhenNotInTransaction(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	assert.PanicsWithValue(t, ErrNoTransaction, func() { db.mustBeInTransaction() })
	assert.NoError(t, mock.ExpectationsWereMet())
}

func Test_recoverPanics_RecoversError(t *testing.T) {
	expectedError := errors.New("some error")
	err := func() (err error) {
		defer recoverPanics(&err)
		panic(expectedError)
	}()
	assert.Equal(t, expectedError, err)
}

func Test_recoverPanics_PanicsOnRuntimeError(t *testing.T) {
	didPanic, panicValue := func() (didPanic bool, panicValue interface{}) {
		defer func() {
			if p := recover(); p != nil {
				didPanic = true
				panicValue = p
			}
		}()

		_ = func() (err error) {
			defer recoverPanics(&err)
			var a []int
			a[0]++ // nolint:govet // runtime error
			return nil
		}()

		return false, nil
	}()

	assert.True(t, didPanic)
	assert.Implements(t, (*runtime.Error)(nil), panicValue)
	assert.Equal(t, "runtime error: index out of range [0] with length 0", panicValue.(error).Error())
}

func TestDB_withNamedLock_ReturnsErrLockWaitTimeoutExceededWhenGetLockTimeouts(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()
	lockName := "lock name"
	timeout := 1234 * time.Millisecond
	expectedTimeout := int(timeout.Round(time.Second).Seconds())

	dbMock.ExpectQuery("^"+regexp.QuoteMeta("SELECT GET_LOCK(?, ?)")+"$").
		WithArgs(lockName, expectedTimeout).
		WillReturnRows(sqlmock.NewRows([]string{"GET_LOCK('" + lockName + "', " + strconv.Itoa(expectedTimeout) + ")"}).AddRow(int64(0)))

	err := db.withNamedLock(lockName, timeout, func(*DB) error {
		return nil
	})
	assert.Equal(t, ErrLockWaitTimeoutExceeded, err)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestDB_withNamedLock_ReturnsErrorWhenDBFails(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()
	lockName := someName
	timeout := 1234 * time.Millisecond
	expectedTimeout := int(timeout.Round(time.Second).Seconds())
	expectedError := errors.New("some error")

	dbMock.ExpectQuery("^"+regexp.QuoteMeta("SELECT GET_LOCK(?, ?)")+"$").
		WithArgs(lockName, expectedTimeout).
		WillReturnError(expectedError)

	err := db.withNamedLock(lockName, timeout, func(*DB) error {
		return nil
	})
	assert.Equal(t, expectedError, err)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestDB_withNamedLock_ReleasesLockOnSuccess(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()
	lockName := someName
	timeout := 1234 * time.Millisecond
	expectedTimeout := int(timeout.Round(time.Second).Seconds())

	dbMock.ExpectQuery("^"+regexp.QuoteMeta("SELECT GET_LOCK(?, ?)")+"$").
		WithArgs(lockName, expectedTimeout).
		WillReturnRows(sqlmock.NewRows([]string{"GET_LOCK(?, ?)"}).AddRow(int64(1)))
	dbMock.ExpectExec("^" + regexp.QuoteMeta("SELECT RELEASE_LOCK(?)") + "$").
		WithArgs(lockName).WillReturnResult(sqlmock.NewResult(-1, -1))

	err := db.withNamedLock(lockName, timeout, func(*DB) error {
		return nil
	})
	assert.NoError(t, err)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestDB_withNamedLock_ReleasesLockOnError(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()
	lockName := someName
	timeout := 1234 * time.Millisecond
	expectedTimeout := int(timeout.Round(time.Second).Seconds())
	expectedError := errors.New("some error")

	dbMock.ExpectQuery("^"+regexp.QuoteMeta("SELECT GET_LOCK(?, ?)")+"$").
		WithArgs(lockName, expectedTimeout).
		WillReturnRows(sqlmock.NewRows([]string{"GET_LOCK(?, ?)"}).AddRow(int64(1)))
	dbMock.ExpectExec("^" + regexp.QuoteMeta("SELECT RELEASE_LOCK(?)") + "$").
		WithArgs(lockName)

	err := db.withNamedLock(lockName, timeout, func(*DB) error {
		return expectedError
	})
	assert.Equal(t, expectedError, err)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestDB_withNamedLock_ReleasesLockOnPanic(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()
	lockName := someName
	timeout := 1234 * time.Millisecond
	expectedTimeout := int(timeout.Round(time.Second).Seconds())
	expectedError := errors.New("some error")

	dbMock.ExpectQuery("^"+regexp.QuoteMeta("SELECT GET_LOCK(?, ?)")+"$").
		WithArgs(lockName, expectedTimeout).
		WillReturnRows(sqlmock.NewRows([]string{"GET_LOCK(?, ?)"}).AddRow(int64(1)))
	dbMock.ExpectExec("^" + regexp.QuoteMeta("SELECT RELEASE_LOCK(?)") + "$").
		WithArgs(lockName).WillReturnResult(sqlmock.NewResult(-1, -1))

	assert.PanicsWithValue(t, expectedError, func() {
		_ = db.withNamedLock(lockName, timeout, func(*DB) error {
			panic(expectedError)
		})
	})
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestDB_withNamedLock_ReturnsReleaseError(t *testing.T) {
	db, dbMock := NewDBMock()
	defer func() { _ = db.Close() }()
	lockName := someName
	timeout := 1234 * time.Millisecond
	expectedTimeout := int(timeout.Round(time.Second).Seconds())
	expectedError := errors.New("some error")

	dbMock.ExpectQuery("^"+regexp.QuoteMeta("SELECT GET_LOCK(?, ?)")+"$").
		WithArgs(lockName, expectedTimeout).
		WillReturnRows(sqlmock.NewRows([]string{"GET_LOCK(?, ?)"}).AddRow(int64(1)))
	dbMock.ExpectExec("^" + regexp.QuoteMeta("SELECT RELEASE_LOCK(?)") + "$").
		WithArgs(lockName).WillReturnError(expectedError)

	err := db.withNamedLock(lockName, timeout, func(*DB) error {
		return nil
	})

	assert.Equal(t, expectedError, err)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestDB_WithWriteLock(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM `myTable` FOR UPDATE").
		WillReturnRows(mock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectCommit()

	db = db.Table("myTable")
	err := db.inTransaction(func(db *DB) error {
		newDB := db.WithWriteLock()
		assert.NotEqual(t, newDB, db)
		assert.NoError(t, newDB.Error())
		var result []interface{}
		assert.NoError(t, newDB.Scan(&result).Error())
		return nil
	})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_WithWriteLock_PanicsWhenNotInTransaction(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	assert.PanicsWithValue(t, ErrNoTransaction, func() { db.WithWriteLock() })
	assert.NoError(t, mock.ExpectationsWereMet())
}

var retryOnDuplicatePrimaryKeyErrorExpectedQueryRegexp = "^" + regexp.QuoteMeta("INSERT INTO users (id) VALUES (?)") + "$"

func TestDB_retryOnDuplicatePrimaryKeyError(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	for i := 1; i < keyTriesCount; i++ {
		mock.ExpectExec(retryOnDuplicatePrimaryKeyErrorExpectedQueryRegexp).WithArgs(i).
			WillReturnError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry '" + strconv.Itoa(i) + "' for key 'PRIMARY'"})
	}
	mock.ExpectExec(retryOnDuplicatePrimaryKeyErrorExpectedQueryRegexp).WithArgs(keyTriesCount).
		WillReturnResult(sqlmock.NewResult(keyTriesCount, 1))

	retryCount := 0
	err := db.retryOnDuplicatePrimaryKeyError(func(db *DB) error {
		retryCount++
		return db.Exec("INSERT INTO users (id) VALUES (?)", retryCount).Error()
	})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_retryOnDuplicatePrimaryKeyError_ErrorsWhenLimitExceeded(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	for i := 1; i < keyTriesCount+1; i++ {
		mock.ExpectExec(retryOnDuplicatePrimaryKeyErrorExpectedQueryRegexp).WithArgs(i).
			WillReturnError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry '" + strconv.Itoa(i) + "' for key 'PRIMARY'"})
	}

	retryCount := 0
	err := db.retryOnDuplicatePrimaryKeyError(func(db *DB) error {
		retryCount++
		return db.Exec("INSERT INTO users (id) VALUES (?)", retryCount).Error()
	})
	assert.Equal(t, errors.New("cannot generate a new id"), err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_retryOnDuplicatePrimaryKeyError_ReturnsOtherErrors(t *testing.T) {
	tests := []struct {
		name          string
		expectedError *mysql.MySQLError
	}{
		{
			name:          "non-primary key duplicate",
			expectedError: &mysql.MySQLError{Number: 1062, Message: "Duplicate entry '' for key 'name'"},
		},
		{
			name:          "error code != 1062",
			expectedError: &mysql.MySQLError{Number: 1063, Message: "Duplicate entry '1' for key 'PRIMARY'"},
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			db, mock := NewDBMock()
			defer func() { _ = db.Close() }()

			mock.ExpectExec(retryOnDuplicatePrimaryKeyErrorExpectedQueryRegexp).WithArgs(1).
				WillReturnError(testCase.expectedError)

			err := db.retryOnDuplicatePrimaryKeyError(func(db *DB) error {
				return db.Exec("INSERT INTO users (id) VALUES (?)", 1).Error()
			})
			assert.Equal(t, testCase.expectedError, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDefault(t *testing.T) {
	assert.Equal(t, gorm.Expr("DEFAULT"), Default())
}

func Test_EscapeLikeString(t *testing.T) {
	type args struct {
		s               string
		escapeCharacter byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "all characters",
			args: args{s: "|some _string_ 100%|", escapeCharacter: '|'},
			want: "||some |_string|_ 100|%||",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := EscapeLikeString(tt.args.s, tt.args.escapeCharacter)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDB_InsertIgnoreMaps_WithEmptyArray(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()
	var dataRows []map[string]interface{}
	assert.NoError(t, db.InsertIgnoreMaps("myTable", dataRows))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_InsertIgnoreMaps(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	dataRows := []map[string]interface{}{
		{"a": int64(1), "b": "value"},
		{"c": int64(2), "d": "another value"},
	}

	expectedError := errors.New("some error")
	mock.ExpectExec(regexp.QuoteMeta("INSERT IGNORE INTO `myTable` (`a`, `b`) VALUES (?, ?), (?, ?)")).
		WithArgs(int64(1), "value", nil, nil).
		WillReturnError(expectedError)

	assert.Equal(t, expectedError, db.InsertIgnoreMaps("myTable", dataRows))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func ptrString(s string) *string { return &s }
