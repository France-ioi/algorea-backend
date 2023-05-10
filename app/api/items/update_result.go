package items

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/formdata"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// resultUpdateRequest is the expected input for result updating
// swagger:model resultUpdateRequest
type resultUpdateRequest struct {
	// Toggle a help request on/off
	HelpRequested bool `json:"help_requested"`
}

// swagger:operation PUT /items/{item_id}/attempts/{attempt_id} items resultUpdate
//
//		---
//		summary: Update attempt result properties
//		description: >
//	  Modifies values of an attempt result's properties a participant is able to modify.
//
//
//				Restrictions:
//
//	    * `{as_team_id}` (if given) should be the current user's team,
//	    * the participant should have a `results` row for the `{item_id}`-`{attempt_id}` pair,
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
//				format: int64
//				required: true
//			- name: as_team_id
//				in: query
//				type: integer
//				format: int64
//			- in: body
//				name: data
//				required: true
//				description: Result properties to modify
//				schema:
//					"$ref": "#/definitions/resultUpdateRequest"
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
func (srv *Service) updateResult(w http.ResponseWriter, r *http.Request) service.APIError {
	var err error

	itemID, err := service.ResolveURLQueryPathInt64Field(r, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	attemptID, err := service.ResolveURLQueryPathInt64Field(r, "attempt_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	participantID := service.ParticipantIDFromContext(r.Context())

	input := resultUpdateRequest{}
	formData := formdata.NewFormData(&input)

	apiError := service.NoError
	err = srv.GetStore(r).InTransaction(func(store *database.DataStore) error {
		resultScope := store.Results().
			Where("participant_id = ?", participantID).
			Where("attempt_id = ?", attemptID).
			Where("item_id = ?", itemID)
		var found bool
		found, err = resultScope.WithWriteLock().HasRows()
		service.MustNotBeError(err)
		if !found {
			apiError = service.InsufficientAccessRightsError
			return apiError.Error // rollback
		}

		err = formData.ParseJSONRequestData(r)
		if err != nil {
			apiError = service.ErrInvalidRequest(err)
			return err // rollback
		}

		data := formData.ConstructMapForDB()
		if len(data) > 0 {
			service.MustNotBeError(resultScope.UpdateColumns(data).Error())
		}
		return nil
	})
	if apiError != service.NoError {
		return apiError
	}
	service.MustNotBeError(err)

	service.MustNotBeError(render.Render(w, r, service.UpdateSuccess(nil)))
	return service.NoError
}
