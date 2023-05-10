package items

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/loginmodule"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation POST /items/{item_id}/attempts/{attempt_id}/publish items resultPublish
//
//		---
//		summary: Publish a result to LTI
//		description: >
//	  Publishes score (divided by 100) obtained for the item within the attempt to LTI (via the login module).
//
//
//				Restrictions:
//
//	    * if `as_team_id` is given, it should be a user's parent team group,
//	    * the current user should have at least 'content' access on each of the `{item_id}` item,
//	    * the current user should have non-empty `login_id`,
//
//	  otherwise the 'forbidden' error is returned.
//		parameters:
//			- name: item_id
//				in: path
//				type: integer
//				format: int64
//				required: true
//			- name: attempt_id
//				in: path
//				type: integer
//				required: true
//			- name: as_team_id
//				description: fails with 'bad request' error if given, this service does not currently support team work
//				in: query
//				type: integer
//		responses:
//			"200":
//				"$ref": "#/responses/publishedOrFailedResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) publishResult(w http.ResponseWriter, r *http.Request) service.APIError {
	var err error

	itemID, err := service.ResolveURLQueryPathInt64Field(r, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(r)
	if user.LoginID == nil {
		return service.InsufficientAccessRightsError
	}
	store := srv.GetStore(r)

	found, err := store.Permissions().MatchingUserAncestors(user).WherePermissionIsAtLeast("view", "content").
		Where("item_id = ?", itemID).HasRows()
	service.MustNotBeError(err)
	if !found {
		return service.InsufficientAccessRightsError
	}

	attemptID, err := service.ResolveURLQueryPathInt64Field(r, "attempt_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	if service.ParticipantIDFromContext(r.Context()) != user.GroupID {
		return service.ErrInvalidRequest(errors.New("the service doesn't support 'as_team_id'"))
	}

	var score float32
	err = store.Results().ByID(user.GroupID, attemptID, itemID).PluckFirst("score_computed", &score).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		service.MustNotBeError(err)
	}

	result, err := loginmodule.NewClient(srv.AuthConfig.GetString("loginModuleURL")).SendLTIResult(
		r.Context(),
		srv.AuthConfig.GetString("clientID"),
		srv.AuthConfig.GetString("clientSecret"),
		*user.LoginID, itemID, score/100.0,
	)
	service.MustNotBeError(err)

	message := "published"
	if !result {
		message = "failed"
	}
	render.Respond(w, r, &service.Response{Success: result, Message: message})
	return service.NoError
}
