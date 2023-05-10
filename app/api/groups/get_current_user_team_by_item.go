package groups

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation GET /current-user/teams/by-item/{item_id} groups teamGetByItemID
//
//		---
//		summary: Get current user's team for a contest
//		description: >
//	  The team identified by `{item_id}` i.e. a group which:
//
//	    * has an active and unexpired attempt with `root_item_id` = `{item_id}`,
//
//	    * is a direct parent (i.e. via `groups_groups`) of the authenticated user’s `selfGroup`,
//
//	    * is of type "Team".
//
//
//	  If there are several matching teams, returns the first one in the order of `groups.id`.
//		parameters:
//			- name: item_id
//				type: integer
//				required: true
//				in: path
//		responses:
//			"200":
//				description: OK. Success response with the team's id
//				schema:
//					type: object
//					properties:
//						group_id:
//							type: integer
//					required: [group_id]
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"404":
//				"$ref": "#/responses/notFoundResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getCurrentUserTeamByItem(w http.ResponseWriter, r *http.Request) service.APIError {
	itemID, err := service.ResolveURLQueryPathInt64Field(r, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	var teamID int64
	user := srv.GetUser(r)
	err = srv.GetStore(r).Groups().TeamGroupForTeamItemAndUser(itemID, user).PluckFirst("groups.id", &teamID).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return service.ErrNotFound(errors.New("no team for this item"))
	}
	service.MustNotBeError(err)

	render.Respond(w, r, &map[string]string{"group_id": strconv.FormatInt(teamID, 10)})
	return service.NoError
}
