package items

import (
	"net/http"

	"github.com/go-chi/render"
	"gorm.io/gorm/clause"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:model officialSessionsListResponseRow
type officialSessionsListResponseRow struct {
	// required: true
	GroupID int64 `json:"group_id,string"`
	// required: true
	Name string `json:"name"`
	// Nullable
	// required: true
	Description *string `json:"description"`
	// required: true
	OpenActivityWhenJoining bool `json:"open_activity_when_joining"`
	// required: true
	// enum: none,view,edit
	RequirePersonalInfoAccessApproval string `json:"require_personal_info_access_approval"`
	// Nullable
	// required: true
	RequireLockMembershipApprovalUntil *database.Time `json:"require_lock_membership_approval_until"`
	// required: true
	RequireWatchApproval bool `json:"require_watch_approval"`
	// required: true
	RequireMembersToJoinParent bool `json:"require_members_to_join_parent"`
	// required: true
	IsPublic bool `json:"is_public"`
	// Nullable
	// required: true
	Organizer *string `json:"organizer"`
	// Nullable
	// required: true
	AddressLine1 *string `json:"address_line1"`
	// Nullable
	// required: true
	AddressLine2 *string `json:"address_line2"`
	// Nullable
	// required: true
	AddressPostcode *string `json:"address_postcode"`
	// Nullable
	// required: true
	AddressCity *string `json:"address_city"`
	// Nullable
	// required: true
	AddressCountry *string `json:"address_country"`
	// Nullable
	// required: true
	ExpectedStart *database.Time `json:"expected_start"`
	// required:true
	CurrentUserIsManager bool `json:"current_user_is_manager"`
	// `True` when there is an active group->user relation in `groups_groups`
	// required:true
	CurrentUserIsMember bool `json:"current_user_is_member"`
	// required:true
	Parents []officialSessionsListResponseRowParent `json:"parents"`
}

type officialSessionsListResponseRowParent struct {
	// required: true
	GroupID int64 `json:"id,string"`
	// required: true
	Name string `json:"name"`
	// required: true
	IsPublic bool `json:"is_public"`
	// required:true
	CurrentUserIsManager bool `json:"current_user_is_manager"`
	// `True` when there is an active group->user relation in `groups_groups`
	// required:true
	CurrentUserIsMember bool `json:"current_user_is_member"`
}

type rawOfficialSession struct {
	GroupID                            int64
	Name                               string
	Description                        *string
	OpenActivityWhenJoining            bool
	RequirePersonalInfoAccessApproval  string
	RequireLockMembershipApprovalUntil *database.Time
	RequireWatchApproval               bool
	RequireMembersToJoinParent         bool
	IsPublic                           bool
	Organizer                          *string
	AddressLine1                       *string `gorm:"address_line1"`
	AddressLine2                       *string `gorm:"address_line2"`
	AddressPostcode                    *string
	AddressCity                        *string
	AddressCountry                     *string
	ExpectedStart                      *database.Time
	CurrentUserIsManager               bool
	CurrentUserIsMember                bool
	Parent                             struct {
		GroupID              *int64
		Name                 string
		IsPublic             bool
		CurrentUserIsManager bool
		CurrentUserIsMember  bool
	} `gorm:"embedded;embeddedPrefix:parent__"`
}

// swagger:operation GET /items/{item_id}/official-sessions items officialSessionsList
//
//		---
//		summary: List all official sessions for an item
//		description: >
//	   Lists the groups having `type`='Session', `is_official_session`=true, `is_public`=true, `root_activity_id`=`{item_id}`
//	   along with their parent groups (public or managed by the current user or having the current user as a member).
//
//
//	   Restrictions:
//	     * the current user should have at least 'info' permission on the item,
//
//	   otherwise the 'forbidden' error is returned.
//		parameters:
//			- name: item_id
//				in: path
//				type: integer
//				format: int64
//				required: true
//			- name: sort
//				in: query
//				default: [expected_start$,name,group_id]
//				type: array
//				items:
//					type: string
//					enum: [group_id,-group_id,expected_start,-expected_start,expected_start$,-expected_start$,name,-name]
//			- name: from.group_id
//				description: Start the page from the official session next to the official session with `groups.id` = `{from.group_id}`
//				in: query
//				type: integer
//				format: int64
//			- name: limit
//				description: Display first N official sessions
//				in: query
//				type: integer
//				maximum: 1000
//				default: 500
//		responses:
//			"200":
//				description: OK. Success response with an array of official sessions
//				schema:
//					type: array
//					items:
//						"$ref": "#/definitions/officialSessionsListResponseRow"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) listOfficialSessions(w http.ResponseWriter, r *http.Request) service.APIError {
	itemID, err := service.ResolveURLQueryPathInt64Field(r, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(r)
	store := srv.GetStore(r)
	found, err := store.Permissions().MatchingUserAncestors(user).
		Where("item_id = ?", itemID).
		WherePermissionIsAtLeast("view", "info").HasRows()
	service.MustNotBeError(err)
	if !found {
		return service.InsufficientAccessRightsError
	}

	idsQuery := store.Groups().Where("type = 'Session'").
		Where("is_official_session").
		Where("is_public").
		Where("root_activity_id = ?", itemID)
	idsQuery = service.NewQueryLimiter().Apply(r, idsQuery)
	idsQuery, apiError := service.ApplySortingAndPaging(
		r, idsQuery,
		&service.SortingAndPagingParameters{
			Fields: service.SortingAndPagingFields{
				"group_id":       {ColumnName: "groups.id"},
				"expected_start": {ColumnName: "groups.expected_start", Nullable: true},
				"name":           {ColumnName: "groups.name"},
			},
			DefaultRules: "expected_start$,name,group_id",
			TieBreakers:  service.SortingAndPagingTieBreakers{"group_id": service.FieldTypeInt64},
		})
	if apiError != service.NoError {
		return apiError
	}

	var ids []interface{}
	service.MustNotBeError(idsQuery.Pluck("groups.id", &ids).Error())

	var rawData []rawOfficialSession
	if len(ids) > 0 {
		service.MustNotBeError(store.Groups().Where("groups.id IN (?)", ids).
			Select(`
				groups.id AS group_id, groups.name, groups.description, groups.open_activity_when_joining,
				groups.require_personal_info_access_approval, groups.require_lock_membership_approval_until,
				groups.require_watch_approval, groups.require_members_to_join_parent,
				groups.is_public, groups.organizer, groups.address_line1, groups.address_line2,
				groups.address_postcode, groups.address_city, groups.address_country, groups.expected_start,
				EXISTS(?) AS current_user_is_member,
				EXISTS(?) AS current_user_is_manager,
				parent.id AS parent__group_id, parent.name AS parent__name, parent.is_public AS parent__is_public,
				parent.id IS NOT NULL AND EXISTS(?) AS parent__current_user_is_member,
				parent.id IS NOT NULL AND EXISTS(?) AS parent__current_user_is_manager`,
				store.ActiveGroupGroups().WhereUserIsMember(user).Where("parent_group_id = groups.id").QueryExpr(),
				store.ActiveGroupAncestors().ManagedByUser(user).Where("groups_ancestors_active.child_group_id = groups.id").QueryExpr(),
				store.ActiveGroupGroups().WhereUserIsMember(user).Where("parent_group_id = parent.id").QueryExpr(),
				store.ActiveGroupAncestors().ManagedByUser(user).
					Where("groups_ancestors_active.child_group_id = parent.id").QueryExpr()).
			Joins("LEFT JOIN groups_groups_active ON groups_groups_active.child_group_id = groups.id").
			Joins("LEFT JOIN `groups` AS parent ON parent.id = groups_groups_active.parent_group_id").
			Having("parent.id IS NULL OR parent.is_public OR parent__current_user_is_member OR parent__current_user_is_manager").
			Clauses(clause.OrderBy{
				Expression: clause.Expr{
					SQL:                "FIELD(groups.id, ?), parent.name, parent.id",
					Vars:               []interface{}{ids},
					WithoutParentheses: true,
				},
			}).
			Scan(&rawData).Error())
	}

	var result []officialSessionsListResponseRow
	srv.fillOfficialSessionsWithParents(rawData, &result)

	render.Respond(w, r, result)
	return service.NoError
}

func (srv *Service) fillOfficialSessionsWithParents(
	rawData []rawOfficialSession, target *[]officialSessionsListResponseRow,
) {
	*target = make([]officialSessionsListResponseRow, 0, len(rawData))
	var currentRow *officialSessionsListResponseRow
	for index := range rawData {
		if index == 0 || rawData[index].GroupID != rawData[index-1].GroupID {
			row := officialSessionsListResponseRow{
				GroupID:                            rawData[index].GroupID,
				Name:                               rawData[index].Name,
				Description:                        rawData[index].Description,
				OpenActivityWhenJoining:            rawData[index].OpenActivityWhenJoining,
				RequirePersonalInfoAccessApproval:  rawData[index].RequirePersonalInfoAccessApproval,
				RequireLockMembershipApprovalUntil: rawData[index].RequireLockMembershipApprovalUntil,
				RequireWatchApproval:               rawData[index].RequireWatchApproval,
				RequireMembersToJoinParent:         rawData[index].RequireMembersToJoinParent,
				IsPublic:                           rawData[index].IsPublic,
				Organizer:                          rawData[index].Organizer,
				AddressLine1:                       rawData[index].AddressLine1,
				AddressLine2:                       rawData[index].AddressLine2,
				AddressPostcode:                    rawData[index].AddressPostcode,
				AddressCity:                        rawData[index].AddressCity,
				AddressCountry:                     rawData[index].AddressCountry,
				ExpectedStart:                      rawData[index].ExpectedStart,
				CurrentUserIsManager:               rawData[index].CurrentUserIsManager,
				CurrentUserIsMember:                rawData[index].CurrentUserIsMember,
				Parents:                            make([]officialSessionsListResponseRowParent, 0, 1),
			}
			*target = append(*target, row)
			currentRow = &(*target)[len(*target)-1]
		}

		if rawData[index].Parent.GroupID != nil {
			parent := officialSessionsListResponseRowParent{
				GroupID:              *rawData[index].Parent.GroupID,
				Name:                 rawData[index].Parent.Name,
				IsPublic:             rawData[index].Parent.IsPublic,
				CurrentUserIsManager: rawData[index].Parent.CurrentUserIsManager,
				CurrentUserIsMember:  rawData[index].Parent.CurrentUserIsMember,
			}
			currentRow.Parents = append(currentRow.Parents, parent)
		}
	}
}
