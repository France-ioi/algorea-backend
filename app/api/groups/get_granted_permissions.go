package groups

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
	"github.com/France-ioi/AlgoreaBackend/app/structures"
)

type grantedPermissionsViewResultRowGroup struct {
	// required: true
	ID int64 `json:"id,string"`
	// required: true
	Name string `json:"name"`
}

type grantedPermissionsViewResultPermissions struct {
	structures.ItemPermissions
	// required: true
	CanMakeSessionOfficial bool `json:"can_make_session_official"`
	// required: true
	CanEnterFrom database.Time `json:"can_enter_from"`
	// required: true
	CanEnterUntil database.Time `json:"can_enter_until"`
}

// swagger:model grantedPermissionsViewResultRow
type grantedPermissionsViewResultRow struct {
	// required: true
	SourceGroup grantedPermissionsViewResultRowGroup `json:"source_group" gorm:"embedded;embeddedPrefix:source_group__"`
	// required: true
	Group grantedPermissionsViewResultRowGroup `json:"group" gorm:"embedded;embeddedPrefix:group__"`
	// required: true
	Item struct {
		// required: true
		ID int64 `json:"id,string"`
		// Nullable
		// required: true
		Title *string `json:"title"`
		// required: true
		// enum: Chapter,Task,Skill
		Type *string `json:"type"`
		// required: true
		LanguageTag string `json:"language_tag"`
		// required: true
		RequiresExplicitEntry bool `json:"requires_explicit_entry"`
	} `json:"item" gorm:"embedded;embeddedPrefix:item__"`
	// required: true
	Permissions grantedPermissionsViewResultPermissions `json:"permissions" gorm:"embedded;embeddedPrefix:permissions__"`
}

