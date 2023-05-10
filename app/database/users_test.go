package database

import (
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_WhereUsersAreDescendantsOfGroup(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	const groupID = 123
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `users` " +
			"JOIN groups_ancestors_active " +
			"ON groups_ancestors_active.child_group_id=users.group_id " +
			"WHERE groups_ancestors_active.ancestor_group_id = ?")).
		WithArgs(groupID).
		WillReturnRows(mock.NewRows([]string{"id"}))

	var result []interface{}
	err := db.Table("users").WhereUsersAreDescendantsOfGroup(groupID).Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataStore_CheckIfTeamParticipationsConflictWithExistingUserMemberships_WithLock(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("^SELECT .+ FOR UPDATE\\b.+ FOR UPDATE$").WillReturnRows(mock.NewRows([]string{}))
	mock.ExpectCommit()
	assert.NoError(t, NewDataStore(db).InTransaction(func(store *DataStore) error {
		_, err := store.CheckIfTeamParticipationsConflictWithExistingUserMemberships(1, 2, true)
		return err
	}))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataStore_CheckIfTeamParticipationsConflictWithExistingUserMemberships_Error(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")
	mock.ExpectBegin()
	mock.ExpectQuery("^SELECT .+$").WillReturnError(expectedError)
	mock.ExpectRollback()
	err := NewDataStore(db).InTransaction(func(store *DataStore) error {
		_, err := store.CheckIfTeamParticipationsConflictWithExistingUserMemberships(1, 2, false)
		return err
	})
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
