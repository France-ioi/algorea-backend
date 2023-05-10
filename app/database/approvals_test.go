package database

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_WithPersonalInfoViewApprovals(t *testing.T) {
	db, mock := NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("^" + regexp.QuoteMeta(
		"SELECT * FROM `users` LEFT JOIN LATERAL ("+
			"SELECT 1 AS approved FROM `groups_ancestors_active` "+
			"JOIN group_managers "+
			"ON group_managers.group_id = `groups_ancestors_active`.ancestor_group_id "+
			"JOIN groups_ancestors_active AS group_ancestors "+
			"ON group_ancestors.ancestor_group_id = group_managers.manager_id AND group_ancestors.child_group_id = ? "+
			"JOIN groups_groups_active ON groups_groups_active.parent_group_id = groups_ancestors_active.child_group_id AND "+
			"groups_groups_active.personal_info_view_approved "+
			"WHERE groups_groups_active.child_group_id = users.group_id "+
			"LIMIT 1"+
			") AS personal_info_view_approvals ON 1") + "$").WithArgs(23).
		WillReturnRows(mock.NewRows([]string{"id", "approved"}).AddRow(123, 1))

	type resultType struct {
		ID       int64
		Approved bool
	}
	var result []resultType
	assert.NoError(t,
		NewDataStore(db).Users().WithPersonalInfoViewApprovals(&User{GroupID: 23}).
			Select("*").Scan(&result).Error())
	assert.Equal(t, []resultType{{ID: 123, Approved: true}}, result)
}
