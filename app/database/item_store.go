package database

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ItemStore implements database operations on items.
type ItemStore struct {
	*DataStore
}

// Visible returns a view of the visible items for the given participant.
func (s *ItemStore) Visible(groupID int64) *DB {
	return s.WhereItemsAreVisible(groupID)
}

// VisibleByID returns a view of the visible item identified by itemID, for the given participant.
func (s *ItemStore) VisibleByID(groupID, itemID int64) *DB {
	return s.Visible(groupID).Where("items.id = ?", itemID)
}

// WhereItemsAreSelfOrDescendantsOf filters items who are self or descendant of a given item.
func (conn *DB) WhereItemsAreSelfOrDescendantsOf(itemAncestorID int64) *DB {
	return conn.
		Joins("LEFT JOIN items_ancestors ON items_ancestors.child_item_id = items.id").
		Where("(items_ancestors.ancestor_item_id = ? OR items.id = ?)", itemAncestorID, itemAncestorID)
}

// IsValidParticipationHierarchyForParentAttempt checks if the given list of item ids is a valid participation hierarchy
// for the given `parentAttemptID` which means all the following statements are true:
//   - the first item in `ids` is a root activity/skill (groups.root_activity_id/root_skill_id)
//     of a group the `groupID` is a descendant of or manages,
//   - `ids` is an ordered list of parent-child items,
//   - the `groupID` group has at least 'content' access on each of the items in `ids`,
//   - the `groupID` group has a started, allowing submission, not ended result for each item but the last,
//     with `parentAttemptID` (or its parent attempt each time we reach a root of an attempt) as the attempt,
//   - if `ids` consists of only one item, the `parentAttemptID` is zero.
func (s *ItemStore) IsValidParticipationHierarchyForParentAttempt(
	ids []int64, groupID, parentAttemptID int64, requireContentAccessToTheLastItem, withWriteLock bool,
) (bool, error) {
	if len(ids) == 0 || len(ids) == 1 && parentAttemptID != 0 {
		return false, nil
	}

	return s.participationHierarchyForParentAttempt(
		ids, groupID, parentAttemptID, true, requireContentAccessToTheLastItem, "1", withWriteLock).HasRows()
}

func (s *ItemStore) participationHierarchyForParentAttempt(
	ids []int64, groupID, parentAttemptID int64, requireAttemptsToBeActive, requireContentAccessToTheLastItem bool,
	columnsList string, withWriteLock bool,
) *DB {
	subQuery := s.itemAttemptChainWithoutAttemptForTail(
		ids, groupID, requireAttemptsToBeActive, requireContentAccessToTheLastItem, withWriteLock)

	if len(ids) > 1 {
		subQuery = subQuery.
			Where(fmt.Sprintf("attempts%d.id = ?", len(ids)-2), parentAttemptID)
	}

	subQuery = subQuery.Select(columnsList)
	visibleItems := s.Permissions().MatchingGroupAncestors(groupID).
		WherePermissionIsAtLeast("view", "info").
		Where("item_id IN (?)", ids).
		Joins("JOIN items ON items.id = permissions.item_id").
		Select(`
			items.id, items.allows_multiple_attempts,
			MAX(permissions.can_view_generated_value) AS can_view_generated_value`).
		Group("items.id")

	return s.Raw("WITH visible_items AS (?) ?", visibleItems.SubQuery(), subQuery.SubQuery())
}

