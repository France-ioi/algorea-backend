package groups

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/France-ioi/mapstructure"
	"github.com/go-chi/render"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:model groupGroupProgressResponseTableCell
type groupGroupProgressResponseTableCell struct {
	// The child’s `group_id`
	// required:true
	GroupID int64 `json:"group_id,string"`
	// required:true
	ItemID int64 `json:"item_id,string"`
	// Average score of all "end-members".
	// The score of an "end-member" is the max of his `results.score` or 0 if no results.
	// required:true
	AverageScore float32 `json:"average_score"`
	// % (float [0,1]) of "end-members" who have validated the task.
	// An "end-member" has validated a task if one of his results has `results.validated` = 1.
	// No results for an "end-member" is considered as not validated.
	// required:true
	ValidationRate float32 `json:"validation_rate"`
	// Average number of hints requested by each "end-member".
	// The number of hints requested of an "end-member" is the `results.hints_cached`
	// of the result with the best score
	// (if several with the same score, we use the first result chronologically on `score_obtained_at`).
	// required:true
	AvgHintsRequested float32 `json:"avg_hints_requested"`
	// Average number of submissions made by each "end-member".
	// The number of submissions made by an "end-member" is the `results.submissions`.
	// of the result with the best score
	// (if several with the same score, we use the first result chronologically on `score_obtained_at`).
	// required:true
	AvgSubmissions float32 `json:"avg_submissions"`
	// Average time spent among all the "end-members" (in seconds). The time spent by an "end-member" is computed as:
	//
	//   1) if no results yet: 0
	//
	//   2) if one result validated: min(`validated_at`) - min(`started_at`)
	//     (i.e., time between the first time it started one (any) result
	//      and the time he first validated the task)
	//
	//   3) if no results validated: `now` - min(`started_at`)
	// required:true
	AvgTimeSpent float32 `json:"avg_time_spent"`
}

