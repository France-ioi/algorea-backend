package groups

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// GroupManagersViewResponseRowUser contains names of a manager.
type GroupManagersViewResponseRowUser struct {
	// Displayed only for users
	Login string `json:"login"`
	// Nullable; displayed only for users
	FirstName *string `json:"first_name"`
	// Nullable; displayed only for users
	LastName *string `json:"last_name"`
}

// GroupManagersViewResponseRowThroughAncestorGroups contains permissions propagated from ancestor groups.
type GroupManagersViewResponseRowThroughAncestorGroups struct {
	// enum: none,memberships,memberships_and_group
	// displayed only when include_managers_of_ancestor_groups=1, note that the group is an ancestor of itself
	CanManageThroughAncestorGroups string `json:"can_manage_through_ancestor_groups"`
	// displayed only when include_managers_of_ancestor_groups=1, note that the group is an ancestor of itself
	CanGrantGroupAccessThroughAncestorGroups bool `json:"can_grant_group_access_through_ancestor_groups"`
	// displayed only when include_managers_of_ancestor_groups=1, note that the group is an ancestor of itself
	CanWatchMembersThroughAncestorGroups bool `json:"can_watch_members_through_ancestor_groups"`
}

// swagger:model groupManagersViewResponseRow
type groupManagersViewResponseRow struct {
	// `groups.id`
	// required: true
	ID int64 `json:"id,string"`
	// `groups.name`
	// required: true
	Name string `json:"name"`

	// only for users
	*GroupManagersViewResponseRowUser
	// only when include_managers_of_ancestor_groups=1
	*GroupManagersViewResponseRowThroughAncestorGroups

	// enum: none,memberships,memberships_and_group
	// required: true
	CanManage string `json:"can_manage"`
	// required: true
	CanGrantGroupAccess bool `json:"can_grant_group_access"`
	// required: true
	CanWatchMembers bool `json:"can_watch_members"`

	Type                                string `json:"-"`
	CanManageValue                      int    `json:"-"`
	CanManageThroughAncestorGroupsValue int    `json:"-"`
}

// swagger:operation GET /groups/{group_id}/managers groups groupManagersView
//
//		---
//		summary: List group managers
//		description: >
//
//	  Lists managers of the given group and (optionally) managers of its ancestors
//	  (rows from the `group_managers` table with `group_id` = `{group_id}`) including managers' names.
//
//
//	  The authenticated user should be a manager of the `group_id` group or a member of the group or of its descendant,
//	  otherwise the 'forbidden' error is returned.
//		parameters:
//			- name: group_id
//				in: path
//				type: integer
//				required: true
//			- name: include_managers_of_ancestor_groups
//				description: If equal to 1, the results include managers of all ancestor groups
//				in: query
//				type: integer
//				enum: [0,1]
//				default: 0
//			- name: sort
//				in: query
//				default: [name,id]
//				type: array
//				items:
//					type: string
//					enum: [name,-name,id,-id]
//			- name: from.id
//				description: Start the page from the manager next to the manager with `groups.id`=`{from.id}`
//				in: query
//				type: integer
//			- name: limit
//				description: Display the first N managers
//				in: query
//				type: integer
//				maximum: 1000
//				default: 500
//		responses:
//			"200":
//				description: OK. The array of group managers
//				schema:
//					type: array
//					items:
//						"$ref": "#/definitions/groupManagersViewResponseRow"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getManagers(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)
	store := srv.GetStore(r)

	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	var includeManagersOfAncestorGroups bool
	if len(r.URL.Query()["include_managers_of_ancestor_groups"]) > 0 {
		includeManagersOfAncestorGroups, err = service.ResolveURLQueryGetBoolField(r, "include_managers_of_ancestor_groups")
		if err != nil {
			return service.ErrInvalidRequest(err)
		}
	}

	found, err := store.Raw("SELECT EXISTS(?) OR EXISTS(?) AS found HAVING found",
		store.GroupAncestors().ManagedByUser(user).Where("groups_ancestors.child_group_id = ?", groupID).SubQuery(),
		store.Groups().AncestorsOfJoinedGroups(store, user).Where("groups_ancestors_active.ancestor_group_id = ?", groupID).SubQuery(),
	).HasRows()
	service.MustNotBeError(err)
	if !found {
		return service.InsufficientAccessRightsError
	}

	query := store.GroupManagers().
		Joins("JOIN `groups` ON groups.id = group_managers.manager_id").
		Joins("LEFT JOIN users ON users.group_id = groups.id")

	if includeManagersOfAncestorGroups {
		query = query.
			Joins("JOIN groups_ancestors_active ON groups_ancestors_active.ancestor_group_id = group_managers.group_id").
			Where("groups_ancestors_active.child_group_id = ?", groupID).
			Select(`groups.id, groups.name, groups.type, users.first_name, users.last_name, users.login,
			        MAX(IF(groups_ancestors_active.is_self, can_manage_value, 1)) AS can_manage_value,
			        MAX(IF(groups_ancestors_active.is_self, can_grant_group_access, 0)) AS can_grant_group_access,
			        MAX(IF(groups_ancestors_active.is_self, can_watch_members, 0)) AS can_watch_members,
			        MAX(can_manage_value) AS can_manage_through_ancestor_groups_value,
			        MAX(can_grant_group_access) AS can_grant_group_access_through_ancestor_groups,
			        MAX(can_watch_members) AS can_watch_members_through_ancestor_groups`).
			Group("groups.id")
	} else {
		query = query.Where("group_managers.group_id = ?", groupID).
			Select(`groups.id, groups.name, groups.type, users.first_name, users.last_name, users.login,
              can_manage_value, can_grant_group_access, can_watch_members`)
	}

	query = service.NewQueryLimiter().Apply(r, query)
	query, apiError := service.ApplySortingAndPaging(
		r, query,
		&service.SortingAndPagingParameters{
			Fields: service.SortingAndPagingFields{
				"name": {ColumnName: "groups.name"},
				"id":   {ColumnName: "groups.id"},
			},
			DefaultRules: "name,id",
			TieBreakers:  service.SortingAndPagingTieBreakers{"id": service.FieldTypeInt64},
		})

	if apiError != service.NoError {
		return apiError
	}

	var result []groupManagersViewResponseRow
	service.MustNotBeError(query.Scan(&result).Error())

	for index := range result {
		result[index].CanManage = store.GroupManagers().CanManageNameByIndex(result[index].CanManageValue)
		if result[index].Type != groupTypeUser {
			result[index].GroupManagersViewResponseRowUser = nil
		}
		if !includeManagersOfAncestorGroups {
			result[index].GroupManagersViewResponseRowThroughAncestorGroups = nil
		} else {
			result[index].CanManageThroughAncestorGroups = store.
				GroupManagers().
				CanManageNameByIndex(result[index].CanManageThroughAncestorGroupsValue)
		}
	}

	render.Respond(w, r, result)
	return service.NoError
}
