package database

import (
	"strings"

	"gorm.io/gorm"
)

// GroupMembershipAction represents an action that changes relation between two groups.
type GroupMembershipAction string

const (
	// InvitationCreated means a pending group admin's invitation for user to join a group was created.
	InvitationCreated GroupMembershipAction = "invitation_created"
	// JoinRequestCreated means a pending user's request to join a group was created.
	JoinRequestCreated GroupMembershipAction = "join_request_created"
	// InvitationAccepted means a user became a member of a group by accepting an invitation.
	InvitationAccepted GroupMembershipAction = "invitation_accepted"
	// JoinRequestAccepted means a user became a member of a group since a group admin accepted his request.
	JoinRequestAccepted GroupMembershipAction = "join_request_accepted"
	// LeaveRequestAccepted means a user left a group since a group admin accepted his leave request.
	LeaveRequestAccepted GroupMembershipAction = "leave_request_accepted"
	// InvitationRefused means a user refused an invitation to join a group.
	InvitationRefused GroupMembershipAction = "invitation_refused"
	// InvitationWithdrawn means an admin withdrew his invitation to join a group.
	InvitationWithdrawn GroupMembershipAction = "invitation_withdrawn"
	// JoinedByBadge means a user has been added into a group because of a badge returned by the login module.
	JoinedByBadge GroupMembershipAction = "joined_by_badge"
	// JoinedByCode means a user joined a group by the group's code.
	JoinedByCode GroupMembershipAction = "joined_by_code"
	// JoinRequestRefused means an admin refused a user's request to join a group.
	JoinRequestRefused GroupMembershipAction = "join_request_refused"
	// JoinRequestWithdrawn means a user withdrew his request to join a group.
	JoinRequestWithdrawn GroupMembershipAction = "join_request_withdrawn"
	// Removed means a user was removed from a group.
	Removed GroupMembershipAction = "removed"
	// Left means a user left a group.
	Left GroupMembershipAction = "left"
	// IsMember means a user is a member of a group.
	IsMember GroupMembershipAction = "is_member"
	// LeaveRequestCreated means a pending user's request to leave a group was created.
	LeaveRequestCreated GroupMembershipAction = "is_member,leave_request_created"
	// LeaveRequestExpired means a pending user's leave request for an expired membership.
	LeaveRequestExpired GroupMembershipAction = "leave_request_created"
	// LeaveRequestRefused means a manager refused a user's request to leave a group.
	LeaveRequestRefused GroupMembershipAction = "leave_request_refused"
	// LeaveRequestWithdrawn means a user withdrew his request to leave a group.
	LeaveRequestWithdrawn GroupMembershipAction = "leave_request_withdrawn"
	// NoRelation means there is no row for the group pair in the groups_groups/group_pending_requests tables.
	NoRelation GroupMembershipAction = ""
)

func (groupMembershipAction GroupMembershipAction) isActive() bool {
	switch groupMembershipAction {
	case JoinedByBadge, InvitationAccepted, JoinRequestAccepted, JoinedByCode, IsMember,
		LeaveRequestCreated, LeaveRequestWithdrawn, LeaveRequestRefused:
		return true
	}
	return false
}

func (groupMembershipAction GroupMembershipAction) isPending() bool {
	switch groupMembershipAction {
	case InvitationCreated, JoinRequestCreated, LeaveRequestCreated, LeaveRequestExpired:
		return true
	}
	return false
}

func (groupMembershipAction GroupMembershipAction) hasApprovals() bool {
	return groupMembershipAction == JoinRequestCreated
}

// PendingType converts the GroupMembershipAction into `group_pending_requests.type`.
func (groupMembershipAction GroupMembershipAction) PendingType() string {
	switch groupMembershipAction {
	case InvitationCreated:
		return "invitation"
	case JoinRequestCreated:
		return "join_request"
	case LeaveRequestCreated:
		return "leave_request"
	}
	panic("groupMembershipAction should be of pending kind in PendingType()")
}

// GroupGroupTransitionAction represents a groups_groups relation transition action.
type GroupGroupTransitionAction int