// swagger:operation GET /groups/{group_id}/group-progress groups groupGroupProgress
//
//		---
//		summary: Get group progress
//		description: >
//	             Returns the current progress of a group on a subset of items.
//
//
//	             For each item from `{parent_item_id}` and its visible children, displays the result
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
//			- name: from.id
//				description: Start the page from the group next to the group with `id`=`{from.id}`
//				in: query
//				type: integer
//			- name: limit
//				description: Display results for the first N groups (sorted by `name`)
//				in: query
//				type: integer
//				maximum: 20
//				default: 10
//		responses:
//			"200":
//				description: >
//					OK. Success response with groups progress on items
//					For each item from `{parent_item_id}` and its visible children, displays the result for each direct child
//					of the given group_id whose type is not in (Team, User). Values are averages of all the group's
//					"end-members" where “end-member” defined as descendants of the group which are either
//					1) teams or
//					2) users who descend from the input group not only through teams (one or more).
//				schema:
//					type: array
//					items:
//						"$ref": "#/definitions/groupGroupProgressResponseTableCell"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getGroupProgress(w http.ResponseWriter, r *http.Request) service.APIError {
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

	// Preselect item IDs since we want to use them twice (for end members stats and for final stats)
	// There should not be many of them
	orderedItemIDListWithDuplicates, uniqueItemIDs, _, itemsSubQuery := preselectIDsOfVisibleItems(store, itemParentIDs, user)

	// Preselect IDs of groups for that we will calculate the final stats.
	// All the "end members" are descendants of these groups.
	// There should not be too many of groups because we paginate on them.
	var ancestorGroupIDs []interface{}
	ancestorGroupIDQuery := store.ActiveGroupGroups().
		Where("groups_groups_active.parent_group_id = ?", groupID).
		Joins(`
			JOIN ` + "`groups`" + ` AS group_child
			ON group_child.id = groups_groups_active.child_group_id AND group_child.type NOT IN('Team', 'User')`)
	ancestorGroupIDQuery, apiError = service.ApplySortingAndPaging(
		r, ancestorGroupIDQuery,
		&service.SortingAndPagingParameters{
			Fields: service.SortingAndPagingFields{
				"name": {ColumnName: "group_child.name"},
				"id":   {ColumnName: "group_child.id"},
			},
			DefaultRules: "name,id",
			TieBreakers:  service.SortingAndPagingTieBreakers{"id": service.FieldTypeInt64},
		})
	if apiError != service.NoError {
		return apiError
	}
	ancestorGroupIDQuery = service.NewQueryLimiter().
		SetDefaultLimit(10).SetMaxAllowedLimit(20).
		Apply(r, ancestorGroupIDQuery)
	service.MustNotBeError(ancestorGroupIDQuery.
		Pluck("group_child.id", &ancestorGroupIDs).Error())
	if len(ancestorGroupIDs) == 0 {
		render.Respond(w, r, []map[string]interface{}{})
		return service.NoError
	}

	endMembers := store.Groups().
		Select("groups.id").
		Joins(`
			JOIN groups_ancestors_active
			ON groups_ancestors_active.ancestor_group_id IN (?) AND
				groups_ancestors_active.child_group_id = groups.id`, ancestorGroupIDs).
		Where("groups.type = 'Team' OR groups.type = 'User'").
		Group("groups.id")

	endMembersStats := store.
		Table("(?) AS end_members", endMembers.SubQuery()).
		Select(`
			end_members.id,
			items.id AS item_id,
			IFNULL(result_with_best_score.score, 0) AS score,
			IFNULL(result_with_best_score.validated, 0) AS validated,
			IFNULL(result_with_best_score.hints_cached, 0) AS hints_cached,
			IFNULL(result_with_best_score.submissions, 0) AS submissions,
			IF(result_with_best_score.participant_id IS NULL,
				0,
				(
					SELECT GREATEST(IF(result_with_best_score.validated,
						TIMESTAMPDIFF(SECOND, MIN(started_at), MIN(validated_at)),
						TIMESTAMPDIFF(SECOND, MIN(started_at), NOW())
					), 0)
					FROM results
					WHERE participant_id = end_members.id AND item_id = items.id
				)
			) AS time_spent`).
		Joins("JOIN (?) AS items", itemsSubQuery).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT score_computed AS score, validated, hints_cached, submissions, participant_id
				FROM results
				WHERE participant_id = end_members.id AND item_id = items.id
				ORDER BY participant_id, item_id, score_computed DESC, score_obtained_at
				LIMIT 1
			) AS result_with_best_score ON 1`)

	var result []*groupGroupProgressResponseTableCell
	// It still takes more than 2 minutes to complete on large data sets
	scanAndBuildProgressResults(
		store.ActiveGroupAncestors().
			Select(`
				groups_ancestors_active.ancestor_group_id AS group_id,
				member_stats.item_id,
				AVG(member_stats.score) AS average_score,
				AVG(member_stats.validated) AS validation_rate,
				AVG(member_stats.hints_cached) AS avg_hints_requested,
				AVG(member_stats.submissions) AS avg_submissions,
				AVG(member_stats.time_spent) AS avg_time_spent`).
			Joins("JOIN (?) AS member_stats ON member_stats.id = groups_ancestors_active.child_group_id", endMembersStats.SubQuery()).
			Where("groups_ancestors_active.ancestor_group_id IN (?)", ancestorGroupIDs).
			Group("groups_ancestors_active.ancestor_group_id, member_stats.item_id").
			Clauses(
				clause.OrderBy{
					Expression: clause.Expr{
						SQL:                "FIELD(groups_ancestors_active.ancestor_group_id, ?)",
						Vars:               []interface{}{ancestorGroupIDs},
						WithoutParentheses: true,
					},
				},
			),
		orderedItemIDListWithDuplicates, len(uniqueItemIDs), &result,
	)

	render.Respond(w, r, result)
	return service.NoError
}

func preselectIDsOfVisibleItems(store *database.DataStore, itemParentIDs []int64, user *database.User) (
	orderedItemIDListWithDuplicates []interface{}, uniqueItemIDs []string, itemOrder []int, itemsSubQuery interface{},
) {
	itemParentIDsAsIntSlice := make([]interface{}, len(itemParentIDs))
	for i, parentID := range itemParentIDs {
		itemParentIDsAsIntSlice[i] = parentID
	}

	var parentChildPairs []struct {
		ParentItemID int64
		ChildItemID  int64
	}

	service.MustNotBeError(store.ItemItems().
		Select("items_items.child_item_id AS id").
		Where("parent_item_id IN (?)", itemParentIDs).
		Joins("JOIN ? AS permissions ON permissions.item_id = items_items.child_item_id",
			store.Permissions().MatchingUserAncestors(user).
				Select("item_id").
				WherePermissionIsAtLeast("view", "info").SubQuery()).
		Order(gorm.Expr(
			"FIELD(items_items.parent_item_id"+strings.Repeat(", ?", len(itemParentIDs))+"), items_items.child_order",
			itemParentIDsAsIntSlice...)).
		Group("items_items.parent_item_id, items_items.child_item_id").
		Select("items_items.parent_item_id, items_items.child_item_id").
		Scan(&parentChildPairs).Error())

	// parent1_id, child1_1_id, ..., parent2_id, child2_1_id, ...
	orderedItemIDListWithDuplicates = make([]interface{}, 0, len(itemParentIDs)+len(parentChildPairs))
	itemOrder = make([]int, 0, len(itemParentIDs)+len(parentChildPairs))
	currentParentIDIndex := 0

	// child_id -> true, will be used to construct a list of unique item ids
	childItemIDMap := make(map[int64]bool, len(parentChildPairs))

	orderedItemIDListWithDuplicates = append(orderedItemIDListWithDuplicates, itemParentIDs[0])
	itemOrder = append(itemOrder, 0)
	currentChildNumber := 0
	for i := range parentChildPairs {
		for itemParentIDs[currentParentIDIndex] != parentChildPairs[i].ParentItemID {
			currentParentIDIndex++
			currentChildNumber = 0
			orderedItemIDListWithDuplicates = append(orderedItemIDListWithDuplicates, itemParentIDs[currentParentIDIndex])
			itemOrder = append(itemOrder, 0)
		}
		orderedItemIDListWithDuplicates = append(orderedItemIDListWithDuplicates, parentChildPairs[i].ChildItemID)
		childItemIDMap[parentChildPairs[i].ChildItemID] = true
		currentChildNumber++
		itemOrder = append(itemOrder, currentChildNumber)
	}

	// Create an unordered list of all the unique item ids (parents and children).
	//	Note: itemParentIDs slice doesn't contain duplicates because resolveAndCheckParentIDs() guarantees that.
	itemIDs := make([]string, len(itemParentIDs), len(childItemIDMap)+len(itemParentIDs))
	for i, parentID := range itemParentIDs {
		itemIDs[i] = strconv.FormatInt(parentID, 10)
	}
	for itemID := range childItemIDMap {
		itemIDs = append(itemIDs, strconv.FormatInt(itemID, 10))
	}

	itemsSubQuery = gorm.Expr(`JSON_TABLE('[` + strings.Join(itemIDs, ", ") + `]', "$[*]" COLUMNS(id BIGINT PATH "$"))`)
	return orderedItemIDListWithDuplicates, itemIDs, itemOrder, itemsSubQuery
}

func appendTableRowToResult(orderedItemIDListWithDuplicates []interface{}, reflResultRowMap reflect.Value, resultPtr interface{}) {
	// resultPtr is *[]*tableCellType
	reflTableCellType := reflect.TypeOf(resultPtr).Elem().Elem().Elem()
	// []*tableCellType
	reflTableRow := reflect.MakeSlice(
		reflect.SliceOf(reflect.PtrTo(reflTableCellType)), len(orderedItemIDListWithDuplicates), len(orderedItemIDListWithDuplicates))

	// Here we fill the table row with cells. As an item can be a child of multiple parents, the row may contain duplicates.
	for index, itemID := range orderedItemIDListWithDuplicates {
		reflTableRow.Index(index).Set(reflResultRowMap.MapIndex(reflect.ValueOf(itemID)))
	}
	reflResultPtr := reflect.ValueOf(resultPtr)
	// this means: *resultPtr = append(*resultPtr, tableRow)
	reflResultPtr.Elem().Set(reflect.AppendSlice(reflResultPtr.Elem(), reflTableRow))
}

// resultPtr should be a pointer to a slice of pointers to table cells.
func scanAndBuildProgressResults(
	query *database.DB, orderedItemIDListWithDuplicates []interface{}, uniqueItemsCount int, resultPtr interface{},
) {
	// resultPtr is *[]*tableCellType
	reflTableCellType := reflect.TypeOf(resultPtr).Elem().Elem().Elem()
	reflDecodedTableCell := reflect.New(reflTableCellType).Elem()
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		// will convert strings with time in DB format to database.Time
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
				if f.Kind() != reflect.String {
					return data, nil
				}
				if t != reflect.TypeOf(database.Time{}) {
					return data, nil
				}

				// Convert it by parsing
				result := &database.Time{}
				err := result.ScanString(data.(string))
				return *result, err
			},
		),
		Result:           reflDecodedTableCell.Addr().Interface(),
		TagName:          "json",
		ZeroFields:       true, // this marks keys with null values as used
		WeaklyTypedInput: true,
	})
	service.MustNotBeError(err)

	// here we will store results for each item: map[int64]*tableCellType
	reflResultRowMap := reflect.MakeMapWithSize(
		reflect.MapOf(reflect.TypeOf(int64(0)), reflect.PtrTo(reflTableCellType)), uniqueItemsCount)
	previousGroupID := int64(-1)
	service.MustNotBeError(query.ScanAndHandleMaps(func(cell map[string]interface{}) error {
		// convert map[string]interface{} into tableCellType and store the result in reflDecodedTableCell
		service.MustNotBeError(decoder.Decode(cell))

		groupID := reflDecodedTableCell.FieldByName("GroupID").Interface().(int64)
		if groupID != previousGroupID {
			if previousGroupID != -1 {
				// Moving to a next row of the results table, so we should insert cells from the map into the results slice
				appendTableRowToResult(orderedItemIDListWithDuplicates, reflResultRowMap, resultPtr)
				// and initialize a new map for cells
				reflResultRowMap = reflect.MakeMapWithSize(reflect.MapOf(reflect.TypeOf(int64(0)), reflect.PtrTo(reflTableCellType)), uniqueItemsCount)
			}
			previousGroupID = groupID
		}

		// as reflDecodedTableCell will be reused on the next step of the loop, we should create a copy
		reflDecodedRowCopy := reflect.New(reflTableCellType).Elem()
		reflDecodedRowCopy.Set(reflDecodedTableCell)
		reflResultRowMap.SetMapIndex(reflDecodedTableCell.FieldByName("ItemID"), reflDecodedRowCopy.Addr())
		return nil
	}).Error())

	// If no end members, return an empty result
	if previousGroupID == -1 {
		reflResultPtr := reflect.ValueOf(resultPtr)
		reflResultPtr.Elem().Set(reflect.MakeSlice(reflect.SliceOf(reflResultPtr.Elem().Type().Elem()), 0, 0))
		return
	}

	// store the last row of the table
	appendTableRowToResult(orderedItemIDListWithDuplicates, reflResultRowMap, resultPtr)
}

func resolveAndCheckParentIDs(store *database.DataStore, r *http.Request, user *database.User) ([]int64, service.APIError) {
	itemParentIDs, err := service.ResolveURLQueryGetInt64SliceField(r, "parent_item_ids")
	if err != nil {
		return nil, service.ErrInvalidRequest(err)
	}
	itemParentIDs = uniqueIDs(itemParentIDs)
	if len(itemParentIDs) > 0 {
		var cnt int
		service.MustNotBeError(store.Permissions().MatchingUserAncestors(user).
			WherePermissionIsAtLeast("watch", "result").Where("item_id IN(?)", itemParentIDs).
			PluckFirst("COUNT(DISTINCT item_id)", &cnt).Error())
		if cnt != len(itemParentIDs) {
			return nil, service.InsufficientAccessRightsError
		}
	}
	return itemParentIDs, service.NoError
}

func uniqueIDs(ids []int64) []int64 {
	idsMap := make(map[int64]bool, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !idsMap[id] {
			result = append(result, id)
			idsMap[id] = true
		}
	}
	return result
}
