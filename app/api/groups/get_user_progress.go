package groups

import (
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:model groupUserProgressResponseTableCell
type groupUserProgressResponseTableCell struct {
	// The user’s self `group_id`
	// required:true
	GroupID int64 `json:"group_id,string"`
	// required:true
	ItemID int64 `json:"item_id,string"`
	// The best score across all user's or user teams' results. If there are no results, the score is 0.
	// required:true
	Score float32 `json:"score"`
	// Whether the user or one of his teams has the item validated
	// required:true
	Validated bool `json:"validated"`
	// Nullable
	// required:true
	LatestActivityAt *database.Time `json:"latest_activity_at"`
	// Number of hints requested for the result with the best score (if multiple, take the first one, chronologically).
	// If there are no results, the number of hints is 0.
	// required:true
	HintsRequested int32 `json:"hints_requested"`
	// Number of submissions for the result with the best score (if multiple, take the first one, chronologically).
	// If there are no results, the number of submissions is 0.
	// required:true
	Submissions int32 `json:"submissions"`
	// Time spent by the user (or his teams) (in seconds):
	//
	//   1) if no results yet: 0
	//
	//   2) if one result validated: min(`validated_at`) - min(`started_at`)
	//     (i.e., time between the first time the user (or one of his teams) started one (any) result
	//      and the time he (or one of his teams) first validated the task)
	//
	//   3) if no results validated: `now` - min(`started_at`)
	// required:true
	TimeSpent int32 `json:"time_spent"`
}

// swagger:operation GET /groups/{group_id}/user-progress groups groupUserProgress
//
//		---
//		summary: Get group progress for users
//		description: >
//	             Returns the current progress of users on a subset of items.
//
//
//	             For each item from `{parent_item_id}` and its visible children,
//	             displays the result of all user self-groups among the descendants of the given group
//	             (including those in teams).
//
//
//	             For each user, only the result corresponding to his best score counts
//	             (across all his teams and his own results) disregarding whether or not
//	             the score was done in a team which is descendant of the input group.
//
//
//	             Restrictions:
//
//	             * The current user should be a manager of the group (or of one of its ancestors)
//	             with `can_watch_members` set to true,
//
//	             * The current user should have `can_watch` >= 'result' on each of `{parent_item_ids}` items,
//
//
//	             otherwise the 'forbidden' error is returned.
//		parameters:
//			- name: group_id
//				in: path
//				type: integer
//				required: true
//			- name: parent_item_ids
//				required: true
//				in: query
//				type: array
//				items:
//					type: integer
//			- name: from.id
//				description: Start the page from the user next to the user with `groups.id`=`{from.id}`
//				in: query
//				type: integer
//			- name: limit
//				description: Display results for the first N users (sorted by `groups.name`)
//				in: query
//				type: integer
//				maximum: 1000
//				default: 500
//		responses:
//			"200":
//				description: OK. Success response with users progress on items
//				schema:
//					type: array
//					items:
//						"$ref": "#/definitions/groupUserProgressResponseTableCell"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getUserProgress(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)
	store := srv.GetStore(r)

	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	if !user.CanWatchGroupMembers(store, groupID) {
		return service.InsufficientAccessRightsError
	}

	itemParentIDs, apiError := resolveAndCheckParentIDs(store, r, user)
	if apiError != service.NoError {
		return apiError
	}
	if len(itemParentIDs) == 0 {
		render.Respond(w, r, []map[string]interface{}{})
		return service.NoError
	}

	// Preselect item IDs since we need them to build the results table (there shouldn't be many)
	orderedItemIDListWithDuplicates, uniqueItemIDs, _, itemsSubQuery := preselectIDsOfVisibleItems(store, itemParentIDs, user)

	// Preselect IDs of end member for that we will calculate the stats.
	// There should not be too many of end members on one page.
	var userIDs []string
	userIDQuery := store.ActiveGroupAncestors().
		Joins("JOIN groups_groups_active ON groups_groups_active.parent_group_id = groups_ancestors_active.child_group_id").
		Joins("JOIN `groups` ON groups.id = groups_groups_active.child_group_id AND groups.type = 'User'").
		Where("groups_ancestors_active.ancestor_group_id = ?", groupID).
		Group("groups.id")
	userIDQuery, apiError = service.ApplySortingAndPaging(
		r, userIDQuery,
		&service.SortingAndPagingParameters{
			Fields: service.SortingAndPagingFields{
				"name": {ColumnName: "groups.name"},
				"id":   {ColumnName: "groups.id"},
			},
			DefaultRules: "name,id",
			TieBreakers:  service.SortingAndPagingTieBreakers{"id": service.FieldTypeInt64},
		})
	if apiError != service.NoError {
		return apiError
	}
	userIDQuery = service.NewQueryLimiter().Apply(r, userIDQuery)
	service.MustNotBeError(userIDQuery.
		Pluck("groups.id", &userIDs).Error())

	if len(userIDs) == 0 {
		render.Respond(w, r, []map[string]interface{}{})
		return service.NoError
	}

	userIDsList := strings.Join(userIDs, ", ")
	var result []*groupUserProgressResponseTableCell
	scanAndBuildProgressResults(
		// nolint:gosec
		joinUserProgressResults(
			store.Raw(`
				SELECT STRAIGHT_JOIN
					items.id AS item_id,
					users.group_id AS group_id, `+userProgressFields+`
				FROM JSON_TABLE('[`+userIDsList+`]', "$[*]" COLUMNS(group_id BIGINT PATH "$")) AS users`).
				Joins("JOIN ? AS items", itemsSubQuery),
			gorm.Expr("users.group_id"),
		).
			Group("users.group_id, items.id").
			Order(gorm.Expr("FIELD(users.group_id, "+userIDsList+")")),
		orderedItemIDListWithDuplicates, len(uniqueItemIDs), &result,
	)

	render.Respond(w, r, result)
	return service.NoError
}

