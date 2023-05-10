package items

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/France-ioi/validator"
	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/formdata"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// Item represents input fields that are common to itemCreate & itemUpdate.
type Item struct {
	// Nullable
	URL *string `json:"url"`
	// Nullable
	Options *string `json:"options" validate:"null|options"`
	// default: false
	EntryFrozenTeams bool `json:"entry_frozen_teams"`
	// default: false
	NoScore bool `json:"no_score"`
	// Identifier to reference the task.
	// Unique
	// Nullable
	TextID                 *string `json:"text_id"`
	DisplayDetailsInParent bool    `json:"display_details_in_parent"`
	ReadOnly               bool    `json:"read_only"`
	// enum: List,Grid
	ChildrenLayout string `json:"children_layout"`
	// enum: forceYes,forceNo,default
	FullScreen   string `json:"full_screen" validate:"oneof=forceYes forceNo default"`
	HintsAllowed bool   `json:"hints_allowed"`
	FixedRanks   bool   `json:"fixed_ranks"`

	// enum: None,All,AllButOne,Categories,One,Manual
	// default: All
	ValidationType string `json:"validation_type" validate:"oneof=None All AllButOne Categories One Manual"`

	// enum: All,Half,One,None
	// default: None
	EntryMinAdmittedMembersRatio string    `json:"entry_min_admitted_members_ratio" validate:"oneof=All Half One None"`
	EnteringTimeMin              time.Time `json:"entering_time_min"`
	EnteringTimeMax              time.Time `json:"entering_time_max"`
	EntryMaxTeamSize             int32     `json:"entry_max_team_size"`
	TitleBarVisible              bool      `json:"title_bar_visible"`
	AllowsMultipleAttempts       bool      `json:"allows_multiple_attempts"`
	// enum: User,Team
	EntryParticipantType string `json:"entry_participant_type" validate:"oneof=User Team"`
	// Nullable
	//
	// MySQL time (max value is 838:59:59), cannot be set for skills
	// pattern: ^\d{1,3}:[0-5]?\d:[0-5]?\d$
	// example: 838:59:59
	Duration *string `json:"duration" validate:"omitempty,duration,cannot_be_set_for_skills,duration_requires_explicit_entry"`
	// should be true when the duration is not null, cannot be set for skill items
	RequiresExplicitEntry   bool `json:"requires_explicit_entry" validate:"cannot_be_set_for_skills,duration_requires_explicit_entry"`
	ShowUserInfos           bool `json:"show_user_infos"`
	UsesAPI                 bool `json:"uses_api"`
	PromptToJoinGroupByCode bool `json:"prompt_to_join_group_by_code"`
}

// ItemWithRequiredType represents common item fields plus the required type field.
type ItemWithRequiredType struct {
	Item `json:"item,squash"` //nolint:staticcheck SA5008: unknown JSON option "squash"
	// Can be equal to 'Skill' only if the parent's type is 'Skill'
	// required: true
	// enum: Chapter,Task,Skill
	Type string `json:"type" validate:"set,oneof=Chapter Task Skill,type_skill"`
}

// swagger:ignore
type newItemString struct {
	// required: true
	Title string `json:"title" validate:"set"`
	// Nullable
	ImageURL *string `json:"image_url"`
	// Nullable
	Subtitle *string `json:"subtitle"`
	// Nullable
	Description *string `json:"description"`
}

type itemParent struct {
	ItemID int64 `json:"item_id,string" validate:"parent_item_id,parent_item_type"`

	// enum: Undefined,Discovery,Application,Validation,Challenge
	// default: Undefined
	Category string `json:"category" validate:"oneof=Undefined Discovery Application Validation Challenge"`
	// default: 1
	ScoreWeight int8 `json:"score_weight"`
	// default: as_info
	// enum: none,as_info,as_content
	ContentViewPropagation string `json:"content_view_propagation" validate:"oneof=none as_info as_content"`
	// default: as_is
	// enum: use_content_view_propagation,as_content_with_descendants,as_is
	UpperViewLevelsPropagation string `json:"upper_view_levels_propagation" validate:"oneof=use_content_view_propagation as_content_with_descendants as_is"` //nolint:lll
	// default: true
	GrantViewPropagation bool `json:"grant_view_propagation"`
	// default: true
	WatchPropagation bool `json:"watch_propagation"`
	// default: true
	EditPropagation bool `json:"edit_propagation"`
}

