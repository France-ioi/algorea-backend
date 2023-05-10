package groups

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
	"github.com/France-ioi/AlgoreaBackend/app/structures"
)

// swagger:model groupsMembersViewResponseRow
type groupsMembersViewResponseRow struct {
	// `groups.id`
	// required: true
	ID          int64          `json:"id,string"`
	MemberSince *database.Time `json:"member_since,omitempty"`
	// the latest `group_membership_changes.action`
	// enum: invitation_accepted,join_request_accepted,joined_by_badge,joined_by_code,added_directly
	Action *string `json:"action,omitempty"`
	// required: true
	User struct {
		// `users.group_id`
		// required: true
		GroupID int64 `json:"group_id,string"`
		// required: true
		Login string `json:"login"`

		*structures.UserPersonalInfo
		ShowPersonalInfo bool `json:"-"`

		// Nullable
		// required: true
		Grade *int32 `json:"grade"`
	} `json:"user" gorm:"embedded;embeddedPrefix:user__"`
}

// swagger:operation GET /groups/{group_id}/members group-memberships groupsMembersView
//
//		---
//		summary: List group members
//		description: >
//
//	  Returns a list of users that are members of the group.
//	  The output contains basic user info (`first_name` and `last_name` are only shown
//	  for the authenticated user or if the user approved access to their personal info for some group
//	  managed by the authenticated user).
//
//
//	  The authenticated user should be a manager of `{group_id}`, otherwise the 'forbidden' error is returned.
//		parameters:
//			- name: group_id
//				in: path
//				type: integer
//				required: true
//			- name: sort
//				in: query
//				default: [-member_since,id]
//				type: array
//				items:
//					type: string
//					enum: [member_since,-member_since,user.login,-user.login,user.grade,-user.grade,id,-id]
//			- name: from.id
//				description: Start the page from the member next to the member with `groups.id`=`{from.id}`
//				in: query
//				type: integer
//			- name: limit
//				description: Display the first N members
//				in: query
//				type: integer
//				maximum: 1000
//				default: 500
//		responses:
//			"200":
//				description: OK. The array of group members
//				schema:
//					type: array
//					items:
//						"$ref": "#/definitions/groupsMembersViewResponseRow"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getMembers(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)
	store := srv.GetStore(r)

	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	if apiError := checkThatUserCanManageTheGroup(store, user, groupID); apiError != service.NoError {
		return apiError
	}

	query := store.GroupGroups().
		Select(`
			groups_groups.child_group_id AS id,
			latest_change.at AS member_since,
			latest_change.action,
			users.group_id AS user__group_id,
			users.login AS user__login,
			users.group_id = ? OR personal_info_view_approvals.approved AS user__show_personal_info,
			IF(users.group_id = ? OR personal_info_view_approvals.approved, users.first_name, NULL) AS user__first_name,
			IF(users.group_id = ? OR personal_info_view_approvals.approved, users.last_name, NULL) AS user__last_name,
			users.grade AS user__grade`, user.GroupID, user.GroupID, user.GroupID).
		Joins("JOIN users ON users.group_id = groups_groups.child_group_id").
		WithPersonalInfoViewApprovals(user).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT at, action FROM group_membership_changes USE INDEX(group_id_member_id_at_desc)
				WHERE group_membership_changes.group_id = groups_groups.parent_group_id AND
					group_membership_changes.member_id = groups_groups.child_group_id
				ORDER BY group_membership_changes.group_id, group_membership_changes.member_id, at DESC
				LIMIT 1
			) AS latest_change ON 1`).
		WhereGroupRelationIsActual().
		Where("groups_groups.parent_group_id = ?", groupID)

	query = service.NewQueryLimiter().Apply(r, query)
	query, apiError := service.ApplySortingAndPaging(
		r, query,
		&service.SortingAndPagingParameters{
			Fields: service.SortingAndPagingFields{
				"user.login":   {ColumnName: "users.login"},
				"user.grade":   {ColumnName: "users.grade"},
				"member_since": {ColumnName: "latest_change.at"},
				"id":           {ColumnName: "groups_groups.child_group_id"},
			},
			DefaultRules: "-member_since,id",
			TieBreakers:  service.SortingAndPagingTieBreakers{"id": service.FieldTypeInt64},
		})

	if apiError != service.NoError {
		return apiError
	}

	var result []groupsMembersViewResponseRow
	service.MustNotBeError(query.Scan(&result).Error())
	for index := range result {
		if !result[index].User.ShowPersonalInfo {
			result[index].User.UserPersonalInfo = nil
		}
	}

	render.Respond(w, r, result)
	return service.NoError
}
