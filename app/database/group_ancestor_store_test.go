package database

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupAncestorStore_ManagedByUser(t *testing.T) {
	for _, test := range []struct {
		tableName     string
		expectedQuery string
		storeFunc     func(*DB) *GroupAncestorStore
	}{
		{
			tableName: "groups_ancestors",
			expectedQuery: "SELECT * FROM `groups_ancestors` " +
				"JOIN group_managers ON group_managers.group_id = `groups_ancestors`.ancestor_group_id " +
				"JOIN groups_ancestors_active AS group_ancestors " +
				"ON group_ancestors.ancestor_group_id = group_managers.manager_id AND " +
				"group_ancestors.child_group_id = ? " +
				"WHERE NOW() < `groups_ancestors`.expires_at",
			storeFunc: func(db *DB) *GroupAncestorStore { return NewDataStore(db).GroupAncestors() },
		},
		{
			tableName: "groups_ancestors_active",
			expectedQuery: "SELECT * FROM `groups_ancestors_active` " +
				"JOIN group_managers ON group_managers.group_id = `groups_ancestors_active`.ancestor_group_id " +
				"JOIN groups_ancestors_active AS group_ancestors " +
				"ON group_ancestors.ancestor_group_id = group_managers.manager_id AND " +
				"group_ancestors.child_group_id = ?",
			storeFunc: func(db *DB) *GroupAncestorStore { return NewDataStore(db).ActiveGroupAncestors() },
		},
	} {
		test := test
		t.Run(test.tableName, func(t *testing.T) {
			db, mock := NewDBMock()
			defer func() { _ = db.Close() }()

			mockUser := &User{GroupID: 2, DefaultLanguage: "fr"}
			mock.ExpectQuery(regexp.QuoteMeta(test.expectedQuery)).WithArgs(2).
				WillReturnRows(mock.NewRows([]string{"id"}))

			var result []interface{}
			err := test.storeFunc(db).ManagedByUser(mockUser).Scan(&result).Error()

			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func ptrInt64(i int64) *int64 { return &i }
