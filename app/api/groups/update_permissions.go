package groups

import (
	"errors"
	"net/http"
	"time"

	"github.com/France-ioi/validator"
	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/formdata"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// Access rights to be set
// swagger:model
type updatePermissionsInput struct {
	// Can be changed to true only if the current user is an owner of the item
	IsOwner bool `json:"is_owner" validate:"is_owner"`
	// The current user should have `permissions_generated.can_grant_view_generated` >= the new value (or 'info' if the new value is 'enter')
	// in order to increase level of this permission
	CanView string `json:"can_view" validate:"oneof=none info content content_with_descendants solution,can_view"`
	// The granted `can_view` should be >= the new value
	// (or 'info' if the new value is 'enter', or 'solution' if the new value is 'solution_with_grant') and
	// the current user should have `permissions_generated.can_grant_view_generated` = 'solution_with_grant'
	// in order to increase level of this permission. Only owners can increase it to 'solution_with_grant'.
	CanGrantView string `json:"can_grant_view" validate:"oneof=none enter content content_with_descendants solution solution_with_grant,can_grant_view"` //nolint:lll
	// The granted `can_view` should be >= 'content' and
	// the current user should have `permissions_generated.can_watch_generated` = 'answer_with_grant'
	// in order to increase level of this permission. Only owners can increase it to 'answer_with_grant'.
	CanWatch string `json:"can_watch" validate:"oneof=none result answer answer_with_grant,can_watch"`
	// The granted `can_view` should be >= 'content' and
	// the current user should have permissions_generated.can_edit_generated = 'all_with_grant'
	// in order to increase level of this permission. Only owners can increase it to 'all_with_grant'.
	CanEdit string `json:"can_edit" validate:"oneof=none children all all_with_grant,can_edit"`
	// The granted `can_view` should be >= 'info' and
	// the current user should be an owner in order to change this field to true.
	CanMakeSessionOfficial bool `json:"can_make_session_official" validate:"can_make_session_official"`
	// The current user should have permissions_generated.can_grant_view_generated >= 'enter' in order to
	// decrease this field's value.
	CanEnterFrom time.Time `json:"can_enter_from" validate:"can_enter_from"`
	// The current user should have permissions_generated.can_grant_view_generated >= 'enter' in order to
	// increase this field's value.
	CanEnterUntil time.Time `json:"can_enter_until" validate:"can_enter_until"`
}

type userPermissions struct {
	CanViewValue           int
	CanGrantViewValue      int
	CanWatchValue          int
	CanEditValue           int
	CanEnterFrom           database.Time
	CanEnterUntil          database.Time
	CanMakeSessionOfficial bool
	IsOwner                bool
}

type managerGeneratedPermissions struct {
	CanGrantViewGeneratedValue int
	CanWatchGeneratedValue     int
	CanEditGeneratedValue      int
	IsOwnerGenerated           bool
}

// swagger:operation PUT /groups/{source_group_id}/permissions/{group_id}/{item_id} groups updatePermissions
//
//		---
//		summary: Update permissions
//		description: Let a manager of a group give/withdraw permissions on an item to a group.
//
//	  * The user giving the access must be a manager (with `can_grant_group_access` permission)
//	    of `{source_group_id}` which should be an ancestor of the `{group_id}`.
//
//	  * The user giving the access must have `permissions_generated.can_grant_view` >= given `can_view`
//	    for the item.
//
//	  * The group must already have access to one of the parents of the item or the item itself. If it does not,
//	    the item must be a root activity/skill for an ancestor of the group.
//		parameters:
//			- name: group_id
//				in: path
//				required: true
//				type: integer
//			- name: source_group_id
//				in: path
//				required: true
//				type: integer
//			- name: item_id
//				in: path
//				required: true
//				type: integer
//			- name: access rights information
//				in: body
//				required: true
//				schema:
//					"$ref": "#/definitions/updatePermissionsInput"
//		responses:
//			"200":
//				"$ref": "#/responses/updatedResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) updatePermissions(w http.ResponseWriter, r *http.Request) service.APIError {
	sourceGroupID, err := service.ResolveURLQueryPathInt64Field(r, "source_group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	itemID, err := service.ResolveURLQueryPathInt64Field(r, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(r)
	apiErr := service.NoError

	err = srv.GetStore(r).InTransaction(func(s *database.DataStore) error {
		apiErr = checkIfUserIsManagerAllowedToGrantPermissionsOnItem(s, user, sourceGroupID, groupID, itemID)
		if apiErr != service.NoError {
			return apiErr.Error
		}

		var currentPermissions userPermissions
		err = s.PermissionsGranted().Where("group_id = ?", groupID).Where("item_id = ?", itemID).
			Where("source_group_id = ?", sourceGroupID).Where("origin = 'group_membership'").
			WithWriteLock().
			Select(`
				IFNULL(MAX(can_view_value), 1) AS can_view_value,
				IFNULL(MAX(can_grant_view_value), 1) AS can_grant_view_value,
				IFNULL(MAX(can_watch_value), 1) AS can_watch_value,
				IFNULL(MAX(can_edit_value), 1) AS can_edit_value,
				IFNULL(MAX(can_make_session_official), 0) AS can_make_session_official,
				IFNULL(MAX(can_enter_from), '9999-12-31 23:59:59') AS can_enter_from,
				IFNULL(MAX(can_enter_until), '9999-12-31 23:59:59') AS can_enter_until,
				IFNULL(MAX(is_owner), 0) AS is_owner`).
			Scan(&currentPermissions).Error()

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			service.MustNotBeError(err)
		}

		var managerPermissions managerGeneratedPermissions
		err = s.Permissions().Select(`
			MAX(can_grant_view_generated_value) AS can_grant_view_generated_value,
			MAX(can_watch_generated_value) AS can_watch_generated_value,
			MAX(can_edit_generated_value) AS can_edit_generated_value,
			MAX(is_owner_generated) AS is_owner_generated`).
			Joins("JOIN groups_ancestors_active AS ancestors ON ancestors.ancestor_group_id = permissions.group_id").
			Where("ancestors.child_group_id = ?", user.GroupID).
			Where("permissions.item_id = ?", itemID).
			Scan(&managerPermissions).Error()
		service.MustNotBeError(err)

		var dataMap map[string]interface{}
		var modified bool
		dataMap, modified, apiErr = parsePermissionsInputData(s, &managerPermissions, &currentPermissions, r)
		if apiErr != service.NoError {
			return apiErr.Error // rollback
		}

		if modified {
			correctPermissionsDataMap(s, dataMap, &currentPermissions)
			savePermissionsIntoDB(groupID, itemID, sourceGroupID, dataMap, s)
		}
		return nil
	})

	if apiErr != service.NoError {
		return apiErr
	}

	service.MustNotBeError(err)

	response := service.Response{Success: true, Message: "updated"}
	render.Respond(w, r, &response)

	return service.NoError
}

