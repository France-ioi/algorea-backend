package groups

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"

	"github.com/France-ioi/validator"
	"github.com/go-chi/render"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/domain"
	"github.com/France-ioi/AlgoreaBackend/app/formdata"
	"github.com/France-ioi/AlgoreaBackend/app/loginmodule"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

type createUserBatchRequestSubgroup struct {
	// required: true
	// should not be a user
	GroupID int64 `json:"group_id,string" validate:"set"`
	// required: true
	// minimum: 1
	Count int `json:"count" validate:"set,min=1"`
}

// swagger:model createUserBatchRequest
type createUserBatchRequest struct {
	// required: true
	GroupPrefix string `json:"group_prefix" validate:"set"`
	// required: true
	// pattern: ^[a-z0-9-]{2,14}$
	CustomPrefix string `json:"custom_prefix" validate:"set,custom_prefix"`
	// required: true
	//	minItems: 1
	Subgroups []createUserBatchRequestSubgroup `json:"subgroups" validate:"set,min=1,dive"`
	// required: true
	//	min: 3
	//	max: 29
	PostfixLength int `json:"postfix_length" validate:"set,min=3,max=29"`
	// required: true
	//	min: 6
	//	max: 50
	PasswordLength int `json:"password_length" validate:"set,min=6,max=50"`
}

var customPrefixRegexp = regexp.MustCompile(`^[a-z0-9-]{2,14}$`)

type subgroupApproval struct {
	RequirePersonalInfoAccessApproval bool
	RequireLockMembershipApproval     bool
	RequireWatchApproval              bool
}

// swagger:operation POST /user-batches groups createUserBatch
//
//		---
//		summary: Create a user batch
//		description: >
//
//	  Creates a batch of users:
//
//	  * creates a new row in users_batches,
//
//	  * creates new users in the login module,
//
//	  * inserts the created users into the `users` table,
//
//	  * adds the created users into groups specified as `subgroups[...].group_id` giving all the required approvals.
//
//
//				Restrictions:
//
//	  * The authenticated user (or one of his group ancestors) should be a manager of the group
//	    (directly, or of one of its ancestors) linked to the `group_prefix`
//	    with at least 'can_manage:memberships', otherwise the 'forbidden' response is returned.
//	  * The 'subgroup.group_id'-s should be descendants of the group linked to the `group_prefix` or be the group itself,
//	    otherwise the 'forbidden' response is returned.
//	  * The 'subgroup.group_id'-s should not be of type 'User', otherwise the 'forbidden' response is returned.
//	  * The `group_prefix.allow_new` should be true, otherwise the 'forbidden' response is returned.
//	  * 32^`postfix_length` should be greater than 2 * sum of `subgroups.count`
//	    (to prevent being unable to generate unique logins), otherwise the 'bad request' response is returned.
//	  * Sum of `subgroups.count` + sum of sizes of existing batches under the same `group_prefix`
//	    should not be greater than `max_users` of the prefix, otherwise the 'bad request' response is returned.
//		parameters:
//			- in: body
//				name: data
//				required: true
//				description: The user batch to create
//				schema:
//					"$ref": "#/definitions/createUserBatchRequest"
//		responses:
//			"201":
//				description: "Created. Success response with the newly created task token"
//				schema:
//						type: object
//						required: [success, message, data]
//						properties:
//							success:
//								description: "true"
//								type: boolean
//								enum: [true]
//							message:
//								description: created
//								type: string
//								enum: [created]
//							data:
//								type: array
//								items:
//									"$ref": "#/definitions/createUserBatchResultRow"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) createUserBatch(w http.ResponseWriter, r *http.Request) service.APIError {
	var err error
	user := srv.GetUser(r)
	store := srv.GetStore(r)

	input := createUserBatchRequest{}
	formData := formdata.NewFormData(&input)
	formData.RegisterValidation("custom_prefix", func(fl validator.FieldLevel) bool {
		return customPrefixRegexp.MatchString(fl.Field().Interface().(string))
	})
	formData.RegisterTranslation("custom_prefix",
		"The custom prefix should only consist of letters/digits/hyphens and be 2-14 characters long")

	err = formData.ParseJSONRequestData(r)
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	numberOfUsersToBeCreated, subgroupsApprovals, apiError := checkCreateUserBatchRequestParameters(store, user, input)
	if apiError != service.NoError {
		return apiError
	}

	err = store.UserBatches().InsertMap(map[string]interface{}{
		"group_prefix":  input.GroupPrefix,
		"custom_prefix": input.CustomPrefix,
		"size":          numberOfUsersToBeCreated,
		"creator_id":    user.GroupID,
		"created_at":    database.Now(),
	})
	if err != nil && database.IsDuplicateEntryError(err) {
		return service.ErrInvalidRequest(errors.New("'custom_prefix' already exists for the given 'group_prefix'"))
	}
	service.MustNotBeError(err)

	result, createdUsers, err := loginmodule.NewClient(srv.AuthConfig.GetString("loginModuleURL")).
		CreateUsers(r.Context(), srv.AuthConfig.GetString("clientID"), srv.AuthConfig.GetString("clientSecret"), &loginmodule.CreateUsersParams{
			Prefix:         fmt.Sprintf("%s_%s_", input.GroupPrefix, input.CustomPrefix),
			Amount:         numberOfUsersToBeCreated,
			PostfixLength:  input.PostfixLength,
			PasswordLength: input.PasswordLength,
			LoginFixed:     func(b bool) *bool { return &b }(true),
			Language:       func(s string) *string { return &s }(user.DefaultLanguage),
		})

	defer func() {
		if p := recover(); p != nil {
			store.UserBatches().Delete("group_prefix = ? AND custom_prefix = ?", input.GroupPrefix, input.CustomPrefix)
			panic(p)
		}
	}()
	service.MustNotBeError(err)
	if !result {
		panic(errors.New("login module failed"))
	}

	users := createBatchUsersInDB(store, input, r, numberOfUsersToBeCreated, createdUsers, subgroupsApprovals, user)

	service.MustNotBeError(render.Render(w, r, service.CreationSuccess(users)))
	return service.NoError
}