const (
	// AdminCreatesInvitation means a group admin invites new users to the group.
	AdminCreatesInvitation GroupGroupTransitionAction = iota
	// UserCreatesJoinRequest means a user creates a request to become a group member.
	UserCreatesJoinRequest
	// UserCreatesAcceptedJoinRequest means a user adds himself into a group that he owns
	// It doesn't check if the user owns the group / all needed approvals are given (a calling service should check that).
	UserCreatesAcceptedJoinRequest
	// UserAcceptsInvitation means a user accepts a group invitation.
	UserAcceptsInvitation
	// AdminAcceptsJoinRequest means a group admin accepts a request to join a group.
	// For this action we check that all the approvals required by the group are given in the join request
	// and set groups_groups.*_approved_at to group_pending_requests.at for each.
	AdminAcceptsJoinRequest
	// AdminAcceptsLeaveRequest means a group admin accepts a request to leave a group.
	AdminAcceptsLeaveRequest
	// AdminRefusesLeaveRequest means a group admin refuses a request to leave a group.
	AdminRefusesLeaveRequest
	// UserRefusesInvitation means a user refuses a group invitation.
	UserRefusesInvitation
	// AdminRefusesJoinRequest means a group admin refuses a request to join the group.
	AdminRefusesJoinRequest
	// AdminRemovesUser means a group admin removes a user from a group. It marks relations as "removed".
	// It doesn't check if a child is a user or not.
	AdminRemovesUser
	// AdminWithdrawsInvitation means a group admin withdraws an invitation.
	AdminWithdrawsInvitation
	// UserLeavesGroup means a user leaves a group.
	UserLeavesGroup
	// UserCreatesLeaveRequest means a user creates a request to leave a group
	// We don't check that groups.require_lock_membership_approval_until & groups_groups.lock_membership_approved_at
	// are not null (a calling service should check that by itself).
	UserCreatesLeaveRequest
	// UserCancelsJoinRequest means a user cancels his request to join a group.
	UserCancelsJoinRequest
	// UserCancelsLeaveRequest means a user cancels his request to leave a group.
	UserCancelsLeaveRequest
	// AdminRemovesDirectRelation removes a direct relation.
	AdminRemovesDirectRelation
	// UserJoinsGroupByBadge means we add a user into a group because of his badge returned by the login module.
	UserJoinsGroupByBadge
	// UserJoinsGroupByCode means a user joins a group using a group's code
	// We don't check the code here (a calling service should check the code by itself).
	UserJoinsGroupByCode
)

type groupGroupTransitionRule struct {
	// Transitions defines all possible transitions for the action. The format is "FromAction->ToAction".
	// Relations that have "from" action not listed here are considered as invalid.
	Transitions map[GroupMembershipAction]GroupMembershipAction

	IfNotEnoughApprovalsDowngradeTo GroupMembershipAction
}

