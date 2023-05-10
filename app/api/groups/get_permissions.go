package groups

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/service"
	"github.com/France-ioi/AlgoreaBackend/app/structures"
)

type permissionsStruct struct {
	structures.ItemPermissions
	// required: true
	CanMakeSessionOfficial bool `json:"can_make_session_official"`
}

// Permissions granted directly to the group via `origin` = 'group_membership' and `source_group_id` = `{source_group_id}`.
type grantedPermissionsStruct struct {
	permissionsStruct
	// required: true
	CanEnterFrom string `json:"can_enter_from"`
	// required: true
	CanEnterUntil string `json:"can_enter_until"`
}

type permissionsWithCanEnterFrom struct {
	permissionsStruct
	// The next time the group can enter the item (>= NOW())
	// required: true
	CanEnterFrom string `json:"can_enter_from"`
}

// Computed permissions for the group
// (respecting permissions of its ancestors).
type computedPermissions struct{ permissionsWithCanEnterFrom }

// Permissions granted to the group or its ancestors
// via `origin` = 'group_membership' excluding the row from `granted`.
type permissionsGrantedViaGroupMembership struct{ permissionsWithCanEnterFrom }

// Permissions granted to the group or its ancestors
// via `origin` = 'item_unlocking'.
type permissionsGrantedViaItemUnlocking struct{ permissionsWithCanEnterFrom }

// Permissions granted to the group or its ancestors
// via `origin` = 'self'.
type permissionsGrantedViaSelf struct{ permissionsWithCanEnterFrom }

// Permissions granted to the group or its ancestors
// via `origin` = 'other'.
type permissionsGrantedViaOther struct{ permissionsWithCanEnterFrom }

// swagger:model permissionsViewResponse
type permissionsViewResponse struct {
	// required: true
	Granted grantedPermissionsStruct `json:"granted"`
	// required: true
	Computed computedPermissions `json:"computed"`
	// required: true
	GrantedViaGroupMembership permissionsGrantedViaGroupMembership `json:"granted_via_group_membership"`
	// required: true
	GrantedViaItemUnlocking permissionsGrantedViaItemUnlocking `json:"granted_via_item_unlocking"`
	// required: true
	GrantedViaSelf permissionsGrantedViaSelf `json:"granted_via_self"`
	// required: true
	GrantedViaOther permissionsGrantedViaOther `json:"granted_via_other"`
}