// NewItemRequest is the expected input for new created item
// swagger:model itemCreateRequest
type NewItemRequest struct {
	// `default_language_tag` of the item
	// required: true
	LanguageTag   string                 `json:"language_tag" validate:"set,language_tag"`
	newItemString `json:"string,squash"` //nolint:staticcheck SA5008: unknown JSON option "squash"

	Parent          itemParent `json:"parent"`
	AsRootOfGroupID int64      `json:"as_root_of_group_id,string" validate:"as_root_of_group_id"`

	// Nullable fields are of pointer types
	ItemWithRequiredType `json:"item,squash"` //nolint:staticcheck SA5008: unknown JSON option "squash"

	Children []itemChild `json:"children" validate:"children,children_allowed,dive,child_type_non_skill"`
}

func (in *NewItemRequest) canCreateItemsRelationsWithoutCycles(store *database.DataStore) bool {
	if len(in.Children) == 0 {
		return true
	}
	ids := make([]int64, len(in.Children))
	for index := range in.Children {
		if in.Children[index].ItemID == in.Parent.ItemID {
			return false
		}
		ids[index] = in.Children[index].ItemID
	}
	var count int64
	service.MustNotBeError(store.ItemAncestors().WithWriteLock().
		Where("child_item_id = ?", in.Parent.ItemID).
		Where("ancestor_item_id IN (?)", ids).Count(&count).Error())
	return count == 0
}

// swagger:operation POST /items items itemCreate
//
//		---
//		summary: Create an item
//		description: >
//
//	  Creates an item with parameters from the input data with `items.default_language_tag` = `language_tag`.
//	  Also it
//
//	    * inserts a row into `items_strings` with given `language_tag`, `title`, `image_url`, `subtitle`, `description`,
//
//	    * gives full access to the item for the current user (creates a new `permissions_granted` row with: `item_id` = `items.id`,
//	      `group_id` = `group_id` of the current user, `source_group_id` = `users.group_id` of the current user, `origin` = 'self',
//	      `is_owner` = 1).
//
//	    * adds new relations for the parent and (optionally) children items into `items_items` and propagates `permissions_generated`.
//	      (The only allowed parent-child relations are skills-*, chapter-task, chapter-chapter.
//	      Otherwise the "bad request" error is returned.)
//
//	    * (if `requires_explicit_entry` is true) creates a participants group, links `participants_group_id` to it,
//	      and gives this group 'can_view:content' permission on the new item.
//
//	  The user should have
//
//	    * `can_view` >= 'content' and `can_edit` >= 'children' on the `parent.item_id`,
//	    * `can_view` != 'none' on the `children` items (if any),
//
//	  otherwise the "bad request" response is returned.
//
//
//	  The current user should not be temporary, otherwise the "forbidden" error response is returned.
//
//
//	  At least one of `parent` and `as_root_of_group_id` should be given, otherwise the "bad request" error response is returned.
//		parameters:
//			- in: body
//				name: data
//				required: true
//				description: The item to create
//				schema:
//					"$ref": "#/definitions/itemCreateRequest"
//		responses:
//			"201":
//				"$ref": "#/responses/createdWithIDResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) createItem(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)
	if user.IsTempUser {
		return service.InsufficientAccessRightsError
	}

	itemID, apiError, err := validateAndInsertItem(srv, r)
	if apiError != service.NoError {
		return apiError
	}
	service.MustNotBeError(err)

	// response
	response := struct {
		ItemID int64 `json:"id,string"`
	}{ItemID: itemID}
	service.MustNotBeError(render.Render(w, r, service.CreationSuccess(&response)))
	return service.NoError
}