func (s *ItemStore) itemAttemptChainWithoutAttemptForTail(ids []int64, groupID int64,
	requireAttemptsToBeActive, requireContentAccessToTheLastItem, withWriteLock bool,
) *DB {
	participantAncestors := s.ActiveGroupAncestors().Where("child_group_id = ?", groupID).
		Joins("JOIN `groups` ON groups.id = groups_ancestors_active.ancestor_group_id")
	groupsManagedByParticipant := s.ActiveGroupAncestors().Where("groups_ancestors_active.child_group_id = ?", groupID).
		Joins("JOIN group_managers ON group_managers.manager_id = groups_ancestors_active.ancestor_group_id").
		Joins("JOIN groups_ancestors_active AS managed_descendants ON managed_descendants.ancestor_group_id = group_managers.group_id").
		Joins("JOIN `groups` ON groups.id = managed_descendants.child_group_id")
	rootActivities := participantAncestors.Select("groups.root_activity_id").Union(
		groupsManagedByParticipant.Select("groups.root_activity_id").SubQuery())
	rootSkills := participantAncestors.Select("groups.root_skill_id").Union(
		groupsManagedByParticipant.Select("groups.root_skill_id").SubQuery())

	if withWriteLock {
		rootActivities = rootActivities.WithWriteLock()
		rootSkills = rootSkills.WithWriteLock()
	}

	subQuery := s.Table("visible_items as items0").Where("items0.id = ?", ids[0]).
		Where("items0.id IN (?) OR items0.id IN (?)", rootActivities.SubQuery(), rootSkills.SubQuery())

	for i := 1; i < len(ids); i++ {
		subQuery = subQuery.Joins(fmt.Sprintf(`
				JOIN results AS results%d ON results%d.participant_id = ? AND
					results%d.item_id = items%d.id AND results%d.started_at IS NOT NULL`, i-1, i-1, i-1, i-1, i-1), groupID).
			Joins(fmt.Sprintf(`
				JOIN attempts AS attempts%d ON attempts%d.participant_id = results%d.participant_id AND
					attempts%d.id = results%d.attempt_id`, i-1, i-1, i-1, i-1, i-1)).
			Joins(
				fmt.Sprintf(
					"JOIN items_items AS items_items%d ON items_items%d.parent_item_id = items%d.id AND items_items%d.child_item_id = ?",
					i, i, i-1, i), ids[i]).
			Joins(fmt.Sprintf("JOIN visible_items AS items%d ON items%d.id = items_items%d.child_item_id", i, i, i)).
			Where(fmt.Sprintf("items%d.can_view_generated_value >= ?", i-1),
				s.PermissionsGranted().ViewIndexByName("content"))

		if i != len(ids)-1 {
			subQuery = subQuery.Where(fmt.Sprintf(
				"IF(attempts%d.root_item_id = items%d.id, attempts%d.parent_attempt_id, attempts%d.id) = attempts%d.id",
				i, i, i, i, i-1))
		}

		if requireAttemptsToBeActive {
			subQuery = subQuery.Where(fmt.Sprintf("attempts%d.ended_at IS NULL AND NOW() < attempts%d.allows_submissions_until", i-1, i-1))
		}
	}

	if requireContentAccessToTheLastItem {
		subQuery = subQuery.Where(fmt.Sprintf("items%d.can_view_generated_value >= ?", len(ids)-1),
			s.PermissionsGranted().ViewIndexByName("content"))
	}

	return subQuery
}

// BreadcrumbsHierarchyForParentAttempt returns attempts ids and 'order' (for items allowing multiple attempts)
// for the given list of item ids (but the last item) if it is a valid participation hierarchy
// for the given `parentAttemptID` which means all the following statements are true:
//   - the first item in `ids` is a root activity/skill (groups.root_activity_id/root_skill_id)
//     of a group the `groupID` is a descendant of or manages,
//   - `ids` is an ordered list of parent-child items,
//   - the `groupID` group has at least 'content' access on each of the items in `ids` except for the last one and
//     at least 'info' access on the last one,
//   - the `groupID` group has a started result for each item but the last,
//     with `parentAttemptID` (or its parent attempt each time we reach a root of an attempt) as the attempt,
//   - if `ids` consists of only one item, the `parentAttemptID` is zero.
func (s *ItemStore) BreadcrumbsHierarchyForParentAttempt(ids []int64, groupID, parentAttemptID int64, withWriteLock bool) (
	attemptIDMap map[int64]int64, attemptNumberMap map[int64]int, err error,
) {
	if len(ids) == 0 || len(ids) == 1 && parentAttemptID != 0 {
		return nil, nil, nil
	}

	defer recoverPanics(&err)

	columnsList := columnsListForBreadcrumbsHierarchy(ids[:len(ids)-1])
	query := s.participationHierarchyForParentAttempt(
		ids, groupID, parentAttemptID, false, false, columnsList, withWriteLock)
	var data []map[string]interface{}
	mustNotBeError(query.Limit(1).ScanIntoSliceOfMaps(&data).Error())
	if len(data) == 0 {
		return nil, nil, nil
	}

	attemptIDMap, attemptNumberMap = resultsForBreadcrumbsHierarchy(ids[:len(ids)-1], data[0])
	return
}

