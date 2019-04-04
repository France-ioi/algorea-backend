package answers

import (
	"errors"
	"fmt"
	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/go-chi/render"
	"net/http"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

func (srv *Service) getAnswers(rw http.ResponseWriter, httpReq *http.Request) service.APIError {
	user := srv.GetUser(httpReq)

	dataQuery := srv.Store.UserAnswers().WithUsers().
		Select(`users_answers.ID, users_answers.sName, users_answers.sType, users_answers.sLangProg,
            users_answers.sSubmissionDate, users_answers.iScore, users_answers.bValidated,
            users.sLogin, users.sFirstName, users.sLastName`).
		Order("sSubmissionDate DESC")

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

		// we should create an index on `users_answers`.`idAttempt` for this query
		dataQuery = dataQuery.Where("idAttempt = ?", attemptID)
	} else { // user_id + item_id
		if result := srv.checkAccessRightsForGetAnswersByUserIDAndItemID(userID, itemID, user); result != service.NoError {
			return result
		}

		dataQuery = dataQuery.Where("idItem = ? AND idUser = ?", itemID, userID)
	}

	var result []rawAnswersData
	if err := dataQuery.Scan(&result).Error(); err != nil {
		return service.ErrUnexpected(err)
	}

	responseData := srv.convertDBDataToResponse(result)

	render.Respond(rw, httpReq, responseData)
	return service.NoError
}

type rawAnswersData struct {
	ID             int64    `sql:"column:ID"`
	Name           *string  `sql:"column:sName"`
	Type           string   `sql:"column:sType"`
	LangProg       *string  `sql:"column:sLangProg"`
	SubmissionDate string   `sql:"column:sSubmissionDate"`
	Score          *float32 `sql:"column:iScore"`
	Validated      *bool    `sql:"column:bValidated"`
	UserLogin      string   `sql:"column:sLogin"`
	UserFirstName  *string  `sql:"column:sFirstName"`
	UserLastName   *string  `sql:"column:sLastName"`
}

type answersResponseAnswerUser struct {
	Login     string  `json:"login"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}

type answersResponseAnswer struct {
	ID             int64    `json:"id"`
	Name           *string  `json:"name,omitempty"`
	Type           string   `json:"type"`
	LangProg       *string  `json:"lang_prog,omitempty"`
	SubmissionDate string   `json:"submission_date"`
	Score          *float32 `json:"score,omitempty"`
	Validated      *bool    `json:"validated,omitempty"`

	User answersResponseAnswerUser `json:"user"`
}

func (srv *Service) convertDBDataToResponse(rawData []rawAnswersData) (response *[]answersResponseAnswer) {
	responseData := make([]answersResponseAnswer, 0, len(rawData))
	for _, row := range rawData {
		responseData = append(responseData, answersResponseAnswer{
			ID:             row.ID,
			Name:           row.Name,
			Type:           row.Type,
			LangProg:       row.LangProg,
			SubmissionDate: row.SubmissionDate,
			Score:          row.Score,
			Validated:      row.Validated,
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
	itemsUserCanAccess := srv.Store.Items().AccessRights(user).
		Having("fullAccess>0 OR partialAccess>0")
	if itemsUserCanAccess.Error() == database.ErrUserNotFound {
		return service.InsufficientAccessRightsError
	}
	service.MustNotBeError(itemsUserCanAccess.Error())

	groupsOwnedByUser := srv.Store.GroupAncestors().OwnedByUser(user).Select("idGroupChild")
	service.MustNotBeError(groupsOwnedByUser.Error())

	groupsWhereUserIsMember := srv.Store.GroupGroups().WhereUserIsMember(user).Select("idGroupParent")
	service.MustNotBeError(groupsWhereUserIsMember.Error())

	if err := srv.Store.GroupAttempts().ByID(attemptID).
		Joins("JOIN ? rights ON rights.idItem = groups_attempts.idItem", itemsUserCanAccess.SubQuery()).
		Where("((groups_attempts.idGroup IN ?) OR (groups_attempts.idGroup IN ?))",
			groupsOwnedByUser.SubQuery(),
			groupsWhereUserIsMember.SubQuery()).
		Count(&count).Error(); err != nil {
		if err == database.ErrUserNotFound {
			return service.InsufficientAccessRightsError
		}
		return service.ErrUnexpected(err)
	}
	if count == 0 {
		return service.InsufficientAccessRightsError
	}
	return service.NoError
}

func (srv *Service) checkAccessRightsForGetAnswersByUserIDAndItemID(userID, itemID int64, user *database.User) service.APIError {
	if userID != user.UserID {
		count := 0
		givenUserSelfGroup := srv.Store.Users().ByID(userID).Select("idGroupSelf")
		service.MustNotBeError(givenUserSelfGroup.Error())
		err := srv.Store.GroupAncestors().OwnedByUser(user).
			Where("idGroupChild=?", givenUserSelfGroup.SubQuery()).
			Count(&count).Error()
		if err == database.ErrUserNotFound {
			return service.InsufficientAccessRightsError
		}
		service.MustNotBeError(err)
		if count == 0 {
			return service.InsufficientAccessRightsError
		}
	}

	accessDetails, err := srv.Store.Items().GetAccessDetailsForIDs(user, []int64{itemID})
	if err == database.ErrUserNotFound {
		return service.InsufficientAccessRightsError
	}
	service.MustNotBeError(err)

	if len(accessDetails) == 0 || accessDetails[0].IsForbidden() {
		return service.ErrNotFound(errors.New("insufficient access rights on the given item id"))
	}

	if accessDetails[0].IsGrayed() {
		return service.ErrForbidden(errors.New("insufficient access rights on the given item id"))
	}

	return service.NoError
}
