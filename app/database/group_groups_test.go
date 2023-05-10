package database

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_WhereGroupRelationIsActual(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `groups_groups` " +
			"WHERE NOW() < groups_groups.expires_at")).
		WillReturnRows(mock.NewRows([]string{"id"}))

	var result []interface{}
	err := db.Table("groups_groups").WhereGroupRelationIsActual().Scan(&result).Error()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupGroupStore_WithGroupsRelationsLock(t *testing.T) {
	assertNamedLockMethod(t, "groups_groups", int(groupsRelationsLockTimeout.Seconds()), "groups_groups",
		func(store *DataStore) func(func(store *DataStore) error) error {
			return func(txFunc func(store *DataStore) error) error {
				return store.GroupGroups().WithGroupsRelationsLock(txFunc)
			}
		})
}
