package items

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/render"
	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/formdata"
	"github.com/France-ioi/AlgoreaBackend/app/logging"
	"github.com/France-ioi/AlgoreaBackend/app/payloads"
	"github.com/France-ioi/AlgoreaBackend/app/service"
	"github.com/France-ioi/AlgoreaBackend/app/token"
)

// swagger:operation POST /items/save-grade items saveGrade
// ---
// summary: Save the grade
// description: >
//
//   Saves the grade returned by a grading app into the `gradings` table and updates the attempt in the DB.
//   When the `score` = 100 (full score), the service unlocks dependent locked items (if any) and updates `bAccessSolutions`
//   of the task token.
//
//
//   Restrictions:
//
//     * `task_token`, `score_token`/`answer_token` should belong to the current user, otherwise the "bad request"
//        response is returned;
//     * `idItemLocal`, `itemUrl`, `idAttempt` of the `task_token` should match ones in the `score_token`/`answer_token`,
//       otherwise the "bad request" response is returned;
//     * the answer should exist and should have not been graded, otherwise the "forbidden" response is returned.
// parameters:
// - in: body
//   name: data
//   required: true
//   schema:
//     type: object
//     required: [task_token]
//     properties:
//       task_token:
//         description: A task token previously generated by AlgoreaBackend
//         type: string
//       score_token:
//         description: A score token generated by the grader (required for platforms supporting tokens)
//         type: string
//       answer_token:
//         description: An answer token generated by AlgoreaBackend (required for platforms not supporting tokens)
//         type: string
//       score:
//         description: A score returned by the grader (required for platforms not supporting tokens)
//         type: number
// responses:
//   "201":
//     description: "Created. Success response with the newly created task token"
//     schema:
//       type: object
//       required: [success, message, data]
//       properties:
//         success:
//           description: "true"
//           type: boolean
//           enum: [true]
//         message:
//           description: created
//           type: string
//           enum: [created]
//         data:
//           type: object
//           required: [task_token, validated]
//           properties:
//             task_token:
//               description: The updated task token
//               type: string
//             validated:
//               description: Whether the full score was obtained on this grading
//               type: boolean
//   "400":
//     "$ref": "#/responses/badRequestResponse"
//   "401":
//     "$ref": "#/responses/unauthorizedResponse"
//   "403":
//     "$ref": "#/responses/forbiddenResponse"
//   "500":
//     "$ref": "#/responses/internalErrorResponse"
func (srv *Service) saveGrade(w http.ResponseWriter, r *http.Request) service.APIError {
	requestData := saveGradeRequestParsed{store: srv.Store, publicKey: srv.TokenConfig.PublicKey}

	var err error
	if err = render.Bind(r, &requestData); err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(r)

	if apiError := checkHintOrScoreTokenRequiredFields(user, requestData.TaskToken, "score_token",
		requestData.ScoreToken.Converted.UserID, requestData.ScoreToken.LocalItemID,
		requestData.ScoreToken.ItemURL, requestData.ScoreToken.AttemptID); apiError != service.NoError {
		return apiError
	}

	var validated, ok bool
	err = srv.Store.InTransaction(func(store *database.DataStore) error {
		validated, ok = saveGradingResultsIntoDB(store, user, &requestData)
		return nil
	})
	service.MustNotBeError(err)

	if !ok {
		return service.ErrForbidden(errors.New("the answer has been already graded or is not found"))
	}

	if validated && requestData.TaskToken.AccessSolutions != nil && !(*requestData.TaskToken.AccessSolutions) {
		requestData.TaskToken.AccessSolutions = ptrBool(true)
	}
	requestData.TaskToken.PlatformName = srv.TokenConfig.PlatformName
	newTaskToken, err := requestData.TaskToken.Sign(srv.TokenConfig.PrivateKey)
	service.MustNotBeError(err)

	service.MustNotBeError(render.Render(w, r, service.CreationSuccess(map[string]interface{}{
		"task_token": newTaskToken,
		"validated":  validated,
	})))
	return service.NoError
}

