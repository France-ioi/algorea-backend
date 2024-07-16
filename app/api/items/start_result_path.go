package items

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation POST /items/{ids}/start-result-path items resultStartPath
//
//	---
//	summary: Start results for an item path
//	description: >
//		Creates new started results (or starts not started existing ones) for an item path if needed and returns the last attempt in the chain.
//
//		Of all possible chains of attempts the service chooses the one having missing/not-started results located closer
//		to the end of the path, preferring chains having less missing/not-started results and having higher values of `attempt_id`.
//		If there is no result for the first item, the service tries to create an attempt chain starting with the zero attempt.
//		The chain of attempts cannot have missing results for items requiring explicit entry or require to start/create results
//		within or below ended/not-allowing-submissions attempts.
//
//		If `as_team_id` is given, the created/updated results are linked to the `as_team_id` group instead of the user's self group.
//
//
//			Restrictions:
//
//		* if `as_team_id` is given, it should be a user's parent team group,
//		* the first item in `{ids}` should be a root activity/skill (groups.root_activity_id/root_skill_id) of a group
//			the participant is a descendant of or manages,
//		* `{ids}` should be an ordered list of parent-child items,
//		* the group starting results should have at least 'content' access on each of the items in `{ids}`,
//
//		otherwise the 'forbidden' error is returned.
//	parameters:
//		- name: ids
//			in: path
//			type: string
//			description: slash-separated list of item IDs
//			required: true
//		- name: as_team_id
//			in: query
//			type: integer
//	responses:
//		"201":
//			description: "Created. Success response with the attempt id for the last item in the path"
//			schema:
//					type: object
//					required: [success, message, data]
//					properties:
//						success:
//							description: "true"
//							type: boolean
//							enum: [true]
//						message:
//							description: updated
//							type: string
//							enum: [updated]
//						data:
//							type: object
//							required: [attempt_id]
//							properties:
//								attempt_id:
//									description: The attempt linked to the last item in the path
//									type: integer
//									format: string
//		"400":
//			"$ref": "#/responses/badRequestResponse"
//		"401":
//			"$ref": "#/responses/unauthorizedResponse"
//		"403":
//			"$ref": "#/responses/forbiddenResponse"
//		"500":
//			"$ref": "#/responses/internalErrorResponse"
func (srv *Service) startResultPath(w http.ResponseWriter, r *http.Request) service.APIError {
	var err error

	ids, err := idsFromRequest(r)
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	participantID := service.ParticipantIDFromContext(r.Context())

	var result []map[string]interface{}
	apiError := service.NoError
	var attemptID int64
	err = srv.GetStore(r).InTransaction(func(store *database.DataStore) error {
		result = getDataForResultPathStart(store, participantID, ids)
		if len(result) == 0 {
			apiError = service.InsufficientAccessRightsError
			return apiError.Error
		}

		data := result[0]
		rowsToInsert := make([]map[string]interface{}, 0, len(data))
		rowsToInsertPropagate := make([]map[string]interface{}, 0, len(data))
		for index, itemID := range ids {
			attemptID = data[fmt.Sprintf("attempt_id%d", index)].(int64)
			if data[fmt.Sprintf("has_started_result%d", index)].(int64) == 1 {
				continue
			}
			rowsToInsert = append(rowsToInsert, map[string]interface{}{
				"item_id":            itemID,
				"participant_id":     participantID,
				"attempt_id":         attemptID,
				"started_at":         database.Now(),
				"latest_activity_at": database.Now(),
			})
			rowsToInsertPropagate = append(rowsToInsertPropagate, map[string]interface{}{
				"item_id":        itemID,
				"participant_id": participantID,
				"attempt_id":     attemptID,
				"state":          "to_be_propagated",
			})
		}
		if len(rowsToInsert) > 0 {
			resultStore := store.Results()
			service.MustNotBeError(resultStore.InsertOrUpdateMaps(rowsToInsert, []string{"started_at", "latest_activity_at"}))
			service.MustNotBeError(resultStore.InsertIgnoreMaps("results_propagate", rowsToInsertPropagate))

			store.SchedulePropagation([]string{"results"})
		}

		return nil
	})
	if apiError != service.NoError {
		return apiError
	}
	service.MustNotBeError(err)

	service.MustNotBeError(render.Render(w, r, service.UpdateSuccess(map[string]interface{}{
		"attempt_id": strconv.FormatInt(attemptID, 10),
	})))
	return service.NoError
}

