package currentuser

import (
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

const minSearchStringLength = 3

// swagger:operation GET /current-user/available-groups groups groupsJoinableSearch
//
//		---
//		summary: Search for groups to join
//		description: >
//	  Searches for groups that can be joined freely, based on a substring of their name.
//	  Returns groups with `is_public` = 1 and `type` != 'User'/'ContestParticipants', whose `name` has `{search}` as a substring,
//	  and for that the current user is not already a member and don’t have pending requests/invitations.
//
//
//				Note: The current implementation may be very slow because it uses `LIKE` with a percentage wildcard
//	  at the beginning. This causes MySQL to explore every row having `is_public`=1. Moreover, actually
//	  it has to examine every row of the `groups` table since there is no index for the `is_public` column.
//	  But since there are not too many groups and the result rows count is limited, the search works almost well.
//		parameters:
//			- name: search
//				in: query
//				type: string
//				minLength: 3
//				required: true
//			- name: sort
//				in: query
//				default: [id]
//				type: array
//				items:
//					type: string
//					enum: [id,-id]
//			- name: from.id
//				description: Start the page from the group next to one with `groups.id`=`{from.id}`
//				in: query
//				type: integer
//			- name: limit
//				description: Display the first N groups
//				in: query
//				type: integer
//				maximum: 1000
//				default: 500
//		responses:
//			"200":
//					description: OK. Success response with an array of found groups
//					schema:
//						type: array
//						items:
//							type: object
//							properties:
//								id:
//									type: string
//									format: int64
//								name:
//									type: string
//								type:
//									type: string
//									enum: [Class,Team,Club,Friends,Other,Session,Base]
//								description:
//									description: Nullable
//									type: string
//							required:
//								- id
//								- name
//								- type
//								- description
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) searchForAvailableGroups(w http.ResponseWriter, r *http.Request) service.APIError {
	searchString, err := service.ResolveURLQueryGetStringField(r, "search")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	searchString = strings.TrimSpace(searchString)

	if utf8.RuneCountInString(searchString) < minSearchStringLength {
		return service.ErrInvalidRequest(
			fmt.Errorf("the search string should be at least %d characters long", minSearchStringLength))
	}

	user := srv.GetUser(r)
	store := srv.GetStore(r)

	skipGroups := store.ActiveGroupGroups().
		Select("groups_groups_active.parent_group_id").
		Where("groups_groups_active.child_group_id = ?", user.GroupID).
		SubQuery()

	skipPending := store.GroupPendingRequests().
		Select("group_pending_requests.group_id").
		Where("group_pending_requests.member_id = ?", user.GroupID).
		Where("group_pending_requests.type IN ('join_request', 'invitation')").
		SubQuery()

	escapedSearchString := database.EscapeLikeString(searchString, '|')
	query := store.Groups().
		Select(`
			groups.id,
			groups.name,
			groups.type,
			groups.description`).
		Where("groups.is_public").
		Where("type != 'User' AND type != 'ContestParticipants'").
		Where("groups.id NOT IN (?)", skipGroups).
		Where("groups.id NOT IN (?)", skipPending).
		Where("groups.name LIKE CONCAT('%', ?, '%') ESCAPE '|'", escapedSearchString)

	query = service.NewQueryLimiter().Apply(r, query)
	query, apiError := service.ApplySortingAndPaging(
		r, query,
		&service.SortingAndPagingParameters{
			Fields:       service.SortingAndPagingFields{"id": {ColumnName: "groups.id"}},
			DefaultRules: "id",
			TieBreakers:  service.SortingAndPagingTieBreakers{"id": service.FieldTypeInt64},
		})
	if apiError != service.NoError {
		return apiError
	}

	var result []map[string]interface{}
	service.MustNotBeError(query.ScanIntoSliceOfMaps(&result).Error())
	convertedResult := service.ConvertSliceOfMapsFromDBToJSON(result)

	render.Respond(w, r, convertedResult)
	return service.NoError
}
