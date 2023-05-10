package answers

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation GET /answers/{answer_id} answers answerGet
//
//		---
//		summary: Get an answer
//		description: >
//	  Returns the answer identified by the given `{answer_id}`.
//
//	  - If the user is a participant
//	    - (s)he should have at least 'content' access rights to the `answers.item_id` and
//	    - be a member of the `answers.participant_id` team or
//	      `answers.participant_id` should be equal to the user's self group.
//
//	  - If the user is an observer
//	    - (s)he should have `can_watch` >= 'answer' permission on the `answers.item_id` and
//	    - be a manager with `can_watch_members` of an ancestor of `answers.participant_id` group.
//
//	  If any of the preconditions fails, the 'forbidden' error is returned.
//		parameters:
//			- name: answer_id
//				in: path
//				type: integer
//				required: true
//				format: int64
//		responses:
//			"200":
//				"$ref": "#/responses/itemAnswerGetResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getAnswer(rw http.ResponseWriter, httpReq *http.Request) service.APIError {
	answerID, err := service.ResolveURLQueryPathInt64Field(httpReq, "answer_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(httpReq)
	var result []map[string]interface{}

	store := srv.GetStore(httpReq)
	usersGroupsQuery := store.ActiveGroupGroups().WhereUserIsMember(user).Select("parent_group_id")

	// a participant should have at least 'content' access to the answers.item_id
	participantItemPerms := store.Permissions().MatchingUserAncestors(user).
		WherePermissionIsAtLeast("view", "content").
		Where("permissions.item_id = answers.item_id").
		Select("1").Limit(1)
	// an observer should have 'can_watch'>='answer' permission on the answers.item_id
	observerItemPerms := store.Permissions().MatchingUserAncestors(user).
		WherePermissionIsAtLeast("watch", "answer").
		Where("permissions.item_id = answers.item_id").
		Select("1").Limit(1)
	// an observer should be able to watch the participant
	observerParticipantPerms := store.ActiveGroupAncestors().ManagedByUser(user).
		Where("groups_ancestors_active.child_group_id = answers.participant_id").
		Where("can_watch_members").
		Select("1").Limit(1)

	err = withGradings(store.Answers().ByID(answerID).
		// 1) the user is the participant or a member of the participant group able to view the item,
		// 2) or an observer with required permissions
		Where(`
			((?) AND (answers.participant_id = ? OR answers.participant_id IN (?))) OR
			((?) AND (?))`,
			participantItemPerms.SubQuery(), user.GroupID, usersGroupsQuery.SubQuery(),
			observerItemPerms.SubQuery(), observerParticipantPerms.SubQuery())).
		ScanIntoSliceOfMaps(&result).Error()
	service.MustNotBeError(err)
	if len(result) == 0 {
		return service.InsufficientAccessRightsError
	}
	convertedResult := service.ConvertSliceOfMapsFromDBToJSON(result)[0]

	render.Respond(rw, httpReq, convertedResult)
	return service.NoError
}