func validateAndInsertItem(srv *Service, r *http.Request) (itemID int64, apiError service.APIError, err error) {
	user := srv.GetUser(r)
	store := srv.GetStore(r)
	err = store.InTransaction(func(store *database.DataStore) error {
		input := NewItemRequest{}
		formData := formdata.NewFormData(&input)

		var parentInfo parentItemInfo
		var childrenInfoMap map[int64]permissionAndType
		registerAddItemValidators(formData, store, user, &parentInfo, &childrenInfoMap)

		err = formData.ParseJSONRequestData(r)
		if err != nil {
			apiError = service.ErrInvalidRequest(err)
			return err // rollback
		}

		if !formData.IsSet("parent") && !formData.IsSet("as_root_of_group_id") {
			apiError = service.ErrInvalidRequest(errors.New("at least one of parent and as_root_of_group_id should be given"))
			return err // rollback
		}

		err = store.ItemItems().WithItemsRelationsLock(func(lockedStore *database.DataStore) error {
			if formData.IsSet("parent") && !input.canCreateItemsRelationsWithoutCycles(lockedStore) {
				apiError = service.ErrForbidden(errors.New("an item cannot become an ancestor of itself"))
				return apiError.Error
			}

			apiError = validateChildrenFieldsAndApplyDefaults(childrenInfoMap, input.Children, formData, nil, lockedStore)
			if apiError != service.NoError {
				return apiError.Error
			}

			// insertion
			itemID, apiError = srv.insertItem(lockedStore, user, formData, &input)
			if apiError != service.NoError {
				return apiError.Error
			}

			return nil
		})

		if apiError != service.NoError {
			return apiError.Error // rollback
		}
		service.MustNotBeError(err)

		setNewItemAsRootActivityOrSkill(store, formData, &input, itemID)
		return nil
	})

	return itemID, apiError, err
}

func setNewItemAsRootActivityOrSkill(store *database.DataStore, formData *formdata.FormData, input *NewItemRequest, itemID int64) {
	if formData.IsSet("as_root_of_group_id") {
		columnName := "root_activity_id"
		if input.Type == skill {
			columnName = "root_skill_id"
		}
		service.MustNotBeError(
			store.Groups().ByID(input.AsRootOfGroupID).UpdateColumn(columnName, itemID).Error())
	}
}

// constructParentItemIDValidator constructs a validator for the Parent.ItemID field.
// The validator checks that the user has rights to manage the parent item's children (can_view >= content & can_edit >= children).
func constructParentItemIDValidator(
	store *database.DataStore, user *database.User, parentInfo *parentItemInfo,
) validator.Func {
	return validator.Func(func(fl validator.FieldLevel) bool {
		err := store.Items().
			JoinsPermissionsForGroupToItemsWherePermissionAtLeast(user.GroupID, "view", "content").
			WherePermissionIsAtLeast("edit", "children").
			Where("items.id = ?", fl.Field().Interface().(int64)).Select("items.type").
			Limit(1).Take(&parentInfo).Error()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false
		}
		service.MustNotBeError(err)
		return true
	})
}

// constructAsRootOfGroupIDValidator constructs a validator for the AsRootOfGroupID field.
// The validator checks that the user has rights to manage the group (can_manage = memberships_and_group).
func constructAsRootOfGroupIDValidator(
	store *database.DataStore, user *database.User, formData *formdata.FormData,
) validator.Func {
	return validator.Func(func(fl validator.FieldLevel) bool {
		if !formData.IsSet("as_root_of_group_id") {
			return true
		}
		found, err := store.Groups().ManagedBy(user).Where("groups.id = ?", fl.Field().Interface().(int64)).
			Where("can_manage = 'memberships_and_group'").WithWriteLock().HasRows()
		service.MustNotBeError(err)
		return found
	})
}

// constructParentItemTypeValidator constructs a validator checking that the parent item is not a Task.
func constructParentItemTypeValidator(parentInfo *parentItemInfo) validator.Func {
	return validator.Func(func(fl validator.FieldLevel) bool {
		return parentInfo.Type != task
	})
}

// constructLanguageTagValidator constructs a validator for the LanguageTag field.
// The validator checks that the language exists.
func constructLanguageTagValidator(store *database.DataStore) validator.Func {
	return validator.Func(func(fl validator.FieldLevel) bool {
		found, err := store.Languages().ByTag(fl.Field().Interface().(string)).WithWriteLock().HasRows()
		service.MustNotBeError(err)
		return found
	})
}

// constructTypeSkillValidator constructs a validator for the Type field.
// The validator checks that the parent item's type is 'Skill' when the item's type is 'Skill'.
func constructTypeSkillValidator(parentInfo *parentItemInfo) validator.Func {
	return validator.Func(func(fl validator.FieldLevel) bool {
		if parentInfo.Type == "" || fl.Field().String() != skill {
			return true
		}
		return parentInfo.Type == skill
	})
}