var groupGroupTransitionRules = map[GroupGroupTransitionAction]groupGroupTransitionRule{
	AdminCreatesInvitation: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			NoRelation:          InvitationCreated,
			InvitationCreated:   InvitationCreated,
			JoinRequestCreated:  JoinRequestAccepted,
			LeaveRequestExpired: InvitationCreated,
		},
		IfNotEnoughApprovalsDowngradeTo: InvitationCreated, // only JoinRequestAccepted requires approvals.
	},
	UserCreatesJoinRequest: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			NoRelation:          JoinRequestCreated,
			JoinRequestCreated:  JoinRequestCreated,
			LeaveRequestExpired: JoinRequestCreated,
		},
	},
	UserCreatesAcceptedJoinRequest: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			NoRelation:          JoinRequestAccepted,
			JoinRequestCreated:  JoinRequestAccepted,
			InvitationCreated:   JoinRequestAccepted,
			LeaveRequestExpired: JoinRequestAccepted,
		},
	},
	UserJoinsGroupByBadge: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			NoRelation:          JoinedByBadge,
			JoinRequestCreated:  JoinedByBadge,
			InvitationCreated:   JoinedByBadge,
			LeaveRequestExpired: JoinedByBadge,
		},
	},
	UserJoinsGroupByCode: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			NoRelation:          JoinedByCode,
			JoinRequestCreated:  JoinedByCode,
			InvitationCreated:   JoinedByCode,
			LeaveRequestExpired: JoinedByCode,
		},
	},
	UserAcceptsInvitation: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			InvitationCreated: InvitationAccepted,
		},
	},
	AdminAcceptsJoinRequest: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			JoinRequestCreated: JoinRequestAccepted,
		},
	},
	AdminAcceptsLeaveRequest: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			LeaveRequestCreated: LeaveRequestAccepted,
		},
	},
	UserRefusesInvitation: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			InvitationCreated: InvitationRefused,
		},
	},
	AdminRefusesJoinRequest: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			JoinRequestCreated: JoinRequestRefused,
		},
	},
	AdminRefusesLeaveRequest: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			LeaveRequestCreated: LeaveRequestRefused,
		},
	},
	AdminRemovesUser: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			IsMember:            Removed,
			LeaveRequestCreated: Removed,
		},
	},
	AdminWithdrawsInvitation: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			InvitationCreated: InvitationWithdrawn,
		},
	},
	UserLeavesGroup: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			IsMember:            Left,
			LeaveRequestCreated: Left,
		},
	},
	UserCreatesLeaveRequest: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			IsMember:            LeaveRequestCreated,
			LeaveRequestCreated: LeaveRequestCreated,
		},
	},
	UserCancelsJoinRequest: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			JoinRequestCreated: JoinRequestWithdrawn,
		},
	},
	UserCancelsLeaveRequest: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			LeaveRequestCreated: LeaveRequestWithdrawn,
		},
	},
	AdminRemovesDirectRelation: {
		Transitions: map[GroupMembershipAction]GroupMembershipAction{
			IsMember:            NoRelation,
			NoRelation:          NoRelation,
			LeaveRequestCreated: NoRelation,
		},
	},
}

// GroupGroupTransitionResult is an enum{cycle, invalid, success, unchanged}.
type GroupGroupTransitionResult string

const (
	// Cycle means that the transition wasn't performed because it would create a cycle in groups_groups graph.
	Cycle GroupGroupTransitionResult = "cycle"
	// Invalid means that the transition is impossible.
	Invalid GroupGroupTransitionResult = "invalid"
	// ApprovalsMissing means that one or more approvals required by the transition are missing.
	ApprovalsMissing GroupGroupTransitionResult = "approvals_missing"
	// Full means that the parent group is full (in terms of `groups.max_participants`) when `enforce_max_participants` is true
	// (The number of participants is computed as the number of non-expired users or teams which are direct children
	//  of the group + invitations (join requests are not counted)).
	Full GroupGroupTransitionResult = "full"
	// Success means that the transition was performed successfully.
	Success GroupGroupTransitionResult = "success"
	// Unchanged means that the transition has been already performed.
	Unchanged GroupGroupTransitionResult = "unchanged"
)

// GroupGroupTransitionResults represents results of mass transition (format: map{ id -> GroupGroupTransitionResult }).
type GroupGroupTransitionResults map[int64]GroupGroupTransitionResult

// GroupApprovals represents all the approvals that can be given by a user to the group managers.
type GroupApprovals struct {
	PersonalInfoViewApproval bool
	LockMembershipApproval   bool
	WatchApproval            bool
}

// FromString initializes GroupApprovals from the given comma-separated list of approvals.
func (approvals *GroupApprovals) FromString(s string) {
	approvalsList := strings.Split(s, ",")
	for _, approval := range approvalsList {
		switch approval {
		case "personal_info_view":
			approvals.PersonalInfoViewApproval = true
		case "lock_membership":
			approvals.LockMembershipApproval = true
		case "watch":
			approvals.WatchApproval = true
		}
	}
}

