package database

import "database/sql"

// computeAllAccess recomputes fields of permissions_generated.
//
// It starts from group-item pairs marked with propagate_to = 'self' in `permissions_propagate`.
//
// 1. can_view_generated, can_grant_view_generated, can_watch_generated, can_edit_generated, is_owner_generated are updated.
//
// 3. Then the loop repeats from step 1 for all children (from items_items) of the processed permissions_generated.
//
// Notes:
//   - The function may loop endlessly if items_items is a cyclic graph.
//   - Processed group-item pairs are removed from permissions_propagate.
func (s *PermissionGrantedStore) computeAllAccess() {
	s.mustBeInTransaction()

	// ------------------------------------------------------------------------------------
	// Here we declare and prepare DB statements that will be used by the function later on
	// ------------------------------------------------------------------------------------
	var stmtMarkChildrenOfChildrenAsSelf, stmtDeleteProcessedChildren, stmtUpdatePermissionsGenerated, stmtMarkSelfAsChildren *sql.Stmt
	var err error

	// marking group-item pairs whose parents are marked with propagate_to = 'children' as 'self'
	const queryMarkChildrenOfChildrenAsSelf = `
		INSERT INTO permissions_propagate (group_id, item_id, propagate_to)
		SELECT
			parents.group_id,
			items_items.child_item_id,
			'self' as propagate_to
		FROM items_items
		JOIN permissions_generated AS parents
			ON parents.item_id = items_items.parent_item_id
		JOIN permissions_propagate AS parents_propagate
			ON parents_propagate.group_id = parents.group_id AND parents_propagate.item_id = parents.item_id
		WHERE parents_propagate.propagate_to = 'children'
		ON DUPLICATE KEY UPDATE propagate_to='self'`
	sqlDB := s.db.Statement.ConnPool.(*sql.Tx)
	stmtMarkChildrenOfChildrenAsSelf, err = sqlDB.Prepare(queryMarkChildrenOfChildrenAsSelf)
	mustNotBeError(err)
	defer func() { mustNotBeError(stmtMarkChildrenOfChildrenAsSelf.Close()) }()

	// deleting 'children' permissions_propagate
	const queryDeleteProcessedChildren = `DELETE FROM permissions_propagate WHERE propagate_to = 'children'`
	stmtDeleteProcessedChildren, err = sqlDB.Prepare(queryDeleteProcessedChildren)
	mustNotBeError(err)
	defer func() { mustNotBeError(stmtDeleteProcessedChildren.Close()) }()

	// computation for group-item pairs marked as 'self' in permissions_propagate (so all of them)
	const queryUpdatePermissionsGenerated = `
		INSERT INTO permissions_generated
			(group_id, item_id, can_view_generated, can_grant_view_generated, can_watch_generated, can_edit_generated, is_owner_generated)
		SELECT
			permissions_propagate.group_id,
			permissions_propagate.item_id,
			IF(MAX(permissions_granted.is_owner), 'solution', GREATEST(
				IFNULL(MAX(permissions_granted.can_view_value), 1),
				IFNULL(MAX(
					CASE
					WHEN parent.can_view_generated IS NULL OR parent.can_view_generated IN ('none', 'info') THEN 1 /* none */
					WHEN parent.can_view_generated = 'content' OR items_items.upper_view_levels_propagation = 'use_content_view_propagation' THEN
						CASE items_items.content_view_propagation
						WHEN 'as_info' THEN 2 /* info */
						WHEN 'as_content' THEN 3 /* content */
						ELSE 1 /* none */
						END
					WHEN items_items.upper_view_levels_propagation = 'as_content_with_descendants' THEN 4 /* content_with_descendants */
					ELSE parent.can_view_generated_value
					END), 1)
			)) AS can_view_generated,
			IF(MAX(permissions_granted.is_owner), 'solution_with_grant', GREATEST(
				IFNULL(MAX(permissions_granted.can_grant_view_value), 1),
				IFNULL(MAX(IF(items_items.grant_view_propagation, LEAST(parent.can_grant_view_generated_value, 5 /* solution */), 1)), 1)
			)) AS can_grant_view_generated,
			IF(MAX(permissions_granted.is_owner), 'answer_with_grant', GREATEST(
				IFNULL(MAX(permissions_granted.can_watch_value), 1),
				IFNULL(MAX(IF(items_items.watch_propagation, LEAST(parent.can_watch_generated_value, 3 /* answer */), 1)), 1)
			)) AS can_watch_generated,
			IF(MAX(permissions_granted.is_owner), 'all_with_grant', GREATEST(
				IFNULL(MAX(permissions_granted.can_edit_value), 1),
				IFNULL(MAX(IF(items_items.edit_propagation, LEAST(parent.can_edit_generated_value, 3 /* all */), 1)), 1)
			)) AS can_edit_generated,
			IFNULL(MAX(permissions_granted.is_owner), 0) AS is_owner_generated
		FROM permissions_propagate
		LEFT JOIN permissions_granted USING (group_id, item_id)
		LEFT JOIN items_items ON items_items.child_item_id = permissions_propagate.item_id
		LEFT JOIN permissions_generated AS parent
		  ON parent.item_id = items_items.parent_item_id AND parent.group_id = permissions_propagate.group_id
		GROUP BY permissions_propagate.group_id, permissions_propagate.item_id
		ON DUPLICATE KEY UPDATE
			can_view_generated = VALUES(can_view_generated),
			can_grant_view_generated = VALUES(can_grant_view_generated),
			can_watch_generated = VALUES(can_watch_generated),
			can_edit_generated = VALUES(can_edit_generated),
			is_owner_generated = VALUES(is_owner_generated)`
	stmtUpdatePermissionsGenerated, err = sqlDB.Prepare(queryUpdatePermissionsGenerated)
	mustNotBeError(err)
	defer func() { mustNotBeError(stmtUpdatePermissionsGenerated.Close()) }()

	// marking 'self' permissions_propagate (so all of them) as 'children'
	// (although all existing rows in permissions_propagate have propagate_to='self' at this moment,
	//  we still need to use WHERE clause in order for MySQL to use indexes,
	//  otherwise the query can take minutes to execute)
	const queryMarkSelfAsChildren = `
		UPDATE permissions_propagate
		SET propagate_to = 'children' WHERE propagate_to='self'`
	stmtMarkSelfAsChildren, err = sqlDB.Prepare(queryMarkSelfAsChildren)
	mustNotBeError(err)
	defer func() { mustNotBeError(stmtMarkSelfAsChildren.Close()) }()

	// ------------------------------------------------------------------------------------
	// Here we execute the statements
	// ------------------------------------------------------------------------------------
	hasChanges := true
	for hasChanges {
		_, err = stmtMarkChildrenOfChildrenAsSelf.Exec()
		mustNotBeError(err)
		_, err = stmtDeleteProcessedChildren.Exec()
		mustNotBeError(err)
		_, err = stmtUpdatePermissionsGenerated.Exec()
		mustNotBeError(err)

		var result sql.Result
		result, err = stmtMarkSelfAsChildren.Exec()
		mustNotBeError(err)
		var rowsAffected int64
		rowsAffected, err = result.RowsAffected()
		mustNotBeError(err)
		hasChanges = rowsAffected > 0
	}
}