func checkCreateUserBatchRequestParameters(store *database.DataStore, user *database.User, input createUserBatchRequest) (
	numberOfUsersToBeCreated int, subgroupsApprovals []subgroupApproval, apiError service.APIError,
) {
	var prefixInfo struct {
		GroupID  int64
		MaxUsers int
	}
	err := store.ActiveGroupAncestors().ManagedByUser(user).
		Joins(`JOIN user_batch_prefixes ON user_batch_prefixes.group_id = groups_ancestors_active.child_group_id AND `+
			`user_batch_prefixes.allow_new AND user_batch_prefixes.group_prefix = ?`, input.GroupPrefix).
		Where("group_managers.can_manage != 'none'").
		Select("user_batch_prefixes.group_id, user_batch_prefixes.max_users").
		Take(&prefixInfo).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil, service.InsufficientAccessRightsError
	}
	service.MustNotBeError(err)

	subgroupIDs := make([]interface{}, 0, len(input.Subgroups))

	for _, subgroup := range input.Subgroups {
		subgroupIDs = append(subgroupIDs, subgroup.GroupID)
		numberOfUsersToBeCreated += subgroup.Count
	}

	// 32^postfix_length should be greater than 2*numberOfUsersToBeCreated
	if float64(input.PostfixLength) <= math.Log(float64(2*numberOfUsersToBeCreated))/math.Log(32) {
		return 0, nil, service.ErrInvalidRequest(errors.New("'postfix_length' is too small"))
	}

	service.MustNotBeError(store.Groups().
		Joins("JOIN groups_ancestors_active ON groups_ancestors_active.child_group_id = groups.id").
		Where("ancestor_group_id = ?", prefixInfo.GroupID).
		Where("groups.id IN(?)", subgroupIDs).
		Where("groups.type != 'User'").
		Select(`
			require_personal_info_access_approval != 'none' AS require_personal_info_access_approval,
			IFNULL(require_lock_membership_approval_until > NOW(), 0) AS require_lock_membership_approval,
			require_watch_approval`).
		Clauses(clause.OrderBy{
			Expression: clause.Expr{SQL: "FIELD(groups.id, ?)", Vars: []interface{}{subgroupIDs}, WithoutParentheses: true},
		}).
		Scan(&subgroupsApprovals).Error())
	if len(subgroupsApprovals) != len(subgroupIDs) {
		return 0, nil, service.InsufficientAccessRightsError
	}

	var currentSumSize int
	service.MustNotBeError(store.UserBatches().Where("group_prefix = ?", input.GroupPrefix).
		PluckFirst("IFNULL(SUM(size), 0)", &currentSumSize).Error())
	if prefixInfo.MaxUsers < numberOfUsersToBeCreated+currentSumSize {
		return 0, nil, service.ErrInvalidRequest(errors.New("'user_batch_prefix.max_users' exceeded"))
	}
	return numberOfUsersToBeCreated, subgroupsApprovals, service.NoError
}