// ToArray converts GroupApprovals to a list of approvals.
func (approvals *GroupApprovals) ToArray() []string {
	approvalsList := make([]string, 0, 3)
	if approvals.PersonalInfoViewApproval {
		approvalsList = append(approvalsList, "personal_info_view")
	}
	if approvals.LockMembershipApproval {
		approvalsList = append(approvalsList, "lock_membership")
	}
	if approvals.WatchApproval {
		approvalsList = append(approvalsList, "watch")
	}
	return approvalsList
}

type stateInfo struct {
	ChildGroupID               int64
	Action                     GroupMembershipAction
	ApprovalsOK                bool
	PersonalInfoViewApprovedAt *Time
	LockMembershipApprovedAt   *Time
	WatchApprovedAt            *Time
}

type requiredApprovalsAndLimits struct {
	RequirePersonalInfoAccessApproval bool
	RequireLockMembershipApproval     bool
	RequireWatchApproval              bool
	EnforceMaxParticipants            bool
	MaxParticipants                   int
}

// Transition performs a groups_groups relation transition according to groupGroupTransitionRules.
func (s *GroupGroupStore) Transition(action GroupGroupTransitionAction,
	parentGroupID int64, childGroupIDs []int64, approvals map[int64]GroupApprovals,
	performedByUserID int64,
) (results GroupGroupTransitionResults, approvalsToRequest map[int64]GroupApprovals, err error) {
	s.mustBeInTransaction()
	defer recoverPanics(&err)

	results = make(map[int64]GroupGroupTransitionResult, len(childGroupIDs))
	approvalsToRequest = make(map[int64]GroupApprovals, len(childGroupIDs))

	mustNotBeError(s.WithGroupsRelationsLock(func(dataStore *DataStore) error {
		var oldActions []stateInfo
		var groupRequiredApprovalsAndLimits requiredApprovalsAndLimits

		mustNotBeError(dataStore.Groups().ByID(parentGroupID).
			Select(`
				require_personal_info_access_approval != 'none' AS require_personal_info_access_approval,
				NOW() < IFNULL(require_lock_membership_approval_until, 0) AS require_lock_membership_approval,
				require_watch_approval, enforce_max_participants, max_participants`).
			WithWriteLock().Scan(&groupRequiredApprovalsAndLimits).Error())

		// Here we get current states for each childGroupID:
		// the current state can be one of
		// ("", "invitation_created", "join_request_created", "is_member", "is_member,leave_request_created")
		// where "is_member" means that childGroupID is a member of the parentGroupID
		mustNotBeError(
			dataStore.Raw(`
				SELECT child_group_id, GROUP_CONCAT(action) AS action,
					MAX(personal_info_view_approved_at) AS personal_info_view_approved_at,
					MAX(lock_membership_approved_at) AS lock_membership_approved_at, MAX(watch_approved_at) AS watch_approved_at
					FROM ((? FOR UPDATE) UNION (? FOR UPDATE)) AS statuses
					GROUP BY child_group_id`,
				dataStore.ActiveGroupGroups().
					Select(`
						child_group_id, 'is_member' AS action,
						NULL AS personal_info_view_approved_at,
						NULL AS lock_membership_approved_at,
						NULL AS watch_approved_at`).
					Where("parent_group_id = ? AND child_group_id IN (?)", parentGroupID, childGroupIDs).QueryExpr(),
				dataStore.GroupPendingRequests().
					Select(`
						member_id,
						CASE group_pending_requests.type
							WHEN 'invitation' THEN 'invitation_created'
							WHEN 'join_request' THEN 'join_request_created'
							WHEN 'leave_request' THEN 'leave_request_created'
							ELSE group_pending_requests.type
						END,
						IF(personal_info_view_approved, at, NULL) AS personal_info_view_approved_at,
						IF(lock_membership_approved, at, NULL) AS lock_membership_approved_at,
						IF(watch_approved, at, NULL) AS watch_approved_at`).
					Joins("JOIN `groups` ON `groups`.`id` = group_pending_requests.group_id").
					Where("group_id = ? AND member_id IN (?)", parentGroupID, childGroupIDs).QueryExpr()).
				Scan(&oldActions).Error())

		oldActionsMap := make(map[int64]stateInfo, len(childGroupIDs))
		for _, oldAction := range oldActions {
			oldActionsMap[oldAction.ChildGroupID] = oldAction
		}

		// build the transition plan depending on the current states (oldActionsMap)
		idsToInsertPending, idsToInsertRelation, idsToCheckCycle, idsToDeletePending,
			idsToDeleteRelation, idsChanged := buildTransitionsPlan(
			parentGroupID, childGroupIDs, results, oldActionsMap, &groupRequiredApprovalsAndLimits, approvals, approvalsToRequest, action)

		performCyclesChecking(dataStore, idsToCheckCycle, parentGroupID, results, idsToInsertPending, idsToInsertRelation,
			idsToDeletePending, idsToDeleteRelation, idsChanged)

		enforceMaxSize(dataStore, action, parentGroupID, &groupRequiredApprovalsAndLimits, results, idsToInsertPending,
			idsToInsertRelation, idsToDeletePending, idsToDeleteRelation, idsChanged)

		shouldCreateNewAncestors := false
		shouldPropagatePermissions := false
		if len(idsToDeletePending) > 0 {
			idsToDeleteSlice := make([]int64, 0, len(idsToDeletePending))
			for id := range idsToDeletePending {
				idsToDeleteSlice = append(idsToDeleteSlice, id)
			}
			mustNotBeError(dataStore.GroupPendingRequests().Delete("group_id = ? AND member_id IN (?)", parentGroupID, idsToDeleteSlice).Error())
		}
		if len(idsToDeleteRelation) > 0 {
			idsToDeleteSlice := make([]int64, 0, len(idsToDeleteRelation))
			for id := range idsToDeleteRelation {
				idsToDeleteSlice = append(idsToDeleteSlice, id)
			}
			mustNotBeError(dataStore.GroupGroups().Delete("parent_group_id = ? AND child_group_id IN (?)", parentGroupID, idsToDeleteSlice).Error())
			result := dataStore.PermissionsGranted().
				Delete("origin = 'group_membership' AND source_group_id = ? AND group_id IN (?)", parentGroupID, idsToDeleteSlice)
			mustNotBeError(result.Error())
			shouldCreateNewAncestors = true
			shouldPropagatePermissions = result.RowsAffected() > 0
		}

		insertGroupPendingRequests(dataStore, idsToInsertPending, parentGroupID, approvals)

		if len(idsToInsertRelation) > 0 {
			insertQuery := `
				INSERT INTO groups_groups (
					parent_group_id, child_group_id, personal_info_view_approved_at,
					lock_membership_approved_at, watch_approved_at
				)`
			valuesTemplate := `(?, ?, ?, ?, ?)`
			insertQuery += " VALUES " +
				strings.Repeat(valuesTemplate+", ", len(idsToInsertRelation)-1) +
				valuesTemplate // #nosec
			insertQuery += " ON DUPLICATE KEY UPDATE expires_at = '9999-12-31 23:59:59'"
			values := make([]interface{}, 0, len(idsToInsertRelation)*6)
			for id := range idsToInsertRelation {
				personalInfoViewApprovedAt, lockMembershipApprovedAt, watchApprovedAt := resolveApprovalTimesForGroupsGroups(
					oldActionsMap, id, approvals,
				)
				values = append(values, parentGroupID, id,
					personalInfoViewApprovedAt, lockMembershipApprovedAt, watchApprovedAt)
			}
			shouldCreateNewAncestors = true
			mustNotBeError(dataStore.Exec(insertQuery, values...).Error())
		}

		insertGroupMembershipChanges(dataStore, idsChanged, parentGroupID, performedByUserID)

		if shouldCreateNewAncestors {
			dataStore.GroupGroups().createNewAncestors()
			if shouldPropagatePermissions {
				dataStore.PermissionsGranted().computeAllAccess()
			}
			dataStore.ScheduleResultsPropagation()
		}
		return nil
	}))
	return results, approvalsToRequest, nil
}