func saveGradingResultsIntoDB(store *database.DataStore, user *database.User,
	requestData *saveGradeRequestParsed) (validated, ok bool) {
	score := requestData.ScoreToken.Converted.Score

	gotFullScore := score == 100
	validated = gotFullScore // currently a validated task is only a task with a full score (score == 100)
	if !saveNewScoreIntoGradings(store, user, requestData, score) {
		return validated, false
	}

	// Build query to update attempts
	columnsToUpdate := []string{
		"tasks_tried",
		"score_obtained_at",
		"score_computed",
		"result_propagation_state",
	}
	newScoreExpression := gorm.Expr(`
			LEAST(GREATEST(
				CASE score_edit_rule
					WHEN 'set' THEN score_edit_value
					WHEN 'diff' THEN ? + score_edit_value
					ELSE ?
				END, score_computed, 0), 100)`, score, score)
	values := []interface{}{
		requestData.ScoreToken.Converted.UserAnswerID, // for join
		1, // tasks_tried
		// for score_computed we compare patched scores
		gorm.Expr(`
			CASE
			  -- New best score or no time saved yet
				-- Note that when the score = 0, score_obtained_at is the time of the first submission
				WHEN score_obtained_at IS NULL OR score_computed < ? THEN answers.created_at
				-- We may get the result of an earlier submission after one with the same score
				WHEN score_computed = ? THEN LEAST(score_obtained_at, answers.created_at)
				-- New score if lower than the best score
				ELSE score_obtained_at
			END`, newScoreExpression, newScoreExpression), // score_obtained_at
		newScoreExpression, // score_computed
		"to_be_propagated", // result_propagation_state
	}
	if validated {
		// Item was validated
		columnsToUpdate = append(columnsToUpdate, "validated_at")
		values = append(values, gorm.Expr("LEAST(IFNULL(validated_at, answers.created_at), answers.created_at)"))
	}

	updateExpr := "SET " + strings.Join(columnsToUpdate, " = ?, ") + " = ?"
	values = append(values, requestData.TaskToken.Converted.AttemptID)
	service.MustNotBeError(
		store.DB.Exec("UPDATE attempts JOIN answers ON answers.id = ? "+ // nolint:gosec
			updateExpr+" WHERE attempts.id = ?", values...).Error()) // nolint:gosec
	service.MustNotBeError(store.Attempts().ComputeAllAttempts())
	return validated, true
}

func saveNewScoreIntoGradings(store *database.DataStore, user *database.User,
	requestData *saveGradeRequestParsed, score float64) bool {
	answerID := requestData.ScoreToken.Converted.UserAnswerID
	gradingStore := store.Gradings()

	insertError := gradingStore.InsertMap(map[string]interface{}{
		"answer_id": answerID, "score": score, "graded_at": database.Now(),
	})

	// ERROR 1452 (23000): Cannot add or update a child row: a foreign key constraint fails (the answer has been removed)
	if e, ok := insertError.(*mysql.MySQLError); ok && e.Number == 1452 {
		return false
	}

	// ERROR 1062 (23000): Duplicate entry (already graded)
	if e, ok := insertError.(*mysql.MySQLError); ok && e.Number == 1062 {
		var oldScore *float64
		service.MustNotBeError(gradingStore.
			Where("answer_id = ?", answerID).PluckFirst("score", &oldScore).Error())
		if oldScore != nil {
			if *oldScore != score {
				fieldsForLoggingMarshaled, _ := json.Marshal(map[string]interface{}{
					"idAttempt":    requestData.TaskToken.Converted.AttemptID,
					"idItem":       requestData.TaskToken.Converted.LocalItemID,
					"idUser":       user.GroupID,
					"idUserAnswer": requestData.ScoreToken.Converted.UserAnswerID,
					"newScore":     score,
					"oldScore":     *oldScore,
				})
				logging.Warnf("A user tries to replay a score token with a different score value (%s)", fieldsForLoggingMarshaled)
			}
			return false
		}
	}
	service.MustNotBeError(insertError)

	return true
}

