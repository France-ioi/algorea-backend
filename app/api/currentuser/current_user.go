package currentuser

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/jinzhu/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/auth"
	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// Service is the mount point for services related to `currentuser`
type Service struct {
	service.Base
}

// SetRoutes defines the routes for this package in a route group
func (srv *Service) SetRoutes(router chi.Router) {
	router.Use(render.SetContentType(render.ContentTypeJSON))
	router.Use(auth.UserMiddleware(srv.Store.Sessions()))

	router.Get("/current-user", service.AppHandler(srv.getInfo).ServeHTTP)
	router.Delete("/current-user", service.AppHandler(srv.delete).ServeHTTP)

	router.Get("/current-user/available-groups", service.AppHandler(srv.searchForAvailableGroups).ServeHTTP)

	router.Get("/current-user/group-invitations", service.AppHandler(srv.getGroupInvitations).ServeHTTP)
	router.Post("/current-user/group-invitations/{group_id}/accept", service.AppHandler(srv.acceptGroupInvitation).ServeHTTP)
	router.Post("/current-user/group-invitations/{group_id}/reject", service.AppHandler(srv.rejectGroupInvitation).ServeHTTP)

	router.Post("/current-user/group-requests/{group_id}", service.AppHandler(srv.sendGroupRequest).ServeHTTP)

	router.Get("/current-user/group-memberships", service.AppHandler(srv.getGroupMemberships).ServeHTTP)
	router.Post("/current-user/group-memberships/by-code", service.AppHandler(srv.joinGroupByCode).ServeHTTP)
	router.Delete("/current-user/group-memberships/{group_id}", service.AppHandler(srv.leaveGroup).ServeHTTP)
	router.Get("/current-user/group-memberships-history", service.AppHandler(srv.getGroupMembershipsHistory).ServeHTTP)

	router.Put("/current-user/notifications-read-at", service.AppHandler(srv.updateNotificationsReadAt).ServeHTTP)
	router.Put("/current-user/refresh", service.AppHandler(srv.refresh).ServeHTTP)

	router.Get("/current-user/full-dump", service.AppHandler(srv.getFullDump).ServeHTTP)
	router.Get("/current-user/dump", service.AppHandler(srv.getDump).ServeHTTP)
}

type userGroupRelationAction string

const (
	acceptInvitationAction           userGroupRelationAction = "acceptInvitation"
	rejectInvitationAction           userGroupRelationAction = "rejectInvitation"
	createGroupRequestAction         userGroupRelationAction = "createRequest"
	createAcceptedGroupRequestAction userGroupRelationAction = "createAcceptedRequest"
	leaveGroupAction                 userGroupRelationAction = "leaveGroup"
	joinGroupByCodeAction            userGroupRelationAction = "joinGroupByCode"
)

func (srv *Service) performGroupRelationAction(w http.ResponseWriter, r *http.Request, action userGroupRelationAction) service.APIError {
	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(r)

	if action == leaveGroupAction {
		var found bool
		found, err = srv.Store.Groups().ByID(groupID).
			Where("lock_user_deletion_until IS NULL OR lock_user_deletion_until <= NOW()").HasRows()
		service.MustNotBeError(err)
		if !found {
			return service.ErrForbidden(errors.New("user deletion is locked for this group"))
		}
	}

	apiError := service.NoError
	var results database.GroupGroupTransitionResults
	err = srv.Store.InTransaction(func(store *database.DataStore) error {
		apiError, results = performUserGroupRelationAction(action, store, user, groupID)
		if apiError != service.NoError {
			return apiError.Error // rollback
		}
		return nil
	})

	if apiError != service.NoError {
		return apiError
	}
	service.MustNotBeError(err)

	return RenderGroupGroupTransitionResult(w, r, results[user.GroupID], action)
}

func performUserGroupRelationAction(action userGroupRelationAction, store *database.DataStore, user *database.User,
	groupID int64) (service.APIError, database.GroupGroupTransitionResults) {
	var err error
	apiError := service.NoError

	if action == createGroupRequestAction {
		var found bool
		found, err = store.Groups().ManagedBy(user).Where("groups.id = ?", groupID).HasRows()
		service.MustNotBeError(err)
		if found {
			action = createAcceptedGroupRequestAction
		}
	}
	if map[userGroupRelationAction]bool{
		createGroupRequestAction: true, acceptInvitationAction: true, createAcceptedGroupRequestAction: true,
	}[action] {
		apiError = checkPreconditionsForGroupRequests(store, user, groupID, action == createGroupRequestAction)
		if apiError != service.NoError {
			return apiError, nil
		}
	}
	var results database.GroupGroupTransitionResults
	results, err = store.GroupGroups().Transition(
		map[userGroupRelationAction]database.GroupGroupTransitionAction{
			acceptInvitationAction:           database.UserAcceptsInvitation,
			rejectInvitationAction:           database.UserRefusesInvitation,
			createGroupRequestAction:         database.UserCreatesRequest,
			createAcceptedGroupRequestAction: database.UserCreatesAcceptedRequest,
			leaveGroupAction:                 database.UserLeavesGroup,
		}[action], groupID, []int64{user.GroupID}, user.GroupID)
	service.MustNotBeError(err)
	return apiError, results
}

func checkPreconditionsForGroupRequests(store *database.DataStore, user *database.User,
	groupID int64, requireFreeAccess bool) service.APIError {
	var parentGroupInfo struct {
		Type       string
		TeamItemID *int64
	}

	// The group should exist (and optionally should have `free_access` = 1)
	query := store.Groups().ByID(groupID).WithWriteLock().Select("type, team_item_id")
	if requireFreeAccess {
		query = query.Where("free_access")
	}
	err := query.Take(&parentGroupInfo).Error()
	if gorm.IsRecordNotFoundError(err) {
		return service.InsufficientAccessRightsError
	}
	service.MustNotBeError(err)

	// If the group is a team and its `team_item_id` is set, ensure that the current user is not a member of
	// another team with the same `team_item_id'.
	if parentGroupInfo.Type == "Team" && parentGroupInfo.TeamItemID != nil {
		var found bool
		found, err = store.Groups().TeamsMembersForItem([]int64{user.GroupID}, *parentGroupInfo.TeamItemID).
			WithWriteLock().
			Where("groups.id != ?", groupID).HasRows()
		service.MustNotBeError(err)
		if found {
			return service.ErrUnprocessableEntity(errors.New("you are already on a team for this item"))
		}
	}
	return service.NoError
}
