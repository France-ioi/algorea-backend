package database

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GroupStore implements database operations on groups.
type GroupStore struct {
	*DataStore
}

// ManagedBy returns a composable query for getting all the groups the user can manage.
//
// The `groups` in the result may be duplicated since
// there can be different paths to a managed group through the `group_managers` table and
// the group ancestry graph.
func (s *GroupStore) ManagedBy(user *User) *DB {
	return s.
		Joins(`
			JOIN groups_ancestors_active
				ON groups_ancestors_active.child_group_id = groups.id`).
		Joins(`
			JOIN group_managers
				ON group_managers.group_id = groups_ancestors_active.ancestor_group_id`).
		Joins(`
			JOIN groups_ancestors_active AS user_ancestors
				ON user_ancestors.ancestor_group_id = group_managers.manager_id AND
					user_ancestors.child_group_id = ?`, user.GroupID)
}

// TeamGroupForTeamItemAndUser returns a composable query for getting a team that
//  1. the given user is a member of
//  2. has an unexpired attempt with root_item_id = `itemID`.
//
// If more than one team is found (which should be impossible), the one with the smallest `groups.id` is returned.
func (s *GroupStore) TeamGroupForTeamItemAndUser(itemID int64, user *User) *DB {
	return s.
		Joins(`JOIN groups_groups_active
			ON groups_groups_active.parent_group_id = groups.id AND
				groups_groups_active.child_group_id = ?`, user.GroupID).
		Joins(`
			JOIN attempts ON attempts.participant_id = groups.id AND
				attempts.root_item_id = ? AND NOW() < attempts.allows_submissions_until`, itemID).
		Where("groups.type = 'Team'").
		Order("groups.id").
		Limit(1) // The current API doesn't allow users to join multiple teams working on the same item
}

// TeamGroupForUser returns a composable query for getting team group of the given user with given id.
func (s *GroupStore) TeamGroupForUser(teamGroupID int64, user *User) *DB {
	return s.
		ByID(teamGroupID).
		Joins(`JOIN groups_groups_active
			ON groups_groups_active.parent_group_id = groups.id AND
				groups_groups_active.child_group_id = ?`, user.GroupID).
		Where("groups.type = 'Team'")
}

// CreateNew creates a new group with given name and type.
// It also runs GroupGroupStore.createNewAncestors().
func (s *GroupStore) CreateNew(name, groupType string) (groupID int64, err error) {
	s.mustBeInTransaction()
	defer recoverPanics(&err)
	mustNotBeError(s.RetryOnDuplicatePrimaryKeyError(func(retryStore *DataStore) error {
		groupID = retryStore.NewID()
		return retryStore.Groups().InsertMap(map[string]interface{}{
			"id":         groupID,
			"name":       name,
			"type":       groupType,
			"created_at": Now(),
		})
	}))
	if groupType == "Team" {
		mustNotBeError(s.Attempts().InsertMap(map[string]interface{}{
			"participant_id": groupID,
			"id":             0,
			"creator_id":     nil,
			"created_at":     Now(),
		}))
	}
	s.GroupGroups().createNewAncestors()
	return groupID, nil
}

// CheckIfEntryConditionsStillSatisfiedForAllActiveParticipations checks whether adding/removal of a user
// specified by userID would keep entry conditions satisfied for all active participations of teamGroupID.
// If at least one entry condition becomes broken for at least one active participation, the method returns false.
// An active participation is one that is started (`results.started_at` is not null for the `root_item_id`),
// still allows submissions and is not ended.
// Entry conditions are defined by items.entry_min_admitted_members_ratio & items.entry_max_team_size
// (for more info see description of the itemGetEntryState service).
// The isAdding parameter specifies if we are going to add or remove a user.
func (s *GroupStore) CheckIfEntryConditionsStillSatisfiedForAllActiveParticipations(
	teamGroupID, userID int64, isAdding, withLock bool,
) (bool, error) {
	found, err := s.GenerateQueryCheckingIfActionBreaksEntryConditionsForActiveParticipations(
		gorm.Expr("?", teamGroupID), userID, isAdding, withLock).HasRows()
	return !found, err
}

