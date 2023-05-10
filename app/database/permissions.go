package database

// WherePermissionIsAtLeast returns a composable query filtered by `can_*_generated_value` >= indexOf(`permissionName`)
// depending on the given permission kind.
func (conn *DB) WherePermissionIsAtLeast(permissionKind, permissionName string) *DB {
	return newDB(conn.ctx, conn.db.Where("?",
		NewDataStore(conn.New()).PermissionsGranted().PermissionIsAtLeastSQLExpr(permissionKind, permissionName)))
}

// HavingMaxPermissionAtLeast returns a composable query filtered by `MAX(can_*_generated_value)` >= indexOf(`permissionName`)
// depending on the given permission kind.
func (conn *DB) HavingMaxPermissionAtLeast(permissionKind, permissionName string) *DB {
	return newDB(conn.ctx, conn.db.
		Having("MAX("+permissionColumnByKind(permissionKind)+") >= ?",
			NewDataStore(conn.New()).PermissionsGranted().PermissionIndexByKindAndName(permissionKind, permissionName)),
	)
}

// JoinsPermissionsForGroupToItemsWherePermissionAtLeast returns a composable query with access rights (as *_generated_value)
// for all the items on that the given group has 'permissionKind' >= `neededPermission`.
func (conn *DB) JoinsPermissionsForGroupToItemsWherePermissionAtLeast(groupID int64, permissionKind, neededPermission string) *DB {
	permissionsQuery := NewDataStore(conn.New()).Permissions().
		AggregatedPermissionsForItemsOnWhichGroupHasPermission(groupID, permissionKind, neededPermission).
		Where("permissions.item_id = items.id") // This condition is needed to filter by item_id before aggregating
	// The JOIN LATERAL allows us to filter permissions on both group_id & item_id here
	// instead of calculating permissions for all the items before joining
	return conn.Joins("JOIN LATERAL (?) AS permissions ON permissions.item_id = items.id", permissionsQuery.SubQuery())
}

func permissionColumnByKind(permissionKind string) string {
	return map[string]string{
		"view":       "can_view_generated_value",
		"grant_view": "can_grant_view_generated_value",
		"watch":      "can_watch_generated_value",
		"edit":       "can_edit_generated_value",
	}[permissionKind]
}