// BreadcrumbsHierarchyForAttempt returns attempts ids and 'order' (for items allowing multiple attempts)
// for the given list of item ids if it is a valid participation hierarchy
// for the given `attemptID` which means all the following statements are true:
//   - the first item in `ids` is an activity/skill item (groups.root_activity_id/root_skill_id) of a group
//     the `groupID` is a descendant of or manages,
//   - `ids` is an ordered list of parent-child items,
//   - the `groupID` group has at least 'content' access on each of the items in `ids` except for the last one and
//     at least 'info' access on the last one,
//   - the `groupID` group has a started result for each item,
//     with `attemptID` (or its parent attempt each time we reach a root of an attempt) as the attempt.
func (s *ItemStore) BreadcrumbsHierarchyForAttempt(ids []int64, groupID, attemptID int64, withWriteLock bool) (
	attemptIDMap map[int64]int64, attemptNumberMap map[int64]int, err error,
) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	defer recoverPanics(&err)

	columnsList := columnsListForBreadcrumbsHierarchy(ids)
	query := s.breadcrumbsHierarchyForAttempt(
		ids, groupID, attemptID, false, columnsList, withWriteLock)
	var data []map[string]interface{}
	mustNotBeError(query.Limit(1).ScanIntoSliceOfMaps(&data).Error())
	if len(data) == 0 {
		return nil, nil, nil
	}

	attemptIDMap, attemptNumberMap = resultsForBreadcrumbsHierarchy(ids, data[0])
	return
}

func columnsListForBreadcrumbsHierarchy(ids []int64) string {
	columnsList := "1"
	if len(ids) > 0 {
		var columnsBuilder strings.Builder
		for idIndex := 0; idIndex < len(ids); idIndex++ {
			if idIndex != 0 {
				_, _ = columnsBuilder.WriteString(", ")
			}
			_, _ = columnsBuilder.WriteString(fmt.Sprintf(`
				IF(items%d.allows_multiple_attempts, (
					SELECT number FROM (
						SELECT results.attempt_id, ROW_NUMBER() OVER (ORDER BY started_at) AS number
						FROM results
						JOIN attempts ON attempts.participant_id = results.participant_id and attempts.id = results.attempt_id
						WHERE results.participant_id = attempts%d.participant_id AND
							results.item_id = items%d.id AND
							attempts.parent_attempt_id <=> attempts%d.parent_attempt_id
					) AS numbers WHERE numbers.attempt_id = attempts%d.id
				), NULL) AS number%d,
				attempts%d.id AS attempt%d`, idIndex, idIndex, idIndex, idIndex, idIndex, idIndex, idIndex, idIndex))
		}
		columnsList = columnsBuilder.String()
	}
	return columnsList
}

func resultsForBreadcrumbsHierarchy(ids []int64, data map[string]interface{}) (
	attemptIDMap map[int64]int64, attemptNumberMap map[int64]int,
) {
	attemptIDMap = make(map[int64]int64, len(ids))
	attemptNumberMap = make(map[int64]int, len(ids))
	for idIndex := 0; idIndex < len(ids); idIndex++ {
		attemptIDMap[ids[idIndex]] = data[fmt.Sprintf("attempt%d", idIndex)].(int64)
		numberKey := fmt.Sprintf("number%d", idIndex)
		if data[numberKey] != nil {
			attemptNumberMap[ids[idIndex]] = int(data[numberKey].(int64))
		}
	}
	return attemptIDMap, attemptNumberMap
}

