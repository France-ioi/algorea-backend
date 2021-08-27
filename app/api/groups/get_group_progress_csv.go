package groups

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

const csvExportGroupProgressBatchSize = 20

// swagger:operation GET /groups/{group_id}/group-progress-csv groups groupGroupProgressCSV
// ---
// summary: Get group progress as a CSV file
// description: >
//              Returns the current progress of a group on a subset of items.
//
//
//              For each item from `{parent_item_id}` and its visible children, displays the average result
//              of each direct child of the given `group_id` whose type is not in (Team, User).
//
//
//              Restrictions:
//
//              * The current user should be a manager of the group (or of one of its ancestors)
//              with `can_watch_members` set to true,
//
//              * The current user should have `can_watch_members` >= 'result' on each of `{parent_item_ids}` items,
//
//
//              otherwise the 'forbidden' error is returned.
// parameters:
// - name: group_id
//   in: path
//   type: integer
//   required: true
// - name: parent_item_ids
//   in: query
//   type: array
//   required: true
//   items:
//     type: integer
// responses:
//   "200":
//     description: OK. Success response with users progress on items
//     content:
//       text/csv:
//         schema:
//            type: string
//     examples:
//            text/csv:
//              Group name;Parent item;1. First child item;2. Second child item
//
//              Our group;30;20;10
//   "400":
//     "$ref": "#/responses/badRequestResponse"
//   "401":
//     "$ref": "#/responses/unauthorizedResponse"
//   "403":
//     "$ref": "#/responses/forbiddenResponse"
//   "500":
//     "$ref": "#/responses/internalErrorResponse"
func (srv *Service) getGroupProgressCSV(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)

	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	if apiError := checkThatUserCanWatchGroupMembers(srv.Store, user, groupID); apiError != service.NoError {
		return apiError
	}

	itemParentIDs, apiError := srv.resolveAndCheckParentIDs(r, user)
	if apiError != service.NoError {
		return apiError
	}

	w.Header().Set("Content-Type", "text/csv")
	itemParentIDsString := make([]string, len(itemParentIDs))
	for i, id := range itemParentIDs {
		itemParentIDsString[i] = strconv.FormatInt(id, 10)
	}
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=groups_progress_for_group_%d_and_child_items_of_%s.csv",
			groupID, strings.Join(itemParentIDsString, "_")))
	if len(itemParentIDs) == 0 {
		_, err := w.Write([]byte("Group name\n"))
		service.MustNotBeError(err)
		return service.NoError
	}

	// Preselect item IDs since we need them to build the results table (there shouldn't be many)
	orderedItemIDListWithDuplicates, uniqueItemIDs, itemOrder, itemsSubQuery := srv.preselectIDsOfVisibleItems(itemParentIDs, user)

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	csvWriter.Comma = ';'

	srv.printTableHeader(user, uniqueItemIDs, orderedItemIDListWithDuplicates, itemOrder, csvWriter,
		[]string{"Group name"})

	// Preselect groups for that we will calculate the stats.
	// All the "end members" are descendants of these groups.
	var groups []struct {
		ID   int64
		Name string
	}

	service.MustNotBeError(srv.Store.ActiveGroupGroups().
		Where("groups_groups_active.parent_group_id = ?", groupID).
		Joins(`
			JOIN ` + "`groups`" + ` AS group_child
			ON group_child.id = groups_groups_active.child_group_id AND group_child.type NOT IN('Team', 'User')`).
		Order("group_child.name, group_child.id").
		Select("group_child.id, group_child.name").
		Scan(&groups).Error())

	if len(groups) == 0 {
		return service.NoError
	}

	ancestorGroupIDs := make([]string, len(groups))
	for i := range groups {
		ancestorGroupIDs[i] = strconv.FormatInt(groups[i].ID, 10)
	}

	for startFromGroup := 0; startFromGroup < len(ancestorGroupIDs); startFromGroup += csvExportGroupProgressBatchSize {
		batchBoundary := startFromGroup + csvExportGroupProgressBatchSize
		if batchBoundary > len(ancestorGroupIDs) {
			batchBoundary = len(ancestorGroupIDs)
		}
		ancestorsInBatch := ancestorGroupIDs[startFromGroup:batchBoundary]
		ancestorsInBatchIDsList := strings.Join(ancestorsInBatch, ", ")
		currentRowNumber := 0
		var rowArray []string
		cellsMap := make(map[int64]string, len(orderedItemIDListWithDuplicates))

		endMembers := srv.Store.Groups().
			Select("groups.id").
			Joins(`
				JOIN groups_ancestors_active
				ON groups_ancestors_active.ancestor_group_id IN (?) AND
					groups_ancestors_active.child_group_id = groups.id`, ancestorsInBatch).
			Where("groups.type = 'Team' OR groups.type = 'User'").
			Group("groups.id")

		endMembersStats := srv.Store.Raw(`
		SELECT
			end_members.id,
			items.id AS item_id,
			IFNULL((
				SELECT score_computed AS score
				FROM results
				WHERE participant_id = end_members.id AND item_id = items.id
				ORDER BY participant_id, item_id, score_computed DESC, score_obtained_at
				LIMIT 1
			), 0) AS score
		FROM ? AS end_members`, endMembers.SubQuery()).
			Joins("JOIN ? AS items", itemsSubQuery)

		service.MustNotBeError(srv.Store.ActiveGroupAncestors().
			Select(`
				groups_ancestors_active.ancestor_group_id AS group_id,
				member_stats.item_id,
				AVG(member_stats.score) AS score`).
			Joins("JOIN ? AS member_stats ON member_stats.id = groups_ancestors_active.child_group_id", endMembersStats.SubQuery()).
			Where("groups_ancestors_active.ancestor_group_id IN (?)", ancestorsInBatch).
			Group("groups_ancestors_active.ancestor_group_id, member_stats.item_id").
			Order(gorm.Expr(
				"FIELD(groups_ancestors_active.ancestor_group_id, " + ancestorsInBatchIDsList + ")")).
			ScanAndHandleMaps(processCSVResultRow(orderedItemIDListWithDuplicates, &currentRowNumber, len(uniqueItemIDs), startFromGroup,
				func(groupNumber int) []string {
					return []string{groups[groupNumber].Name}
				}, &rowArray, &cellsMap, csvWriter)).Error())
	}

	return service.NoError
}