type saveGradeRequestParsed struct {
	TaskToken   *token.Task
	ScoreToken  *token.Score
	AnswerToken *token.Answer

	store     *database.DataStore
	publicKey *rsa.PublicKey
}

type saveGradeRequest struct {
	TaskToken   *string            `json:"task_token"`
	ScoreToken  formdata.Anything  `json:"score_token"`
	Score       *float64           `json:"score"`
	AnswerToken *formdata.Anything `json:"answer_token"`
}

// UnmarshalJSON unmarshals the items/saveGrade request data from JSON
func (requestData *saveGradeRequestParsed) UnmarshalJSON(raw []byte) error {
	var wrapper saveGradeRequest
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return err
	}
	if wrapper.TaskToken == nil {
		return errors.New("missing task_token")
	}
	requestData.TaskToken = &token.Task{PublicKey: requestData.publicKey}
	if err := requestData.TaskToken.UnmarshalString(*wrapper.TaskToken); err != nil {
		return fmt.Errorf("invalid task_token: %s", err.Error())
	}
	return requestData.unmarshalScoreToken(&wrapper)
}

func (requestData *saveGradeRequestParsed) unmarshalScoreToken(wrapper *saveGradeRequest) error {
	err := token.UnmarshalDependingOnItemPlatform(requestData.store, requestData.TaskToken.Converted.LocalItemID,
		&requestData.ScoreToken, wrapper.ScoreToken.Bytes(), "score_token")
	if err != nil && !token.IsUnexpectedError(err) {
		return err
	}
	service.MustNotBeError(err)
	if requestData.ScoreToken == nil {
		err = requestData.reconstructScoreTokenData(wrapper)
		if err != nil {
			return err
		}
	}
	return nil
}

func (requestData *saveGradeRequestParsed) unmarshalAnswerToken(wrapper *saveGradeRequest) error {
	if wrapper.AnswerToken == nil {
		return errors.New("missing answer_token")
	}
	requestData.AnswerToken = &token.Answer{PublicKey: requestData.publicKey}
	if err := requestData.AnswerToken.UnmarshalJSON(wrapper.AnswerToken.Bytes()); err != nil {
		return fmt.Errorf("invalid answer_token: %s", err.Error())
	}
	if requestData.AnswerToken.UserID != requestData.TaskToken.UserID {
		return errors.New("wrong idUser in answer_token")
	}
	if requestData.AnswerToken.LocalItemID != requestData.TaskToken.LocalItemID {
		return errors.New("wrong idItemLocal in answer_token")
	}
	if requestData.AnswerToken.ItemURL != requestData.TaskToken.ItemURL {
		return errors.New("wrong itemUrl in answer_token")
	}
	if requestData.AnswerToken.AttemptID != requestData.TaskToken.AttemptID {
		return errors.New("wrong idAttempt in answer_token")
	}
	return nil
}

func (requestData *saveGradeRequestParsed) reconstructScoreTokenData(wrapper *saveGradeRequest) error {
	if err := requestData.unmarshalAnswerToken(wrapper); err != nil {
		return err
	}
	if wrapper.Score == nil {
		return errors.New("missing score")
	}
	userAnswerID, err := strconv.ParseInt(requestData.AnswerToken.UserAnswerID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid idUserAnswer in answer_token: %s", err.Error())
	}
	requestData.ScoreToken = &token.Score{
		Converted: payloads.ScoreTokenConverted{
			Score:        *wrapper.Score,
			UserID:       requestData.TaskToken.Converted.UserID,
			UserAnswerID: userAnswerID,
		},
		ItemURL:     requestData.TaskToken.ItemURL,
		AttemptID:   requestData.AnswerToken.AttemptID,
		LocalItemID: requestData.AnswerToken.LocalItemID,
	}
	return nil
}

// Bind of saveGradeRequestParsed does nothing.
func (requestData *saveGradeRequestParsed) Bind(*http.Request) error {
	return nil
}