func (s *ItemStore) breadcrumbsHierarchyForAttempt(
	ids []int64, groupID, attemptID int64, requireContentAccessToTheLastItem bool,
	columnsList string, withWriteLock bool,
) *DB {
	lastItemIndex := len(ids) - 1
	subQuery := s.
		itemAttemptChainWithoutAttemptForTail(ids, groupID, false, requireContentAccessToTheLastItem, withWriteLock).
		Where(fmt.Sprintf("attempts%d.id = ?", lastItemIndex), attemptID)
	subQuery = subQuery.
		Joins(fmt.Sprintf(`
				JOIN results AS results%d ON results%d.participant_id = ? AND
					results%d.item_id = items%d.id AND results%d.started_at IS NOT NULL`,
			lastItemIndex, lastItemIndex, lastItemIndex, lastItemIndex, lastItemIndex), groupID).
		Joins(fmt.Sprintf(`
				JOIN attempts AS attempts%d ON attempts%d.participant_id = results%d.participant_id AND
					attempts%d.id = results%d.attempt_id`, lastItemIndex, lastItemIndex, lastItemIndex, lastItemIndex, lastItemIndex))
	if len(ids) > 1 {
		subQuery = subQuery.Where(fmt.Sprintf(
			"IF(attempts%d.root_item_id = items%d.id, attempts%d.parent_attempt_id, attempts%d.id) = attempts%d.id",
			lastItemIndex, lastItemIndex, lastItemIndex, lastItemIndex, lastItemIndex-1))
	}

	subQuery = subQuery.Select(columnsList)
	visibleItems := s.Permissions().MatchingGroupAncestors(groupID).
		WherePermissionIsAtLeast("view", "info").
		Where("item_id IN(?)", ids).
		Joins("JOIN items ON items.id = permissions.item_id").
		Select(`
			items.id, items.allows_multiple_attempts,
			MAX(permissions.can_view_generated_value) AS can_view_generated_value`).
		Group("items.id")

	if withWriteLock {
		subQuery = subQuery.WithWriteLock()
		visibleItems = visibleItems.WithWriteLock()
	}
	return s.Raw("WITH visible_items AS (?) ?", visibleItems.SubQuery(), subQuery.SubQuery())
}

// CheckSubmissionRights checks if the participant group can submit an answer for the given item (task),
// i.e. the item (task) exists and is not read-only and the participant has at least content:view permission on the item.
func (s *ItemStore) CheckSubmissionRights(participantID, itemID int64) (hasAccess bool, reason, err error) {
	s.mustBeInTransaction()
	recoverPanics(&err)

	var readOnly bool
	err = s.WhereGroupHasPermissionOnItems(participantID, "view", "content").
		Where("id = ?", itemID).
		WithWriteLock().
		PluckFirst("read_only", &readOnly).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, errors.New("no access to the task item"), nil
	}
	mustNotBeError(err)

	if readOnly {
		return false, errors.New("item is read-only"), nil
	}

	return true, nil, nil
}

// ContestManagedByUser returns a composable query
// for getting a contest with the given item id managed by the given user.
func (s *ItemStore) ContestManagedByUser(contestItemID int64, user *User) *DB {
	return s.ByID(contestItemID).Where("items.duration IS NOT NULL").
		JoinsPermissionsForGroupToItemsWherePermissionAtLeast(user.GroupID, "view", "content").
		WherePermissionIsAtLeast("grant_view", "enter").
		WherePermissionIsAtLeast("watch", "result")
}

// DeleteItem deletes an item. Note the method fails if the item has children.
func (s *ItemStore) DeleteItem(itemID int64) (err error) {
	s.mustBeInTransaction()

	return s.ItemItems().WithItemsRelationsLock(func(s *DataStore) error {
		mustNotBeError(s.WithForeignKeyChecksDisabled(func(s *DataStore) error {
			return s.ItemStrings().Where("item_id = ?", itemID).Delete().Error()
		}))
		mustNotBeError(s.Items().ByID(itemID).Delete().Error())
		mustNotBeError(s.ItemItems().After())
		return nil
	})
}

// getAncestorsRequestHelpPropagationQuery gets all ancestors of an itemID while request_help_propagation = 1.
func (s *ItemStore) getAncestorsRequestHelpPropagationQuery(itemID int64) *DB {
	return s.Raw(`
		WITH RECURSIVE items_ancestors_request_help_propagation(item_id) AS
		(
			SELECT ?
			UNION ALL
			SELECT items.id FROM items
			JOIN items_items ON items_items.parent_item_id = items.id AND	items_items.request_help_propagation = 1
			JOIN items_ancestors_request_help_propagation ON items_ancestors_request_help_propagation.item_id = items_items.child_item_id
		)
		SELECT item_id FROM items_ancestors_request_help_propagation
	`, itemID)
}

// GetItemIDFromTextID gets the item_id from the text_id of an item.
func (s *ItemStore) GetItemIDFromTextID(textID string) (itemID int64, err error) {
	err = s.Select("items.id AS id").
		Where("text_id = ?", textID).
		PluckFirst("id", &itemID).Error()

	return itemID, err
}
