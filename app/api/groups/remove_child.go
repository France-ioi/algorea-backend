package groups

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation DELETE /groups/{parent_group_id}/relations/{child_group_id} group-memberships groupRemoveChild
//
//		---
//		summary: Remove a subgroup
//		description: >
//	  Removes a child group from a parent group and optionally deletes that group and its subgroups.
//
//
//	  Deletes the relation from `groups_groups`. If `{delete_orphans}` is true and the child group becomes an orphan
//	  then the service also deletes that group, its parent-child relations, and recursively deletes each
//	  new orphaned group.
//
//
//	  If `{delete_orphans}` is false and the child removal makes it an orphan (it doesn't have other parent groups),
//	  the service deletes the relation keeping the orphaned child group.
//
//
//	  If a group gets deleted, the service also deletes `groups_groups`, `attempts`, `results`,
//	  `group_membership_changes`, `group_pending_requests`,
//	  `permissions_granted`, `permissions_generated`, and `filters` linked to the group.
//	  Access rights are updated accordingly too.
//
//
//	  Restrictions (otherwise the 'forbidden' error is returned):
//	    * the authenticated user should be a manager of both `{parent_group_id}` and `{child_group_id}`,
//	    * the authenticated user should have `can_manage` >= 'memberships' on the `{parent_group_id}`,
//	    * the parent group should not be of type "User" or "Team",
//	    * the child group should not be of types "Base" or "User"
//	      (since there are more appropriate services for removing users from groups: groupLeave and groupRemoveMembers).
//		parameters:
//			- name: parent_group_id
//				in: path
//				type: integer
//				format: int64
//				required: true
//			- name: child_group_id
//				in: path
//				type: integer
//				format: int64
//				required: true
//			- name: delete_orphans
//				in: query
//				type: integer
//				enum: [0,1]
//				default: 0
//		responses:
//			"201":
//				"$ref": "#/responses/deletedResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) removeChild(w http.ResponseWriter, r *http.Request) service.APIError {
	parentGroupID, err := service.ResolveURLQueryPathInt64Field(r, "parent_group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}
	childGroupID, err := service.ResolveURLQueryPathInt64Field(r, "child_group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	shouldDeleteOrphans := false
	if len(r.URL.Query()["delete_orphans"]) > 0 {
		shouldDeleteOrphans, err = service.ResolveURLQueryGetBoolField(r, "delete_orphans")
		if err != nil {
			return service.ErrInvalidRequest(err)
		}
	}

	user := srv.GetUser(r)
	apiErr := service.NoError

	err = srv.GetStore(r).InTransaction(func(s *database.DataStore) error {
		apiErr = checkThatUserHasRightsForDirectRelation(s, user, parentGroupID, childGroupID, deleteRelation)
		if apiErr != service.NoError {
			return apiErr.Error // rollback
		}

		// Check that the relation exists
		var found bool
		found, err = s.ActiveGroupGroups().WithWriteLock().
			Where("parent_group_id = ?", parentGroupID).
			Where("child_group_id = ?", childGroupID).
			HasRows()
		service.MustNotBeError(err)
		if !found {
			apiErr = service.InsufficientAccessRightsError
			return apiErr.Error // rollback
		}

		return s.GroupGroups().DeleteRelation(parentGroupID, childGroupID, shouldDeleteOrphans)
	})

	if apiErr != service.NoError {
		return apiErr
	}

	service.MustNotBeError(err)
	service.MustNotBeError(render.Render(w, r, service.DeletionSuccess(nil)))
	return service.NoError
}