func enforceMaxSize(dataStore *DataStore, action GroupGroupTransitionAction, parentGroupID int64,
	limits *requiredApprovalsAndLimits, results GroupGroupTransitionResults, idsToInsertPending map[int64]GroupMembershipAction,
	idsToInsertRelation, idsToDeletePending, idsToDeleteRelation map[int64]bool, idsChanged map[int64]GroupMembershipAction,
) {
	if !limits.EnforceMaxParticipants || !map[GroupGroupTransitionAction]bool{
		UserJoinsGroupByBadge: true, UserJoinsGroupByCode: true, UserCreatesJoinRequest: true, UserCreatesAcceptedJoinRequest: true,
		AdminCreatesInvitation: true, AdminAcceptsJoinRequest: true,
	}[action] {
		return
	}

	changedIDsList := make([]int64, 0, len(idsChanged))
	for id := range idsChanged {
		changedIDsList = append(changedIDsList, id)
	}
	var activeRelationsCount int64
	mustNotBeError(dataStore.ActiveGroupGroups().Where("parent_group_id = ?", parentGroupID).
		Joins("JOIN `groups` ON groups.id = child_group_id").
		Where("groups.type IN ('User', 'Team')").
		Where("child_group_id NOT IN(?)", changedIDsList).WithWriteLock().Count(&activeRelationsCount).Error())
	var invitationsCount int64
	mustNotBeError(dataStore.GroupPendingRequests().
		Where("group_id = ?", parentGroupID).
		Where("type = 'invitation'").Where("member_id NOT IN(?)", changedIDsList).
		WithWriteLock().Count(&invitationsCount).Error())

	membersCount := activeRelationsCount + invitationsCount
	for _, itemAction := range idsChanged {
		if itemAction.isActive() || itemAction == InvitationCreated {
			membersCount++
		}
	}
	if membersCount > int64(limits.MaxParticipants) || membersCount == int64(limits.MaxParticipants) && action == UserCreatesJoinRequest {
		deleteIDsFromTransitionPlan(changedIDsList, Full, results,
			idsToInsertPending, idsToInsertRelation, idsToDeletePending, idsToDeleteRelation, idsChanged)
	}
}