const userProgressFields = `
	IFNULL(MAX(result_with_best_score.score_computed), 0) AS score,
	IFNULL(MAX(result_with_best_score.validated), 0) AS validated,
	MAX(last_result.latest_activity_at) AS latest_activity_at,
	IFNULL(MAX(result_with_best_score.hints_cached), 0) AS hints_requested,
	IFNULL(MAX(result_with_best_score.submissions), 0) AS submissions,
	IF(MAX(result_with_best_score.participant_id) IS NULL,
		0,
		GREATEST(IF(MAX(result_with_best_score.validated),
			TIMESTAMPDIFF(SECOND, MIN(first_result.started_at), MIN(first_validated_result.validated_at)),
			TIMESTAMPDIFF(SECOND, MIN(first_result.started_at), NOW())
		), 0)
	) AS time_spent`

func joinUserProgressResults(db *database.DB, userID interface{}) *database.DB {
	return db.
		Joins(`
			LEFT JOIN LATERAL (
				SELECT STRAIGHT_JOIN groups.id
				FROM groups_groups_active
				JOIN `+"`groups`"+` ON groups.id = groups_groups_active.parent_group_id
				WHERE groups.type = 'Team' AND groups_groups_active.child_group_id = ?
			) teams ON 1`, userID).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT participant_id, attempt_id, score_computed, score_obtained_at
				FROM results AS result_with_best_score_for_user
				WHERE participant_id = ? AND item_id = items.id
				ORDER BY participant_id, item_id, score_computed DESC, score_obtained_at
				LIMIT 1
			) AS result_with_best_score_for_user ON 1`, userID).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT participant_id, attempt_id, score_computed, score_obtained_at
				FROM results AS result_with_best_score_for_team
				WHERE participant_id = teams.id AND item_id = items.id
				ORDER BY participant_id, item_id, score_computed DESC, score_obtained_at
				LIMIT 1
			) AS result_with_best_score_for_team ON 1`).
		Joins(`
			LEFT JOIN results AS result_with_best_score
			ON result_with_best_score.participant_id = IF(result_with_best_score_for_team.score_computed IS NOT NULL AND
				result_with_best_score_for_user.score_computed IS NOT NULL AND (
				result_with_best_score_for_team.score_computed > result_with_best_score_for_user.score_computed OR
					(
						result_with_best_score_for_team.score_computed = result_with_best_score_for_user.score_computed AND
						result_with_best_score_for_team.score_obtained_at < result_with_best_score_for_user.score_obtained_at
					)
				) OR result_with_best_score_for_user.score_computed IS NULL,
				result_with_best_score_for_team.participant_id,
				result_with_best_score_for_user.participant_id
			) AND result_with_best_score.attempt_id = IF(result_with_best_score_for_team.score_computed IS NOT NULL AND
				result_with_best_score_for_user.score_computed IS NOT NULL AND (
				result_with_best_score_for_team.score_computed > result_with_best_score_for_user.score_computed OR
					(
						result_with_best_score_for_team.score_computed = result_with_best_score_for_user.score_computed AND
						result_with_best_score_for_team.score_obtained_at < result_with_best_score_for_user.score_obtained_at
					)
				) OR result_with_best_score_for_user.score_computed IS NULL,
				result_with_best_score_for_team.attempt_id,
				result_with_best_score_for_user.attempt_id
			) AND result_with_best_score.item_id = items.id`).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT participant_id, attempt_id, latest_activity_at
				FROM results AS last_result_of_user USE INDEX (participant_id_item_id_latest_activity_at_desc)
				WHERE participant_id = ? AND item_id = items.id AND latest_activity_at IS NOT NULL
				ORDER BY participant_id, item_id, latest_activity_at DESC LIMIT 1
			) AS last_result_of_user ON 1`, userID).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT participant_id, attempt_id, latest_activity_at
				FROM results AS last_result_of_team USE INDEX (participant_id_item_id_latest_activity_at_desc)
				WHERE participant_id = teams.id AND item_id = items.id AND latest_activity_at IS NOT NULL
				ORDER BY participant_id, item_id, latest_activity_at DESC LIMIT 1
			) AS last_result_of_team ON 1`).
		Joins(`
			JOIN LATERAL (
				SELECT
					IF(
						(
							last_result_of_team.participant_id IS NOT NULL AND
							last_result_of_user.participant_id IS NOT NULL AND
							last_result_of_team.latest_activity_at > last_result_of_user.latest_activity_at
						) OR last_result_of_user.participant_id IS NULL,
						last_result_of_team.latest_activity_at,
						last_result_of_user.latest_activity_at
					) AS latest_activity_at
				)  AS last_result ON 1`).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT participant_id, attempt_id, started_at
				FROM results AS first_result_of_user USE INDEX (participant_id_item_id_started_started_at)
				WHERE participant_id = ? AND item_id = items.id AND started = 1
				ORDER BY participant_id, item_id, started, started_at LIMIT 1
			) AS first_result_of_user ON 1`, userID).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT participant_id, attempt_id, started_at
				FROM results AS first_result_of_team USE INDEX (participant_id_item_id_started_started_at)
				WHERE participant_id = teams.id AND item_id = items.id AND started = 1
				ORDER BY participant_id, item_id, started, started_at LIMIT 1
			) AS first_result_of_team ON 1`).
		Joins(`
			JOIN LATERAL (
				SELECT
					IF(
						(
							first_result_of_team.participant_id IS NOT NULL AND
							first_result_of_user.participant_id IS NOT NULL AND
							first_result_of_team.started_at < first_result_of_user.started_at
						) OR first_result_of_user.participant_id IS NULL,
						first_result_of_team.started_at,
						first_result_of_user.started_at
					) AS started_at
			) AS first_result ON 1`).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT participant_id, attempt_id, validated_at
				FROM results AS first_validated_result_of_user USE INDEX (participant_id_item_id_validated_validated_at)
				WHERE participant_id = ? AND item_id = items.id AND validated = 1
				ORDER BY participant_id, item_id, validated, validated_at LIMIT 1
			) AS first_validated_result_of_user ON 1`, userID).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT participant_id, attempt_id, validated_at
				FROM results AS first_validated_result_of_team USE INDEX (participant_id_item_id_validated_validated_at)
				WHERE participant_id = teams.id AND item_id = items.id AND validated = 1
				ORDER BY participant_id, item_id, validated, validated_at LIMIT 1
			) AS first_validated_result_of_team ON 1`).
		Joins(`
			JOIN LATERAL (
				SELECT
					IF(
						(
							first_validated_result_of_team.participant_id IS NOT NULL AND
							first_validated_result_of_user.participant_id IS NOT NULL AND
							first_validated_result_of_team.validated_at < first_validated_result_of_user.validated_at
						) OR first_result_of_user.participant_id IS NULL,
						first_validated_result_of_team.validated_at,
						first_validated_result_of_user.validated_at
					) AS validated_at
				) AS first_validated_result ON 1`)
}