// GenerateQueryCheckingIfActionBreaksEntryConditionsForActiveParticipations generates an SQL query
// checking whether adding/removal of a user
// specified by userID would break any of entry conditions for any of active participations of teamGroupID.
// If at least one entry condition becomes broken for at least one active participation, the query returns a row with 1.
// An active participation is one that is started (`results.started_at` is not null for the `root_item_id`),
// still allows submissions and is not ended.
// Entry conditions are defined by items.entry_min_admitted_members_ratio & items.entry_max_team_size
// (for more info see description of the itemGetEntryState service).
// The isAdding parameter specifies if we are going to add or remove a user.
func (s *GroupStore) GenerateQueryCheckingIfActionBreaksEntryConditionsForActiveParticipations(
	teamGroupIDExpr clause.Expr, userID int64, isAdding, withLock bool,
) *DB {
	activeTeamParticipationsQuery := s.Attempts().
		Joins(`
			JOIN results ON results.participant_id = attempts.participant_id AND results.attempt_id = attempts.id AND
				results.item_id = attempts.root_item_id`).
		Where("attempts.participant_id = ?", teamGroupIDExpr).
		Where("root_item_id IS NOT NULL").
		Where("started").
		Where("NOW() < allows_submissions_until").
		Where("ended_at IS NULL").
		Select("item_id, MIN(started_at) AS started_at").
		Group("attempts.root_item_id")

	updatedMemberIDsQuery := s.ActiveGroupGroups().Where("parent_group_id = ?", teamGroupIDExpr).
		Select("child_group_id")

	if withLock {
		activeTeamParticipationsQuery = activeTeamParticipationsQuery.WithWriteLock()
		updatedMemberIDsQuery = updatedMemberIDsQuery.WithWriteLock()
	}

	if isAdding {
		updatedMemberIDsQuery = updatedMemberIDsQuery.UnionAll(s.Raw("SELECT ?", userID).QueryExpr())
	} else {
		updatedMemberIDsQuery = updatedMemberIDsQuery.Where("child_group_id != ?", userID)
	}

	membersPreconditionsQuery := s.ActiveGroupAncestors().
		Where("groups_ancestors_active.child_group_id IN (?)", updatedMemberIDsQuery.QueryExpr()).
		Joins("JOIN (?) AS active_participations", activeTeamParticipationsQuery.SubQuery()).
		Joins("JOIN items ON items.id = active_participations.item_id").
		Joins(`
			LEFT JOIN permissions_granted ON permissions_granted.group_id = groups_ancestors_active.ancestor_group_id AND
				permissions_granted.item_id = items.id`).
		Group("items.id, groups_ancestors_active.child_group_id").
		Select(`
			items.id AS item_id,
			items.entry_max_team_size,
			items.entry_min_admitted_members_ratio,
			IFNULL(
				MAX(permissions_granted.can_enter_from <= active_participations.started_at AND
				    active_participations.started_at < permissions_granted.can_enter_until),
				0) AS can_enter`)

	if withLock {
		membersPreconditionsQuery = membersPreconditionsQuery.WithWriteLock()
	}

	return s.Raw(`
		SELECT 1 FROM (?) members_preconditions
		GROUP BY item_id
		HAVING NOT (
			MIN(entry_min_admitted_members_ratio) = 'None' OR
			MIN(entry_min_admitted_members_ratio) = 'All' AND SUM(can_enter) = COUNT(*) OR
			MIN(entry_min_admitted_members_ratio) = 'Half' AND COUNT(*) <= SUM(can_enter) * 2 OR
			MIN(entry_min_admitted_members_ratio) = 'One' AND SUM(can_enter) >= 1
		) OR MIN(entry_max_team_size) < COUNT(*)`, membersPreconditionsQuery.QueryExpr()).Limit(1)
}

// DeleteGroup deletes a group and emerging orphaned groups.
func (s *GroupStore) DeleteGroup(groupID int64) (err error) {
	s.mustBeInTransaction()
	defer recoverPanics(&err)

	mustNotBeError(s.GroupGroups().WithGroupsRelationsLock(func(s *DataStore) error {
		s.GroupGroups().deleteGroupAndOrphanedDescendants(groupID)
		return nil
	}))
	return nil
}

// AncestorsOfJoinedGroups returns a query selecting all group ancestors ids of a user.
func (s *GroupStore) AncestorsOfJoinedGroups(store *DataStore, user *User) *DB {
	return store.ActiveGroupGroups().
		Where("groups_groups_active.child_group_id = ?", user.GroupID).
		Joins("JOIN groups_ancestors_active ON groups_ancestors_active.child_group_id = groups_groups_active.parent_group_id").
		Joins("JOIN `groups` AS ancestor_group ON ancestor_group.id = groups_ancestors_active.ancestor_group_id").
		Where("ancestor_group.type != 'ContestParticipants'").
		Select("groups_ancestors_active.ancestor_group_id")
}

// ManagedUsersAndAncestorsOfManagedGroups returns all groups which are ancestors of managed groups,
// and all users who are descendants from managed groups.
func (s *GroupStore) ManagedUsersAndAncestorsOfManagedGroups(store *DataStore, user *User) *DB {
	return store.ActiveGroupAncestors().ManagedByUser(user).
		Joins("JOIN `groups` ON groups.id = groups_ancestors_active.child_group_id").
		Joins(`
			JOIN groups_ancestors_active AS ancestors_of_managed
				ON ancestors_of_managed.child_group_id = groups_ancestors_active.child_group_id AND
				   (groups.type != 'User' OR ancestors_of_managed.is_self)`).
		Joins("JOIN `groups` AS ancestor_group ON ancestor_group.id = ancestors_of_managed.ancestor_group_id").
		Where("ancestor_group.type != 'ContestParticipants'").
		Select("ancestors_of_managed.ancestor_group_id")
}

// PickVisibleGroups returns a query filtering only group which are visible.
func (s *GroupStore) PickVisibleGroups(db *DB, user *User) *DB {
	AncestorsOfJoinedGroupsQuery := s.AncestorsOfJoinedGroups(NewDataStore(db.New()), user).QueryExpr()
	ManagedUsersAndAncestorsOfManagedGroupsQuery := s.ManagedUsersAndAncestorsOfManagedGroups(NewDataStore(db.New()), user).QueryExpr()

	return db.Where("groups.is_public OR groups.id IN(?) OR groups.id IN(?)",
		AncestorsOfJoinedGroupsQuery, ManagedUsersAndAncestorsOfManagedGroupsQuery)
}

// IsVisibleFor checks whether a group is visible to a user.
func (s *GroupStore) IsVisibleFor(groupID int64, user *User) bool {
	isVisible, err := s.PickVisibleGroups(s.Groups().DB, user).
		Where("groups.id = ?", groupID).
		Select("1").
		Limit(1).
		HasRows()
	mustNotBeError(err)

	return isVisible
}
