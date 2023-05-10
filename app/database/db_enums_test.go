package database

import (
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_getFromEnumUnderLock_WipesOutAllMapsOnError(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	ClearAllDBEnums()
	defer ClearAllDBEnums()

	expectedError := errors.New("some error")
	mock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT SUBSTRING(COLUMN_TYPE, 6, LENGTH(COLUMN_TYPE)-6) FROM `information_schema`.`COLUMNS`  "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1")+"$").
		WithArgs("permissions_granted", "can_view").
		WillReturnRows(mock.NewRows([]string{"value"}).
			AddRow("'none','info','content','content_with_descendants','solution'"))
	mock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT SUBSTRING(COLUMN_TYPE, 6, LENGTH(COLUMN_TYPE)-6) FROM `information_schema`.`COLUMNS`  "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1")+"$").
		WithArgs("permissions_granted", "can_grant_view").
		WillReturnError(expectedError)

	assert.PanicsWithValue(t, expectedError, func() {
		db.getFromEnumUnderLock(func() interface{} { return nil })
	})
	assert.Nil(t, enumValueIndex2Name)
	assert.Nil(t, enumValueName2Index)
}

func fakeDBEnums(enumName string, name2index map[string]int, index2name map[int]string) {
	enumIndex := enumName2Number[enumName]
	if len(enumValueIndex2Name) <= enumIndex {
		enumValueIndex2Name = append(enumValueIndex2Name, make([]map[int]string, enumIndex+1-len(enumValueIndex2Name))...)
		enumValueName2Index = append(enumValueName2Index, make([]map[string]int, enumIndex+1-len(enumValueName2Index))...)
	}
	enumValueName2Index[enumIndex] = name2index
	enumValueIndex2Name[enumIndex] = index2name
}