func registerOptionalValidator(data *formdata.FormData, tag string, validatorFunc func(fl validator.FieldLevel) bool) {
	data.RegisterValidation(tag, data.ValidatorSkippingUnsetFields(validatorFunc))
	data.RegisterTranslation(tag, "the value is not permitted")
}

func parsePermissionsInputData(s *database.DataStore, managerPermissions *managerGeneratedPermissions,
	currentPermissions *userPermissions, r *http.Request) (
	dataMap map[string]interface{}, modified bool, apiError service.APIError,
) {
	data := formdata.NewFormData(&updatePermissionsInput{})
	modifiedPtr := registerPermissionsValidators(s, managerPermissions, currentPermissions, data)

	err := data.ParseJSONRequestData(r)
	if err != nil {
		return nil, false, service.ErrInvalidRequest(err)
	}
	return data.ConstructMapForDB(), *modifiedPtr, service.NoError
}

func registerPermissionsValidators(
	s *database.DataStore, managerPermissions *managerGeneratedPermissions, currentPermissions *userPermissions,
	data *formdata.FormData,
) *bool {
	var modified bool
	registerIsOwnerValidator(data, managerPermissions, currentPermissions, &modified)
	registerCanViewValidator(data, managerPermissions, currentPermissions, &modified, s)
	registerCanGrantViewValidator(data, managerPermissions, currentPermissions, &modified, s)
	registerCanWatchValidator(data, managerPermissions, currentPermissions, &modified, s)
	registerCanEditValidator(data, managerPermissions, currentPermissions, &modified, s)
	registerCanMakeSessionOfficialValidator(data, managerPermissions, currentPermissions, &modified, s)
	registerCanEnterFromValidator(data, managerPermissions, currentPermissions, &modified, s)
	registerCanEnterUntilValidator(data, managerPermissions, currentPermissions, &modified, s)
	return &modified
}

