package contests

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation GET /contests/{item_id}/groups/{group_id}/members/additional-times contests contestListMembersAdditionalTime
//
//		---
//		summary: List additional times on a contest
//		description: >
//	               For all descendant
//
//	                 * teams if `items.entry_participant_type` = 'Team'
//	                 * end-users groups otherwise
//
//	               linked to the item via `attempts.root_item_id`
//	               and able to view (at least 'can_view:info') or enter (`can_enter_from` < `can_enter_until`) the item,
//	               the service returns their `group_id`, `name`, `type` and `additional_time` & `total_additional_time`.
//
//
//	               * `additional_time` defaults to 0 if no such `groups_contest_items`
//
//	               * `total_additional_time` is the sum of additional times of this group on the item through all its
//	                 `groups_ancestors` (even from different branches, but each ancestors counted only once), defaulting to 0
//
//	               Restrictions:
//	                 * `item_id` should be a timed contest;
//	                 * the authenticated user should have `can_view` >= 'content', `can_grant_view` >= 'enter',
//	                   and `can_watch` >= 'result' on the input item;
//	                 * the authenticated user should be a manager of the `group_id`
//	                   with `can_grant_group_access` and `can_watch_members` permissions.
//		parameters:
//			- name: item_id
//				description: "`id` of a timed contest"
//				in: path
//				type: integer
//				required: true
//			- name: group_id
//				in: path
//				type: integer
//				required: true
//			- name: from.id
//				description: Start the page from the group next to the group with `id`=`{from.id}`
//				in: query
//				type: integer
//			- name: sort
//				in: query
//				default: [name,id]
//				type: array
//				items:
//					type: string
//					enum: [name,-name,id,-id]
//			- name: limit
//				description: Display the first N groups
//				in: query
//				type: integer
//				maximum: 1000
//				default: 500
//		responses:
//			"200":
//				description: OK. Success response with contests info
//				schema:
//					type: array
//					items:
//	        	"$ref": "#/definitions/contestInfo"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getMembersAdditionalTimes(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)

	itemID, err := service.ResolveURLQueryPathInt64Field(r, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	store := srv.GetStore(r)
	participantType, err := getParticipantTypeForContestManagedByUser(store, itemID, user)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return service.InsufficientAccessRightsError
	}
	service.MustNotBeError(err)

	ok, err := store.Groups().ManagedBy(user).Where("groups.id = ?", groupID).
		Having("MAX(can_grant_group_access) AND MAX(can_watch_members)").HasRows()
	service.MustNotBeError(err)
	if !ok {
		return service.InsufficientAccessRightsError
	}

	query := store.ActiveGroupAncestors().Where("groups_ancestors_active.ancestor_group_id = ?", groupID).
		Joins(`
			JOIN `+"`groups`"+` AS found_group
				ON found_group.id = groups_ancestors_active.child_group_id AND found_group.type = ?`, participantType).
		Joins(`
			JOIN attempts
				ON attempts.participant_id = found_group.id AND attempts.root_item_id = ?`, itemID).
		Joins(`
			JOIN groups_ancestors_active AS found_group_ancestors
				ON found_group_ancestors.child_group_id = found_group.id`).
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
			LEFT JOIN groups_contest_items AS main_group_contest_item ON main_group_contest_item.group_id = found_group.id AND
				main_group_contest_item.item_id = ?`, itemID).
		Select(`
				found_group.id AS group_id,
				found_group.name,
				found_group.type,
				IFNULL(TIME_TO_SEC(MAX(main_group_contest_item.additional_time)), 0) AS additional_time,
				IFNULL(SUM(TIME_TO_SEC(groups_contest_items.additional_time)), 0) AS total_additional_time`).
		Group("found_group.id").
		Having(`
			MAX(permissions_generated.can_view_generated_value) >= ? OR
			MAX(permissions_granted.can_enter_from < permissions_granted.can_enter_until)`,
			store.PermissionsGranted().ViewIndexByName("info"))

	query = service.NewQueryLimiter().Apply(r, query)
	query, apiError := service.ApplySortingAndPaging(
		r, query,
		&service.SortingAndPagingParameters{
			Fields: service.SortingAndPagingFields{
				"name": {ColumnName: "found_group.name"},
				"id":   {ColumnName: "found_group.id"},
			},
			DefaultRules: "name,id",
			TieBreakers:  service.SortingAndPagingTieBreakers{"id": service.FieldTypeInt64},
		})
	if apiError != service.NoError {
		return apiError
	}

	var result []contestInfo
	service.MustNotBeError(query.Scan(&result).Error())

	render.Respond(w, r, result)
	return service.NoError
}