// swagger:operation GET /groups/{source_group_id}/permissions/{group_id}/{item_id} groups permissionsView
//
//		---
//		summary: View permissions
//		description: Lets a manager of a group view permissions on an item for the group.
//
//	  * The current user must be a manager (with `can_grant_group_access` permission)
//	    of `{source_group_id}` which should be an ancestor of the `{group_id}`.
//
//	  * The current user must have `can_grant_view` > 'none' or
//	    `can_watch` = 'answer_with_grant' or `can_edit` = 'all_with_grant' on `{item_id}` on the item.
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
//		responses:
//			"200":
//				description: OK. Permissions for the group
//				schema:
//					"$ref": "#/definitions/permissionsViewResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getPermissions(w http.ResponseWriter, r *http.Request) service.APIError {
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
	store := srv.GetStore(r)

	apiErr := checkIfUserIsManagerAllowedToGrantPermissionsToGroupID(store, user, sourceGroupID, groupID)
	if apiErr != service.NoError {
		return apiErr
	}

	found, err := store.Permissions().MatchingUserAncestors(user).
		Where("? OR can_watch_generated = 'answer_with_grant' OR can_edit_generated = 'all_with_grant'",
			store.PermissionsGranted().PermissionIsAtLeastSQLExpr("grant_view", enter)).
		Where("item_id = ?", itemID).HasRows()
	service.MustNotBeError(err)
	if !found {
		return service.InsufficientAccessRightsError
	}

	var permissions []map[string]interface{}
	const canMakeSessionOfficialColumn = "IFNULL(MAX(can_make_session_official), 0) AS can_make_session_official"
	grantedPermissions := store.PermissionsGranted().
		Where("group_id = ?", groupID).
		Where("item_id = ?", itemID).
		Where("source_group_id = ?", sourceGroupID).
		Where("origin = 'group_membership'").
		Select(`
			IFNULL(MAX(can_view_value), 1) AS can_view_value,
			IFNULL(MAX(can_grant_view_value), 1) AS can_grant_view_value,
			IFNULL(MAX(can_watch_value), 1) AS can_watch_value,
			IFNULL(MAX(can_edit_value), 1) AS can_edit_value,
			IFNULL(MAX(can_enter_from), '9999-12-31 23:59:59') AS can_enter_from,
			IFNULL(MAX(can_enter_until), '9999-12-31 23:59:59') AS can_enter_until,
			IFNULL(MAX(is_owner), 0) AS is_owner, ` + canMakeSessionOfficialColumn)

	generatedPermissions := store.Permissions().
		Joins("JOIN groups_ancestors_active AS ancestors ON ancestors.ancestor_group_id = permissions.group_id").
		Where("ancestors.child_group_id = ?", groupID).
		Where("item_id = ?", itemID).
		Select(`
			IFNULL(MAX(can_view_generated_value), 1) AS can_view_generated_value,
			IFNULL(MAX(can_grant_view_generated_value), 1) AS can_grant_view_generated_value,
			IFNULL(MAX(can_watch_generated_value), 1) AS can_watch_generated_value,
			IFNULL(MAX(can_edit_generated_value), 1) AS can_edit_generated_value,
			IFNULL(MAX(is_owner_generated), 0) AS is_owner_generated`)

	ancestorPermissions := store.PermissionsGranted().
		Joins("JOIN groups_ancestors_active ON groups_ancestors_active.ancestor_group_id = permissions_granted.group_id").
		Where("groups_ancestors_active.child_group_id = ?", groupID).
		Where("item_id = ?", itemID)
	const canEnterFromColumn = `
		IFNULL(MIN(IF(
			NOW() BETWEEN can_enter_from AND can_enter_until,
			NOW(),
			IF(can_enter_from BETWEEN NOW() AND can_enter_until, can_enter_from, '9999-12-31 23:59:59')
		)), '9999-12-31 23:59:59') AS can_enter_from`
	grantedPermissionsWithAncestors := ancestorPermissions.
		Select(`
			IFNULL(MAX(can_view_value), 1) AS can_view_value,
			IFNULL(MAX(can_grant_view_value), 1) AS can_grant_view_value,
			IFNULL(MAX(can_watch_value), 1) AS can_watch_value,
			IFNULL(MAX(can_edit_value), 1) AS can_edit_value,
			IFNULL(MAX(is_owner), 0) AS is_owner, ` + canMakeSessionOfficialColumn + ", " + canEnterFromColumn)

	aggregatedPermissions := ancestorPermissions.Select(canEnterFromColumn + ", " + canMakeSessionOfficialColumn)

	grantedPermissionsGroupMembership := grantedPermissionsWithAncestors.
		Where("origin = 'group_membership'").
		Where("NOT (group_id = ? AND source_group_id = ?)", groupID, sourceGroupID)
	grantedPermissionsItemUnlocking := grantedPermissionsWithAncestors.Where("origin = 'item_unlocking'")
	grantedPermissionsSelf := grantedPermissionsWithAncestors.Where("origin = 'self'")
	grantedPermissionsOther := grantedPermissionsWithAncestors.Where("origin = 'other'")

	err = store.
		Raw(`
			SELECT
				grp.can_view_value AS granted_directly_can_view_value, grp.can_grant_view_value AS granted_directly_can_grant_view_value,
				grp.can_watch_value AS granted_directly_can_watch_value, grp.can_edit_value AS granted_directly_can_edit_value,
				grp.can_make_session_official AS granted_directly_can_make_session_official, grp.can_enter_from AS granted_directly_can_enter_from,
				grp.can_enter_until AS granted_directly_can_enter_until, grp.is_owner AS granted_directly_is_owner,

				gep.can_view_generated_value AS generated_can_view_value, gep.can_grant_view_generated_value AS generated_can_grant_view_value,
				gep.can_watch_generated_value AS generated_can_watch_value, gep.can_edit_generated_value AS generated_can_edit_value,
				gep.is_owner_generated AS generated_is_owner,
				grp_aggregated.can_enter_from AS generated_can_enter_from,
				grp_aggregated.can_make_session_official AS generated_can_make_session_official,

				grp_membership.can_view_value AS granted_anc_membership_can_view_value,
				grp_membership.can_grant_view_value AS granted_anc_membership_can_grant_view_value,
				grp_membership.can_watch_value AS granted_anc_membership_can_watch_value,
				grp_membership.can_edit_value AS granted_anc_membership_can_edit_value,
				grp_membership.can_make_session_official AS granted_anc_membership_can_make_session_official,
				grp_membership.can_enter_from AS granted_anc_membership_can_enter_from,
				grp_membership.is_owner AS granted_anc_membership_is_owner,

				grp_unlocking.can_view_value AS granted_anc_unlocking_can_view_value,
				grp_unlocking.can_grant_view_value AS granted_anc_unlocking_can_grant_view_value,
				grp_unlocking.can_watch_value AS granted_anc_unlocking_can_watch_value,
				grp_unlocking.can_edit_value AS granted_anc_unlocking_can_edit_value,
				grp_unlocking.can_make_session_official AS granted_anc_unlocking_can_make_session_official,
				grp_unlocking.can_enter_from AS granted_anc_unlocking_can_enter_from,
				grp_unlocking.is_owner AS granted_anc_unlocking_is_owner,

				grp_self.can_view_value AS granted_anc_self_can_view_value, grp_self.can_grant_view_value AS granted_anc_self_can_grant_view_value,
				grp_self.can_watch_value AS granted_anc_self_can_watch_value, grp_self.can_edit_value AS granted_anc_self_can_edit_value,
				grp_self.can_make_session_official AS granted_anc_self_can_make_session_official,
				grp_self.can_enter_from AS granted_anc_self_can_enter_from,
				grp_self.is_owner AS granted_anc_self_is_owner,

				grp_other.can_view_value AS granted_anc_other_can_view_value, grp_other.can_grant_view_value AS granted_anc_other_can_grant_view_value,
				grp_other.can_watch_value AS granted_anc_other_can_watch_value, grp_other.can_edit_value AS granted_anc_other_can_edit_value,
				grp_other.can_make_session_official AS granted_anc_other_can_make_session_official,
				grp_other.can_enter_from AS granted_anc_other_can_enter_from,
				grp_other.is_owner AS granted_anc_other_is_owner
			FROM (?) AS grp, (?) AS gep, (?) AS grp_membership, (?) AS grp_unlocking, (?) AS grp_self, (?) AS grp_other, (?) AS grp_aggregated`,
			grantedPermissions.SubQuery(), generatedPermissions.SubQuery(), grantedPermissionsGroupMembership.SubQuery(),
			grantedPermissionsItemUnlocking.SubQuery(), grantedPermissionsSelf.SubQuery(), grantedPermissionsOther.SubQuery(),
			aggregatedPermissions.SubQuery()).
		ScanIntoSliceOfMaps(&permissions).Error()
	service.MustNotBeError(err)

	permissionsRow := permissions[0]
	permissionsGrantedStore := store.PermissionsGranted()

	render.Respond(w, r, &permissionsViewResponse{
		Granted: grantedPermissionsStruct{
			permissionsStruct: permissionsStruct{
				ItemPermissions: structures.ItemPermissions{
					CanView:      permissionsGrantedStore.ViewNameByIndex(int(permissionsRow["granted_directly_can_view_value"].(int64))),
					CanGrantView: permissionsGrantedStore.GrantViewNameByIndex(int(permissionsRow["granted_directly_can_grant_view_value"].(int64))),
					CanWatch:     permissionsGrantedStore.WatchNameByIndex(int(permissionsRow["granted_directly_can_watch_value"].(int64))),
					CanEdit:      permissionsGrantedStore.EditNameByIndex(int(permissionsRow["granted_directly_can_edit_value"].(int64))),
					IsOwner:      permissionsRow["granted_directly_is_owner"].(int64) == 1,
				},
				CanMakeSessionOfficial: permissionsRow["granted_directly_can_make_session_official"].(int64) == 1,
			},
			CanEnterFrom:  service.ConvertDBTimeToJSONTime(permissionsRow["granted_directly_can_enter_from"]),
			CanEnterUntil: service.ConvertDBTimeToJSONTime(permissionsRow["granted_directly_can_enter_until"]),
		},
		Computed: computedPermissions{permissionsWithCanEnterFrom{
			permissionsStruct: permissionsStruct{
				ItemPermissions: structures.ItemPermissions{
					CanView:      permissionsGrantedStore.ViewNameByIndex(int(permissionsRow["generated_can_view_value"].(int64))),
					CanGrantView: permissionsGrantedStore.GrantViewNameByIndex(int(permissionsRow["generated_can_grant_view_value"].(int64))),
					CanWatch:     permissionsGrantedStore.WatchNameByIndex(int(permissionsRow["generated_can_watch_value"].(int64))),
					CanEdit:      permissionsGrantedStore.EditNameByIndex(int(permissionsRow["generated_can_edit_value"].(int64))),
					IsOwner:      permissionsRow["generated_is_owner"].(int64) == 1,
				},
				CanMakeSessionOfficial: permissionsRow["generated_can_make_session_official"].(int64) == 1,
			},
			CanEnterFrom: service.ConvertDBTimeToJSONTime(permissionsRow["generated_can_enter_from"]),
		}},
		GrantedViaGroupMembership: permissionsGrantedViaGroupMembership{permissionsWithCanEnterFrom{
			permissionsStruct: permissionsStruct{
				ItemPermissions: structures.ItemPermissions{
					CanView:      permissionsGrantedStore.ViewNameByIndex(int(permissionsRow["granted_anc_membership_can_view_value"].(int64))),
					CanGrantView: permissionsGrantedStore.GrantViewNameByIndex(int(permissionsRow["granted_anc_membership_can_grant_view_value"].(int64))),
					CanWatch:     permissionsGrantedStore.WatchNameByIndex(int(permissionsRow["granted_anc_membership_can_watch_value"].(int64))),
					CanEdit:      permissionsGrantedStore.EditNameByIndex(int(permissionsRow["granted_anc_membership_can_edit_value"].(int64))),
					IsOwner:      permissionsRow["granted_anc_membership_is_owner"].(int64) == 1,
				},
				CanMakeSessionOfficial: permissionsRow["granted_anc_membership_can_make_session_official"].(int64) == 1,
			},
			CanEnterFrom: service.ConvertDBTimeToJSONTime(permissionsRow["granted_anc_membership_can_enter_from"]),
		}},
		GrantedViaItemUnlocking: permissionsGrantedViaItemUnlocking{permissionsWithCanEnterFrom{
			permissionsStruct: permissionsStruct{
				ItemPermissions: structures.ItemPermissions{
					CanView:      permissionsGrantedStore.ViewNameByIndex(int(permissionsRow["granted_anc_unlocking_can_view_value"].(int64))),
					CanGrantView: permissionsGrantedStore.GrantViewNameByIndex(int(permissionsRow["granted_anc_unlocking_can_grant_view_value"].(int64))),
					CanWatch:     permissionsGrantedStore.WatchNameByIndex(int(permissionsRow["granted_anc_unlocking_can_watch_value"].(int64))),
					CanEdit:      permissionsGrantedStore.EditNameByIndex(int(permissionsRow["granted_anc_unlocking_can_edit_value"].(int64))),
					IsOwner:      permissionsRow["granted_anc_unlocking_is_owner"].(int64) == 1,
				},
				CanMakeSessionOfficial: permissionsRow["granted_anc_unlocking_can_make_session_official"].(int64) == 1,
			},
			CanEnterFrom: service.ConvertDBTimeToJSONTime(permissionsRow["granted_anc_unlocking_can_enter_from"]),
		}},
		GrantedViaSelf: permissionsGrantedViaSelf{permissionsWithCanEnterFrom{
			permissionsStruct: permissionsStruct{
				ItemPermissions: structures.ItemPermissions{
					CanView:      permissionsGrantedStore.ViewNameByIndex(int(permissionsRow["granted_anc_self_can_view_value"].(int64))),
					CanGrantView: permissionsGrantedStore.GrantViewNameByIndex(int(permissionsRow["granted_anc_self_can_grant_view_value"].(int64))),
					CanWatch:     permissionsGrantedStore.WatchNameByIndex(int(permissionsRow["granted_anc_self_can_watch_value"].(int64))),
					CanEdit:      permissionsGrantedStore.EditNameByIndex(int(permissionsRow["granted_anc_self_can_edit_value"].(int64))),
					IsOwner:      permissionsRow["granted_anc_self_is_owner"].(int64) == 1,
				},
				CanMakeSessionOfficial: permissionsRow["granted_anc_self_can_make_session_official"].(int64) == 1,
			},
			CanEnterFrom: service.ConvertDBTimeToJSONTime(permissionsRow["granted_anc_self_can_enter_from"]),
		}},
		GrantedViaOther: permissionsGrantedViaOther{permissionsWithCanEnterFrom{
			permissionsStruct: permissionsStruct{
				ItemPermissions: structures.ItemPermissions{
					CanView:      permissionsGrantedStore.ViewNameByIndex(int(permissionsRow["granted_anc_other_can_view_value"].(int64))),
					CanGrantView: permissionsGrantedStore.GrantViewNameByIndex(int(permissionsRow["granted_anc_other_can_grant_view_value"].(int64))),
					CanWatch:     permissionsGrantedStore.WatchNameByIndex(int(permissionsRow["granted_anc_other_can_watch_value"].(int64))),
					CanEdit:      permissionsGrantedStore.EditNameByIndex(int(permissionsRow["granted_anc_other_can_edit_value"].(int64))),
					IsOwner:      permissionsRow["granted_anc_other_is_owner"].(int64) == 1,
				},
				CanMakeSessionOfficial: permissionsRow["granted_anc_other_can_make_session_official"].(int64) == 1,
			},
			CanEnterFrom: service.ConvertDBTimeToJSONTime(permissionsRow["granted_anc_other_can_enter_from"]),
		}},
	})
	return service.NoError
}
