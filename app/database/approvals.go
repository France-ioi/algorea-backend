package database

// WithPersonalInfoViewApprovals returns a subview with information
// on if a user (in `users` table) approved viewing their personal info for the given manager
// (`groups_groups.personal_info_view_approved` is true for a group managed by the given manager).
// The approvals can be checked as `personal_info_view_approvals.approved`.
func (conn *DB) WithPersonalInfoViewApprovals(manager *User) *DB {
	return conn.Joins("LEFT JOIN LATERAL (?) AS personal_info_view_approvals ON 1",
		NewDataStore(conn.New()).ActiveGroupAncestors().ManagedByUser(manager).
			Joins(`
				JOIN groups_groups_active
					ON groups_groups_active.parent_group_id = groups_ancestors_active.child_group_id AND
					   groups_groups_active.personal_info_view_approved`).
			Where("groups_groups_active.child_group_id = users.group_id").
			Select("1 AS approved").
			Limit(1).
			SubQuery())
}