// constructDurationRequiresExplicitEntryValidator constructs a validator for the RequiresExplicitEntry field.
// The validator checks that when the duration is given and is not null, the field is true.
func constructDurationRequiresExplicitEntryValidator() validator.Func {
	return validator.Func(func(fl validator.FieldLevel) bool {
		data := fl.Parent().Addr().Interface().(*Item)
		return data.RequiresExplicitEntry || !fl.Field().IsValid()
	})
}

// constructCannotBeSetForSkillsValidator constructs a validator checking that the field is not set for skill items.
func constructCannotBeSetForSkillsValidator() validator.Func {
	return validator.Func(func(fl validator.FieldLevel) bool {
		return fl.Field().IsZero() ||
			fl.Top().Elem().FieldByName("Type").String() != skill
	})
}

func constructItemOptionsValidator() validator.Func {
	return func(fl validator.FieldLevel) bool {
		return json.Valid([]byte(fl.Field().String()))
	}
}

// constructChildrenValidator constructs a validator for the Children field.
// The validator checks that there are no duplicates in the list and
// all the children items are visible to the user (can_view != 'none').
func constructChildrenValidator(store *database.DataStore, user *database.User,
	childrenInfoMap *map[int64]permissionAndType, oldPropagationLevelsMap *map[int64]*itemsRelationData, // nolint:gocritic
	itemID *int64,
) validator.Func {
	return validator.Func(func(fl validator.FieldLevel) bool {
		children := fl.Field().Interface().([]itemChild)

		if len(children) == 0 {
			return true
		}

		idsMap := make(map[int64]bool, len(children))
		ids := make([]int64, len(children))
		for index := range children {
			idsMap[children[index].ItemID] = true
			ids[index] = children[index].ItemID
		}
		if len(idsMap) != len(children) {
			return false
		}

		(*childrenInfoMap) = generateChildrenInfoMap(store, user, ids)

		if oldPropagationLevelsMap != nil || itemID != nil {
			(*oldPropagationLevelsMap) = generateOldPropagationLevelsMap(store, itemID)
		}

		for _, id := range ids {
			if _, ok := (*childrenInfoMap)[id]; !ok {
				// the child item candidate is not visible and there are no existing items_items
				if oldPropagationLevelsMap == nil {
					return false
				}
				// the child item candidate is not visible and it is not a child item
				if _, ok = (*oldPropagationLevelsMap)[id]; !ok {
					return false
				}
				// the child item candidate is not visible, but it is a child item already:
				// stub the permissions with nones
				(*childrenInfoMap)[id] = permissionAndType{
					Permission: &Permission{
						ItemID:                     id,
						CanViewGeneratedValue:      1,
						CanGrantViewGeneratedValue: 1,
						CanWatchGeneratedValue:     1,
						CanEditGeneratedValue:      1,
					},
				}
			}
		}
		return true
	})
}

func generateOldPropagationLevelsMap(store *database.DataStore, itemID *int64) map[int64]*itemsRelationData {
	var oldRelations []itemsRelationData
	service.MustNotBeError(store.ItemItems().ChildrenOf(*itemID).WithWriteLock().
		Select(`child_item_id AS item_id, category, score_weight,
				        content_view_propagation_value, upper_view_levels_propagation_value,
						    grant_view_propagation, watch_propagation, edit_propagation`).
		Scan(&oldRelations).Error())
	oldPropagationLevelsMap := make(map[int64]*itemsRelationData, len(oldRelations))
	for index := range oldRelations {
		oldPropagationLevelsMap[oldRelations[index].ItemID] = &oldRelations[index]
	}
	return oldPropagationLevelsMap
}

func generateChildrenInfoMap(store *database.DataStore, user *database.User, ids []int64) map[int64]permissionAndType {
	var childrenInfo []permissionAndType
	service.MustNotBeError(store.Items().
		JoinsPermissionsForGroupToItemsWherePermissionAtLeast(user.GroupID, "view", "info").
		Where("items.id IN (?)", ids).
		WithWriteLock().
		Select("permissions.*, items.type").
		Scan(&childrenInfo).Error())

	childrenInfoMap := make(map[int64]permissionAndType, len(childrenInfo))
	for index := range childrenInfo {
		(childrenInfoMap)[childrenInfo[index].ItemID] = childrenInfo[index]
	}
	return childrenInfoMap
}