func registerIsOwnerValidator(data *formdata.FormData, managerPermissions *managerGeneratedPermissions,
	currentPermissions *userPermissions, modified *bool,
) {
	registerOptionalValidator(data, "is_owner", func(fl validator.FieldLevel) bool {
		newValue := fl.Field().Bool()
		if newValue && newValue != currentPermissions.IsOwner && !managerPermissions.IsOwnerGenerated {
			return false
		}
		if currentPermissions.IsOwner != newValue {
			*modified = true
		}
		return true
	})
}

func registerCanViewValidator(data *formdata.FormData, managerPermissions *managerGeneratedPermissions, currentPermissions *userPermissions,
	modified *bool, s *database.DataStore,
) {
	registerOptionalValidator(data, "can_view", func(fl validator.FieldLevel) bool {
		newValue := fl.Field().String()
		if !checkIfPossibleToModifyCanView(newValue, currentPermissions, managerPermissions, s) {
			return false
		}
		newValueIndex := s.PermissionsGranted().ViewIndexByName(newValue)
		if currentPermissions.CanViewValue != newValueIndex {
			currentPermissions.CanViewValue = newValueIndex
			*modified = true
		}
		return true
	})
}

func registerCanGrantViewValidator(
	data *formdata.FormData, managerPermissions *managerGeneratedPermissions, currentPermissions *userPermissions,
	modified *bool, s *database.DataStore,
) {
	registerOptionalValidator(data, "can_grant_view", func(fl validator.FieldLevel) bool {
		newValue := fl.Field().String()
		if !checkIfPossibleToModifyCanGrantView(newValue, currentPermissions, managerPermissions, s) {
			return false
		}
		newValueIndex := s.PermissionsGranted().GrantViewIndexByName(newValue)
		if currentPermissions.CanGrantViewValue != newValueIndex {
			currentPermissions.CanGrantViewValue = newValueIndex
			*modified = true
		}
		return true
	})
}

func registerCanWatchValidator(
	data *formdata.FormData, managerPermissions *managerGeneratedPermissions, currentPermissions *userPermissions,
	modified *bool, s *database.DataStore,
) {
	registerOptionalValidator(data, "can_watch", func(fl validator.FieldLevel) bool {
		newValue := fl.Field().String()
		if !checkIfPossibleToModifyCanWatch(newValue, currentPermissions, managerPermissions, s) {
			return false
		}
		newValueIndex := s.PermissionsGranted().WatchIndexByName(newValue)
		if currentPermissions.CanWatchValue != newValueIndex {
			currentPermissions.CanWatchValue = newValueIndex
			*modified = true
		}
		return true
	})
}

func registerCanEditValidator(
	data *formdata.FormData, managerPermissions *managerGeneratedPermissions, currentPermissions *userPermissions,
	modified *bool, s *database.DataStore,
) {
	registerOptionalValidator(data, "can_edit", func(fl validator.FieldLevel) bool {
		newValue := fl.Field().String()
		if !checkIfPossibleToModifyCanEdit(newValue, currentPermissions, managerPermissions, s) {
			return false
		}
		newValueIndex := s.PermissionsGranted().EditIndexByName(newValue)
		if currentPermissions.CanEditValue != newValueIndex {
			currentPermissions.CanEditValue = newValueIndex
			*modified = true
		}
		return true
	})
}

