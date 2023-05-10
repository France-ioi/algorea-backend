//go:build !prod

package database

import (
	"fmt"
	"os"
	"regexp"

	"github.com/DATA-DOG/go-sqlmock"
)

// NewDBMock generate a DB mock the database engine.
func NewDBMock() (*DB, sqlmock.Sqlmock) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		fmt.Println("Unable to create the mock db: ", err)
		os.Exit(1)
	}

	db, err := Open(dbMock)
	if err != nil {
		fmt.Println("Unable to create the gorm connection to the mock: ", err)
		os.Exit(1)
	}

	return db, mock
}

// MockDBEnumQueries stubs all the db queries for loading permission enums.
func MockDBEnumQueries(sqlMock sqlmock.Sqlmock) {
	sqlMock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT SUBSTRING(COLUMN_TYPE, 6, LENGTH(COLUMN_TYPE)-6) FROM `information_schema`.`COLUMNS`  "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1")+"$").
		WithArgs("permissions_granted", "can_view").
		WillReturnRows(sqlMock.NewRows([]string{"value"}).
			AddRow("'none','info','content','content_with_descendants','solution'"))
	sqlMock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT SUBSTRING(COLUMN_TYPE, 6, LENGTH(COLUMN_TYPE)-6) FROM `information_schema`.`COLUMNS`  "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1")+"$").
		WithArgs("permissions_granted", "can_grant_view").
		WillReturnRows(sqlMock.NewRows([]string{"value"}).
			AddRow("'none','enter','content','content_with_descendants','solution','solution_with_grant'"))
	sqlMock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT SUBSTRING(COLUMN_TYPE, 6, LENGTH(COLUMN_TYPE)-6) FROM `information_schema`.`COLUMNS`  "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1")+"$").
		WithArgs("permissions_granted", "can_watch").
		WillReturnRows(sqlMock.NewRows([]string{"value"}).
			AddRow("'none','result','answer','answer_with_grant'"))
	sqlMock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT SUBSTRING(COLUMN_TYPE, 6, LENGTH(COLUMN_TYPE)-6) FROM `information_schema`.`COLUMNS`  "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1")+"$").
		WithArgs("permissions_granted", "can_edit").
		WillReturnRows(sqlMock.NewRows([]string{"value"}).
			AddRow("'none','children','all','all_with_grant'"))
	sqlMock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT SUBSTRING(COLUMN_TYPE, 6, LENGTH(COLUMN_TYPE)-6) FROM `information_schema`.`COLUMNS`  "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1")+"$").
		WithArgs("group_managers", "can_manage").
		WillReturnRows(sqlMock.NewRows([]string{"value"}).
			AddRow("'none','memberships','memberships_and_group'"))
	sqlMock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT SUBSTRING(COLUMN_TYPE, 6, LENGTH(COLUMN_TYPE)-6) FROM `information_schema`.`COLUMNS`  "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1")+"$").
		WithArgs("items_items", "content_view_propagation").
		WillReturnRows(sqlMock.NewRows([]string{"value"}).
			AddRow("'none','as_info','as_content'"))
	sqlMock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT SUBSTRING(COLUMN_TYPE, 6, LENGTH(COLUMN_TYPE)-6) FROM `information_schema`.`COLUMNS`  "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1")+"$").
		WithArgs("items_items", "upper_view_levels_propagation").
		WillReturnRows(sqlMock.NewRows([]string{"value"}).
			AddRow("'use_content_view_propagation','as_content_with_descendants','as_is'"))
}

// ClearAllDBEnums clears all cached permission enums.
func ClearAllDBEnums() {
	enumValueIndex2Name = nil
	enumValueName2Index = nil
}
