package answers

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/formdata"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation POST /items/{item_id}/attempts/{attempt_id}/answers answers itemAnswerSave
// ---
// summary: Save an answer
// description: Allows user to "save" a current snapshot of an answer manually.
//
//   * The authenticated user should have at least 'content' access to the `{item_id}`.
//
//   * `{as_team_id}` (if given) should be the user's team.
//
//   * There should be a row in the `results` table with `attempt_id` = `{attempt_id}`,
//     `participant_id` = the user's group (or `{as_team_id}` if given), `item_id` = `{item_id}`
// parameters:
// - name: attempt_id
//   in: path
//   type: integer
//   required: true
// - name: item_id
//   in: path
//   type: integer
//   required: true
// - name: as_team_id
//   in: query
//   type: integer
//   format: int64
// - name: answer information
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/answerData"
// responses:
//   "201":
//     description: Created. The request has successfully saved the answer.
//     schema:
//       "$ref": "#/definitions/createdResponse"
//   "400":
//     "$ref": "#/responses/badRequestResponse"
//   "401":
//     "$ref": "#/responses/unauthorizedResponse"
//   "403":
//     "$ref": "#/responses/forbiddenResponse"
//   "500":
//     "$ref": "#/responses/internalErrorResponse"
func (srv *Service) save(rw http.ResponseWriter, httpReq *http.Request) service.APIError {
	return srv.saveAnswerWithType(rw, httpReq, false)
}

func (srv *Service) saveAnswerWithType(rw http.ResponseWriter, httpReq *http.Request, isCurrent bool) service.APIError {
	attemptID, err := service.ResolveURLQueryPathInt64Field(httpReq, "attempt_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}
	itemID, err := service.ResolveURLQueryPathInt64Field(httpReq, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	var requestData answerData
	formData := formdata.NewFormData(&requestData)
	err = formData.ParseJSONRequestData(httpReq)
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(httpReq)

	groupID := user.GroupID
	var found bool
	if len(httpReq.URL.Query()["as_team_id"]) != 0 {
		groupID, err = service.ResolveURLQueryGetInt64Field(httpReq, "as_team_id")
		if err != nil {
			return service.ErrInvalidRequest(err)
		}

		found, err = srv.Store.Results().ExistsForUserTeam(user, groupID, attemptID, itemID)
	} else {
		found, err = srv.Store.Results().ByID(groupID, attemptID, itemID).HasRows()
	}
	service.MustNotBeError(err)
	if !found {
		return service.InsufficientAccessRightsError
	}

	err = srv.Store.InTransaction(func(store *database.DataStore) error {
		answersStore := store.Answers()

		answerType := "Saved"
		if isCurrent {
			answerType = "Current"

			service.MustNotBeError(answersStore.Where("answers.author_id = ?", user.GroupID).
				Where("answers.attempt_id = ?", attemptID).
				Where("answers.participant_id = ?", groupID).
				Where("answers.item_id = ?", itemID).
				Where("answers.type = 'Current'").
				Delete().Error())
		}

		return answersStore.RetryOnDuplicatePrimaryKeyError(func(store *database.DataStore) error {
			answerID := store.NewID()
			return store.Answers().InsertMap(map[string]interface{}{
				"id":             answerID,
				"author_id":      user.GroupID,
				"attempt_id":     attemptID,
				"participant_id": groupID,
				"item_id":        itemID,
				"type":           answerType,
				"state":          requestData.State,
				"answer":         requestData.Answer,
				"created_at":     database.Now(),
			})
		})
	})
	service.MustNotBeError(err)

	var result render.Renderer
	if isCurrent {
		result = service.UpdateSuccess(nil)
	} else {
		result = service.CreationSuccess(nil)
	}

	service.MustNotBeError(render.Render(rw, httpReq, result))
	return service.NoError
}