func registerCanMakeSessionOfficialValidator(data *formdata.FormData, managerPermissions *managerGeneratedPermissions,
	currentPermissions *userPermissions, modified *bool, s *database.DataStore,
) {
	registerOptionalValidator(data, "can_make_session_official", func(fl validator.FieldLevel) bool {
		newValue := fl.Field().Bool()
		if !checkIfPossibleToModifyCanMakeSessionOfficial(newValue, currentPermissions, managerPermissions, s) {
			return false
		}
		if currentPermissions.CanMakeSessionOfficial != newValue {
			currentPermissions.CanMakeSessionOfficial = newValue
			*modified = true
		}
		return true
	})
}

func registerCanEnterFromValidator(data *formdata.FormData, managerPermissions *managerGeneratedPermissions,
	currentPermissions *userPermissions, modified *bool, s *database.DataStore,
) {
	registerOptionalValidator(data, "can_enter_from", func(fl validator.FieldLevel) bool {
		newValue := fl.Field().Interface().(time.Time)
		if !checkIfPossibleToModifyCanEnterFrom(newValue, currentPermissions, managerPermissions, s) {
			return false
		}
		if !newValue.Equal(time.Time(currentPermissions.CanEnterFrom)) {
			*modified = true
		}
		return true
	})
}

func registerCanEnterUntilValidator(
	data *formdata.FormData, managerPermissions *managerGeneratedPermissions, currentPermissions *userPermissions,
	modified *bool, s *database.DataStore,
) {
	registerOptionalValidator(data, "can_enter_until", func(fl validator.FieldLevel) bool {
		newValue := fl.Field().Interface().(time.Time)
		if !checkIfPossibleToModifyCanEnterUntil(newValue, currentPermissions, managerPermissions, s) {
			return false
		}
		if !newValue.Equal(time.Time(currentPermissions.CanEnterUntil)) {
			*modified = true
		}
		return true
	})
}

const (
	none              = "none"
	enter             = "enter"
	info              = "info"
	content           = "content"
	solution          = "solution"
	solutionWithGrant = "solution_with_grant"
	answer            = "answer" //nolint:unused
	answerWithGrant   = "answer_with_grant"
	allWithGrant      = "all_with_grant"
)

func checkIfUserIsManagerAllowedToGrantPermissionsOnItem(s *database.DataStore, user *database.User,
	sourceGroupID, groupID, itemID int64,
) service.APIError {
	apiError := checkIfUserIsManagerAllowedToGrantPermissionsToGroupID(s, user, sourceGroupID, groupID)
	if apiError != service.NoError {
		return apiError
	}

	return checkIfItemOrOneOfItsParentsIsVisibleToGroupOrItemIsRoot(s, groupID, itemID)
}

func checkIfUserIsManagerAllowedToGrantPermissionsToGroupID(
	s *database.DataStore, user *database.User, sourceGroupID, groupID int64,
) service.APIError {
	// the authorized user should be a manager of the sourceGroupID with `can_grant_group_access' permission and
	// the 'sourceGroupID' should be an ancestor of 'groupID'
	found, err := s.Groups().ManagedBy(user).Where("groups.id = ?", sourceGroupID).
		Where("groups.type != 'User'").
		Joins(`
				JOIN groups_ancestors_active AS descendants
					ON descendants.ancestor_group_id = groups.id AND descendants.child_group_id = ?`, groupID).
		Where("group_managers.can_grant_group_access").
		HasRows()
	service.MustNotBeError(err)
	if !found {
		return service.InsufficientAccessRightsError
	}
	return service.NoError
}