func resolveApprovalTimesForGroupsGroups(oldActionsMap map[int64]stateInfo, id int64, approvals map[int64]GroupApprovals) (
	personalInfoViewApprovedAt, lockMembershipApprovedAt, watchApprovedAt interface{},
) {
	personalInfoViewApprovedAt = gorm.Expr("NULL")
	lockMembershipApprovedAt = gorm.Expr("NULL")
	watchApprovedAt = gorm.Expr("NULL")
	if oldActionsMap[id].Action.hasApprovals() {
		personalInfoViewApprovedAt = oldActionsMap[id].PersonalInfoViewApprovedAt
		lockMembershipApprovedAt = oldActionsMap[id].LockMembershipApprovedAt
		watchApprovedAt = oldActionsMap[id].WatchApprovedAt
	} else {
		if approvals[id].PersonalInfoViewApproval {
			personalInfoViewApprovedAt = Now()
		}
		if approvals[id].LockMembershipApproval {
			lockMembershipApprovedAt = Now()
		}
		if approvals[id].WatchApproval {
			watchApprovedAt = Now()
		}
	}
	return personalInfoViewApprovedAt, lockMembershipApprovedAt, watchApprovedAt
}

func insertGroupPendingRequests(dataStore *DataStore, idsToInsertPending map[int64]GroupMembershipAction,
	parentGroupID int64, approvals map[int64]GroupApprovals,
) {
	if len(idsToInsertPending) > 0 {
		insertQuery := `
			INSERT INTO group_pending_requests
				(group_id, member_id, ` + "`type`" + `, personal_info_view_approved,
				 lock_membership_approved, watch_approved)`
		valuesTemplate := "(?, ?, ?, ?, ?, ?)"
		insertQuery += " VALUES " +
			strings.Repeat(valuesTemplate+", ", len(idsToInsertPending)-1) +
			valuesTemplate // #nosec
		values := make([]interface{}, 0, len(idsToInsertPending)*6)
		for id, groupMembershipAction := range idsToInsertPending {
			values = append(values, parentGroupID, id, groupMembershipAction.PendingType(),
				approvals[id].PersonalInfoViewApproval, approvals[id].LockMembershipApproval,
				approvals[id].WatchApproval)
		}
		mustNotBeError(dataStore.db.Exec(insertQuery, values...).Error)
	}
}

