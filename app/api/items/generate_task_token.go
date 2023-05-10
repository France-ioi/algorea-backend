package items

import (
	"errors"
	"fmt"
	"hash/crc64"
	"net/http"
	"strconv"

	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
	"github.com/France-ioi/AlgoreaBackend/app/token"
)

// swagger:operation POST /items/{item_id}/attempts/{attempt_id}/generate-task-token items itemTaskTokenGenerate
//
//		---
//		summary: Generate a task token
//		description: >
//	  Generate a task token with the refreshed attempt.
//
//
//	  * `latest_activity_at` of `results` is set to the current time.
//
//	  * Then the service returns a task token with fresh data for the attempt for the given item.
//
//	  * `bAccessSolutions` of the token is true if ether the participant has `can_view` >= 'solution' on the item or
//	    the item has been validated for the participant in the given attempt.
//
//
//				Restrictions:
//
//	    * if `{as_team_id}` is given, it should be a team and the current user should be a member of this team,
//	    * the user (or `{as_team_id}`) should have at least 'content' access to the item,
//	    * the item should be a 'Task',
//	    * there should be a row in the `results` table with `participant_id` equal to the user's group (or `{as_team_id}`),
//	      `attempt_id` = `{attempt_id}`, `item_id` = `{item_id}`, `started_at` set,
//	    * the attempt with (`participant_id`, `{attempt_id}`) should have allows_submissions_until in the future,
//
//	  otherwise the 'forbidden' error is returned.
//		parameters:
//			- name: attempt_id
//				in: path
//				type: integer
//				required: true
//			- name: item_id
//				in: path
//				type: integer
//				required: true
//			- name: as_team_id
//				in: query
//				type: integer
//				format: int64
//		responses:
//			"200":
//				description: "OK. Success response with the fresh task token"
//				schema:
//					type: object
//					required: [success, message, data]
//					properties:
//						success:
//							description: "true"
//							type: boolean
//							enum: [true]
//						message:
//							description: updated
//							type: string
//							enum: [updated]
//						data:
//							type: object
//							required: [task_token]
//							properties:
//								task_token:
//									type: string
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) generateTaskToken(w http.ResponseWriter, r *http.Request) service.APIError {
	var err error

	attemptID, err := service.ResolveURLQueryPathInt64Field(r, "attempt_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}
	itemID, err := service.ResolveURLQueryPathInt64Field(r, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(r)
	participantID := service.ParticipantIDFromContext(r.Context())

	var itemInfo struct {
		AccessSolutions   bool
		HintsAllowed      bool
		TextID            *string
		URL               string
		SupportedLangProg *string
	}

	var resultInfo struct {
		HintsRequested   *string
		HintsCachedCount int32 `gorm:"column:hints_cached"`
		Validated        bool
	}
	apiError := service.NoError
	err = srv.GetStore(r).InTransaction(func(store *database.DataStore) error {
		// the group should have can_view >= 'content' permission on the item
		err = store.Items().ByID(itemID).
			Joins("JOIN groups_ancestors_active ON groups_ancestors_active.child_group_id = ?", participantID).
			Joins(`
				JOIN permissions_generated
					ON permissions_generated.item_id = items.id AND
						 permissions_generated.group_id = groups_ancestors_active.ancestor_group_id`).
			WherePermissionIsAtLeast("view", "content").
			Where("items.type = 'Task'").
			Select(`
					can_view_generated_value = ? AS access_solutions,
					hints_allowed, text_id, url, supported_lang_prog`,
				store.PermissionsGranted().ViewIndexByName("solution")).
			Take(&itemInfo).Error()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apiError = service.InsufficientAccessRightsError
			return apiError.Error // rollback
		}
		service.MustNotBeError(err)

		resultScope := store.Results().
			Where("results.participant_id = ?", participantID).
			Where("results.attempt_id = ?", attemptID).
			Where("results.item_id = ?", itemID)

		// load the result data
		err = resultScope.WithWriteLock().
			Select("hints_requested, hints_cached, validated").
			Joins("JOIN attempts ON attempts.participant_id = results.participant_id AND attempts.id = results.attempt_id").
			Where("NOW() < attempts.allows_submissions_until").
			Where("results.started").
			Take(&resultInfo).Error()

		if errors.Is(err, gorm.ErrRecordNotFound) {
			apiError = service.InsufficientAccessRightsError
			return err // rollback
		}
		service.MustNotBeError(err)

		// update results
		service.MustNotBeError(resultScope.UpdateColumn("latest_activity_at", database.Now()).Error())
		service.MustNotBeError(store.Results().MarkAsToBePropagated(participantID, attemptID, itemID))

		return nil
	})
	if apiError != service.NoError {
		return apiError
	}
	service.MustNotBeError(err)

	fullAttemptID := fmt.Sprintf("%d/%d", participantID, attemptID)
	randomSeed := crc64.Checksum([]byte(fullAttemptID), crc64.MakeTable(crc64.ECMA))

	accessSolutions := itemInfo.AccessSolutions || resultInfo.Validated

	taskToken := token.Task{
		AccessSolutions:    &accessSolutions,
		SubmissionPossible: ptrBool(true),
		HintsAllowed:       &itemInfo.HintsAllowed,
		HintsRequested:     resultInfo.HintsRequested,
		HintsGivenCount:    ptrString(strconv.Itoa(int(resultInfo.HintsCachedCount))),
		IsAdmin:            ptrBool(false),
		ReadAnswers:        ptrBool(true),
		UserID:             strconv.FormatInt(user.GroupID, 10),
		LocalItemID:        strconv.FormatInt(itemID, 10),
		ItemID:             itemInfo.TextID,
		AttemptID:          fullAttemptID,
		ItemURL:            itemInfo.URL,
		SupportedLangProg:  itemInfo.SupportedLangProg,
		RandomSeed:         strconv.FormatUint(randomSeed, 10),
		PlatformName:       srv.TokenConfig.PlatformName,
		Login:              &user.Login,
	}
	signedTaskToken, err := taskToken.Sign(srv.TokenConfig.PrivateKey)
	service.MustNotBeError(err)

	render.Respond(w, r, service.UpdateSuccess(map[string]interface{}{
		"task_token": signedTaskToken,
	}))
	return service.NoError
}

func ptrString(s string) *string { return &s }
func ptrBool(b bool) *bool       { return &b }