func checkIfItemOrOneOfItsParentsIsVisibleToGroupOrItemIsRoot(s *database.DataStore, groupID, itemID int64) service.APIError {
	// at least one of the item's parents should be visible to the group
	found, err := s.Permissions().MatchingGroupAncestors(groupID).
		WherePermissionIsAtLeast("view", info).
		Joins("JOIN items_items ON items_items.parent_item_id = permissions.item_id").
		Where("items_items.child_item_id = ?", itemID).
		HasRows()
	service.MustNotBeError(err)
	if !found {
		// if not, the item itself should be visible to the group
		found, err = s.Permissions().MatchingGroupAncestors(groupID).WherePermissionIsAtLeast("view", info).
			Where("item_id = ?", itemID).HasRows()
		service.MustNotBeError(err)
		if !found {
			// if not, the item should be a root item for one of the group's ancestors
			found, err = s.Groups().
				Joins("JOIN groups_ancestors_active ON groups_ancestors_active.ancestor_group_id = groups.id").
				Where("groups_ancestors_active.child_group_id = ?", groupID).
				Where("root_activity_id = ? OR root_skill_id = ?", itemID, itemID).HasRows()
			service.MustNotBeError(err)
			if !found {
				return service.InsufficientAccessRightsError
			}
		}
	}
	return service.NoError
}

func checkIfPossibleToModifyCanView(viewPermissionToSet string, currentPermissions *userPermissions,
	managerPermissions *managerGeneratedPermissions, store *database.DataStore,
) bool {
	permissionGrantedStore := store.PermissionsGranted()
	if permissionGrantedStore.ViewIndexByName(viewPermissionToSet) <= currentPermissions.CanViewValue {
		return true
	}

	requiredGrantViewPermission := viewPermissionToSet
	if requiredGrantViewPermission == info { // no "info" in can_grant_view
		requiredGrantViewPermission = enter
	}

	// permissions_generated.can_grant_view_generated should be >= data["can_view"] for the manager
	return permissionGrantedStore.GrantViewIndexByName(requiredGrantViewPermission) <= managerPermissions.CanGrantViewGeneratedValue
}

func checkIfPossibleToModifyCanGrantView(grantViewPermissionToSet string, currentPermissions *userPermissions,
	managerPermissions *managerGeneratedPermissions, store *database.DataStore,
) bool {
	permissionGrantedStore := store.PermissionsGranted()
	if permissionGrantedStore.GrantViewIndexByName(grantViewPermissionToSet) <= currentPermissions.CanGrantViewValue {
		return true
	}

	requiredUserViewPermission := grantViewPermissionToSet
	switch requiredUserViewPermission {
	case enter: // no "enter" in can_view
		requiredUserViewPermission = info
	case solutionWithGrant:
		requiredUserViewPermission = solution
	}

	// permissions_granted.can_view should be >= requiredUserViewPermission for the user
	if currentPermissions.CanViewValue < permissionGrantedStore.ViewIndexByName(requiredUserViewPermission) {
		return false
	}

	if grantViewPermissionToSet == solutionWithGrant && !managerPermissions.IsOwnerGenerated {
		return false
	}

	return managerPermissions.CanGrantViewGeneratedValue == permissionGrantedStore.GrantViewIndexByName(solutionWithGrant)
}

func checkIfPossibleToModifyCanWatch(watchPermissionToSet string, currentPermissions *userPermissions,
	managerPermissions *managerGeneratedPermissions, store *database.DataStore,
) bool {
	permissionGrantedStore := store.PermissionsGranted()
	if permissionGrantedStore.WatchIndexByName(watchPermissionToSet) <= currentPermissions.CanWatchValue {
		return true
	}

	if currentPermissions.CanViewValue < permissionGrantedStore.ViewIndexByName(content) {
		return false
	}

	if watchPermissionToSet == answerWithGrant && !managerPermissions.IsOwnerGenerated {
		return false
	}

	return managerPermissions.CanWatchGeneratedValue == permissionGrantedStore.WatchIndexByName(answerWithGrant)
}

func checkIfPossibleToModifyCanEdit(editPermissionToSet string, currentPermissions *userPermissions,
	managerPermissions *managerGeneratedPermissions, store *database.DataStore,
) bool {
	permissionGrantedStore := store.PermissionsGranted()
	if permissionGrantedStore.EditIndexByName(editPermissionToSet) <= currentPermissions.CanEditValue {
		return true
	}

	if currentPermissions.CanViewValue < permissionGrantedStore.ViewIndexByName(content) {
		return false
	}

	if editPermissionToSet == allWithGrant && !managerPermissions.IsOwnerGenerated {
		return false
	}

	return managerPermissions.CanEditGeneratedValue == permissionGrantedStore.EditIndexByName(allWithGrant)
}

