package groups

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/formdata"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation PUT /groups/{group_id}/managers/{manager_id} groups groupManagerEdit
//
//		---
//		summary: Change permissions of a group manager
//		description: >
//
//	  Modifies permissions of a group manager.
//
//
//	  The authenticated user should have 'can_manage:memberships_and_group' permission on the group
//	  and the `{group_id}`-`{manager_id}` pair should exist in `group_managers,
//	  otherwise the "forbidden" error is returned.
//		parameters:
//			- name: group_id
//				in: path
//				required: true
//				type: integer
//			- name: manager_id
//				in: path
//				required: true
//				type: integer
//			- in: body
//				name: data
//				required: true
//				description: New permissions of the manager
//				schema:
//					"$ref": "#/definitions/createGroupManagerRequest"
//		responses:
//			"200":
//				"$ref": "#/responses/updatedResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) updateGroupManager(w http.ResponseWriter, r *http.Request) service.APIError {
	var err error
	user := srv.GetUser(r)

	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}
	managerID, err := service.ResolveURLQueryPathInt64Field(r, "manager_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	input := createGroupManagerRequest{}
	formData := formdata.NewFormData(&input)
	err = formData.ParseJSONRequestData(r)
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	apiError := service.NoError
	err = srv.GetStore(r).InTransaction(func(store *database.DataStore) error {
		var found bool
		// 1) the authenticated user should have can_manage:memberships_and_group permission on the groupID
		// 2) there should be a row in group_managers for the given groupID-managerID pair
		found, err = store.Groups().ManagedBy(user).WithWriteLock().
			Where("groups.id = ?", groupID).
			Joins(`
				JOIN group_managers AS this_manager
					ON this_manager.group_id = groups.id AND this_manager.manager_id = ?`, managerID).
			Where("group_managers.can_manage = 'memberships_and_group'").HasRows()
		service.MustNotBeError(err)
		if !found {
			apiError = service.InsufficientAccessRightsError
			return apiError.Error // rollback
		}

		values := formData.ConstructMapForDB()
		return store.GroupManagers().
			Where("group_id = ?", groupID).
			Where("manager_id = ?", managerID).
			UpdateColumns(values).Error()
	})

	if apiError != service.NoError {
		return apiError
	}
	service.MustNotBeError(err)

	service.MustNotBeError(render.Render(w, r, service.UpdateSuccess(nil)))
	return service.NoError
}
