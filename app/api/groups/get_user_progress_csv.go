package groups

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

const csvExportBatchSize = 500

// swagger:operation GET /groups/{group_id}/user-progress-csv groups groupUserProgressCSV
//
//		---
//		summary: Get group progress for users as a CSV file
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
//		responses:
//			"200":
//				description: OK. Success response with users progress on items
//				content:
//					text/csv:
//						schema:
//						type: string
//				examples:
//					text/csv:
//						Login;Last name;First name;Parent item;1. First child item;2. Second child item
//
//						johnd;Doe;John;30;20;10
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getUserProgressCSV(w http.ResponseWriter, r *http.Request) service.APIError {
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
		fmt.Sprintf("attachment; filename=users_progress_for_group_%d_and_child_items_of_%s.csv",
			groupID, strings.Join(itemParentIDsString, "_")))
	if len(itemParentIDs) == 0 {
		_, err := w.Write([]byte("Login;First name;Last name\n"))
		service.MustNotBeError(err)
		return service.NoError
	}

	// Preselect item IDs since we need them to build the results table (there shouldn't be many)
	orderedItemIDListWithDuplicates, uniqueItemIDs, itemOrder, itemsSubQuery := preselectIDsOfVisibleItems(store, itemParentIDs, user)

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	csvWriter.Comma = ';'

	printTableHeader(store, user, uniqueItemIDs, orderedItemIDListWithDuplicates, itemOrder, csvWriter,
		[]string{"Login", "First name", "Last name"})

	// Preselect end member for that we will calculate the stats.
	var users []struct {
		ID        int64
		FirstName string
		LastName  string
		Login     string
	}
	service.MustNotBeError(store.ActiveGroupAncestors().
		Joins("JOIN groups_groups_active ON groups_groups_active.parent_group_id = groups_ancestors_active.child_group_id").
		Joins("JOIN `users` ON users.group_id = groups_groups_active.child_group_id").
		Where("groups_ancestors_active.ancestor_group_id = ?", groupID).
		Group("users.group_id").
		Order("login, first_name, last_name, users.group_id").
		Select(`
			users.group_id AS id,
			users.login,
			IF(users.group_id = ? OR MAX(personal_info_view_approvals.approved), users.first_name, NULL) AS first_name,
			IF(users.group_id = ? OR MAX(personal_info_view_approvals.approved), users.last_name, NULL) AS last_name`,
			user.GroupID, user.GroupID).
		WithPersonalInfoViewApprovals(user).
		Scan(&users).Error())

	if len(users) == 0 {
		return service.NoError
	}
	userIDs := make([]string, len(users))
	for i := range users {
		userIDs[i] = strconv.FormatInt(users[i].ID, 10)
	}

	for startFromUser := 0; startFromUser < len(userIDs); startFromUser += csvExportBatchSize {
		batchBoundary := startFromUser + csvExportBatchSize
		if batchBoundary > len(userIDs) {
			batchBoundary = len(userIDs)
		}
		userIDsList := strings.Join(userIDs[startFromUser:batchBoundary], ", ")
		userNumber := startFromUser
		service.MustNotBeError(
			// nolint:gosec
			joinUserProgressResultsForCSV(
				store.Raw(`
				SELECT STRAIGHT_JOIN
					items.id AS item_id,
					users.group_id AS group_id, MAX(result_with_best_score.score_computed) AS score
				FROM JSON_TABLE('[`+userIDsList+`]', "$[*]" COLUMNS(group_id BIGINT PATH "$")) AS users`).
					Joins("JOIN ? AS items", itemsSubQuery),
				gorm.Expr("users.group_id"),
			).
				Group("users.group_id, items.id").
				Order(gorm.Expr("FIELD(users.group_id, " + userIDsList + ")")).
				ScanAndHandleMaps(
					processCSVResultRow(
						orderedItemIDListWithDuplicates, len(uniqueItemIDs), &userNumber,
						func(_ int64) []string {
							return []string{users[userNumber].Login, users[userNumber].LastName, users[userNumber].FirstName}
						}, csvWriter)).Error())
	}

	return service.NoError
}