// constructChildrenAllowedValidator constructs a validator checking that the new item can have children (is not a Task).
func constructChildrenAllowedValidator(
	defaultItemType string, childrenInfoMap *map[int64]permissionAndType,
) validator.Func { // nolint:gocritic
	return validator.Func(func(fl validator.FieldLevel) bool {
		if len(*childrenInfoMap) == 0 {
			return true
		}
		var itemType string
		itemTypeField := fl.Top().Elem().FieldByName("Type")
		if itemTypeField.IsValid() {
			itemType = itemTypeField.String()
		} else {
			itemType = defaultItemType
		}
		return itemType != task
	})
}

// constructChildTypeNonSkillValidator constructs a validator for the Children field that check
// if a child's type is not 'Skill' when the item's type is not 'Skill'.
func constructChildTypeNonSkillValidator(childrenInfoMap *map[int64]permissionAndType) validator.Func { // nolint:gocritic
	return validator.Func(func(fl validator.FieldLevel) bool {
		child := fl.Field().Interface().(itemChild)

		itemType := fl.Top().Elem().FieldByName("Type").String()
		if itemType == skill {
			return true
		}
		return (*childrenInfoMap)[child.ItemID].Type != skill
	})
}

type parentItemInfo struct {
	Type string
}

func registerAddItemValidators(formData *formdata.FormData, store *database.DataStore, user *database.User,
	parentInfo *parentItemInfo, childrenInfoMap *map[int64]permissionAndType,
) { // nolint:gocritic
	formData.RegisterValidation("parent_item_id",
		formData.ValidatorSkippingUnsetFields(constructParentItemIDValidator(store, user, parentInfo)))
	formData.RegisterTranslation("parent_item_id",
		"should exist and the user should be able to manage its children")
	formData.RegisterValidation("parent_item_type", constructParentItemTypeValidator(parentInfo))
	formData.RegisterTranslation("parent_item_type", "parent item cannot be Task")

	formData.RegisterValidation("as_root_of_group_id", constructAsRootOfGroupIDValidator(store, user, formData))
	formData.RegisterTranslation("as_root_of_group_id", "should exist and the user should be able to manage the group")

	registerLanguageTagValidator(formData, store)
	formData.RegisterValidation("type_skill", constructTypeSkillValidator(parentInfo))
	formData.RegisterTranslation("type_skill", "type can be equal to 'Skill' only if the parent item is a skill")
	formData.RegisterValidation("duration_requires_explicit_entry", constructDurationRequiresExplicitEntryValidator())
	formData.RegisterTranslation("duration_requires_explicit_entry", "requires_explicit_entry should be true when the duration is not null")
	formData.RegisterValidation("cannot_be_set_for_skills", constructCannotBeSetForSkillsValidator())
	formData.RegisterTranslation("cannot_be_set_for_skills", "cannot be set for skill items")
	registerChildrenValidator(formData, store, user, "", childrenInfoMap, nil, nil)
	formData.RegisterValidation("child_type_non_skill", constructChildTypeNonSkillValidator(childrenInfoMap))
	formData.RegisterTranslation("child_type_non_skill", "a skill cannot be a child of a non-skill item")
	formData.RegisterValidation("options", constructItemOptionsValidator())
	formData.RegisterTranslation("null|options", "options should be a valid JSON or null")
}

func registerLanguageTagValidator(formData *formdata.FormData, store *database.DataStore) {
	formData.RegisterValidation("language_tag", constructLanguageTagValidator(store))
	formData.RegisterTranslation("language_tag", "no such language")
}

func registerChildrenValidator(formData *formdata.FormData, store *database.DataStore, user *database.User,
	itemType string, childrenInfoMap *map[int64]permissionAndType, oldPropagationLevelsMap *map[int64]*itemsRelationData, // nolint:gocritic
	itemID *int64,
) {
	formData.RegisterValidation("children", constructChildrenValidator(store, user, childrenInfoMap, oldPropagationLevelsMap, itemID))
	formData.RegisterTranslation("children",
		"children IDs should be unique and each should be visible to the user")

	formData.RegisterValidation("children_allowed", constructChildrenAllowedValidator(itemType, childrenInfoMap))
	formData.RegisterTranslation("children_allowed", "a task cannot have children items")
}

