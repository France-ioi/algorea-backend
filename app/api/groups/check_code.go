package groups

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/domain"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// Only if valid.
type groupCodeCheckResponseGroup struct {
	// required: true
	Name string `json:"name"`
	// required: true
	// enum: none,view,edit
	RequirePersonalInfoAccessApproval string `json:"require_personal_info_access_approval"`
	// Nullable
	// required: true
	RequireLockMembershipApprovalUntil *database.Time `json:"require_lock_membership_approval_until"`
	// required: true
	RequireWatchApproval bool `json:"require_watch_approval"`
	// Nullable
	// required: true
	RootActivityID *int64 `json:"root_activity_id,string"`
	// Nullable
	// required: true
	RootSkillID *int64 `json:"root_skill_id,string"`
	// required: true
	Managers []struct {
		// required: true
		ID int64 `json:"id,string"`
		// required: true
		Login string `json:"login"`
		// Nullable
		// required: true
		FirstName *string `json:"first_name"`
		// Nullable
		// required: true
		LastName *string `json:"last_name"`
	} `json:"managers" gorm:"-"`
}

// swagger:model groupCodeCheckResponse
type groupCodeCheckResponse struct {
	// required: true
	Valid bool `json:"valid"`
	// Only if invalid
	// enum: no_group,frozen_membership,already_member,conflicting_team_participation,team_conditions_not_met
	Reason string                       `json:"reason,omitempty"`
	Group  *groupCodeCheckResponseGroup `json:"group,omitempty"`
}

// swagger:operation GET /groups/is-code-valid groups groupsCodeCheck
//
//		---
//		summary: Check if the group code is valid
//		description: >
//	  Checks if it is possible for the current user (or for a new user if the current user is temporary)
//	  to join a group with the given code.
//	  The service returns false:
//
//	  * if there is no group with `code_expires_at` > NOW() (or NULL), `code` = `{code}`, and `type` != 'User'
//	    (`reason` = 'no_group');
//
//	  * if the group is a team and the user is already on a team that has attempts for same contest
//	    while the contest doesn't allow multiple attempts or that has active attempts for the same contest
//	    (`reason` = 'conflicting_team_participation'),
//
//	  * if the group membership is frozen (`reason` = 'frozen_membership');
//
//	  * if there is already an active row in `groups_groups` with the found group as a parent
//	    and the user’s id as a child (`reason` = 'already_member');
//
//	  * if the group is a team and joining breaks entry conditions of at least one of the team's participations
//	    (i.e. any of `entry_min_admitted_members_ratio` or `entry_max_team_size` would not be satisfied)
//	    (`reason` = 'team_conditions_not_met').
//
//	  Otherwise, the service returns true.
//		parameters:
//			- name: code
//				in: query
//				type: string
//				required: true
//		responses:
//			"200":
//				description: OK. Validity of the code and additional info
//				schema:
//					"$ref": "#/definitions/groupCodeCheckResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) checkCode(w http.ResponseWriter, r *http.Request) service.APIError {
	code, err := service.ResolveURLQueryGetStringField(r, "code")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(r)
	store := srv.GetStore(r)
	userIDToCheck := user.GroupID
	if user.IsTempUser {
		userIDToCheck = domain.ConfigFromContext(r.Context()).AllUsersGroupID
	}

	valid, reason, groupID := checkGroupCodeForUser(store, userIDToCheck, code)
	response := groupCodeCheckResponse{
		Valid:  valid,
		Reason: string(reason),
	}

	if valid {
		var groupInfo groupCodeCheckResponseGroup
		service.MustNotBeError(store.Groups().ByID(groupID).Select(`
			id, name, require_personal_info_access_approval, require_lock_membership_approval_until,
			require_watch_approval, root_activity_id, root_skill_id`).Take(&groupInfo).Error())
		response.Group = &groupInfo
		service.MustNotBeError(store.GroupManagers().
			Select("users.group_id AS id, login, first_name, last_name").
			Where("group_managers.group_id = ?", groupID).
			Joins("JOIN users ON users.group_id = group_managers.manager_id").
			Order("users.login, users.group_id").Scan(&response.Group.Managers).Error())
	}

	render.Respond(w, r, &response)
	return service.NoError
}

type groupCodeFailReason string

const (
	okReason                           groupCodeFailReason = ""
	noGroupReason                      groupCodeFailReason = "no_group"
	frozenMembershipReason             groupCodeFailReason = "frozen_membership"
	alreadyMemberReason                groupCodeFailReason = "already_member"
	conflictingTeamParticipationReason groupCodeFailReason = "conflicting_team_participation"
	teamConditionsNotMetReason         groupCodeFailReason = "team_conditions_not_met"
)

func checkGroupCodeForUser(store *database.DataStore, userIDToCheck int64, code string) (
	valid bool, reason groupCodeFailReason, groupID int64,
) {
	info, err := store.GetGroupJoiningByCodeInfoByCode(code, false)
	service.MustNotBeError(err)
	if info == nil {
		return false, noGroupReason, 0
	}
	if info.FrozenMembership {
		return false, frozenMembershipReason, info.GroupID
	}

	alreadyMember, err := store.ActiveGroupGroups().
		Where("parent_group_id = ?", info.GroupID).Where("child_group_id = ?", userIDToCheck).
		HasRows()
	service.MustNotBeError(err)
	if alreadyMember {
		return false, alreadyMemberReason, info.GroupID
	}

	if info.Type != "Team" {
		return true, okReason, info.GroupID
	}

	found, err := store.CheckIfTeamParticipationsConflictWithExistingUserMemberships(info.GroupID, userIDToCheck, false)
	service.MustNotBeError(err)
	if found {
		return false, conflictingTeamParticipationReason, info.GroupID
	}

	ok, err := store.Groups().CheckIfEntryConditionsStillSatisfiedForAllActiveParticipations(info.GroupID, userIDToCheck, true, false)
	service.MustNotBeError(err)
	if !ok {
		return false, teamConditionsNotMetReason, info.GroupID
	}
	return true, okReason, info.GroupID
}
