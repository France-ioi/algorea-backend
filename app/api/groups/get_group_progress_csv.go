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

const csvExportGroupProgressBatchSize = 20

type idName struct {
	ID   int64
	Name string
}

// swagger:operation GET /groups/{group_id}/group-progress-csv groups groupGroupProgressCSV
//
//		---
//		summary: Get group progress as a CSV file
//		description: >
//	             Returns the current progress of a group on a subset of items.
//
//
//	             For each item from `{parent_item_id}` and its visible children, displays the average result
//	             of each direct child of the given `group_id` whose type is not in (Team, User).
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
//				type: array
//				required: true
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
//						Group name;Parent item;1. First child item;2. Second child item
//
//						Our group;30;20;10
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getGroupProgressCSV(w http.ResponseWriter, r *http.Request) service.APIError {
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
		fmt.Sprintf("attachment; filename=groups_progress_for_group_%d_and_child_items_of_%s.csv",
			groupID, strings.Join(itemParentIDsString, "_")))
	if len(itemParentIDs) == 0 {
		_, err := w.Write([]byte("Group name\n"))
		service.MustNotBeError(err)
		return service.NoError
	}

	// Preselect item IDs since we need them to build the results table (there shouldn't be many)
	orderedItemIDListWithDuplicates, uniqueItemIDs, itemOrder, itemsSubQuery := preselectIDsOfVisibleItems(store, itemParentIDs, user)

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	csvWriter.Comma = ';'

	printTableHeader(store, user, uniqueItemIDs, orderedItemIDListWithDuplicates, itemOrder, csvWriter,
		[]string{"Group name"})

	// Preselect groups for that we will calculate the stats.
	// All the "end members" are descendants of these groups.
	var groups []idName
	service.MustNotBeError(store.ActiveGroupGroups().
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

		endMembers := store.Groups().
			Select("groups.id").
			Joins(`
				JOIN groups_ancestors_active
				ON groups_ancestors_active.ancestor_group_id IN (?) AND
					groups_ancestors_active.child_group_id = groups.id`, ancestorsInBatch).
			Where("groups.type = 'Team' OR groups.type = 'User'").
			Group("groups.id")

		endMembersStats := store.Raw(`
		SELECT
			end_members.id,
			items.id AS item_id,
			(
				SELECT score_computed AS score
				FROM results
				WHERE participant_id = end_members.id AND item_id = items.id
				ORDER BY participant_id, item_id, score_computed DESC, score_obtained_at
				LIMIT 1
			) AS score
		FROM ? AS end_members`, endMembers.SubQuery()).
			Joins("JOIN ? AS items", itemsSubQuery)

		groupNumber := startFromGroup
		service.MustNotBeError(store.ActiveGroupAncestors().
			Select(`
				groups_ancestors_active.ancestor_group_id AS group_id,
				member_stats.item_id,
				IF(MAX(member_stats.score IS NOT NULL), AVG(IFNULL(member_stats.score, 0)), '') AS score`).
			Joins("JOIN ? AS member_stats ON member_stats.id = groups_ancestors_active.child_group_id", endMembersStats.SubQuery()).
			Where("groups_ancestors_active.ancestor_group_id IN (?)", ancestorsInBatch).
			Group("groups_ancestors_active.ancestor_group_id, member_stats.item_id").
			Order(gorm.Expr(
				"FIELD(groups_ancestors_active.ancestor_group_id, " + ancestorsInBatchIDsList + ")")).
			ScanAndHandleMaps(processCSVResultRow(orderedItemIDListWithDuplicates, len(uniqueItemIDs), &groupNumber,
				generateGroupNameAndWriteEmptyRowsForSkippedGroups(&groupNumber, groups, len(orderedItemIDListWithDuplicates), csvWriter),
				csvWriter)).Error())
		writeEmptyRowsForSkippedGroupsAtTheEnd(groupNumber, batchBoundary, groups, len(orderedItemIDListWithDuplicates), csvWriter)
	}

	return service.NoError
}

func generateGroupNameAndWriteEmptyRowsForSkippedGroups(
	groupNumber *int, groups []idName, numberOfItems int, csvWriter *csv.Writer,
) func(groupID int64) []string {
	return func(groupID int64) []string {
		for ; groups[*groupNumber].ID != groupID; *groupNumber++ {
			writeEmptyGroupProgressResultRow(csvWriter, groups[*groupNumber].Name, numberOfItems)
		}
		return []string{groups[*groupNumber].Name}
	}
}

func writeEmptyRowsForSkippedGroupsAtTheEnd(groupNumber, batchBoundary int, groups []idName, numberOfItems int, csvWriter *csv.Writer) {
	for ; groupNumber < batchBoundary; groupNumber++ {
		writeEmptyGroupProgressResultRow(csvWriter, groups[groupNumber].Name, numberOfItems)
	}
}

func writeEmptyGroupProgressResultRow(csvWriter *csv.Writer, groupName string, numberOfItems int) {
	service.MustNotBeError(
		csvWriter.Write(
			append([]string{groupName}, make([]string, numberOfItems)...)))
}