func printTableHeader(
	store *database.DataStore, user *database.User, uniqueItemIDs []string, orderedItemIDListWithDuplicates []interface{},
	itemOrder []int, csvWriter *csv.Writer, firstColumns []string,
) {
	var items []struct {
		ID           int64  `json:"id"`
		ParentItemID int64  `json:"parent_item_id"`
		Title        string `json:"-"`
	}
	service.MustNotBeError(store.Items().
		JoinsUserAndDefaultItemStrings(user).
		Where("items.id IN (?)", uniqueItemIDs).
		Select("id, COALESCE(user_strings.title, default_strings.title) AS title").
		Scan(&items).Error())
	itemTitlesMap := make(map[int64]string, len(uniqueItemIDs))
	for i := range items {
		itemTitlesMap[items[i].ID] = items[i].Title
	}
	itemTitles := make([]string, 0, len(orderedItemIDListWithDuplicates)+3)
	itemTitles = append(itemTitles, firstColumns...)
	for i, itemID := range orderedItemIDListWithDuplicates {
		title := itemTitlesMap[itemID.(int64)]
		if itemOrder[i] != 0 {
			title = fmt.Sprintf("%d. %s", itemOrder[i], title)
		}
		itemTitles = append(itemTitles, title)
	}
	service.MustNotBeError(csvWriter.Write(itemTitles))
}

func processCSVResultRow(
	orderedItemIDListWithDuplicates []interface{},
	uniqueItemsCount int,
	groupNumber *int,
	generateGroupNamesFunc func(groupID int64) []string,
	csvWriter *csv.Writer,
) func(m map[string]interface{}) error {
	var rowArray []string
	var cellsMap map[int64]string
	currentRowNumber := 0
	return func(m map[string]interface{}) error {
		var score string
		switch v := m["score"].(type) {
		case string:
			score = v
		case nil:
			score = ""
		}

		itemID := convertToInt64(m["item_id"])
		groupID := convertToInt64(m["group_id"])

		if currentRowNumber%uniqueItemsCount == 0 {
			groupNames := generateGroupNamesFunc(groupID)
			rowArray = make([]string, 0, len(orderedItemIDListWithDuplicates)+len(groupNames))
			cellsMap = make(map[int64]string, len(orderedItemIDListWithDuplicates))
			rowArray = append(rowArray, groupNames...)
			*groupNumber++
		}

		cellsMap[itemID] = score

		if currentRowNumber%uniqueItemsCount == uniqueItemsCount-1 {
			for _, id := range orderedItemIDListWithDuplicates {
				rowArray = append(rowArray, cellsMap[id.(int64)])
			}
			service.MustNotBeError(csvWriter.Write(rowArray))
		}
		currentRowNumber++
		return nil
	}
}

func convertToInt64(value interface{}) int64 {
	var err error
	var result int64
	switch v := value.(type) {
	case string:
		result, err = strconv.ParseInt(v, 10, 64)
		service.MustNotBeError(err)
	case int64:
		result = v
	}
	return result
}

func joinUserProgressResultsForCSV(db *database.DB, userID interface{}) *database.DB {
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
			JOIN LATERAL (
				SELECT
					IF(
						result_with_best_score_for_team.score_computed IS NOT NULL AND
						result_with_best_score_for_user.score_computed IS NOT NULL AND (
							result_with_best_score_for_team.score_computed > result_with_best_score_for_user.score_computed OR
							(
								result_with_best_score_for_team.score_computed = result_with_best_score_for_user.score_computed AND
								result_with_best_score_for_team.score_obtained_at < result_with_best_score_for_user.score_obtained_at
							)
						) OR result_with_best_score_for_user.score_computed IS NULL,
						result_with_best_score_for_team.score_computed,
						result_with_best_score_for_user.score_computed
					) AS score_computed
			) AS result_with_best_score ON 1`)
}
