package database

// User represents data associated with the user (from the `users` table).
type User struct {
	GroupID             int64
	Login               string
	LoginID             *int64
	DefaultLanguage     string
	IsAdmin             bool
	IsTempUser          bool `gorm:"column:temp_user"`
	AccessGroupID       *int64
	NotificationsReadAt *Time
}

// Clone returns a deep copy of the given User structure.
func (u *User) Clone() *User {
	result := *u
	if result.NotificationsReadAt != nil {
		notificationReadDateCopy := *result.NotificationsReadAt
		result.NotificationsReadAt = &notificationReadDateCopy
	}
	if result.LoginID != nil {
		loginIDCopy := *result.LoginID
		result.LoginID = &loginIDCopy
	}
	if result.AccessGroupID != nil {
		accessGroupIDCopy := *result.AccessGroupID
		result.AccessGroupID = &accessGroupIDCopy
	}
	return &result
}

// HasItemPermission checks whether the user have a certain permission on an item.
func (u *User) HasItemPermission(s *DataStore, itemID int64, permissionType, permissionValue string) bool {
	userHasPermission, err := s.Permissions().MatchingUserAncestors(u).
		Where("permissions.item_id = ?", itemID).
		WherePermissionIsAtLeast(permissionType, permissionValue).
		Select("1").
		Limit(1).
		HasRows()
	mustNotBeError(err)

	return userHasPermission
}

// CanWatchItemAnswer checks whether the user has can_watch >= answer on an item.
func (u *User) CanWatchItemAnswer(s *DataStore, itemID int64) bool {
	return u.HasItemPermission(s, itemID, "watch", "answer")
}

// CanWatchItemResult checks whether the user has can_watch >= result on an item.
func (u *User) CanWatchItemResult(s *DataStore, itemID int64) bool {
	return u.HasItemPermission(s, itemID, "watch", "result")
}

// CanViewItemContent checks whether the user has can_view >= content on an item.
func (u *User) CanViewItemContent(s *DataStore, itemID int64) bool {
	return u.HasItemPermission(s, itemID, "view", "content")
}

// CanRequestHelpTo checks whether the user can request help on an item to a group.
func (u *User) CanRequestHelpTo(s *DataStore, itemID, helperGroupID int64) bool {
	// in order to verify that the user “can request help to” a group on an item, we need to verify whether
	// one of the ancestors (including himself) of User has the can_request_help_to(Group) on Item,
	// recursively on Item’s ancestors while request_help_propagation=1, for each Group being a descendant of Group.

	itemAncestorsRequestHelpPropagationQuery := s.Items().getAncestorsRequestHelpPropagationQuery(itemID)

	canRequestHelpTo, err := s.Users().
		Joins("JOIN groups_ancestors_active ON groups_ancestors_active.child_group_id = ?", u.GroupID).
		Joins(`JOIN permissions_granted ON
			permissions_granted.group_id = groups_ancestors_active.ancestor_group_id AND
			(permissions_granted.item_id = ? OR permissions_granted.item_id IN (?))`, itemID, itemAncestorsRequestHelpPropagationQuery.SubQuery()).
		Joins(`JOIN groups_ancestors_active AS groups_ancestors_can_request_help_to ON
			groups_ancestors_can_request_help_to.child_group_id = ?`, helperGroupID).
		Where("permissions_granted.can_request_help_to = groups_ancestors_can_request_help_to.ancestor_group_id").
		Select("1").
		Limit(1).
		HasRows()
	mustNotBeError(err)

	return canRequestHelpTo
}

// GetManagedGroupsWithCanGrantGroupAccessIds retrieves all group ids that the user manages and for which
// he can_grant_group_access.
func (u *User) GetManagedGroupsWithCanGrantGroupAccessIds(store *DataStore) []int64 {
	var managedGroupsWithCanGrantGroupAccessIds []int64

	store.ActiveGroupAncestors().ManagedByUser(u).
		Group("groups_ancestors_active.child_group_id").
		Having("MAX(group_managers.can_grant_group_access)").
		Select("groups_ancestors_active.child_group_id AS id").
		Pluck("id", &managedGroupsWithCanGrantGroupAccessIds)

	return managedGroupsWithCanGrantGroupAccessIds
}

// CanWatchGroupMembers checks whether the user can watch a group / a participant.
func (u *User) CanWatchGroupMembers(s *DataStore, groupID int64) bool {
	userCanWatchGroupMembers, err := s.ActiveGroupAncestors().ManagedByUser(u).
		Where("groups_ancestors_active.child_group_id = ?", groupID).
		Where("group_managers.can_watch_members").
		Select("1").
		Limit(1).
		HasRows()
	mustNotBeError(err)

	return userCanWatchGroupMembers
}

// HasStartedResultOnItem checks whether the user has a started result on an item.
func (u *User) HasStartedResultOnItem(s *DataStore, itemID int64) bool {
	hasStartedResultOntem, err := s.Items().
		Where("items.id = ?", itemID).
		WhereUserHaveStartedResultOnItem(u).
		Limit(1).
		HasRows()
	mustNotBeError(err)

	return hasStartedResultOntem
}

// HasValidatedItem checks whether the user has validated an item.
func (u *User) HasValidatedItem(s *DataStore, itemID int64) bool {
	hasValidatedItem, err := s.Results().
		Where("results.item_id = ?", itemID).
		Where("results.validated").
		Limit(1).
		HasRows()
	mustNotBeError(err)

	return hasValidatedItem
}
