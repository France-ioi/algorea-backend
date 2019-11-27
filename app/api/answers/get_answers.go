package answers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation GET /answers answers users attempts items itemAnswersView
// ---
// summary: List answers
// description: Return answers (i.e., history of submissions and current answer)
//   for a given item and user, or from a given attempt.
//
//   * One of (`user_id`, `item_id`) pair or `attempt_id` is required.
//
//   * The user should have at least 'content' access to the item.
//
//   * If `item_id` and `user_id` are given, the authenticated user should have `group_id` equal to the input `user_id`
//   or be an owner of a group containing the input `user_id`.
//
//   * If `attempt_id` is given, the authenticated user should be a member of the group
//   or an owner of the group attached to the attempt.
// parameters:
// - name: user_id
//   in: query
//   type: integer
// - name: item_id
//   in: query
//   type: integer
// - name: attempt_id
//   in: query
//   type: integer
// - name: sort
//   in: query
//   default: [-submitted_at,id]
//   type: array
//   items:
//     type: string
//     enum: [submitted_at,-submitted_at,id,-id]
// - name: from.submitted_at
//   description: Start the page from the answer next to the answer with `submitted_at` = `from.submitted_at`
//                and `users_answers.id` = `from.id`
//                (`from.id` is required when `from.submitted_at` is present)
//   in: query
//   type: string
// - name: from.id
//   description: Start the page from the answer next to the answer with `submitted_at`=`from.submitted_at`
//                and `users_answers.id`=`from.id`
//                (`from.submitted_at` is required when from.id is present)
//   in: query
//   type: integer
// - name: limit
//   description: Display the first N answers
//   in: query
//   type: integer
//   maximum: 1000
//   default: 500
// responses:
//   "200":
//     description: OK. Success response with an array of answers
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/answersResponseAnswer"
//   "400":
//     "$ref": "#/responses/badRequestResponse"
//   "401":
//     "$ref": "#/responses/unauthorizedResponse"
//   "403":
//     "$ref": "#/responses/forbiddenResponse"
//   "404":
//     "$ref": "#/responses/notFoundResponse"
//   "500":
//     "$ref": "#/responses/internalErrorResponse"
func (srv *Service) getAnswers(rw http.ResponseWriter, httpReq *http.Request) service.APIError {
	user := srv.GetUser(httpReq)

	dataQuery := srv.Store.UserAnswers().WithUsers().
		Select(`users_answers.id, users_answers.name, users_answers.type, users_answers.lang_prog,
		        users_answers.submitted_at, users_answers.score, users_answers.validated,
		        users.login, users.first_name, users.last_name`)

	userID, userIDError := service.ResolveURLQueryGetInt64Field(httpReq, "user_id")
	itemID, itemIDError := service.ResolveURLQueryGetInt64Field(httpReq, "item_id")

	if userIDError != nil || itemIDError != nil { // attempt_id
		attemptID, attemptIDError := service.ResolveURLQueryGetInt64Field(httpReq, "attempt_id")
		if attemptIDError != nil {
			return service.ErrInvalidRequest(fmt.Errorf("either user_id & item_id or attempt_id must be present"))
		}

		if result := srv.checkAccessRightsForGetAnswersByAttemptID(attemptID, user); result != service.NoError {
			return result
		}

		// we should create an index on `users_answers`.`attempt_id` for this query
		dataQuery = dataQuery.Where("attempt_id = ?", attemptID)
	} else { // user_id + item_id
		if result := srv.checkAccessRightsForGetAnswersByUserIDAndItemID(userID, itemID, user); result != service.NoError {
			return result
		}

		dataQuery = dataQuery.Where("item_id = ? AND user_id = ?", itemID, userID)
	}

	dataQuery, apiError := service.ApplySortingAndPaging(httpReq, dataQuery, map[string]*service.FieldSortingParams{
		"submitted_at": {ColumnName: "users_answers.submitted_at", FieldType: "time"},
		"id":           {ColumnName: "users_answers.id", FieldType: "int64"},
	}, "-submitted_at,id", false)
	if apiError != service.NoError {
		return apiError
	}
	dataQuery = service.NewQueryLimiter().Apply(httpReq, dataQuery)

	var result []rawAnswersData
	service.MustNotBeError(dataQuery.Scan(&result).Error())

	responseData := srv.convertDBDataToResponse(result)

	render.Respond(rw, httpReq, responseData)
	return service.NoError
}

