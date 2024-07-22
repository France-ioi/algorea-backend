package database

import (
	"time"

	"github.com/France-ioi/AlgoreaBackend/app/logging"
)

const (
	propagateLockName    = "listener_propagate"
	propagateLockTimeout = 10 * time.Second
)

// propagate recomputes fields of results
// For results marked as 'to_be_propagated'/'to_be_recomputed':
//  1. We mark all their ancestors in results as 'to_be_recomputed'
//     (we consider a row in results as an ancestor if
//     a) it has the same value in group_id
//     b) its item_id is an ancestor of the original row's item_id
//     c) its attempt_id is equal to the original row's attempt_id for original rows with root_item_id != item_id or
//     its attempt_id is equal to the original row's parent_attempt_id for original rows with root_item_id = item_id).
//  2. We process all objects that are marked as 'to_be_recomputed' and that have no children marked as 'to_be_recomputed'.
//     Then, if an object has children, we update
//     latest_activity_at, tasks_tried, tasks_with_help, validated_at.
//     This step is repeated until no records are updated.
//  3. We insert new permissions_granted for each unlocked item according to corresponding item_dependencies.
//
// The `results_propagation` rows are marked in code as well as in the following SQL Triggers:
// - after_insert_groups_groups/items_items
// - after_insert_permissions_generated
// - after_update_groups_groups/items_items
// - after_update_permissions_generated
// - before_delete_items_items.
func (s *ResultStore) propagate() (err error) {
	var groupsUnlocked int64
	defer recoverPanics(&err)
	// Use a lock so that we don't execute the listener multiple times in parallel
	mustNotBeError(s.WithNamedLock(propagateLockName, propagateLockTimeout, func(s *DataStore) error {
		s.setResultsPropagationFromResultsPropagateItems()

		mustNotBeError(s.InTransaction(func(s *DataStore) error {
			initTransactionTime := time.Now()

			// We mark as 'to_be_recomputed' results of all ancestors of items marked as 'to_be_recomputed'/'to_be_propagated'
			// with appropriate attempt_id.
			// Also, we insert missing results for chapters having descendants with results marked as 'to_be_recomputed'/'to_be_propagated'.
			// We only create results for chapters which are (or have ancestors which are) visible to the group that attempted
			// to solve the descendant items. Chapters requiring explicit entry or placed outside of the scope
			// of the attempts' root item are skipped).
			// (This query can take more than 30 seconds to run when executed for the first time after the db migration)
			mustNotBeError(s.Exec("DROP TEMPORARY TABLE IF EXISTS results_to_mark").Error())
			mustNotBeError(s.Exec(`
			CREATE TEMPORARY TABLE results_to_mark (
				participant_id BIGINT(20) NOT NULL,
				attempt_id BIGINT(20) NOT NULL,
				item_id BIGINT(20) NOT NULL,
				result_exists TINYINT(1) NOT NULL,
				KEY result_exists (result_exists)
			)`).Error())
			defer func() { mustNotBeError(s.Exec("DROP TEMPORARY TABLE results_to_mark").Error()) }()

			result := s.db.Exec(`
			INSERT INTO results_to_mark (participant_id, attempt_id, item_id, result_exists)
			WITH RECURSIVE results_to_insert (participant_id, attempt_id, item_id, result_exists) AS (
					SELECT results.participant_id,
								 IF(attempts.root_item_id = results.item_id, attempts.parent_attempt_id, results.attempt_id) AS attempt_id,
								 items_items.parent_item_id AS item_id,
								 existing.participant_id IS NOT NULL AS result_exists
					FROM results
					JOIN results_propagate USING(participant_id, attempt_id, item_id)
					JOIN attempts ON attempts.participant_id = results.participant_id AND attempts.id = results.attempt_id
					JOIN items_items ON items_items.child_item_id = results.item_id
					JOIN items ON items.id = items_items.parent_item_id
					LEFT JOIN results AS existing
						ON existing.participant_id = results.participant_id AND
							 existing.attempt_id = IF(attempts.root_item_id = results.item_id, attempts.parent_attempt_id, results.attempt_id) AND
							 existing.item_id = items_items.parent_item_id
					LEFT JOIN results_propagate AS existing_propagate
						ON existing_propagate.participant_id = existing.participant_id AND existing_propagate.attempt_id = existing.attempt_id AND
               existing_propagate.item_id = existing.item_id
					WHERE NOT (items.requires_explicit_entry AND existing.participant_id IS NULL) AND
								(existing.participant_id IS NULL OR existing_propagate.state IS NULL OR existing_propagate.state != 'to_be_recomputed')
				UNION
					SELECT results_to_insert.participant_id,
								 IF(attempts.root_item_id = results_to_insert.item_id, attempts.parent_attempt_id, results_to_insert.attempt_id) AS attempt_id,
								 items_items.parent_item_id AS item_id,
								 existing.participant_id IS NOT NULL AS result_exists
					FROM results_to_insert
					JOIN attempts ON attempts.participant_id = results_to_insert.participant_id AND attempts.id = results_to_insert.attempt_id
					JOIN items_items ON items_items.child_item_id = results_to_insert.item_id
					JOIN items ON items.id = items_items.parent_item_id
					LEFT JOIN results AS existing
						ON existing.participant_id = results_to_insert.participant_id AND
							 existing.attempt_id =
						     IF(attempts.root_item_id = results_to_insert.item_id, attempts.parent_attempt_id, results_to_insert.attempt_id) AND
							 existing.item_id = items_items.parent_item_id
					LEFT JOIN results_propagate AS existing_propagate
						ON existing_propagate.participant_id = existing.participant_id AND existing_propagate.attempt_id = existing.attempt_id AND
               existing_propagate.item_id = existing.item_id
					WHERE NOT (items.requires_explicit_entry AND existing.participant_id IS NULL) AND
								(existing.participant_id IS NULL OR existing_propagate.state IS NULL OR existing_propagate.state != 'to_be_recomputed')
			)
			SELECT
				results_to_insert.participant_id, results_to_insert.attempt_id, results_to_insert.item_id,
        MAX(results_to_insert.result_exists) AS result_exists
			FROM results_to_insert
			JOIN attempts ON attempts.participant_id = results_to_insert.participant_id AND attempts.id = results_to_insert.attempt_id
			LEFT JOIN items_ancestors AS root_item_descendant
				ON root_item_descendant.ancestor_item_id = attempts.root_item_id AND root_item_descendant.child_item_id = results_to_insert.item_id
			WHERE result_exists OR ((
				EXISTS(
					SELECT 1
					FROM permissions_generated
					JOIN groups_ancestors_active
						ON groups_ancestors_active.ancestor_group_id = permissions_generated.group_id
					WHERE
						permissions_generated.item_id = results_to_insert.item_id AND
						permissions_generated.can_view_generated != 'none' AND
						groups_ancestors_active.child_group_id = results_to_insert.participant_id
				) OR EXISTS(
					SELECT 1
					FROM permissions_generated
					JOIN groups_ancestors_active
						ON groups_ancestors_active.ancestor_group_id = permissions_generated.group_id
					WHERE
						permissions_generated.item_id IN (
							SELECT grand_ancestors.ancestor_item_id
							FROM items_ancestors AS grand_ancestors
							WHERE grand_ancestors.child_item_id = results_to_insert.item_id
						) AND permissions_generated.can_view_generated != 'none' AND
						groups_ancestors_active.child_group_id = results_to_insert.participant_id
			)) AND (
				attempts.root_item_id IS NULL OR attempts.root_item_id = results_to_insert.item_id OR
				root_item_descendant.ancestor_item_id IS NOT NULL))
			GROUP BY results_to_insert.participant_id, results_to_insert.attempt_id, results_to_insert.item_id
		`)
			mustNotBeError(result.Error)

			if result.RowsAffected > 0 {
				mustNotBeError(s.Exec(`
				INSERT IGNORE INTO results (participant_id, attempt_id, item_id, latest_activity_at)
				SELECT
					results_to_mark.participant_id, results_to_mark.attempt_id, results_to_mark.item_id, '1000-01-01 00:00:00'
				FROM results_to_mark
				WHERE NOT result_exists`).Error())

				mustNotBeError(s.Exec(`
				INSERT INTO results_propagate (participant_id, attempt_id, item_id, state)
				SELECT
					results_to_mark.participant_id, results_to_mark.attempt_id, results_to_mark.item_id, 'to_be_recomputed'
				FROM results_to_mark
				ON DUPLICATE KEY UPDATE state = 'to_be_recomputed'`).Error())
			}

			logging.Debugf(
				"Duration of step of results propagation: %d rows affected, took %v",
				result.RowsAffected,
				time.Since(initTransactionTime),
			)

			return nil
		}))

		s.PropagationStepTransaction(func(store *DataStore) bool {
			initTransactionTime := time.Now()

			// We process only those objects that were marked as 'to_be_recomputed' and
			// that have no children (within the attempt or child attempts) marked as 'to_be_recomputed'.
			// This way we prevent infinite looping as we never process items that are ancestors of themselves
			//
			// For every object, we compute all the characteristics based on the children:
			//  - latest_activity_at as the max of children's
			//  - tasks_with_help, tasks_tried as the sum of children's per-item maximums
			//  - children_validated as the number of children items with validated == 1
			//  - validated, depending on the items_items.category and items.validation_type
			//    (an item should have at least one validated child to become validated itself by the propagation)
			const updateQuery = `
					UPDATE results_propagate AS target_propagate ` +
				// process only those results marked as 'to_be_recomputed' that do not have child results marked as 'to_be_recomputed'
				`JOIN (
						SELECT *
						FROM (
							WITH marked_to_be_recomputed AS (SELECT participant_id, attempt_id, item_id FROM results_propagate WHERE state='to_be_recomputed')
							SELECT DISTINCT inner_parent.participant_id, inner_parent.attempt_id, inner_parent.item_id
							FROM marked_to_be_recomputed AS inner_parent
							WHERE
								NOT EXISTS (
									SELECT 1
									FROM items_items
									JOIN marked_to_be_recomputed AS children
										ON children.participant_id = inner_parent.participant_id AND
										   children.attempt_id = inner_parent.attempt_id AND
										   children.item_id = items_items.child_item_id
									WHERE items_items.parent_item_id = inner_parent.item_id
								) AND NOT EXISTS (
									SELECT 1
									FROM items_items
									JOIN attempts
										ON attempts.participant_id = inner_parent.participant_id AND
										   attempts.parent_attempt_id = inner_parent.attempt_id AND
										   attempts.root_item_id = items_items.child_item_id
									JOIN marked_to_be_recomputed AS children
										ON children.participant_id = inner_parent.participant_id AND
											 children.attempt_id = attempts.id AND
										   children.item_id = items_items.child_item_id
									WHERE items_items.parent_item_id = inner_parent.item_id
								)
						) AS tmp2
					) AS tmp USING(participant_id, attempt_id, item_id)
					JOIN results AS target_results USING(participant_id, attempt_id, item_id)
					JOIN items
						ON items.id = target_propagate.item_id
					LEFT JOIN LATERAL (
						SELECT
							target_results.participant_id,
							MAX(aggregated_children_results.latest_activity_at) AS latest_activity_at,
							SUM(aggregated_children_results.tasks_tried) AS tasks_tried,
							SUM(aggregated_children_results.tasks_with_help) AS tasks_with_help,
							SUM(aggregated_children_results.validated) AS children_validated,
							SUM(IFNULL(NOT aggregated_children_results.validated, 1)) AS children_non_validated,
							SUM(items_items.category = 'Validation' AND IFNULL(NOT aggregated_children_results.validated, 1))
								AS children_non_validated_categories,
							MAX(aggregated_children_results.validated_at) AS max_validated_at,
							MAX(IF(items_items.category = 'Validation', aggregated_children_results.validated_at, NULL))
								AS max_validated_at_categories,
							SUM(IFNULL(aggregated_children_results.score_computed, 0) * items_items.score_weight) /
								COALESCE(NULLIF(SUM(items_items.score_weight), 0), 1) AS average_score
						FROM items_items ` +
				// We use LEFT JOIN LATERAL to aggregate results grouped by target_results.participant_id & items_items.child_item_id.
				// The usual LEFT JOIN conditions in the ON clause would group results before joining which would produce
				// wrong results.
				`	LEFT JOIN LATERAL (
							SELECT
								MAX(validated) AS validated,
								MIN(validated_at) AS validated_at,
								MAX(latest_activity_at) AS latest_activity_at,
								MAX(tasks_tried) AS tasks_tried,
								MAX(tasks_with_help) AS tasks_with_help,
								MAX(score_computed) AS score_computed
							FROM results AS children_results
							JOIN attempts
								ON attempts.participant_id = children_results.participant_id AND
								   attempts.id = children_results.attempt_id
							WHERE children_results.participant_id = target_results.participant_id AND
								children_results.item_id = items_items.child_item_id AND
							  (children_results.attempt_id = target_results.attempt_id OR
							    (attempts.root_item_id = items_items.child_item_id AND
									 attempts.parent_attempt_id = target_results.attempt_id))
							GROUP BY children_results.participant_id, children_results.item_id
						) AS aggregated_children_results ON 1
						JOIN items ON(
							items.id = items_items.child_item_id
						)
						WHERE items_items.parent_item_id = target_results.item_id AND NOT items.no_score
						GROUP BY items_items.parent_item_id
					) AS children_stats ON 1
					SET
						target_results.latest_activity_at = GREATEST(
							IFNULL(children_stats.latest_activity_at, '1000-01-01 00:00:00'),
							target_results.latest_activity_at
						),
						target_results.tasks_tried = IFNULL(children_stats.tasks_tried, 0),
						target_results.tasks_with_help = IFNULL(children_stats.tasks_with_help, 0),
						target_results.validated_at = CASE
							WHEN children_stats.participant_id IS NULL THEN NULL
							WHEN items.validation_type = 'Categories' AND children_stats.children_non_validated_categories = 0
								THEN children_stats.max_validated_at_categories
							WHEN items.validation_type = 'All' AND children_stats.children_non_validated = 0
								THEN children_stats.max_validated_at
							WHEN items.validation_type = 'AllButOne' AND children_stats.children_non_validated < 2
								THEN children_stats.max_validated_at
							WHEN items.validation_type = 'One' AND children_stats.children_validated > 0
								THEN children_stats.max_validated_at
							ELSE NULL
						END,
						target_results.score_computed = IFNULL(IF(items.no_score OR children_stats.average_score IS NULL,
							0,
							LEAST(GREATEST(CASE target_results.score_edit_rule
								WHEN 'set' THEN target_results.score_edit_value
								WHEN 'diff' THEN children_stats.average_score + target_results.score_edit_value
								ELSE children_stats.average_score
							END, 0), 100)), 0),
						target_propagate.state = 'to_be_propagated'`

			rowsAffected := store.Exec(updateQuery).RowsAffected()

			logging.Debugf("Duration of step of results propagation: %d rows affected, took %v", rowsAffected, time.Since(initTransactionTime))

			return rowsAffected == 0
		})

		mustNotBeError(s.InTransaction(func(s *DataStore) error {
			initTransactionTime := time.Now()

			canViewContentIndex := s.PermissionsGranted().ViewIndexByName("content")
			result := s.db.Exec(`
			INSERT INTO permissions_granted
				(group_id, item_id, source_group_id, origin, can_view, can_enter_from, latest_update_at)
				SELECT
					results.participant_id,
					item_dependencies.dependent_item_id AS item_id,
					results.participant_id,
					'item_unlocking',
					IF(items.requires_explicit_entry, 'none', 'content'),
					IF(items.requires_explicit_entry, NOW(), '9999-12-31 23:59:59'),
					NOW()
				FROM results_propagate
				JOIN results USING(participant_id, attempt_id, item_id)
				JOIN item_dependencies ON item_dependencies.item_id = results.item_id AND
					item_dependencies.score <= results.score_computed AND item_dependencies.grant_content_view
				JOIN items ON items.id = item_dependencies.dependent_item_id
				WHERE results_propagate.state = 'to_be_propagated'
			ON DUPLICATE KEY UPDATE
				latest_update_at = IF(
					VALUES(can_view) = 'content' AND can_view_value < ? OR
					VALUES(can_enter_from) <> '9999-12-31 23:59:59' AND can_enter_from > VALUES(can_enter_from) OR
					VALUES(can_enter_from) <> '9999-12-31 23:59:59' AND can_enter_until <> '9999-12-31 23:59:59',
					NOW(), latest_update_at),
				can_view = IF(VALUES(can_view) = 'content' AND can_view_value < ?, 'content', can_view),
				can_enter_from = IF(
					VALUES(can_enter_from) <> '9999-12-31 23:59:59' AND can_enter_from > VALUES(can_enter_from),
					VALUES(can_enter_from), can_enter_from)`,
				canViewContentIndex, canViewContentIndex)

			mustNotBeError(result.Error)
			groupsUnlocked += result.RowsAffected

			err = s.db.Exec(`DELETE FROM results_propagate WHERE state = 'to_be_propagated'`).Error
			mustNotBeError(err)

			logging.Debugf(
				"Duration of final step of results propagation: %d rows affected, took %v",
				result.RowsAffected,
				time.Since(initTransactionTime),
			)

			return nil
		}))

		return nil
	}))

	// If items have been unlocked, need to recompute access
	if groupsUnlocked > 0 {
		mustNotBeError(s.InTransaction(func(s *DataStore) error {
			// generate permissions_generated from permissions_granted
			s.SchedulePropagation([]string{"permissions"})
			// we should compute attempts again as new permissions were set and
			// triggers on permissions_generated likely marked some attempts as 'to_be_propagated'
			s.SchedulePropagation([]string{"results"})

			return nil
		}))
	}

	return nil
}