type resultRowUser struct {
	// required: true
	UserID int64 `json:"user_id,string"`
	// required: true
	Login string `json:"login"`
	// required: true
	Password string `json:"password"`
}

// swagger:model createUserBatchResultRow
type resultRow struct {
	// required: true
	GroupID int64 `json:"group_id,string"`
	// required: true
	Users []resultRowUser `json:"users"`
}

func createBatchUsersInDB(store *database.DataStore, input createUserBatchRequest, r *http.Request, numberOfUsersToBeCreated int,
	createdUsers []loginmodule.CreateUsersResponseDataRow, subgroupsApprovals []subgroupApproval, user *database.User,
) []*resultRow {
	result := make([]*resultRow, 0, len(subgroupsApprovals))

	service.MustNotBeError(store.InTransaction(func(store *database.DataStore) error {
		domainConfig := domain.ConfigFromContext(r.Context())

		relationsToCreate := make([]map[string]interface{}, 0, 2*numberOfUsersToBeCreated)
		usersToCreate := make([]map[string]interface{}, 0, numberOfUsersToBeCreated)
		attemptsToCreate := make([]map[string]interface{}, 0, numberOfUsersToBeCreated)
		usersInSubgroup := 0
		var currentResultRow *resultRow
		currentSubgroupIndex := -1
		for _, createdUser := range createdUsers {
			createdUser := createdUser
			if currentSubgroupIndex == -1 || usersInSubgroup == input.Subgroups[currentSubgroupIndex].Count {
				currentSubgroupIndex++
				currentResultRow = &resultRow{
					GroupID: input.Subgroups[currentSubgroupIndex].GroupID,
					Users:   make([]resultRowUser, 0, input.Subgroups[currentSubgroupIndex].Count),
				}
				result = append(result, currentResultRow)
				usersInSubgroup = 0
			}

			var userGroupID int64
			service.MustNotBeError(store.RetryOnDuplicatePrimaryKeyError(func(retryIDStore *database.DataStore) error {
				userGroupID = retryIDStore.NewID()
				return retryIDStore.Groups().InsertMap(map[string]interface{}{
					"id":          userGroupID,
					"name":        createdUser.Login,
					"type":        groupTypeUser,
					"description": createdUser.Login,
					"created_at":  database.Now(),
					"is_open":     false,
					"send_emails": false,
				})
			}))

			var personalInfoApprovedAt, lockMembershipApprovedAt, watchApprovedAt interface{}
			if subgroupsApprovals[currentSubgroupIndex].RequirePersonalInfoAccessApproval {
				personalInfoApprovedAt = database.Now()
			}
			if subgroupsApprovals[currentSubgroupIndex].RequireLockMembershipApproval {
				lockMembershipApprovedAt = database.Now()
			}
			if subgroupsApprovals[currentSubgroupIndex].RequireWatchApproval {
				watchApprovedAt = database.Now()
			}
			relationsToCreate = append(relationsToCreate,
				map[string]interface{}{
					"parent_group_id":                domainConfig.AllUsersGroupID,
					"child_group_id":                 userGroupID,
					"personal_info_view_approved_at": nil, "lock_membership_approved_at": nil, "watch_approved_at": nil,
				},
				map[string]interface{}{
					"parent_group_id":                input.Subgroups[currentSubgroupIndex].GroupID,
					"child_group_id":                 userGroupID,
					"personal_info_view_approved_at": personalInfoApprovedAt,
					"lock_membership_approved_at":    lockMembershipApprovedAt,
					"watch_approved_at":              watchApprovedAt,
				},
			)

			usersToCreate = append(usersToCreate, map[string]interface{}{
				"temp_user":        0,
				"registered_at":    database.Now(),
				"group_id":         userGroupID,
				"login_id":         createdUser.ID,
				"login":            createdUser.Login,
				"default_language": user.DefaultLanguage,
				"creator_id":       user.GroupID,
			})

			attemptsToCreate = append(attemptsToCreate, map[string]interface{}{
				"participant_id": userGroupID,
				"id":             0,
				"creator_id":     userGroupID,
				"created_at":     database.Now(),
			})

			usersInSubgroup++
			currentResultRow.Users = append(currentResultRow.Users, resultRowUser{
				UserID:   userGroupID,
				Login:    createdUser.Login,
				Password: createdUser.Password,
			})
		}
		service.MustNotBeError(store.Users().InsertMaps(usersToCreate))
		service.MustNotBeError(store.Attempts().InsertMaps(attemptsToCreate))
		return store.GroupGroups().CreateRelationsWithoutChecking(relationsToCreate)
	}))
	return result
}
