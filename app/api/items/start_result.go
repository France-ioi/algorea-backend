package items

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation POST /items/{ids}/start-result items resultStart
//
//	---
//	summary: Start a result
//	description: >
//		Creates a new started result for the given item and attempt or sets `started_at` of an existing result (if it hasn't been set).
//		If `as_team_id` is given, the created result is linked to the `as_team_id` group instead of the user's self group.
//
//
//			Restrictions:
//
//		* if `as_team_id` is given, it should be a user's parent team group,
//		* the first item in `{ids}` should be a root activity/skill (groups.root_activity_id/root_skill_id) of a group
//			the participant is a descendant of or manages,
//		* the last item in `{ids}` should not require explicit entry (`items.requires_explicit_entry` should be false),
//		* `{ids}` should be an ordered list of parent-child items,
//		* the group starting the result should have at least 'content' access on each of the items in `{ids}`,
//		* the participant should have a started, allowing submission, not ended result for each item but the last,
//			with `{attempt_id}` (or its parent attempt each time we reach a root of an attempt) as the attempt,
//		* if `{ids}` consists of only one item, the `{attempt_id}` should be zero,
//
//		otherwise the 'forbidden' error is returned.
//	parameters:
//		- name: ids
//			in: path
//			type: string
//			description: slash-separated list of item IDs
//			required: true
//		- name: attempt_id
//			in: query
//			type: integer
//			required: true
//		- name: as_team_id
//			in: query
//			type: integer
//	responses:
//		"201":
//			"$ref": "#/responses/updatedResponse"
//		"400":
//			"$ref": "#/responses/badRequestResponse"
//		"401":
//			"$ref": "#/responses/unauthorizedResponse"
//		"403":
//			"$ref": "#/responses/forbiddenResponse"
//		"500":
//			"$ref": "#/responses/internalErrorResponse"
func (srv *Service) startResult(w http.ResponseWriter, r *http.Request) service.APIError {
	var err error

	ids, err := idsFromRequest(r)
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	attemptID, err := service.ResolveURLQueryGetInt64Field(r, "attempt_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	participantID := service.ParticipantIDFromContext(r.Context())

	apiError := service.NoError
	err = srv.GetStore(r).InTransaction(func(store *database.DataStore) error {
		var ok bool
		ok, err = store.Items().IsValidParticipationHierarchyForParentAttempt(ids, participantID, attemptID, true, true)
		service.MustNotBeError(err)
		if !ok {
			apiError = service.InsufficientAccessRightsError
			return apiError.Error // rollback
		}

		itemID := ids[len(ids)-1]
		var found bool
		found, err = store.Items().ByID(itemID).
			Where("NOT items.requires_explicit_entry").WithWriteLock().HasRows()
		service.MustNotBeError(err)
		if !found {
			apiError = service.InsufficientAccessRightsError
			return apiError.Error // rollback
		}

		result := store.Exec(`
			INSERT INTO results (participant_id, attempt_id, item_id, started_at, latest_activity_at)
			VALUES (?, ?, ?, NOW(), NOW())
			ON DUPLICATE KEY UPDATE
				latest_activity_at = IF(started_at IS NULL, NOW(), latest_activity_at),
				started_at = IFNULL(started_at, NOW())`,
			participantID, attemptID, itemID)
		service.MustNotBeError(result.Error())

		if result.RowsAffected() != 0 {
			resultStore := store.Results()
			service.MustNotBeError(resultStore.MarkAsToBePropagated(participantID, attemptID, itemID, false))

			store.SchedulePropagation([]string{"results"})
		}
		return nil
	})
	if apiError != service.NoError {
		return apiError
	}
	service.MustNotBeError(err)

	service.MustNotBeError(render.Render(w, r, service.UpdateSuccess(nil)))
	return service.NoError
}