// setResultsPropagationFromResultsPropagateItems inserts results_propagate rows from results_propagate_items.
func (s *DataStore) setResultsPropagationFromResultsPropagateItems() {
	const maxInserts = 5000

	const fromResultsWithItemIDToPropagateQueryPart = `
		FROM results
		WHERE item_id IN (SELECT item_id FROM results_propagate_items ORDER BY item_id)
	`

	mustNotBeError(s.InTransaction(func(s *DataStore) error {
		initTransactionTime := time.Now()

		// First we count the number of rows that need to be processed, then we use LIMIT/OFFSET.
		// Because we want the transaction to take a reasonable time to execute,
		// we also don't want to have too many small transactions running, that's why we don't loop on item_id.
		//
		// We can't use rowsAffected to know whether all rows are updated, because it counts:
		// - 0 when a row exists and is not updated
		// - 1 when a row is inserted
		// - 2 when a row exists and is updated
		// So we can have rowsAffected == 0 if nothing has to be updated in the first `maxInserts` rows,
		// even when there is still work to be done at a later offset.
		var res struct {
			Count int64 `json:"count"`
		}
		mustNotBeError(
			s.Raw(`SELECT COUNT(*) AS count ` + fromResultsWithItemIDToPropagateQueryPart).
				Scan(&res).
				Error(),
		)

		totalRowsAffected := int64(0)
		offset := int64(0)
		for ; offset < res.Count; offset += maxInserts {
			rowsAffected := s.Exec(`
							INSERT INTO results_propagate
								(
								 	SELECT participant_id, attempt_id, item_id, 'to_be_propagated' AS state
									`+fromResultsWithItemIDToPropagateQueryPart+`
									ORDER BY participant_id, attempt_id, item_id
									LIMIT ? OFFSET ?
								)
							ON DUPLICATE KEY UPDATE state = 'to_be_recomputed';
					`, maxInserts, offset).
				RowsAffected()

			totalRowsAffected += rowsAffected
		}

		mustNotBeError(
			s.Exec("DELETE FROM `results_propagate_items`").
				Error(),
		)

		logging.Debugf(
			"Duration of step of results propagation insertion from results_propagate_items: took %v with %d rows affected and last offset %d",
			time.Since(initTransactionTime),
			totalRowsAffected,
			offset,
		)

		return nil
	}))
}
