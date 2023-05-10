package users

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
	"github.com/France-ioi/AlgoreaBackend/app/structures"
)

// ManagerPermissionsPart contains fields related to permissions for managing the user.
// These fields are only displayed if the current user is a manager of the user.
// swagger:ignore
type ManagerPermissionsPart struct {
	CurrentUserIsManager bool `json:"-"`
	// returned only if the current user is a manager
	CurrentUserCanGrantUserAccess bool `json:"current_user_can_grant_user_access"`
	// returned only if the current user is a manager
	CurrentUserCanWatchUser bool `json:"current_user_can_watch_user"`
}

// swagger:model
type userViewResponse struct {
	// required: true
	GroupID int64 `json:"group_id,string"`
	// required: true
	TempUser bool `json:"temp_user"`
	// required: true
	Login string `json:"login"`
	// Nullable
	// required: true
	FreeText *string `json:"free_text"`
	// Nullable
	// required: true
	WebSite *string `json:"web_site"`

	*structures.UserPersonalInfo
	ShowPersonalInfo bool `json:"-"`

	// list of ancestor (excluding the user himself) groups that the current user (or his ancestor groups) is manager of
	// required:true
	AncestorsCurrentUserIsManagerOf []structures.GroupShortInfo `json:"ancestors_current_user_is_manager_of" gorm:"-"`

	*ManagerPermissionsPart

	// required: true
	IsCurrentUser bool `json:"is_current_user"`
}

// swagger:operation GET /users/{user_id} users userViewByID
//
//	---
//	summary: Get profile info for a user by ID
//	description: Returns data from the `users` table for the given `{user_id}`
//              (`first_name` and `last_name` are only shown for the authenticated user or
//               if the user approved access to their personal info for some group
//               managed by the authenticated user) along with some permissions if the current user is a manager.
//	parameters:
//		- name: user_id
//			in: path
//			type: integer
//			format: int64
//			required: true
//	responses:
//		"200":
//			description: OK. Success response with user's data
//			schema:
//				"$ref": "#/definitions/userViewResponse"
//		"401":
//			"$ref": "#/responses/unauthorizedResponse"
//		"403":
//			"$ref": "#/responses/forbiddenResponse"
//		"404":
//			"$ref": "#/responses/notFoundResponse"
//		"500":
//			"$ref": "#/responses/internalErrorResponse"

// swagger:operation GET /users/by-login/{login} users userViewByLogin
//
//		---
//		summary: Get profile info for a user by login
//		description: Returns data from the `users` table for the given `{login}`
//	             (`first_name` and `last_name` are only shown for the authenticated user or
//	              if the user approved access to their personal info for some group
//	              managed by the authenticated user) along with some permissions if the current user is a manager.
//		parameters:
//			- name: login
//				in: path
//				type: string
//				required: true
//		responses:
//			"200":
//				description: OK. Success response with user's data
//				schema:
//					"$ref": "#/definitions/userViewResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"404":
//				"$ref": "#/responses/notFoundResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getUser(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)

	var scope *database.DB
	store := srv.GetStore(r)
	if userLogin := chi.URLParam(r, "login"); userLogin != "" {
		scope = store.Users().Where("login = ?", userLogin)
	} else {
		userID, err := service.ResolveURLQueryPathInt64Field(r, "user_id")
		if err != nil {
			return service.ErrInvalidRequest(err)
		}
		scope = store.Users().ByID(userID)
	}

	var userInfo userViewResponse
	err := scope.
		Select(`
			group_id, temp_user, login, free_text, web_site,
			users.group_id = ? OR personal_info_view_approvals.approved AS show_personal_info,
			IF(users.group_id = ? OR personal_info_view_approvals.approved, users.first_name, NULL) AS first_name,
			IF(users.group_id = ? OR personal_info_view_approvals.approved, users.last_name, NULL) AS last_name,
			manager_access.found AS current_user_is_manager,
			IF(manager_access.found, manager_access.can_grant_group_access, 0) AS current_user_can_grant_user_access,
			IF(manager_access.found, manager_access.can_watch_members, 0) AS current_user_can_watch_user`,
			user.GroupID, user.GroupID, user.GroupID).
		WithPersonalInfoViewApprovals(user).
		Joins(`
			LEFT JOIN LATERAL (?) AS manager_access ON 1`,
			store.GroupAncestors().ManagedByUser(user).
				Select(`
					1 AS found,
					MAX(can_manage_value) AS can_manage_value,
					MAX(can_grant_group_access) AS can_grant_group_access,
					MAX(can_watch_members) AS can_watch_members,
					groups_ancestors.child_group_id`).
				Where("groups_ancestors.child_group_id = users.group_id").
				Group("groups_ancestors.child_group_id").SubQuery()).
		Take(&userInfo).Error()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return service.ErrNotFound(errors.New("no such user"))
	}
	service.MustNotBeError(err)

	if !userInfo.ShowPersonalInfo {
		userInfo.UserPersonalInfo = nil
	}

	if userInfo.CurrentUserIsManager {
		service.MustNotBeError(store.Groups().ManagedBy(user).
			Joins(`
				JOIN groups_ancestors_active AS groups_ancestors
					ON groups_ancestors.ancestor_group_id = groups.id AND
						 NOT groups_ancestors.is_self AND
						 groups_ancestors.child_group_id = ?`, userInfo.GroupID).
			Group("groups.id").
			Order("groups.name").
			Select("groups.id, groups.name").
			Scan(&userInfo.AncestorsCurrentUserIsManagerOf).Error())
	} else {
		userInfo.ManagerPermissionsPart = nil
		userInfo.AncestorsCurrentUserIsManagerOf = make([]structures.GroupShortInfo, 0)
	}

	userInfo.IsCurrentUser = userInfo.GroupID == user.GroupID

	render.Respond(w, r, &userInfo)
	return service.NoError
}
