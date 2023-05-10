package groups

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/France-ioi/validator"
	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/formdata"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// Information of the group to be modified
// swagger:model
type groupUpdateInput struct {
	Name  string `json:"name" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	Grade int32  `json:"grade" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	// Nullable
	Description *string `json:"description" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	IsOpen      bool    `json:"is_open" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	// If changed from true to false, is automatically switches all requests to join this group from requestSent to requestRefused
	IsPublic bool `json:"is_public" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	// Duration after the first use of the code when it will expire (in seconds)
	// Nullable
	CodeLifetime *int32 `json:"code_lifetime" validate:"changing_requires_can_manage_at_least=memberships,null|gte=0"`
	// Nullable
	CodeExpiresAt *database.Time `json:"code_expires_at" validate:"changing_requires_can_manage_at_least=memberships"`
	// Nullable
	RootActivityID *int64 `json:"root_activity_id" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	// Nullable
	RootSkillID *int64 `json:"root_skill_id" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	// Can be set only if root_activity_id is set and
	// the current user has the 'can_make_session_official' permission on the activity item
	IsOfficialSession       bool `json:"is_official_session" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	OpenActivityWhenJoining bool `json:"open_activity_when_joining" validate:"changing_requires_can_manage_at_least=memberships_and_group"`

	// Can be changed only from false to true
	// (changing auto-rejects all pending join/leave requests and withdraws all pending invitations)
	FrozenMembership bool `json:"frozen_membership"  validate:"changing_requires_can_manage_at_least=memberships,frozen_membership"`
	// Nullable; cannot be set to null when enforce_max_participant is true
	MaxParticipants *int `json:"max_participants" validate:"changing_requires_can_manage_at_least=memberships,max_participants"`
	// Cannot be set to true when max_participants is null
	EnforceMaxParticipants bool `json:"enforce_max_participants" validate:"changing_requires_can_manage_at_least=memberships,enforce_max_participants"` //nolint:lll

	// enum: none,view,edit
	// Cannot be changed to 'edit'
	RequirePersonalInfoAccessApproval  string         `json:"require_personal_info_access_approval" validate:"changing_requires_can_manage_at_least=memberships_and_group,require_personal_info_access_approval,oneof=none view edit"` //nolint:lll
	RequireLockMembershipApprovalUntil *database.Time `json:"require_lock_membership_approval_until" validate:"changing_requires_can_manage_at_least=memberships_and_group"`                                                           //nolint:lll
	RequireWatchApproval               bool           `json:"require_watch_approval" validate:"changing_requires_can_manage_at_least=memberships_and_group"`                                                                           //nolint:lll
	RequireMembersToJoinParent         bool           `json:"require_members_to_join_parent" validate:"changing_requires_can_manage_at_least=memberships_and_group"`                                                                   //nolint:lll

	// Nullable
	Organizer *string `json:"organizer" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	// Nullable
	AddressLine1 *string `json:"address_line1" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	// Nullable
	AddressLine2 *string `json:"address_line2" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	// Nullable
	AddressPostcode *string `json:"address_postcode" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	// Nullable
	AddressCity *string `json:"address_city" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	// Nullable
	AddressCountry *string `json:"address_country" validate:"changing_requires_can_manage_at_least=memberships_and_group"`
	// Nullable
	ExpectedStart *database.Time `json:"expected_start" validate:"changing_requires_can_manage_at_least=memberships_and_group"`

	CanManageValue int `json:"-"`
}

// swagger:operation PUT /groups/{group_id} groups groupUpdate
//
//	---
//	summary: Update a group
//	description: Updates group information.
//
//		Requires the user to be a manager of the group, otherwise the 'forbidden' error is returned.
//
//
//		The user should have `can_manage` >= 'memberships' in order to modify `code_expires_at`, `code_lifetime`,
//		`frozen_membership`, `max_participants`, or `enforce_max_participants`, otherwise the "bad request" error is returned.
//		In order to modify the values of other fields, the user should have `can_manage` >= 'memberships_and_group'.
//
//
//		If the `root_activity_id` item is provided and is not null, the item should not be a skill, and
//		the user should have at least 'can_view:info' permission on it, otherwise the 'forbidden' error is returned.
//
//
//		If the `root_skill_id` item is provided and is not null, the item should be a skill, and the user should have at least
//		'can_view:info' permission on it, otherwise the 'forbidden' error is returned.
//
//
//		 If `is_official_session` is being changed to true, the user should have at least
//		'can_make_session_official' permission on the activity item, otherwise the 'forbidden' error is returned.
//
//
//		Setting `is_official_session` to true while keeping `root_activity_id` not set or setting `root_activity_id` to null for
//		an official session will cause the "bad request" error.
//
//
//		Setting `enforce_max_participants` to true while keeping `max_participants` null or setting `max_participants` to null
//		while keeping `enforce_max_participants` = true will cause the "bad request" error.
//
//
//		Changing `require_personal_info_access_approval` to 'edit' will cause the "bad request" error.
//	parameters:
//		- name: group_id
//			in: path
//			required: true
//			type: integer
//		- name: group information
//			in: body
//			required: true
//			schema:
//				"$ref": "#/definitions/groupUpdateInput"
//	responses:
//		"200":
//			"$ref": "#/responses/updatedResponse"
//		"400":
//			"$ref": "#/responses/badRequestResponse"
//		"401":
//			"$ref": "#/responses/unauthorizedResponse"
//		"403":
//			"$ref": "#/responses/forbiddenResponse"
//		"500":
//			"$ref": "#/responses/internalErrorResponse"
func (srv *Service) updateGroup(w http.ResponseWriter, r *http.Request) service.APIError {
	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(r)

	apiErr := service.NoError

	err = srv.GetStore(r).InTransaction(func(s *database.DataStore) error {
		groupStore := s.Groups()

		var currentGroupData groupUpdateInput

		err = groupStore.ManagedBy(user).
			Select(`
				groups.name, groups.grade, groups.description, groups.is_open, groups.is_public,
				groups.code_lifetime, groups.code_expires_at, groups.root_activity_id, groups.root_skill_id,
				groups.is_official_session, groups.open_activity_when_joining, groups.require_members_to_join_parent,
				groups.frozen_membership, groups.organizer, groups.address_line1, groups.address_line2,
				groups.address_postcode, groups.address_city, groups.address_country, groups.expected_start,
				groups.max_participants, groups.enforce_max_participants, groups.require_personal_info_access_approval,
				groups.require_lock_membership_approval_until, groups.require_watch_approval,
				groups.require_members_to_join_parent,
				MAX(can_manage_value) AS can_manage_value`).WithWriteLock().
			Where("groups.id = ?", groupID).Group("groups.id").Take(&currentGroupData).Error()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apiErr = service.InsufficientAccessRightsError
			return apiErr.Error // rollback
		}
		service.MustNotBeError(err)

		var formData *formdata.FormData
		formData, err = validateUpdateGroupInput(r, &currentGroupData, s)
		if err != nil {
			apiErr = service.ErrInvalidRequest(err)
			return apiErr.Error // rollback
		}

		dbMap := formData.ConstructMapForDB()

		apiErr = validateRootActivityIDAndIsOfficial(s, user, currentGroupData.RootActivityID, currentGroupData.IsOfficialSession, dbMap)
		if apiErr != service.NoError {
			return apiErr.Error // rollback
		}
		apiErr = validateRootSkillID(s, user, currentGroupData.RootSkillID, dbMap)
		if apiErr != service.NoError {
			return apiErr.Error // rollback
		}

		service.MustNotBeError(refuseSentGroupRequestsIfNeeded(
			groupStore, groupID, user.GroupID, dbMap, currentGroupData.IsPublic, currentGroupData.FrozenMembership))

		// update the group
		service.MustNotBeError(groupStore.Where("id = ?", groupID).Updates(dbMap).Error())

		return nil // commit
	})

	if apiErr != service.NoError {
		return apiErr
	}
	service.MustNotBeError(err)

	response := service.Response{Success: true, Message: "updated"}
	render.Respond(w, r, &response)

	return service.NoError
}

func validateRootActivityIDAndIsOfficial(
	store *database.DataStore, user *database.User, oldRootActivityID *int64, oldIsOfficialSession bool,
	dbMap map[string]interface{},
) service.APIError {
	rootActivityIDToCheck := oldRootActivityID
	rootActivityID, rootActivityIDSet := dbMap["root_activity_id"]
	rootActivityIDChanged := rootActivityIDSet && !int64PtrEqualValues(oldRootActivityID, rootActivityID.(*int64))
	if rootActivityIDChanged {
		rootActivityIDToCheck = rootActivityID.(*int64)
		if rootActivityIDToCheck != nil {
			apiError := validateRootActivityID(store, user, rootActivityIDToCheck)
			if apiError != service.NoError {
				return apiError
			}
		}
	}

	if isTryingToChangeOfficialSessionActivity(dbMap, oldIsOfficialSession, rootActivityIDChanged) {
		if rootActivityIDToCheck == nil {
			return service.ErrInvalidRequest(errors.New("the root_activity_id should be set for official sessions"))
		}
		found, err := store.PermissionsGranted().WithWriteLock().
			Joins(`
				JOIN groups_ancestors_active ON groups_ancestors_active.ancestor_group_id = permissions_granted.group_id AND
				     groups_ancestors_active.child_group_id = ?`, user.GroupID).
			Where("permissions_granted.can_make_session_official OR permissions_granted.is_owner").
			Where("permissions_granted.item_id = ?", rootActivityIDToCheck).
			HasRows()
		service.MustNotBeError(err)
		if !found {
			return service.ErrForbidden(errors.New("not enough permissions for attaching the group to the activity as an official session"))
		}
	}
	return service.NoError
}

func validateRootActivityID(store *database.DataStore, user *database.User, rootActivityIDToCheck *int64) service.APIError {
	if rootActivityIDToCheck != nil {
		found, errorInTransaction := store.Items().ByID(*rootActivityIDToCheck).Where("type != 'Skill'").WithWriteLock().
			WhereUserHasViewPermissionOnItems(user, "info").HasRows()
		service.MustNotBeError(errorInTransaction)
		if !found {
			return service.ErrForbidden(errors.New("no access to the root activity or it is a skill"))
		}
	}
	return service.NoError
}

func validateRootSkillID(store *database.DataStore, user *database.User, oldRootSkillID *int64,
	dbMap map[string]interface{},
) service.APIError {
	newRootSkillIDInterface, newRootSkillIDSet := dbMap["root_skill_id"]
	if newRootSkillIDSet && newRootSkillIDInterface != nil {
		newRootSkillID := newRootSkillIDInterface.(*int64)

		rootSkillIDChanged := !int64PtrEqualValues(oldRootSkillID, newRootSkillID)
		if rootSkillIDChanged && newRootSkillID != nil {
			found, errorInTransaction := store.Items().
				ByID(*newRootSkillID).
				Where("type = 'Skill'").
				WithWriteLock().
				WhereUserHasViewPermissionOnItems(user, "info").
				HasRows()
			service.MustNotBeError(errorInTransaction)
			if !found {
				return service.ErrForbidden(errors.New("no access to the root skill or it is not a skill"))
			}
		}
	}

	return service.NoError
}

// refuseSentGroupRequestsIfNeeded automatically refuses all requests to join this group
// (removes them from group_pending_requests and inserts appropriate group_membership_changes
// with `action` = 'join_request_refused') if is_public is changed from true to false.
func refuseSentGroupRequestsIfNeeded(
	store *database.GroupStore, groupID, initiatorID int64, dbMap map[string]interface{},
	previousIsPublicValue, previousFrozenMembershipValue bool,
) error {
	var shouldRefusePending bool

	pendingTypesToHandle := make([]string, 0, 3)
	pendingTypesToHandle = append(pendingTypesToHandle, "join_request")

	// if is_public is going to be changed from true to false
	if newIsPublic, ok := dbMap["is_public"]; ok && !newIsPublic.(bool) && previousIsPublicValue {
		shouldRefusePending = true
	}
	if newFrozenMembership, ok := dbMap["frozen_membership"]; ok && newFrozenMembership.(bool) && !previousFrozenMembershipValue {
		shouldRefusePending = true
		pendingTypesToHandle = append(pendingTypesToHandle, "leave_request", "invitation")
	}

	if shouldRefusePending {
		service.MustNotBeError(store.Exec(`
			INSERT INTO group_membership_changes (group_id, member_id, action, at, initiator_id)
			SELECT group_id, member_id,
				CASE type
					WHEN 'join_request' THEN 'join_request_refused'
					WHEN 'leave_request' THEN 'leave_request_refused'
					WHEN 'invitation' THEN 'invitation_withdrawn'
				END,
				NOW(), ?
			FROM group_pending_requests
			WHERE group_id = ? AND type IN (?)
			FOR UPDATE`, initiatorID, groupID, pendingTypesToHandle).Error())
		// refuse sent group requests
		return store.GroupPendingRequests().
			Where("type IN (?)", pendingTypesToHandle).
			Where("group_id = ?", groupID).
			Delete().Error()
	}
	return nil
}

func validateUpdateGroupInput(
	r *http.Request, currentGroupData *groupUpdateInput, store *database.DataStore,
) (*formdata.FormData, error) {
	input := &groupUpdateInput{}
	formData := formdata.NewFormData(input)
	formData.SetOldValues(currentGroupData)
	formData.RegisterValidation("frozen_membership", constructFrozenMembershipValidator(formData))
	formData.RegisterTranslation("frozen_membership", "can only be changed from false to true")

	formData.RegisterValidation("changing_requires_can_manage_at_least",
		constructChangingRequiresCanManageAtLeastValidator(formData, store, currentGroupData))
	formData.RegisterTranslation("changing_requires_can_manage_at_least", "only managers with 'can_manage' >= '%[3]s' can modify this field")

	formData.RegisterValidation("require_personal_info_access_approval", constructRequirePersonalInfoAccessApprovalValidator(formData))
	formData.RegisterTranslation("require_personal_info_access_approval", "cannot be changed to 'edit'")

	formData.RegisterValidation("max_participants", constructMaxParticipantsValidator(formData, currentGroupData))
	formData.RegisterTranslation("max_participants", "cannot be set to null when 'enforce_max_participants' is true")

	formData.RegisterValidation("enforce_max_participants", constructEnforceMaxParticipantsValidator(formData, currentGroupData))
	formData.RegisterTranslation("enforce_max_participants", "cannot be set to true when 'max_participants' is null")

	formData.RegisterTranslation("null|gte=0", "can be null or an integer between 0 and 2147483647 inclusively")

	err := formData.ParseJSONRequestData(r)
	return formData, err
}

func int64PtrEqualValues(a, b *int64) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func isTryingToChangeOfficialSessionActivity(dbMap map[string]interface{}, oldIsOfficialSession, rootActivityIDChanged bool) bool {
	isOfficialSession, isOfficialSessionSet := dbMap["is_official_session"]
	isOfficialSessionChanged := isOfficialSessionSet && oldIsOfficialSession != isOfficialSession.(bool)
	return isOfficialSessionChanged && isOfficialSession.(bool) ||
		!isOfficialSessionChanged && oldIsOfficialSession && rootActivityIDChanged
}

func constructFrozenMembershipValidator(formData *formdata.FormData) validator.Func {
	return formData.ValidatorSkippingUnchangedFields(func(fl validator.FieldLevel) bool {
		// return false if frozen_membership is going to be changed from true to false
		return fl.Field().Bool()
	})
}

func constructChangingRequiresCanManageAtLeastValidator(formData *formdata.FormData, store *database.DataStore,
	currentGroupData *groupUpdateInput,
) validator.Func {
	return formData.ValidatorSkippingUnchangedFields(func(fl validator.FieldLevel) bool {
		return currentGroupData.CanManageValue >= store.GroupManagers().CanManageIndexByName(fl.Param())
	})
}

func constructRequirePersonalInfoAccessApprovalValidator(formData *formdata.FormData) validator.Func {
	return formData.ValidatorSkippingUnchangedFields(func(fl validator.FieldLevel) bool {
		return fl.Field().Interface() != "edit"
	})
}

func constructMaxParticipantsValidator(formData *formdata.FormData, currentGroupData *groupUpdateInput) validator.Func {
	return formData.ValidatorSkippingUnchangedFields(func(fl validator.FieldLevel) bool {
		if fl.Field().Kind() == reflect.Ptr {
			enforceMaxParticipants := currentGroupData.EnforceMaxParticipants
			if formData.IsSet("enforce_max_participants") {
				enforceMaxParticipants = fl.Top().Elem().FieldByName("EnforceMaxParticipants").Interface().(bool)
			}
			if enforceMaxParticipants {
				return false
			}
		}
		return true
	})
}

func constructEnforceMaxParticipantsValidator(formData *formdata.FormData, currentGroupData *groupUpdateInput) validator.Func {
	return formData.ValidatorSkippingUnchangedFields(func(fl validator.FieldLevel) bool {
		if fl.Field().Interface().(bool) {
			maxParticipants := currentGroupData.MaxParticipants
			if formData.IsSet("max_participants") {
				maxParticipants = fl.Top().Elem().FieldByName("MaxParticipants").Interface().(*int)
			}
			if maxParticipants == nil {
				return false
			}
		}
		return true
	})
}
