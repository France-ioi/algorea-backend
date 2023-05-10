package answers

import (
	"errors"
	"fmt"
	"hash/crc64"
	"net/http"
	"strconv"

	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/service"
	"github.com/France-ioi/AlgoreaBackend/app/token"
)

// swagger:operation POST /answers/{answer_id}/generate-task-token answers answerTaskTokenGenerate
//
//	---
//	summary: Generate a task token
//	description: >
//
//		Generate a read-only task token from an answer
//
//
//		* Then the service returns a task token for the attempt for the given item.
//
//		* `bAccessSolutions` of the token is true if either the participant has `can_view` >= 'solution' on the item or
//		the item has been validated by the participant.
//
//		Restrictions:
//
//			* the answer should exist
//			* the item of the answer should be a "Task"
//			* the current user must have a started result on the item (whatever the attempt)
//			* if the participant of the answer is either the current-user or a team which the current-user is member of,
//				the current user must be allowed to "view >= 'content'" the item
//			* otherwise:
//				the current user must be allowed to "watch" the participant of the answer
//				the current user must be allowed to "watch answer" for the item
//
//
//			otherwise the 'forbidden' error is returned.
//		parameters:
//			- name: answer_id
//				in: path
//				type: integer
//				required: true
//		responses:
//			"200":
//				description: "OK. Success response with the task token"
//				schema:
//						type: object
//						required: [success, message, data]
//						properties:
//							success:
//								description: "true"
//								type: boolean
//								enum: [true]
//							message:
//								description: updated
//								type: string
//								enum: [updated]
//							data:
//								type: object
//								required: [task_token]
//								properties:
//									task_token:
//										type: string
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) generateTaskToken(w http.ResponseWriter, r *http.Request) service.APIError {
	answerID, err := service.ResolveURLQueryPathInt64Field(r, "answer_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(r)

	var answerInfos struct {
		AccessSolutions   bool
		TextID            *string
		URL               string
		SupportedLangProg *string
		AuthorID          int64
		AuthorLogin       *string
		ParticipantID     int64
		AttemptID         int64
		ItemID            int64
		HintsRequested    *string
		HintsCachedCount  int32 `gorm:"column:hints_cached"`
		Validated         bool
	}

	store := srv.GetStore(r)
	usersGroupsQuery := store.ActiveGroupGroups().WhereUserIsMember(user).Select("parent_group_id")

	// a participant should have at least 'content' access to the answers.item_id
	participantItemPerms := store.Permissions().MatchingUserAncestors(user).
		WherePermissionIsAtLeast("view", "content").
		Where("permissions.item_id = answers.item_id").
		Select("1").
		Limit(1)

	// an observer should have 'can_watch'>='answer' permission on the answers.item_id
	observerItemPerms := store.Permissions().MatchingUserAncestors(user).
		WherePermissionIsAtLeast("watch", "answer").
		Where("permissions.item_id = answers.item_id").
		Select("1").
		Limit(1)

	// an observer should be able to watch the participant
	observerParticipantPerms := store.ActiveGroupAncestors().ManagedByUser(user).
		Joins("JOIN `groups` ON groups.id = groups_ancestors_active.child_group_id").
		Where("groups_ancestors_active.child_group_id = answers.participant_id").
		Where("can_watch_members").
		Select("1").
		Limit(1)

	err = store.Answers().WithItems().WithResults().
		Joins("JOIN groups_ancestors_active ON groups_ancestors_active.child_group_id = ?", user.GroupID).
		Joins(`
				JOIN permissions_generated
					ON permissions_generated.item_id = items.id AND
						 permissions_generated.group_id = groups_ancestors_active.ancestor_group_id`).
		Joins("JOIN users AS author_users ON author_users.group_id = answers.author_id").
		Joins(`
			JOIN results AS requester_results ON requester_results.participant_id = ? AND
				requester_results.item_id = answers.item_id`, user.GroupID).
		Select(`
					permissions_generated.can_view_generated_value = ? AS access_solutions,
					items.text_id AS text_id,
					items.url AS url,
					items.supported_lang_prog AS supported_lang_prog,
					answers.author_id AS author_id,
					answers.participant_id AS participant_id,
					answers.attempt_id AS attempt_id,
					answers.item_id AS item_id,
					results.hints_requested AS hints_requested,
					results.hints_cached AS hints_cached,
					requester_results.validated AS validated,
					author_users.login AS author_login`,
			store.PermissionsGranted().ViewIndexByName("solution")).
		// 1) if the participant of the answer is either the current-user or a team which the current-user is member of,
		//    the current user must be allowed to "view >= 'content'" the item
		// 2) or an observer who can "watch" the participant and "watch answer" the item
		Where(`
				((?) AND (answers.participant_id = ? OR answers.participant_id IN (?))) OR
				((?) AND (?))`,
			participantItemPerms.SubQuery(), user.GroupID, usersGroupsQuery.SubQuery(),
			observerItemPerms.SubQuery(), observerParticipantPerms.SubQuery()).
		Where("answers.id = ?", answerID).
		Where("items.type = 'Task'").
		WhereUserHaveStartedResultOnItem(user).
		Limit(1).
		Take(&answerInfos).Error()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return service.InsufficientAccessRightsError
	}
	service.MustNotBeError(err)

	fullAttemptID := fmt.Sprintf("%d/%d", answerInfos.ParticipantID, answerInfos.AttemptID)
	randomSeed := crc64.Checksum([]byte(fullAttemptID), crc64.MakeTable(crc64.ECMA))

	accessSolutions := answerInfos.AccessSolutions || answerInfos.Validated

	taskToken := token.Task{
		AccessSolutions:    &accessSolutions,
		SubmissionPossible: ptrBool(false),
		HintsAllowed:       ptrBool(false),
		HintsRequested:     answerInfos.HintsRequested,
		HintsGivenCount:    ptrString(strconv.Itoa(int(answerInfos.HintsCachedCount))),
		IsAdmin:            ptrBool(false),
		ReadAnswers:        ptrBool(true),
		UserID:             strconv.FormatInt(answerInfos.AuthorID, 10),
		LocalItemID:        strconv.FormatInt(answerInfos.ItemID, 10),
		ItemID:             answerInfos.TextID,
		AttemptID:          fullAttemptID,
		ItemURL:            answerInfos.URL,
		SupportedLangProg:  answerInfos.SupportedLangProg,
		RandomSeed:         strconv.FormatUint(randomSeed, 10),
		PlatformName:       srv.TokenConfig.PlatformName,
		Login:              answerInfos.AuthorLogin,
	}
	signedTaskToken, err := taskToken.Sign(srv.TokenConfig.PrivateKey)
	service.MustNotBeError(err)

	render.Respond(w, r, service.CreationSuccess(map[string]interface{}{
		"task_token": signedTaskToken,
	}))
	return service.NoError
}

func ptrString(s string) *string { return &s }
func ptrBool(b bool) *bool       { return &b }
