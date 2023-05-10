package items

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"gorm.io/gorm/clause"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation GET /items/{ids}/breadcrumbs items itemBreadcrumbsGet
//
//		---
//		summary: Get item breadcrumbs
//		description: >
//
//	  Returns brief item information for items listed in `ids` in the user's preferred language (if exist) or the items'
//	  default language.
//
//
//				Restrictions:
//	    * the list of item IDs should be a valid path from a root item
//	     (`items.id`=`groups.root_activity_id|root_skill_id` for one of the participant's ancestor groups or managed groups),
//	    * `as_team_id` (if given) should be the current user's team,
//	    * the participant should have at least 'content' access on each listed item except the last one through that path,
//	      and at least 'info' access on the last item,
//	    * all the results within the ancestry of `attempt_id`/`parent_attempt_id` on the items' path
//	      (except for the last item if `parent_attempt_id` is given) should be started (`started_at` is not null),
//
//	    otherwise the 'forbidden' error is returned.
//		parameters:
//			- name: ids
//				in: path
//				type: string
//				description: slash-separated list of IDs
//				required: true
//			- name: parent_attempt_id
//				description: "`id` of an attempt for the second to the last item in the path.
//	                This parameter is incompatible with `attempt_id`."
//				in: query
//				type: integer
//			- name: attempt_id
//				description: "`id` of an attempt for the last item in the path.
//	                This parameter is incompatible with `parent_attempt_id`."
//				in: query
//				type: integer
//			- name: as_team_id
//				in: query
//				type: integer
//				format: int64
//		responses:
//			"200":
//				description: OK. Breadcrumbs data
//				schema:
//					type: array
//					items:
//						type: object
//						properties:
//							item_id:
//								type: string
//								format: int64
//							type:
//								type: string
//								enum: [Chapter,Task,Skill]
//							title:
//								type: string
//							language_tag:
//								type: string
//							attempt_id:
//								description: the attempt for this item (result) within ancestry of `attempt_id` or `parent_attempt_id`
//	         	                 (skipped for the last item if `parent_attempt_id` is used)
//								type: string
//								format: int64
//							attempt_number:
//								description: the order of this attempt result among the other results (within the parent attempt)
//	                	         sorted by `started_at`
//	                  	       (only for items allowing multiple submissions;
//	                    	     skipped for the last item if `parent_attempt_id` is used)
//								type: string
//								format: int64
//						required: [item_id, type, title, language_tag]
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getBreadcrumbs(w http.ResponseWriter, r *http.Request) service.APIError {
	// Get IDs from request and validate it.
	ids, groupID, attemptID, parentAttemptID, attemptIDSet, user, apiError := srv.parametersForGetBreadcrumbs(r)
	if apiError != service.NoError {
		return apiError
	}

	var attemptIDMap map[int64]int64
	var attemptNumberMap map[int64]int
	var err error
	store := srv.GetStore(r)
	if attemptIDSet {
		attemptIDMap, attemptNumberMap, err = store.Items().BreadcrumbsHierarchyForAttempt(ids, groupID, attemptID, false)
	} else {
		attemptIDMap, attemptNumberMap, err = store.Items().BreadcrumbsHierarchyForParentAttempt(ids, groupID, parentAttemptID, false)
	}
	service.MustNotBeError(err)
	if attemptIDMap == nil {
		return service.ErrForbidden(errors.New("item ids hierarchy is invalid or insufficient access rights"))
	}

	idsInterface := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsInterface = append(idsInterface, id)
	}
	var result []map[string]interface{}
	service.MustNotBeError(store.Items().Select(`
			items.id AS item_id,
			items.type,
			COALESCE(user_strings.title, default_strings.title) AS title,
			COALESCE(user_strings.language_tag, default_strings.language_tag) AS language_tag`).
		JoinsUserAndDefaultItemStrings(user).
		Where("items.id IN (?)", ids).
		Clauses(clause.OrderBy{
			Expression: clause.Expr{SQL: "FIELD(items.id, ?)", Vars: []interface{}{idsInterface}, WithoutParentheses: true},
		}).
		ScanIntoSliceOfMaps(&result).Error())

	for index := range result {
		if itemAttemptID, ok := attemptIDMap[result[index]["item_id"].(int64)]; ok {
			result[index]["attempt_id"] = itemAttemptID
		}
		if itemAttemptNumber, ok := attemptNumberMap[result[index]["item_id"].(int64)]; ok {
			result[index]["attempt_order"] = itemAttemptNumber
		}
	}
	render.Respond(w, r, service.ConvertSliceOfMapsFromDBToJSON(result))
	return service.NoError
}

func (srv *Service) parametersForGetBreadcrumbs(r *http.Request) (
	ids []int64, participantID, attemptID, parentAttemptID int64, attemptIDSet bool, user *database.User, apiError service.APIError,
) {
	var err error
	ids, err = idsFromRequest(r)
	if err != nil {
		return nil, 0, 0, 0, false, nil, service.ErrInvalidRequest(err)
	}

	attemptID, parentAttemptID, attemptIDSet, apiError = attemptIDOrParentAttemptID(r)
	if apiError != service.NoError {
		return nil, 0, 0, 0, false, nil, apiError
	}

	user = srv.GetUser(r)
	participantID = service.ParticipantIDFromContext(r.Context())
	return ids, participantID, attemptID, parentAttemptID, attemptIDSet, user, service.NoError
}

func attemptIDOrParentAttemptID(r *http.Request) (
	attemptID, parentAttemptID int64, attemptIDSet bool, apiError service.APIError,
) {
	var err error
	attemptIDSet = len(r.URL.Query()["attempt_id"]) != 0
	parentAttemptIDSet := len(r.URL.Query()["parent_attempt_id"]) != 0
	if attemptIDSet {
		if parentAttemptIDSet {
			return 0, 0, false, service.ErrInvalidRequest(errors.New("only one of attempt_id and parent_attempt_id can be given"))
		}
		attemptID, err = service.ResolveURLQueryGetInt64Field(r, "attempt_id")
		if err != nil {
			return 0, 0, false, service.ErrInvalidRequest(err)
		}
	}
	if parentAttemptIDSet {
		parentAttemptID, err = service.ResolveURLQueryGetInt64Field(r, "parent_attempt_id")
		if err != nil {
			return 0, 0, false, service.ErrInvalidRequest(err)
		}
	}
	if !attemptIDSet && !parentAttemptIDSet {
		return 0, 0, false, service.ErrInvalidRequest(errors.New("one of attempt_id and parent_attempt_id should be given"))
	}
	return attemptID, parentAttemptID, attemptIDSet, service.NoError
}

func idsFromRequest(r *http.Request) ([]int64, error) {
	return service.ResolveURLQueryPathInt64SliceFieldWithLimit(r, "ids", 10)
}