func (srv *Service) insertItem(store *database.DataStore, user *database.User, formData *formdata.FormData,
	newItemRequest *NewItemRequest,
) (itemID int64, apiError service.APIError) {
	itemMap := formData.ConstructPartialMapForDB("ItemWithRequiredType")
	stringMap := formData.ConstructPartialMapForDB("newItemString")

	err := store.WithForeignKeyChecksDisabled(func(fkStore *database.DataStore) error {
		return fkStore.RetryOnDuplicatePrimaryKeyError(func(s *database.DataStore) error {
			itemID = s.NewID()

			itemMap["id"] = itemID
			itemMap["default_language_tag"] = newItemRequest.LanguageTag
			return s.Items().InsertMap(itemMap)
		})
	})
	if err != nil && database.IsDuplicateEntryError(err) {
		return 0, service.ErrForbidden(formdata.FieldErrors{"text_id": []string{
			"text_id must be unique",
		}})
	}
	service.MustNotBeError(err)

	if itemMap["requires_explicit_entry"] == true {
		participantsGroupID := createContestParticipantsGroup(store, itemID)
		service.MustNotBeError(store.Items().ByID(itemID).
			UpdateColumn("participants_group_id", participantsGroupID).Error())
	}

	service.MustNotBeError(store.PermissionsGranted().InsertMap(
		map[string]interface{}{
			"item_id":         itemID,
			"group_id":        user.GroupID,
			"source_group_id": user.GroupID,
			"origin":          "self",
			"is_owner":        true,
		}))

	stringMap["item_id"] = itemID
	stringMap["language_tag"] = newItemRequest.LanguageTag
	service.MustNotBeError(store.ItemStrings().InsertMap(stringMap))

	setItemRequestDefaults(newItemRequest, formData)

	parentChildSpecLength := len(newItemRequest.Children)
	if formData.IsSet("parent") {
		parentChildSpecLength++
	}
	if parentChildSpecLength > 0 {
		parentChildSpec := make([]*insertItemItemsSpec, 0, parentChildSpecLength)
		if formData.IsSet("parent.item_id") {
			var order int32
			service.MustNotBeError(store.ItemItems().WithWriteLock().
				Where("parent_item_id = ?", newItemRequest.Parent.ItemID).
				PluckFirst("IFNULL(MAX(`child_order`), 0)+1", &order).Error())

			parentChildSpec = append(parentChildSpec,
				&insertItemItemsSpec{
					ParentItemID: newItemRequest.Parent.ItemID, ChildItemID: itemID,
					Order:    order,
					Category: newItemRequest.Parent.Category, ScoreWeight: newItemRequest.Parent.ScoreWeight,
					ContentViewPropagation: valueOrDefault(
						formData, "parent.content_view_propagation", newItemRequest.Parent.ContentViewPropagation, asInfo).(string),
					UpperViewLevelsPropagation: valueOrDefault(
						formData, "parent.upper_view_levels_propagation", newItemRequest.Parent.UpperViewLevelsPropagation, asIs).(string),
					GrantViewPropagation: valueOrDefault(
						formData, "parent.grant_view_propagation", newItemRequest.Parent.GrantViewPropagation, true).(bool),
					WatchPropagation: valueOrDefault(
						formData, "parent.watch_propagation", newItemRequest.Parent.WatchPropagation, true).(bool),
					EditPropagation: valueOrDefault(
						formData, "parent.edit_propagation", newItemRequest.Parent.EditPropagation, true).(bool),
				})
		}
		parentChildSpec = append(parentChildSpec,
			constructItemsItemsForChildren(newItemRequest.Children, itemID)...)
		insertItemItems(store, parentChildSpec)
	}
	service.MustNotBeError(store.ItemItems().After())

	return itemID, service.NoError
}

// setItemRequestDefaults sets the default values of a newItemRequest which are not set.
func setItemRequestDefaults(newItemRequest *NewItemRequest, formData *formdata.FormData) {
	if !formData.IsSet("parent.category") {
		newItemRequest.Parent.Category = undefined
	}
	if !formData.IsSet("parent.score_weight") {
		newItemRequest.Parent.ScoreWeight = 1
	}

	for index := range newItemRequest.Children {
		if !formData.IsSet(fmt.Sprintf("children[%d].category", index)) {
			newItemRequest.Children[index].Category = undefined
		}
		if !formData.IsSet(fmt.Sprintf("children[%d].score_weight", index)) {
			newItemRequest.Children[index].ScoreWeight = 1
		}
	}
}

func valueOrDefault(formData *formdata.FormData, fieldName string, value, defaultValue interface{}) interface{} {
	if formData.IsSet(fieldName) {
		return value
	}
	return defaultValue
}
