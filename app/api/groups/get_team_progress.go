package groups

import (
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/jinzhu/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:model groupTeamProgressResponseRow
type groupTeamProgressResponseRow struct {
	// The team’s `group_id`
	// required:true
	GroupID int64 `json:"group_id,string"`
	// required:true
	ItemID int64 `json:"item_id,string"`
	// Current score. If there are no attempts, the score is 0
	// required:true
	Score float32 `json:"score"`
	// Whether the team has the item validated
	// required:true
	Validated bool `json:"validated"`
	// Nullable
	// required:true
	LastActivityDate *database.Time `json:"last_activity_date"`
	// Number of hints requested for the attempt with the best score (if multiple, take the first one, chronologically).
	// If there are no attempts, the number of hints is 0.
	// required:true
	HintsRequested int32 `json:"hints_requested"`
	// Number of submissions for the attempt with the best score (if multiple, take the first one, chronologically).
	// If there are no attempts, the number of submissions is 0.
	// required:true
	SubmissionsAttempts int32 `json:"submissions_attempts"`
	// Time spent by the team (in seconds):
	//
	//   1) if no attempts yet: 0
	//
	//   2) if one attempt validated: min(`validation_date`) - min(`start_date`)
	//     (i.e., time between the first time it started one (any) attempt
	//      and the time he first validated the task)
	//
	//   3) if no attempts validated: `now` - min(`start_date`)
	// required:true
	TimeSpent int32 `json:"time_spent"`
}

// swagger:operation GET /groups/{group_id}/team-progress groups attempts items groupTeamProgress
// ---
// summary: Display the current progress of teams on a subset of items
// description: For all children of items from the parent_item_id list,
//              display the result of each team among the descendants of the group.
// parameters:
// - name: group_id
//   in: path
//   type: integer
//   required: true
// - name: parent_item_ids
//   in: query
//   required: true
//   type: array
//   items:
//     type: integer
// - name: from.name
//   description: Start the page from the team next to the team with `name` = `from.name` and `id` = `from.id`
//                (`from.id` is required when `from.name` is present)
//   in: query
//   type: string
// - name: from.id
//   description: Start the page from the team next to the team with `name`=`from.name` and `id`=`from.id`
//                (`from.name` is required when from.id is present)
//   in: query
//   type: integer
// - name: limit
//   description: Display results for the first N teams (sorted by `name`)
//   in: query
//   type: integer
//   maximum: 1000
//   default: 500
// responses:
//   "200":
//     description: OK. Success response with teams progress on items
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/groupTeamProgressResponseRow"
//   "400":
//     "$ref": "#/responses/badRequestResponse"
//   "401":
//     "$ref": "#/responses/unauthorizedResponse"
//   "403":
//     "$ref": "#/responses/forbiddenResponse"
//   "500":
//     "$ref": "#/responses/internalErrorResponse"
func (srv *Service) getTeamProgress(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)

	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	if apiError := checkThatUserOwnsTheGroup(srv.Store, user, groupID); apiError != service.NoError {
		return apiError
	}

	itemParentIDs, err := service.ResolveURLQueryGetInt64SliceField(r, "parent_item_ids")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	itemsVisibleToUserSubQuery := srv.Store.GroupItems().AccessRightsForItemsVisibleToUser(user).SubQuery()

	// Preselect IDs of end member for that we will calculate the stats.
	// There should not be too many of end members on one page.
	var teamIDs []interface{}
	teamIDQuery := srv.Store.GroupAncestors().
		Joins("JOIN `groups` ON groups.id = groups_ancestors.group_child_id AND groups.type = 'Team'").
		Where("groups_ancestors.group_ancestor_id = ?", groupID).
		Where("groups_ancestors.group_child_id != groups_ancestors.group_ancestor_id")
	teamIDQuery, apiError := service.ApplySortingAndPaging(r, teamIDQuery, map[string]*service.FieldSortingParams{
		// Note that we require the 'from.name' request parameter although the service does not return group names
		"name": {ColumnName: "groups.name", FieldType: "string"},
		"id":   {ColumnName: "groups.id", FieldType: "int64"},
	}, "name,id")
	if apiError != service.NoError {
		return apiError
	}
	teamIDQuery = service.NewQueryLimiter().Apply(r, teamIDQuery)
	service.MustNotBeError(teamIDQuery.
		Pluck("groups.id", &teamIDs).Error())

	if len(teamIDs) == 0 {
		render.Respond(w, r, []map[string]interface{}{})
		return service.NoError
	}

	itemsQuery := srv.Store.ItemItems().
		Select("items_items.item_child_id").
		Where("item_parent_id IN (?)", itemParentIDs).
		Joins("JOIN ? AS visible ON visible.item_id = items_items.item_child_id", itemsVisibleToUserSubQuery)

	var result []groupTeamProgressResponseRow
	service.MustNotBeError(srv.Store.Groups().
		Select(`
			items.id AS item_id,
			groups.id AS group_id,
			IFNULL(attempt_with_best_score.score, 0) AS score,
			IFNULL(attempt_with_best_score.validated, 0) AS validated,
			(SELECT MAX(last_activity_date) FROM groups_attempts WHERE group_id = groups.id AND item_id = items.id) AS last_activity_date,
			IFNULL(attempt_with_best_score.hints_cached, 0) AS hints_requested,
			IFNULL(attempt_with_best_score.submissions_attempts, 0) AS submissions_attempts,
			IF(attempt_with_best_score.group_id IS NULL,
				0,
				(
					SELECT IF(attempt_with_best_score.validated,
						TIMESTAMPDIFF(SECOND, MIN(start_date), MIN(validation_date)),
						TIMESTAMPDIFF(SECOND, MIN(start_date), NOW())
					)
					FROM groups_attempts
					WHERE group_id = groups.id AND item_id = items.id
				)
			) AS time_spent`).
		Joins(`JOIN items ON items.id IN ?`, itemsQuery.SubQuery()).
		Joins(`
			LEFT JOIN groups_attempts AS attempt_with_best_score
			ON attempt_with_best_score.id = (
				SELECT id FROM groups_attempts
				WHERE group_id = groups.id AND item_id = items.id
				ORDER BY group_id, item_id, minus_score, best_answer_date LIMIT 1
			)`).
		Where("groups.id IN (?)", teamIDs).
		Order(gorm.Expr(
			"FIELD(groups.id"+strings.Repeat(", ?", len(teamIDs))+")",
			teamIDs...)).
		Order("items.id").
		Scan(&result).Error())

	render.Respond(w, r, result)
	return service.NoError
}
