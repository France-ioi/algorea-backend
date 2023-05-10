package database

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLanguageStore_ByTag(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	const tag = "sl"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `languages` WHERE languages.tag = ?")).
		WithArgs(tag).
		WillReturnRows(mock.NewRows([]string{"tag"}))

	var result []interface{}
	err := NewDataStore(db).Languages().ByTag(tag).Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
