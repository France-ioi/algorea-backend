package database

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupStore_OwnedBy(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mockUser := NewMockUser(1, &UserData{SelfGroupID: 2, OwnedGroupID: 3, DefaultLanguageID: 4})

	mock.ExpectQuery(regexp.QuoteMeta("SELECT `groups`.* FROM `groups` " +
		"JOIN groups_ancestors ON groups_ancestors.idGroupChild = groups.ID " +
		"WHERE (groups_ancestors.idGroupAncestor=?)")).
		WithArgs(3).
		WillReturnRows(mock.NewRows([]string{"ID"}))

	var result []interface{}
	err := NewDataStore(db).Groups().OwnedBy(mockUser).Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupStore_OwnedBy_HandlesError(t *testing.T) {
	testMethodHandlesUserNotFoundError(t, func(db *DB, user *User) []interface{} {
		var result []interface{}
		err := NewDataStore(db).Groups().OwnedBy(user).Scan(&result).Error()
		return []interface{}{err}
	}, []interface{}{ErrUserNotFound})
}