func checkIfPossibleToModifyCanMakeSessionOfficial(canMakeSessionOfficalToSet bool, currentPermissions *userPermissions,
	managerPermissions *managerGeneratedPermissions, store *database.DataStore,
) bool {
	if !canMakeSessionOfficalToSet || canMakeSessionOfficalToSet == currentPermissions.CanMakeSessionOfficial {
		return true
	}

	if currentPermissions.CanViewValue < store.PermissionsGranted().ViewIndexByName(info) {
		return false
	}

	return managerPermissions.IsOwnerGenerated
}

func checkIfPossibleToModifyCanEnterFrom(canEnterFromToSet time.Time, currentPermissions *userPermissions,
	managerPermissions *managerGeneratedPermissions, store *database.DataStore,
) bool {
	if !time.Time(currentPermissions.CanEnterFrom).After(canEnterFromToSet) {
		return true
	}

	return store.PermissionsGranted().GrantViewIndexByName(enter) <= managerPermissions.CanGrantViewGeneratedValue
}

func checkIfPossibleToModifyCanEnterUntil(canEnterUntilToSet time.Time, currentPermissions *userPermissions,
	managerPermissions *managerGeneratedPermissions, store *database.DataStore,
) bool {
	if !time.Time(currentPermissions.CanEnterUntil).Before(canEnterUntilToSet) {
		return true
	}

	return store.PermissionsGranted().GrantViewIndexByName(enter) <= managerPermissions.CanGrantViewGeneratedValue
}

func correctPermissionsDataMap(store *database.DataStore, dataMap map[string]interface{}, currentPermissions *userPermissions) {
	permissionGrantedStore := store.PermissionsGranted()

	currentCanViewValue := currentPermissions.CanViewValue
	if currentPermissions.CanGrantViewValue == permissionGrantedStore.GrantViewIndexByName(solutionWithGrant) &&
		currentCanViewValue != permissionGrantedStore.ViewIndexByName(solution) ||
		currentPermissions.CanGrantViewValue < permissionGrantedStore.GrantViewIndexByName(solutionWithGrant) &&
			currentCanViewValue < currentPermissions.CanGrantViewValue {
		dataMap["can_grant_view"] = permissionGrantedStore.GrantViewNameByIndex(currentCanViewValue)
	}
	if currentCanViewValue < permissionGrantedStore.ViewIndexByName(content) {
		if permissionGrantedStore.WatchIndexByName(none) < currentPermissions.CanWatchValue {
			dataMap["can_watch"] = none
		}
		if permissionGrantedStore.EditIndexByName(none) < currentPermissions.CanEditValue {
			dataMap["can_edit"] = none
		}
		if currentPermissions.CanMakeSessionOfficial && currentCanViewValue < permissionGrantedStore.ViewIndexByName(info) {
			dataMap["can_make_session_official"] = false
		}
	}
}

func savePermissionsIntoDB(groupID, itemID, sourceGroupID int64, dbMap map[string]interface{}, s *database.DataStore) {
	dbMap["latest_update_at"] = database.Now()

	columnsToUpdate := make([]string, 0, len(dbMap))
	for key := range dbMap {
		columnsToUpdate = append(columnsToUpdate, key)
	}

	dbMap["group_id"] = groupID
	dbMap["item_id"] = itemID
	dbMap["source_group_id"] = sourceGroupID
	dbMap["origin"] = "group_membership"

	permissionGrantedStore := s.PermissionsGranted()
	service.MustNotBeError(permissionGrantedStore.InsertOrUpdateMap(dbMap, columnsToUpdate))
	service.MustNotBeError(permissionGrantedStore.After())
	if dbMap["can_view"] != nil && dbMap["can_view"] != none || dbMap["is_owner"] != nil && dbMap["is_owner"].(bool) {
		// permissionGrantedStore.After() implicitly (via triggers) marks some attempts as to_be_propagated
		// when an item becomes visible, so we should propagate attempts here
		s.ScheduleResultsPropagation()
	}
}
