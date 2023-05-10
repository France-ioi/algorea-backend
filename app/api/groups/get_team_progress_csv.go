package groups

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation GET /groups/{group_id}/team-progress-csv groups groupTeamProgressCSV
//
//		---
//		summary: Get group progress for teams as a CSV file
//		description: >
//	             Returns the current progress of teams on a subset of items.
//
//
//	             For each item from `{parent_item_id}` and its visible children,
//	             displays the result of each team among the descendants of the group.
//
//
//	             Restrictions:
//
//	             * The current user should be a manager of the group (or of one of its ancestors)
//	             with `can_watch_members` set to true,
//
//	             * The current user should have `can_watch_members` >= 'result' on each of `{parent_item_ids}` items,
//
//
//	             otherwise the 'forbidden' error is returned.
//		parameters:
//			- name: group_id
//				in: path
//				type: integer
//				required: true
//			- name: parent_item_ids
//				in: query
//				required: true
//				type: array
//				items:
//					type: integer
//		responses:
//			"200":
//				description: OK. Success response with users progress on items
//				content:
//					text/csv:
//						schema:
//						type: string
//				examples:
//					text/csv:
//						Team name;Parent item;1. First child item;2. Second child item
//
//						Our team;30;20;10
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getTeamProgressCSV(w http.ResponseWriter, r *http.Request) service.APIError {
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

	w.Header().Set("Content-Type", "text/csv")
	itemParentIDsString := make([]string, len(itemParentIDs))
	for i, id := range itemParentIDs {
		itemParentIDsString[i] = strconv.FormatInt(id, 10)
	}
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=teams_progress_for_group_%d_and_child_items_of_%s.csv",
			groupID, strings.Join(itemParentIDsString, "_")))
	if len(itemParentIDs) == 0 {
		_, err := w.Write([]byte("Team name\n"))
		service.MustNotBeError(err)
		return service.NoError
	}

	// Preselect item IDs since we need them to build the results table (there shouldn't be many)
	orderedItemIDListWithDuplicates, uniqueItemIDs, itemOrder, itemsSubQuery := preselectIDsOfVisibleItems(store, itemParentIDs, user)

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	csvWriter.Comma = ';'

	printTableHeader(store, user, uniqueItemIDs, orderedItemIDListWithDuplicates, itemOrder, csvWriter,
		[]string{"Team name"})

	// Preselect teams for that we will calculate the stats.
	var teams []struct {
		ID   int64
		Name string
	}
	service.MustNotBeError(store.ActiveGroupAncestors().
		Joins("JOIN `groups` ON groups.id = groups_ancestors_active.child_group_id AND groups.type = 'Team'").
		Where("groups_ancestors_active.ancestor_group_id = ?", groupID).
		Where("groups_ancestors_active.child_group_id != groups_ancestors_active.ancestor_group_id").
		Order("groups.name, groups.id").
		Select("groups.id, groups.name").
		Scan(&teams).Error())

	if len(teams) == 0 {
		return service.NoError
	}
	teamIDs := make([]string, len(teams))
	for i := range teams {
		teamIDs[i] = strconv.FormatInt(teams[i].ID, 10)
	}

	for startFromTeam := 0; startFromTeam < len(teamIDs); startFromTeam += csvExportBatchSize {
		batchBoundary := startFromTeam + csvExportBatchSize
		if batchBoundary > len(teamIDs) {
			batchBoundary = len(teamIDs)
		}
		teamIDsList := strings.Join(teamIDs[startFromTeam:batchBoundary], ", ")
		teamNumber := startFromTeam
		// nolint:gosec
		service.MustNotBeError(store.Raw(`
				SELECT
				items.id AS item_id,
				groups.id AS group_id,
				result_with_best_score.score_computed AS score
				FROM JSON_TABLE('[`+teamIDsList+`]', "$[*]" COLUMNS(id BIGINT PATH "$")) AS `+"`groups`").
			Joins("JOIN ? AS items", itemsSubQuery).
			Joins(`
				LEFT JOIN LATERAL (
					SELECT score_computed, validated, hints_cached, submissions, participant_id
					FROM results
					WHERE participant_id = groups.id AND item_id = items.id
					ORDER BY participant_id, item_id, score_computed DESC, score_obtained_at
					LIMIT 1
				) AS result_with_best_score ON 1`).
			Order(gorm.Expr(
				"FIELD(groups.id, " + teamIDsList + ")")).
			ScanAndHandleMaps(
				processCSVResultRow(
					orderedItemIDListWithDuplicates, len(uniqueItemIDs), &teamNumber,
					func(_ int64) []string {
						return []string{teams[teamNumber].Name}
					}, csvWriter)).Error())
	}

	return service.NoError
}
