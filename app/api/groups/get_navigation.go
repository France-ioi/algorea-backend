package groups

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

type groupNavigationViewResponseChild struct {
	// required:true
	ID int64 `json:"id,string"`
	// required:true
	Name string `json:"name"`
	// required:true
	// enum: Class,Team,Club,Friends,Other,Session,Base
	Type string `json:"type"`
	// whether the user is a member of this group or one of its descendants
	// required:true
	// enum: none,direct,descendant
	CurrentUserMembership string `json:"current_user_membership"`
	// whether the user (or its ancestor) is a manager of this group,
	// or a manager of one of this group's ancestors (so is implicitly manager of this group) or,
	// a manager of one of this group's non-user descendants, or none of above
	// required: true
	// enum: none,direct,ancestor,descendant
	CurrentUserManagership string `json:"current_user_managership"`
}

// swagger:model groupNavigationViewResponse
type groupNavigationViewResponse struct {
	// required:true
	ID int64 `json:"id,string"`
	// required:true
	Name string `json:"name"`
	// required:true
	// enum: Class,Team,Club,Friends,Other,Session,Base
	Type string `json:"type"`
	// required:true
	Children []groupNavigationViewResponseChild `json:"children" gorm:"-"`
}

// swagger:operation GET /groups/{group_id}/navigation group-memberships groupNavigationView
//
//		---
//		summary: Get navigation data
//		description: >
//
//	  Lists child groups visible to the user, so either 1) ancestors of a group he joined, or
//	  2) ancestors of a non-user group he manages, or 3) descendants of a group he manages, or
//	  4) groups with `is_public` = 1. Ordered alphabetically by name.
//
//
//	  The input group should be visible to the current user with the same definition as above,
//	  otherwise the 'forbidden' error is returned. If the group is a user, the 'forbidden' error is returned as well.
//
//		parameters:
//			- name: group_id
//				in: path
//				type: integer
//				required: true
//			- name: limit
//				description: Display the first N children
//				in: query
//				type: integer
//				maximum: 1000
//				default: 500
//		responses:
//			"200":
//				description: OK. Success response with an array root groups
//				schema:
//					"$ref": "#/definitions/groupNavigationViewResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getNavigation(w http.ResponseWriter, r *http.Request) service.APIError {
	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(r)
	store := srv.GetStore(r)

	var result groupNavigationViewResponse
	err = store.Groups().PickVisibleGroups(store.Groups().ByID(groupID), user).
		Where("groups.type != 'User'").
		Select("id, name, type").Take(&result).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return service.InsufficientAccessRightsError
	}
	service.MustNotBeError(err)

	query := store.Groups().PickVisibleGroups(store.Groups().DB, user).
		Joins(`
			JOIN groups_groups_active
				ON groups_groups_active.child_group_id = groups.id AND groups_groups_active.parent_group_id = ?`, groupID).
		Where("groups.type != 'User'").
		Order("name")
	query = service.NewQueryLimiter().Apply(r, query)

	service.MustNotBeError(selectGroupsDataForMenu(store, query, user, "").Scan(&result.Children).Error())

	render.Respond(w, r, result)
	return service.NoError
}