func insertGroupMembershipChanges(dataStore *DataStore, idsChanged map[int64]GroupMembershipAction,
	parentGroupID, performedByUserID int64,
) {
	if len(idsChanged) > 0 {
		insertQuery := "INSERT INTO group_membership_changes (group_id, member_id, action, at, initiator_id)"
		valuesTemplate := "(?, ?, ?, NOW(3), ?)"
		const paramsCount = 4
		insertQuery += " VALUES " +
			strings.Repeat(valuesTemplate+", ", len(idsChanged)-1) +
			valuesTemplate // #nosec
		mustNotBeError(dataStore.retryOnDuplicatePrimaryKeyError(func(db *DB) error {
			values := make([]interface{}, 0, len(idsChanged)*paramsCount)
			for id, toAction := range idsChanged {
				values = append(values, parentGroupID, id, toAction[strings.LastIndex(string(toAction), ",")+1:], performedByUserID)
			}
			return dataStore.db.Exec(insertQuery, values...).Error
		}))
	}
}

func performCyclesChecking(s *DataStore, idsToCheckCycle map[int64]bool, parentGroupID int64,
	results GroupGroupTransitionResults, idsToInsertPending map[int64]GroupMembershipAction, idsToInsertRelation,
	idsToDeletePending, idsToDeleteRelation map[int64]bool, idsChanged map[int64]GroupMembershipAction,
) {
	if len(idsToCheckCycle) > 0 {
		idsToCheckCycleSlice := make([]int64, 0, len(idsToCheckCycle))
		for id := range idsToCheckCycle {
			idsToCheckCycleSlice = append(idsToCheckCycleSlice, id)
		}
		var cycleIDs []int64
		mustNotBeError(s.GroupAncestors().
			WithWriteLock().
			Where("child_group_id = ? AND ancestor_group_id IN (?)", parentGroupID, idsToCheckCycleSlice).
			Pluck("ancestor_group_id", &cycleIDs).Error())

		deleteIDsFromTransitionPlan(cycleIDs, Cycle, results,
			idsToInsertPending, idsToInsertRelation, idsToDeletePending, idsToDeleteRelation, idsChanged)
	}
}

func deleteIDsFromTransitionPlan(ids []int64, status GroupGroupTransitionResult,
	results GroupGroupTransitionResults, idsToInsertPending map[int64]GroupMembershipAction, idsToInsertRelation,
	idsToDeletePending, idsToDeleteRelation map[int64]bool, idsChanged map[int64]GroupMembershipAction,
) {
	for _, groupID := range ids {
		results[groupID] = status
		delete(idsToInsertRelation, groupID)
		delete(idsToInsertPending, groupID)
		delete(idsToDeletePending, groupID)
		delete(idsToDeleteRelation, groupID)
		delete(idsChanged, groupID)
	}
}