// swagger:operation GET /groups/{group_id}/granted_permissions groups grantedPermissionsView
//
//		---
//		summary: View granted permissions
//		description:
//	  List all permissions granted to a group and its ancestors or to its descendants.
//	  Only permissions granted on items for which the current user has
//	  `can_grant_view` > 'none' or `can_watch` = 'answer_with_grant' or `can_edit` = 'all_with_grant' are displayed.
//
//
//	  When `{descendants}` is 0, source groups of permissions are ancestors of the `group_id` group (including the group itself)
//	  managed by the current user with `can_grant_group_access` permission.
//
//	  When `{descendants}` is 1, source groups of permissions are ancestors of the `group_id` group (including the group itself)
//	  or descendants of the `group_id` group managed by the current user with `can_grant_group_access` permission.
//
//	  * The current user must be a manager (with `can_grant_group_access` permission) of `{group_id}`.
//		parameters:
//			- name: group_id
//				in: path
//				required: true
//				type: integer
//			- name: descendants
//				description: If equal to 1, the results are permissions granted to the group's descendants (not including the group itself),
//	               otherwise the results are permissions granted to the group's ancestors (including the group itself).
//				in: query
//				type: integer
//				enum: [0,1]
//				default: 0
//			- name: sort
//				in: query
//				default: [item.title,source_group.name,group.name]
//				type: array
//				items:
//					type: string
//					enum: [source_group.name,-source_group.name,group.name,-group.name,
//	           item.title,-item.title,source_group.id,-source_group.id,group.id,-group.id,item.id,-item.id]
//			- name: from.source_group.id
//				description: Start the page from permissions next to the permissions with `source_group_id`=`{from.source_group.id}`
//	               (`{from.item.id}` and `{from.group.id}` should be given too when `{from.source_group.id}` is given)
//				in: query
//				type: integer
//			- name: from.group.id
//				description: Start the page from permissions next to the permissions with `group_id`=`{from.group.id}`
//	               (`{from.item.id}` and `{from.source_group.id}` should be given too when `{from.group.id}` is given)
//				in: query
//				type: integer
//			- name: from.item.id
//				description: Start the page from permissions next to the permissions with `item_id`=`{from.item.id}`
//	               (`{from.group.id}` and `{from.source_group.id}` should be given too when `{from.item.id}` is given)
//				in: query
//				type: integer
//			- name: limit
//				description: Display the first N permissions
//				in: query
//				type: integer
//				maximum: 1000
//				default: 500
//		responses:
//			"200":
//				description: OK. Granted permissions
//				schema:
//					type: array
//					items:
//						"$ref": "#/definitions/grantedPermissionsViewResultRow"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getGrantedPermissions(w http.ResponseWriter, r *http.Request) service.APIError {
	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	var forDescendants bool
	if len(r.URL.Query()["descendants"]) > 0 {
		forDescendants, err = service.ResolveURLQueryGetBoolField(r, "descendants")
		if err != nil {
			return service.ErrInvalidRequest(err)
		}
	}

	user := srv.GetUser(r)
	store := srv.GetStore(r)

	found, err := store.Groups().ManagedBy(user).Where("groups.id = ?", groupID).
		Where("groups.type != 'User'").Where("can_grant_group_access").HasRows()
	service.MustNotBeError(err)
	if !found {
		return service.InsufficientAccessRightsError
	}

	itemsQuery := store.Permissions().MatchingUserAncestors(user).
		Where("? OR can_watch_generated = 'answer_with_grant' OR can_edit_generated = 'all_with_grant'",
			store.PermissionsGranted().PermissionIsAtLeastSQLExpr("grant_view", "enter")).
		Select("DISTINCT item_id AS id")

	// Used to be a subquery, but it failed with MySQL-8.0.26 due to obscure bugs introduced in this version.
	// It works when doing the query first and using the result in the second query.
	// See commit 5a25fbded8134c93c72dc853f72071943a1bd24c
	managedGroupsWithCanGrantGroupAccessIds := user.GetManagedGroupsWithCanGrantGroupAccessIds(store)

	var sourceGroupsQuery, groupsQuery *database.DB
	if forDescendants {
		ancestorsAndDescendantsQuery := store.ActiveGroupAncestors().
			Select("ancestor_group_id AS id").
			Where("child_group_id = ?", groupID).
			Union(
				store.ActiveGroupAncestors().
					Select("child_group_id AS id").
					Where("ancestor_group_id = ?", groupID).SubQuery())

		sourceGroupsQuery = store.Groups().
			Where("groups.type != 'User'").
			Where("id IN (?)", managedGroupsWithCanGrantGroupAccessIds).
			Where("id IN ?", ancestorsAndDescendantsQuery.SubQuery()).
			Select("groups.id, groups.name")

		groupsQuery = store.Groups().
			Joins("JOIN groups_ancestors_active ON groups_ancestors_active.child_group_id = groups.id").
			Where("ancestor_group_id = ?", groupID).
			Where("NOT is_self").
			Select("groups.id, groups.name")
	} else {
		sourceGroupsQuery = store.ActiveGroupAncestors().
			Where("child_group_id = ?", groupID).
			Where("ancestor_group_id IN (?)", managedGroupsWithCanGrantGroupAccessIds).
			Joins("JOIN `groups` ON groups.id = ancestor_group_id").
			Where("groups.type != 'User'").
			Select("groups.id, groups.name")

		groupsQuery = store.Groups().
			Joins("JOIN groups_ancestors_active ON groups_ancestors_active.ancestor_group_id = groups.id").
			Where("child_group_id = ?", groupID).
			Select("groups.id, groups.name")
	}

	var permissions []grantedPermissionsViewResultRow
	query := store.PermissionsGranted().
		Joins("JOIN ? AS source_group ON source_group.id = source_group_id", sourceGroupsQuery.SubQuery()).
		Joins("JOIN ? AS target_group ON target_group.id = group_id", groupsQuery.SubQuery()).
		Joins("JOIN items ON items.id = item_id").
		Where("items.id IN (?)", itemsQuery.SubQuery()).
		Where("origin = 'group_membership'").
		JoinsUserAndDefaultItemStrings(user).
		Where(`
			can_view != 'none' OR can_grant_view != 'none' OR can_watch != 'none' OR can_edit != 'none' OR
			can_make_session_official OR is_owner OR
			can_enter_from != '9999-12-31 23:59:59' OR can_enter_until != '9999-12-31 23:59:59'`).
		Select(`
			target_group.id AS group__id, target_group.name AS group__name,
			source_group.id AS source_group__id, source_group.name AS source_group__name,
			items.id AS item__id,
			IF(user_strings.language_tag IS NULL, default_strings.title, user_strings.title) AS item__title,
			COALESCE(user_strings.language_tag, default_strings.language_tag) AS item__language_tag,
			items.requires_explicit_entry AS item__requires_explicit_entry,
			items.type AS item__type,
			can_view AS permissions__can_view, can_grant_view AS permissions__can_grant_view,
			can_watch AS permissions__can_watch, can_edit AS permissions__can_edit,
			can_make_session_official AS permissions__can_make_session_official,
			is_owner AS permissions__is_owner, can_enter_from AS permissions__can_enter_from,
			can_enter_until AS permissions__can_enter_until`)

	query = service.NewQueryLimiter().Apply(r, query)
	query, apiError := service.ApplySortingAndPaging(
		r, query,
		&service.SortingAndPagingParameters{
			Fields: service.SortingAndPagingFields{
				"source_group.name": {ColumnName: "source_group.name"},
				"group.name":        {ColumnName: "target_group.name"},
				"item.title": {
					ColumnName: "IF(user_strings.language_tag IS NULL, default_strings.title, user_strings.title)",
					Nullable:   true,
				},
				"source_group.id": {ColumnName: "permissions_granted.source_group_id"},
				"group.id":        {ColumnName: "permissions_granted.group_id"},
				"item.id":         {ColumnName: "permissions_granted.item_id"},
			},
			DefaultRules: "item.title,source_group.name,group.name",
			TieBreakers: service.SortingAndPagingTieBreakers{
				"source_group.id": service.FieldTypeInt64,
				"group.id":        service.FieldTypeInt64,
				"item.id":         service.FieldTypeInt64,
			},
		})
	if apiError != service.NoError {
		return apiError
	}

	service.MustNotBeError(query.Scan(&permissions).Error())
	render.Respond(w, r, permissions)
	return service.NoError
}