func hasAccessToItemPath(store *database.DataStore, participantID int64, ids []int64) bool {
	var count int64
	store.Permissions().
		MatchingGroupAncestors(participantID).
		Joins(`JOIN items ON items.id = permissions.item_id`).
		Where("items.id IN (?)", ids).
		WherePermissionIsAtLeast("view", "content").
		Select("COUNT(DISTINCT items.id) AS count").
		Count(&count)

	return count == int64(len(ids))
}

func getDataForResultPathStart(store *database.DataStore, participantID int64, ids []int64) []map[string]interface{} {
	var result []map[string]interface{}
	if !hasAccessToItemPath(store, participantID, ids) {
		return result
	}

	participantAncestors := store.ActiveGroupAncestors().Where("child_group_id = ?", participantID).
		Joins("JOIN `groups` ON groups.id = groups_ancestors_active.ancestor_group_id").
		Select("root_activity_id, root_skill_id")
	groupsManagedByParticipant := store.ActiveGroupAncestors().ManagedByGroup(participantID).
		Joins("JOIN `groups` ON groups.id = groups_ancestors_active.child_group_id").
		Select("root_activity_id, root_skill_id")
	rootActivities := participantAncestors.Select("groups.root_activity_id").Union(
		groupsManagedByParticipant.Select("groups.root_activity_id").SubQuery())
	rootSkills := participantAncestors.Select("groups.root_skill_id").Union(
		groupsManagedByParticipant.Select("groups.root_skill_id").SubQuery())

	query := store.Table("items as items0").WithWriteLock()
	for i := 0; i < len(ids); i++ {
		query = query.Where(fmt.Sprintf("items%d.id = ?", i), ids[i])
	}
	query = query.Where("items0.id IN ? OR items0.id IN ?", rootActivities.SubQuery(), rootSkills.SubQuery())

	var score string
	var columns string
	attemptIsActiveCondition := "1"
	var columnsForOrder string
	for i := 0; i < len(ids); i++ {
		var previousAttemptCondition string
		if i > 0 {
			score += " + "
			const comma = ", "
			columns += comma
			previousAttemptCondition = fmt.Sprintf(` AND
					IF(attempts%d.root_item_id = items%d.id, attempts%d.parent_attempt_id, attempts%d.id) = attempts%d.id`, i, i, i, i, i-1)
		}

		columnsForOrder += fmt.Sprintf(", attempts%d.id DESC", i)
		attemptIsActiveCondition = fmt.Sprintf(
			"attempts%d.ended_at IS NULL AND NOW() < attempts%d.allows_submissions_until AND %s", i, i, attemptIsActiveCondition)
		score += fmt.Sprintf("((results%d.started_at IS NULL) << %d)", i, len(ids)-i-1)
		query = query.
			Joins(fmt.Sprintf(`
				JOIN attempts AS attempts%d ON attempts%d.participant_id = ? AND
					(NOT items%d.requires_explicit_entry OR attempts%d.root_item_id = items%d.id)`+previousAttemptCondition, i, i, i, i, i),
				participantID).
			Joins(fmt.Sprintf(`
					LEFT JOIN results AS results%d ON results%d.participant_id = attempts%d.participant_id AND
						attempts%d.id = results%d.attempt_id AND results%d.item_id = items%d.id`, i, i, i, i, i, i, i)).
			Where(
				fmt.Sprintf("(NOT items%d.requires_explicit_entry OR results%d.attempt_id IS NOT NULL) AND (results%d.started_at IS NOT NULL OR %s)",
					i, i, i, attemptIsActiveCondition))

		if i != len(ids)-1 {
			query = query.Joins(fmt.Sprintf(
				"JOIN items_items AS items_items%d ON items_items%d.parent_item_id = items%d.id AND items_items%d.child_item_id = ?",
				i+1, i+1, i, i+1), ids[i+1]).
				Joins(fmt.Sprintf("JOIN items AS items%d ON items%d.id = items_items%d.child_item_id", i+1, i+1, i+1))
		}
		columns += fmt.Sprintf("attempts%d.id AS attempt_id%d, results%d.started_at IS NOT NULL AS has_started_result%d", i, i, i, i)
	}
	query = query.Select(columns).Where("results0.attempt_id IS NOT NULL OR attempts0.id = 0").
		Order(score + columnsForOrder).Limit(1)

	service.MustNotBeError(
		query.
			ScanIntoSliceOfMaps(&result).Error())

	return result
}