func buildTransitionsPlan(parentGroupID int64, childGroupIDs []int64, results GroupGroupTransitionResults,
	oldActionsMap map[int64]stateInfo, groupRequiredApprovals *requiredApprovalsAndLimits,
	approvals, approvalsToRequest map[int64]GroupApprovals, action GroupGroupTransitionAction,
) (idsToInsertPending map[int64]GroupMembershipAction, idsToInsertRelation, idsToCheckCycle,
	idsToDeletePending, idsToDeleteRelation map[int64]bool, idsChanged map[int64]GroupMembershipAction,
) {
	idsToCheckCycle = make(map[int64]bool, len(childGroupIDs))
	idsToDeletePending = make(map[int64]bool, len(childGroupIDs))
	idsToDeleteRelation = make(map[int64]bool, len(childGroupIDs))
	idsToInsertPending = make(map[int64]GroupMembershipAction, len(childGroupIDs))
	idsToInsertRelation = make(map[int64]bool, len(childGroupIDs))
	idsChanged = make(map[int64]GroupMembershipAction, len(childGroupIDs))
	for _, id := range childGroupIDs {
		results[id] = Invalid
		if id == parentGroupID {
			continue
		}

		oldAction := oldActionsMap[id]

		if toAction, toActionOK := groupGroupTransitionRules[action].Transitions[oldAction.Action]; toActionOK {
			if toAction.isActive() && !oldAction.Action.isActive() || toAction.hasApprovals() {
				if ok, approvalsNeeded := approvalsOK(&oldAction, groupRequiredApprovals, approvals[id]); !ok {
					if groupGroupTransitionRules[action].IfNotEnoughApprovalsDowngradeTo != NoRelation {
						toAction = groupGroupTransitionRules[action].IfNotEnoughApprovalsDowngradeTo
					} else {
						results[id] = ApprovalsMissing
						if approvalsNeeded != (GroupApprovals{}) {
							approvalsToRequest[id] = approvalsNeeded
						}
						continue
					}
				}
			}

			buildOneTransition(id, oldAction, toAction, results, idsToInsertPending, idsToInsertRelation, idsToCheckCycle,
				idsToDeletePending, idsToDeleteRelation, idsChanged)
		}
	}
	return idsToInsertPending, idsToInsertRelation, idsToCheckCycle, idsToDeletePending, idsToDeleteRelation, idsChanged
}

func buildOneTransition(id int64, oldAction stateInfo, toAction GroupMembershipAction,
	results GroupGroupTransitionResults,
	idsToInsertPending map[int64]GroupMembershipAction, idsToInsertRelation, idsToCheckCycle, idsToDeletePending,
	idsToDeleteRelation map[int64]bool, idsChanged map[int64]GroupMembershipAction,
) {
	if toAction != oldAction.Action {
		if toAction != NoRelation {
			idsChanged[id] = toAction
		}
		results[id] = Success
		if oldAction.Action.isActive() {
			if !toAction.isActive() {
				idsToDeleteRelation[id] = true
			}
		} else {
			if toAction.isActive() {
				idsToInsertRelation[id] = true
			}
			if toAction.isActive() || toAction.isPending() {
				idsToCheckCycle[id] = true
			}
		}
		if oldAction.Action.isPending() {
			idsToDeletePending[id] = true
		}
		if toAction.isPending() {
			idsToInsertPending[id] = toAction
		}
	} else {
		results[id] = Unchanged
	}
}

func approvalsOK(oldAction *stateInfo, groupRequiredApprovals *requiredApprovalsAndLimits, approvals GroupApprovals) (
	ok bool, approvalsToRequest GroupApprovals,
) {
	var approvalsToCheck GroupApprovals
	if oldAction.Action.hasApprovals() {
		approvalsToCheck.PersonalInfoViewApproval = oldAction.PersonalInfoViewApprovedAt != nil
		approvalsToCheck.LockMembershipApproval = oldAction.LockMembershipApprovedAt != nil
		approvalsToCheck.WatchApproval = oldAction.WatchApprovedAt != nil
	} else {
		approvalsToCheck = approvals
	}
	approvalsToRequest.PersonalInfoViewApproval = groupRequiredApprovals.RequirePersonalInfoAccessApproval &&
		!approvalsToCheck.PersonalInfoViewApproval
	approvalsToRequest.LockMembershipApproval = groupRequiredApprovals.RequireLockMembershipApproval &&
		!approvalsToCheck.LockMembershipApproval
	approvalsToRequest.WatchApproval = groupRequiredApprovals.RequireWatchApproval && !approvalsToCheck.WatchApproval
	return approvalsToRequest == GroupApprovals{}, approvalsToRequest
}
