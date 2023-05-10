package contests

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation GET /contests/{item_id}/groups/by-name contests contestGetGroupByName
//
//		---
//		summary: Get a group by name
//		description: >
//	               Return one group matching the name and satisfying:
//
//	                 * the group can view (at least 'can_view:info') or enter (`can_enter_from` < `can_enter_until`) the item;
//	                 * the authenticated user is a manager of the group with `can_grant_group_access` and `can_watch_members` permissions;
//	                 * the `groups.name` (matching `login` if a "User" group) is matching the input `name` parameter (case-insensitive)
//
//	               If there are several groups or users matching, returns the first one (by `id`).
//
//
//	               If the contest is a team-only contest (`items.entry_participant_type` = 'Team') and the name matches an end-user,
//	               returns his team instead of user’s group.
//
//
//	               Restrictions:
//	                 * `item_id` should be a timed contest;
//	                 * the authenticated user should have `can_view` >= 'content, `can_grant_view` >= 'enter', and `can_watch` >= 'result'
//	                   on the input item.
//
//	               Otherwise, the "Forbidden" response is returned.
//
//
//	               __NOTE__: This service is only here for transition between the former interface and the new one.
//	                     This way of searching only by `name`/`login` and getting one result is not really convenient,
//	                     but matching the former UI. This service will have to be removed as soon as
//	                     the new interface is used.
//
//		parameters:
//			- name: item_id
//				description: "`id` of a timed contest"
//				in: path
//				type: integer
//				required: true
//			- name: name
//				in: query
//				type: string
//				required: true
//		responses:
//			"200":
//				description: OK. Success response with the `group_id`, `additional_time`, `total_additional_time`
//				schema:
//					"$ref": "#/definitions/contestInfo"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getGroupByName(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)

	itemID, err := service.ResolveURLQueryPathInt64Field(r, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	groupName, err := service.ResolveURLQueryGetStringField(r, "name")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	store := srv.GetStore(r)
	participantType, err := getParticipantTypeForContestManagedByUser(store, itemID, user)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return service.InsufficientAccessRightsError
	}
	service.MustNotBeError(err)

	groupsManagedByUserSubQuery := store.GroupAncestors().ManagedByUser(user).
		Group("groups_ancestors.child_group_id").
		Having("MAX(can_grant_group_access) AND MAX(can_watch_members)").
		Select("groups_ancestors.child_group_id").SubQuery()
	query := store.Groups().
		Joins(`
			JOIN groups_ancestors_active AS found_group_ancestors
				ON found_group_ancestors.child_group_id = groups.id`).
		Joins(`
			LEFT JOIN permissions_generated ON permissions_generated.group_id = found_group_ancestors.ancestor_group_id AND
				permissions_generated.item_id = ?`, itemID).
		Joins(`
			LEFT JOIN permissions_granted ON permissions_granted.group_id = found_group_ancestors.ancestor_group_id AND
				permissions_granted.item_id = ?`, itemID).
		Joins(`
			LEFT JOIN groups_contest_items ON groups_contest_items.group_id = found_group_ancestors.ancestor_group_id AND
				groups_contest_items.item_id = ?`, itemID).
		Joins(`
			LEFT JOIN groups_contest_items AS main_group_contest_item ON main_group_contest_item.group_id = groups.id AND
				main_group_contest_item.item_id = ?`, itemID).
		Where("groups.id IN (?)", groupsManagedByUserSubQuery).
		Where("groups.type = ?", participantType).
		Select(`
			groups.id AS group_id,
			groups.name,
			groups.type,
			IFNULL(TIME_TO_SEC(MAX(main_group_contest_item.additional_time)), 0) AS additional_time,
			IFNULL(SUM(TIME_TO_SEC(groups_contest_items.additional_time)), 0) AS total_additional_time`).
		Having(`
			MAX(permissions_generated.can_view_generated_value) >= ? OR
			MAX(permissions_granted.can_enter_from < permissions_granted.can_enter_until)`,
			store.PermissionsGranted().ViewIndexByName("info")).
		Group("groups.id").
		Order("groups.id")

	if participantType == team {
		query = query.
			Joins(`
				LEFT JOIN groups_groups_active
					ON groups_groups_active.parent_group_id = groups.id`).
			Joins(`
				LEFT JOIN `+"`groups`"+` AS user_group
					ON user_group.id = groups_groups_active.child_group_id AND user_group.type = 'User' AND
						user_group.name = ? AND LENGTH(user_group.name) = LENGTH(?)`, groupName, groupName).
			Group("groups.id, user_group.id").
			Having("MAX(user_group.id) IS NOT NULL OR (groups.name = ? AND LENGTH(groups.name) = LENGTH(?))", groupName, groupName)
	} else {
		query = query.Where("groups.name = ? AND LENGTH(groups.name) = LENGTH(?)", groupName, groupName)
	}

	var result contestInfo
	if err = query.Take(&result).Error(); errors.Is(err, gorm.ErrRecordNotFound) {
		return service.InsufficientAccessRightsError
	}
	service.MustNotBeError(err)

	render.Respond(w, r, result)
	return service.NoError
}
