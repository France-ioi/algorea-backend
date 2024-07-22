package database

import (
	"time"

	"github.com/France-ioi/AlgoreaBackend/app/logging"
)

const groups = "groups"

// createNewAncestorsQueries contains the SQL queries needed for createNewAncestors.
type createNewAncestorsQueries struct {
	markAsProcessingQuery string
	recomputeQueries      []string
	markAsDoneQuery       string
}

// createNewAncestors inserts new rows into
// the objectName_ancestors table (items_ancestors or groups_ancestors)
// for all rows marked with ancestors_computation_state="todo" in objectName_propagate
// (items_propagate or groups_propagate) and their descendants.
//
// Note: rows in *_propagate tables with `ancestors_computation_state`="todo"
// are added in the database in SQL triggers:
// - after_insert_items/groups
// - after_update_groups_groups
// - before_insert_items_items/groups_groups
// - before_delete_items_items/groups_groups.
func (s *DataStore) createNewAncestors(objectName, singleObjectName string) { /* #nosec */
	mustNotBeError(s.InTransaction(func(s *DataStore) error {
		initTransactionTime := time.Now()

		s.createNewAncestorsInsideTransactionInitStep(objectName, singleObjectName)

		logging.Debugf("Duration of %v_ancestors propagation init step: %v", objectName, time.Since(initTransactionTime))

		return nil
	}))

	queries := s.constructCreateNewAncestorsQueries(objectName, singleObjectName)

	s.PropagationStepTransaction(func(store *DataStore) bool {
		initStepTransactionTime := time.Now()

		rowsAffected := store.createNewAncestorsInsideTransactionStep(queries)

		logging.Debugf(
			"Duration of %v_ancestors propagation step: %d rows affected, took %v",
			objectName,
			rowsAffected,
			time.Since(initStepTransactionTime),
		)

		return rowsAffected == 0
	})
}

// createNewAncestorsInsideTransaction does the sql work of createNewAncestors.
// It has to be called in a transaction.
// Normally, createNewAncestors is called AFTER transactions.
// But there is a case where we need to call it inside: when we import the badges of the user.
// In this case, there is a verification that there are no cycles that needs the groups ancestors to be propagated.
// For now, since we keep the whole work of createNewAncestors in a single transaction, we can use this function
// when we need to propagate inside a transaction, and createNewAncestors for the normal propagation.
// In the future, we might want to split the steps here each into its own transaction.
// At that time, we'll need a better way to either:
// - Remove the need for badges cycles detection to not depend on group ancestors
// - Or refactor those two functions in a different way.
func (s *DataStore) createNewAncestorsInsideTransaction(objectName, singleObjectName string) {
	s.mustBeInTransaction()

	s.createNewAncestorsInsideTransactionInitStep(objectName, singleObjectName)

	queries := s.constructCreateNewAncestorsQueries(objectName, singleObjectName)

	hasChanges := true
	for hasChanges {
		rowsAffected := s.createNewAncestorsInsideTransactionStep(queries)

		hasChanges = rowsAffected > 0
	}
}

// createNewAncestorsInsideTransactionInitStep does the sql work of the initialization step of createNewAncestors.
func (s *DataStore) createNewAncestorsInsideTransactionInitStep(objectName, singleObjectName string) {
	s.mustBeInTransaction()

	// We mark as 'todo' all descendants of objects marked as 'todo'
	query := `
		INSERT INTO  ` + objectName + `_propagate (id, ancestors_computation_state)
		SELECT descendants.id, 'todo'
		FROM ` + QuoteName(objectName) + ` AS descendants
		JOIN ` + QuoteName(objectName+"_ancestors") + `
			ON descendants.id = ` + QuoteName(objectName+"_ancestors") + ".child_" + singleObjectName + `_id
		JOIN ` + QuoteName(objectName+"_propagate") + ` AS ancestors
			ON ancestors.id = ` + QuoteName(objectName+"_ancestors") + ".ancestor_" + singleObjectName + `_id
		WHERE ancestors.ancestors_computation_state = 'todo'
		FOR UPDATE
		ON DUPLICATE KEY UPDATE ancestors_computation_state = 'todo'` /* #nosec */

	mustNotBeError(s.db.Exec(query).Error)
}

// createNewAncestorsInsideTransactionStep does the sql work of a step of createNewAncestors.
func (s *DataStore) createNewAncestorsInsideTransactionStep(queries createNewAncestorsQueries) int64 {
	s.mustBeInTransaction()

	mustNotBeError(s.Exec(queries.markAsProcessingQuery).Error())
	for i := 0; i < len(queries.recomputeQueries); i++ {
		mustNotBeError(s.Exec(queries.recomputeQueries[i]).Error())
	}

	return s.Exec(queries.markAsDoneQuery).RowsAffected()
}

