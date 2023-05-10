package groups

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/logging"
	"github.com/France-ioi/AlgoreaBackend/app/loginmodule"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation DELETE /user-batches/{group_prefix}/{custom_prefix} groups userBatchRemove
//
//		---
//		summary: Remove a user batch
//		description: |
//	  Lets a group manager remove user batches and all users having "{group_prefix}_{custom_prefix}_" as login prefix.
//
//
//	  If the preconditions are satisfied, the service
//
//	    * requests the login module to delete the users with "{group_prefix}\_{custom_prefix}\_" as prefix
//	      (/platform_api/accounts_manager/delete with the `prefix` parameter);
//
//	    * deletes all users with "{group_prefix}\_{custom_prefix}\_" as prefix
//	      (ignoring the membership locks on groups that the authenticated user manages (but not others!));
//
//	    * deletes the user batch entry.
//
//	  As we do not lock the DB between the preconditions checking and the actual deletion
//	  with the call to the login module in the middle, there is possibility of deleting users
//	  that haven't been checked or haven't been removed from the login module.
//
//	  If the local user deletion fails because of DB failure, there might be inconsistency between the DB
//	  and the login module which can be fixed by retrying the request with the same parameters.
//
//				Preconditions:
//
//	  * The authenticated user should be a manager of the `group_prefix`'s group (or its ancestor)
//	    with `can_manage` >= 'memberships', otherwise the 'forbidden' error is returned.
//
//	  * If there are users with locked membership in groups the current user cannot manage,
//	    the 'unprocessable entity' error is returned.
//		parameters:
//			- name: group_prefix
//				in: path
//				type: string
//				required: true
//			- name: custom_prefix
//				in: path
//				type: string
//				required: true
//		responses:
//			"200":
//				"$ref": "#/responses/deletedResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"422":
//				"$ref": "#/responses/unprocessableEntityResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) removeUserBatch(w http.ResponseWriter, r *http.Request) service.APIError {
	groupPrefix := chi.URLParam(r, "group_prefix")
	customPrefix := chi.URLParam(r, "custom_prefix")

	user := srv.GetUser(r)
	store := srv.GetStore(r)
	managedByUser := store.ActiveGroupAncestors().ManagedByUser(user).
		Where("can_manage != 'none'").
		Select("groups_ancestors_active.child_group_id AS id")

	// The user batch should exist and the current user should be a manager of the group
	// linked to the group_prefix
	found, err := store.UserBatches().
		Joins("JOIN user_batch_prefixes USING(group_prefix)").
		Joins("JOIN (?) AS managed_groups ON managed_groups.id = user_batch_prefixes.group_id", managedByUser.SubQuery()).
		Where("group_prefix = ?", groupPrefix).
		Where("custom_prefix = ?", customPrefix).
		HasRows()
	service.MustNotBeError(err)
	if !found {
		return service.InsufficientAccessRightsError
	}

	// There should not be users with locked membership in the groups the current user cannot manage
	found, err = store.Users().
		Joins(`
			JOIN groups_groups_active
				ON groups_groups_active.child_group_id = users.group_id`).
		Joins("JOIN `groups` AS parent_group ON parent_group_id = groups_groups_active.parent_group_id").
		Where("NOW() < parent_group.require_lock_membership_approval_until AND groups_groups_active.lock_membership_approved").
		Where("login LIKE CONCAT(?, '\\_', ?, '\\_%')", groupPrefix, customPrefix).
		Where("parent_group.id NOT IN(?)", managedByUser.QueryExpr()).
		HasRows()
	service.MustNotBeError(err)
	if found {
		logging.Warnf(
			"User with group_id = %d failed to delete a user batch because of locked membership (group_prefix = '%s', custom_prefix = '%s')",
			user.GroupID, groupPrefix, customPrefix)
		return service.ErrUnprocessableEntity(errors.New("there are users with locked membership"))
	}

	result, err := loginmodule.NewClient(srv.AuthConfig.GetString("loginModuleURL")).
		DeleteUsers(
			r.Context(),
			srv.AuthConfig.GetString("clientID"),
			srv.AuthConfig.GetString("clientSecret"),
			groupPrefix+"_"+customPrefix+"_",
		)
	service.MustNotBeError(err)

	if !result {
		return service.ErrUnexpected(errors.New("login module failed"))
	}
	service.MustNotBeError(store.Users().DeleteWithTrapsByScope(func(store *database.DataStore) *database.DB {
		return store.Users().Where("login LIKE CONCAT(?, '\\_', ?, '\\_%')", groupPrefix, customPrefix)
	}))
	service.MustNotBeError(
		store.UserBatches().
			Where("group_prefix = ?", groupPrefix).
			Where("custom_prefix = ?", customPrefix).Delete().Error())

	service.MustNotBeError(render.Render(w, r, service.DeletionSuccess(nil)))
	return service.NoError
}
