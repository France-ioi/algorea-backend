package answers

import (
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/jinzhu/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation GET /answers/{answer_id} answers itemAnswerGet
// ---
// summary: Get an answer
// description: Display an answer
//
//   * The user should have at least partial access rights to the users_answers.idItem item
//
//   * The user should be able to see answers related to his group's attempts, so
//
//     (a) if items.bHasAttempts = 1, then the user should be a member of the groups_attempts.idGroup team<br/>
//     (b) if items.bHasAttempts = 0, then groups_attempts.idGroup should be equal to the user's self group
// parameters:
// - name: answer_id
//   in: path
//   type: integer
//   required: true
// responses:
//   "200":
//     "$ref": "#/responses/itemAnswerGetResponse"
//   "400":
//     "$ref": "#/responses/badRequestPOSTPUTPATCHResponse"
//   "401":
//     "$ref": "#/responses/unauthorizedResponse"
//   "403":
//     "$ref": "#/responses/forbiddenResponse"
//   "500":
//     "$ref": "#/responses/internalErrorResponse"
func (srv *Service) get(rw http.ResponseWriter, httpReq *http.Request) service.APIError {
	userAnswerID, err := service.ResolveURLQueryPathInt64Field(httpReq, "answer_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(httpReq)
	var result []map[string]interface{}
	err = srv.Store.UserAnswers().Visible(user).
		Where("users_answers.ID = ?", userAnswerID).
		Select(`users_answers.ID, users_answers.idUser, users_answers.idItem, users_answers.idAttempt,
			users_answers.sType, users_answers.sState, users_answers.sAnswer,
			users_answers.sSubmissionDate, users_answers.iScore, users_answers.bValidated,
			users_answers.sGradingDate, users_answers.idUserGrader`).
		ScanIntoSliceOfMaps(&result).Error()
	if gorm.IsRecordNotFoundError(err) {
		return service.InsufficientAccessRightsError
	}
	service.MustNotBeError(err)
	if len(result) == 0 {
		return service.InsufficientAccessRightsError
	}
	convertedResult := service.ConvertSliceOfMapsFromDBToJSON(result)[0]

	render.Respond(rw, httpReq, convertedResult)
	return service.NoError
}

// OK. Normal response of the itemAnswersGet service
// swagger:response itemAnswerGetResponse
type itemAnswerGetResponse struct { //nolint:unused,deadcode
	// description: The returned answer
	// in:body
	Body struct {
		// required:true
		ID int64 `json:"id,string"`
		// required:true
		UserID int64 `json:"user_id,string"`
		// required:true
		ItemID int64 `json:"item_id,string"`
		// Nullable
		// format:integer
		AttemptID *string `json:"attempt_id,string"`
		// required:true
		// enum:Submission,Saved,Current
		Type string `json:"type"`
		// Nullable
		State *string `json:"state,omitempty"`
		// Nullable
		Answer *string `json:"answer,omitempty"`
		// required:true
		SubmissionDate time.Time `json:"submission_date"`
		// Nullable
		Score *float32 `json:"score,omitempty"`
		// Nullable
		Validated *bool `json:"validated,omitempty"`
		// Nullable
		GradingDate *time.Time `json:"grading_date,omitempty"`
		// Nullable
		// format:integer
		UserGraderID *string `json:"user_grader_id,omitempty"`
	}
}