// constructCreateNewAncestorsQueries constructs the SQL queries needed for the main steps of createNewAncestors.
func (s *DataStore) constructCreateNewAncestorsQueries(objectName, singleObjectName string) (queries createNewAncestorsQueries) {
	relationsTable := objectName + "_" + objectName

	var additionalJoin string
	if objectName == groups {
		additionalJoin = " JOIN `groups` AS parent ON parent.id = groups_groups.parent_group_id AND parent.type != 'Team' "
	}
	// Next queries will be executed in the loop

	// We mark as "processing" all objects that were marked as 'todo' and that have no parents not marked as 'done'
	// This way we prevent infinite looping as we never process objects that are descendants of themselves

	/* #nosec */
	queries.markAsProcessingQuery = `
		UPDATE ` + objectName + `_propagate AS children
		SET children.ancestors_computation_state='processing'
		WHERE children.ancestors_computation_state = 'todo' AND NOT EXISTS (
			SELECT 1 FROM (
				SELECT 1
				FROM ` + relationsTable + `
					JOIN ` + objectName + `_propagate
						ON ` + objectName + `_propagate.id = ` + relationsTable + `.parent_` + singleObjectName + `_id AND
							 ` + objectName + `_propagate.ancestors_computation_state <> 'done'
					` + additionalJoin + `
				WHERE ` + relationsTable + `.child_` + singleObjectName + `_id = children.id
				FOR UPDATE
			) has_undone_parents FOR UPDATE
		)`

	expiresAtColumn := ""
	expiresAtValueJoin := ""
	ignore := "IGNORE"

	if objectName == groups {
		expiresAtColumn = ", expires_at"
		expiresAtValueJoin = ", LEAST(groups_ancestors_join.expires_at, groups_groups.expires_at)"
		ignore = ""
	}

	// For every object marked as 'processing', we compute all its ancestors
	queries.recomputeQueries = make([]string, 0, 3)
	queries.recomputeQueries = append(queries.recomputeQueries, `
		DELETE `+objectName+`_ancestors
		FROM `+objectName+`_ancestors
			JOIN `+objectName+`_propagate
				ON `+objectName+`_propagate.id = `+objectName+`_ancestors.child_`+singleObjectName+`_id
		WHERE `+objectName+`_propagate.ancestors_computation_state = 'processing'`, `
		INSERT `+ignore+` INTO `+objectName+`_ancestors
		(
			ancestor_`+singleObjectName+`_id,
			child_`+singleObjectName+`_id`+`
			`+expiresAtColumn+`
		)
		SELECT
			`+objectName+`_ancestors_join.ancestor_`+singleObjectName+`_id,
			`+relationsTable+`.child_`+singleObjectName+`_id
			`+expiresAtValueJoin+`
		FROM `+relationsTable+` AS `+relationsTable+`
		`+additionalJoin+`
		JOIN `+objectName+`_ancestors AS `+objectName+`_ancestors_join ON (
			`+objectName+`_ancestors_join.child_`+singleObjectName+`_id = `+relationsTable+`.parent_`+singleObjectName+`_id
		)
		JOIN `+objectName+`_propagate ON (
			`+relationsTable+`.child_`+singleObjectName+`_id = `+objectName+`_propagate.id
		)
		WHERE
			`+objectName+`_propagate.ancestors_computation_state = 'processing'`) // #nosec
	if objectName == groups {
		queries.recomputeQueries[1] += `
				AND NOW() < groups_groups.expires_at AND
				NOW() < LEAST(groups_ancestors_join.expires_at, groups_groups.expires_at)
			ON DUPLICATE KEY UPDATE
				expires_at = GREATEST(groups_ancestors.expires_at, LEAST(groups_ancestors_join.expires_at, groups_groups.expires_at))`
		queries.recomputeQueries = append(queries.recomputeQueries, `
			INSERT IGNORE INTO `+objectName+`_ancestors
			(
				ancestor_`+singleObjectName+`_id,
				child_`+singleObjectName+`_id
			)
			SELECT
				groups_propagate.id AS ancestor_group_id,
				groups_propagate.id AS child_group_id
			FROM groups_propagate
			WHERE groups_propagate.ancestors_computation_state = 'processing'
			FOR UPDATE`) // #nosec
	} else {
		queries.recomputeQueries[1] += ` FOR UPDATE`
		queries.recomputeQueries = append(queries.recomputeQueries, `
			INSERT IGNORE INTO items_ancestors (ancestor_item_id, child_item_id)
			SELECT items_items.parent_item_id, items_items.child_item_id
			FROM items_items
			JOIN items_propagate ON items_items.child_item_id = items_propagate.id
			WHERE items_propagate.ancestors_computation_state = 'processing'
			FOR UPDATE`) // #nosec
	}

	// Objects marked as 'processing' are now marked as 'done'
	queries.markAsDoneQuery = `
		UPDATE ` + objectName + `_propagate
		SET ancestors_computation_state = 'done'
		WHERE ancestors_computation_state = 'processing'` // #nosec

	return queries
}
