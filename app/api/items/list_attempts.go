package items

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
	"github.com/France-ioi/AlgoreaBackend/app/structures"
)

// swagger:model attemptsListResponseRow
type attemptsListResponseRow struct {
	// required: true
	ID int64 `json:"id,string"`
	// required: true
	CreatedAt database.Time `json:"created_at"`
	// required: true
	ScoreComputed float32 `json:"score_computed"`
	// required: true
	Validated bool `json:"validated"`
	// Nullable
	// required: true
	StartedAt *database.Time `json:"started_at"`
	// Nullable
	// required: true
	EndedAt *database.Time `json:"ended_at"`
	// required: true
	AllowsSubmissionsUntil database.Time `json:"allows_submissions_until"`
	// required: true
	LatestActivityAt database.Time `json:"latest_activity_at"`
	// required: true
	HelpRequested bool `json:"help_requested"`
	UserCreator   *struct {
		// required: true
		Login string `json:"login"`

		*structures.UserPersonalInfo
		ShowPersonalInfo bool `json:"-"`

		// required: true
		GroupID *int64 `json:"group_id,string"`
	} `json:"user_creator" gorm:"embedded;embeddedPrefix:user_creator__"`
}

// swagger:operation GET /items/{item_id}/attempts items attemptsList
//
//		---
//		summary: List attempts/results for an item
//		description: >
//	   Returns attempts of the current participant (the current user or `{as_team_id}` team) with their results
//	   for the given item within the parent attempt.
//
//
//	   `first_name` and `last_name` of attempt creators are only visible to attempt creators themselves and
//	   to managers of those attempt creators' groups to which they provided view access to personal data.
//
//
//	   Restrictions:
//	     * `{as_team_id}` (if given) should be the current user's team,
//	     * the participant should have at least 'content' access on the item,
//	     * if `{attempt_id}` is given, it should exist for the participant in order to determine `{parent_attempt_id}`
//	       (we assume that the 'zero attempt' always exists and it is its own parent attempt),
//
//	   otherwise the 'forbidden' error is returned.
//		parameters:
//			- name: item_id
//				in: path
//				type: integer
//				format: int64
//				required: true
//			- name: parent_attempt_id
//				description: "`id` of a parent attempt. This parameter is incompatible with `attempt_id`."
//				in: query
//				type: integer
//			- name: attempt_id
//				description: "`id` of an attempt for the `{item_id}`.
//	                This parameter is incompatible with `parent_attempt_id`."
//				in: query
//				type: integer
//			- name: as_team_id
//				in: query
//				type: integer
//			- name: sort
//				in: query
//				default: [id]
//				type: array
//				items:
//					type: string
//					enum: [id,-id]
//			- name: from.id
//				description: Start the page from the attempt next to the attempt with `results.attempt_id` = `{from.id}`
//				in: query
//				type: integer
//				format: int64
//			- name: limit
//				description: Display first N attempts
//				in: query
//				type: integer
//				maximum: 1000
//				default: 500
//		responses:
//			"200":
//				description: OK. Success response with an array of attempts
//				schema:
//					type: array
//					items:
//						"$ref": "#/definitions/attemptsListResponseRow"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) listAttempts(w http.ResponseWriter, r *http.Request) service.APIError {
	itemID, groupID, parentAttemptID, apiError := srv.resolveParametersForListAttempts(r)
	if apiError != service.NoError {
		return apiError
	}
	user := srv.GetUser(r)

	query := srv.GetStore(r).Results().Where("results.participant_id = ?", groupID).
		Where("item_id = ?", itemID).
		Joins("JOIN attempts ON attempts.participant_id = results.participant_id AND attempts.id = results.attempt_id").
		Joins("LEFT JOIN users ON users.group_id = attempts.creator_id").
		Where("attempts.id = ? OR attempts.parent_attempt_id = ?", parentAttemptID, parentAttemptID).
		WithPersonalInfoViewApprovals(user).
		Select(`
			attempts.id, attempts.created_at, attempts.allows_submissions_until,
			results.score_computed, results.validated, attempts.ended_at,
			results.started_at, results.latest_activity_at, results.help_requested,
			users.login AS user_creator__login,
			users.group_id = ? OR personal_info_view_approvals.approved AS user_creator__show_personal_info,
			IF(users.group_id = ? OR personal_info_view_approvals.approved, users.first_name, NULL) AS user_creator__first_name,
			IF(users.group_id = ? OR personal_info_view_approvals.approved, users.last_name, NULL) AS user_creator__last_name,
			users.group_id AS user_creator__group_id`, user.GroupID, user.GroupID, user.GroupID)
	query = service.NewQueryLimiter().Apply(r, query)
	query, apiError = service.ApplySortingAndPaging(
		r, query,
		&service.SortingAndPagingParameters{
			Fields: service.SortingAndPagingFields{
				"id": {ColumnName: "results.attempt_id"},
			},
			DefaultRules: "id",
			TieBreakers:  service.SortingAndPagingTieBreakers{"id": service.FieldTypeInt64},
		})
	if apiError != service.NoError {
		return apiError
	}
	var result []attemptsListResponseRow
	service.MustNotBeError(query.Scan(&result).Error())

	for index := range result {
		if result[index].UserCreator.GroupID == nil {
			result[index].UserCreator = nil
		} else if !result[index].UserCreator.ShowPersonalInfo {
			result[index].UserCreator.UserPersonalInfo = nil
		}
	}

	render.Respond(w, r, result)
	return service.NoError
}

func (srv *Service) resolveParametersForListAttempts(r *http.Request) (
	itemID, participantID, parentAttemptID int64, apiError service.APIError,
) {
	itemID, err := service.ResolveURLQueryPathInt64Field(r, "item_id")
	if err != nil {
		return 0, 0, 0, service.ErrInvalidRequest(err)
	}

	attemptID, parentAttemptID, attemptIDSet, apiError := attemptIDOrParentAttemptID(r)
	if apiError != service.NoError {
		return 0, 0, 0, apiError
	}

	participantID = service.ParticipantIDFromContext(r.Context())
	store := srv.GetStore(r)

	if attemptIDSet {
		if attemptID != 0 {
			var result struct{ ParentAttemptID int64 }
			err = store.Attempts().
				Where("attempts.participant_id = ? AND attempts.id = ?", participantID, attemptID).
				Select("IF(attempts.root_item_id = ?, attempts.parent_attempt_id, attempts.id) AS parent_attempt_id", itemID).
				Take(&result).Error()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, 0, 0, service.InsufficientAccessRightsError
			}
			service.MustNotBeError(err)
			parentAttemptID = result.ParentAttemptID
		}
	}

	found, err := store.Permissions().MatchingGroupAncestors(participantID).
		WherePermissionIsAtLeast("view", "content").
		Where("item_id = ?", itemID).HasRows()

	service.MustNotBeError(err)
	if !found {
		return 0, 0, 0, service.InsufficientAccessRightsError
	}
	return itemID, participantID, parentAttemptID, service.NoError
}