// swagger:ignore
type rawAnswersData struct {
	ID            int64
	Name          *string
	Type          string
	LangProg      *string
	SubmittedAt   database.Time
	Score         *float32
	Validated     *bool
	UserLogin     string  `sql:"column:login"`
	UserFirstName *string `sql:"column:first_name"`
	UserLastName  *string `sql:"column:last_name"`
}

type answersResponseAnswerUser struct {
	// required: true
	Login string `json:"login"`
	// Nullable
	// required: true
	FirstName *string `json:"first_name"`
	// Nullable
	// required: true
	LastName *string `json:"last_name"`
}

// swagger:model
type answersResponseAnswer struct {
	// `users_answers.id`
	// required: true
	ID int64 `json:"id,string"`
	// Nullable
	// required: true
	Name *string `json:"name"`
	// required: true
	// enum: Submission,Saved,Current
	Type string `json:"type"`
	// Nullable
	// required: true
	LangProg *string `json:"lang_prog"`
	// required: true
	SubmittedAt database.Time `json:"submitted_at"`
	// Nullable
	// required: true
	Score *float32 `json:"score"`
	// Nullable
	// required: true
	Validated *bool `json:"validated"`

	// required: true
	User answersResponseAnswerUser `json:"user"`
}

func (srv *Service) convertDBDataToResponse(rawData []rawAnswersData) (response *[]answersResponseAnswer) {
	responseData := make([]answersResponseAnswer, 0, len(rawData))
	for _, row := range rawData {
		responseData = append(responseData, answersResponseAnswer{
			ID:          row.ID,
			Name:        row.Name,
			Type:        row.Type,
			LangProg:    row.LangProg,
			SubmittedAt: row.SubmittedAt,
			Score:       row.Score,
			Validated:   row.Validated,
			User: answersResponseAnswerUser{
				Login:     row.UserLogin,
				FirstName: row.UserFirstName,
				LastName:  row.UserLastName,
			},
		})
	}
	return &responseData
}

func (srv *Service) checkAccessRightsForGetAnswersByAttemptID(attemptID int64, user *database.User) service.APIError {
	var count int64
	itemsUserCanAccess := srv.Store.Permissions().WithViewPermissionForUser(user, "content")

	groupsManagedByUser := srv.Store.GroupAncestors().ManagedByUser(user).Select("child_group_id")
	groupsWhereUserIsMember := srv.Store.GroupGroups().WhereUserIsMember(user).Select("parent_group_id")

	service.MustNotBeError(srv.Store.GroupAttempts().ByID(attemptID).
		Joins("JOIN ? rights ON rights.item_id = groups_attempts.item_id", itemsUserCanAccess.SubQuery()).
		Where("(groups_attempts.group_id IN ?) OR (groups_attempts.group_id IN ?) OR groups_attempts.group_id = ?",
			groupsManagedByUser.SubQuery(),
			groupsWhereUserIsMember.SubQuery(),
			user.GroupID).
		Count(&count).Error())
	if count == 0 {
		return service.InsufficientAccessRightsError
	}
	return service.NoError
}

func (srv *Service) checkAccessRightsForGetAnswersByUserIDAndItemID(userID, itemID int64, user *database.User) service.APIError {
	if userID != user.GroupID {
		count := 0
		err := srv.Store.GroupAncestors().ManagedByUser(user).
			Where("child_group_id=?", userID).
			Count(&count).Error()
		service.MustNotBeError(err)
		if count == 0 {
			return service.InsufficientAccessRightsError
		}
	}

	accessDetails, err := srv.Store.Items().GetAccessDetailsForIDs(user, []int64{itemID})
	service.MustNotBeError(err)

	if len(accessDetails) == 0 || accessDetails[0].IsForbidden() {
		return service.ErrNotFound(errors.New("insufficient access rights on the given item id"))
	}

	if accessDetails[0].IsInfo() {
		return service.ErrForbidden(errors.New("insufficient access rights on the given item id"))
	}

	return service.NoError
}
