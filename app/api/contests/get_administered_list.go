package contests

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

type parentTitle struct {
	// Nullable
	// required: true
	Title *string `json:"title"`
	// required: true
	LanguageTag string `json:"language_tag"`
}

// swagger:model
type contestAdminListRow struct {
	// required: true
	ItemID int64 `json:"id,string"`
	// Nullable
	// required: true
	Title *string `gorm:"column:title_translation" json:"title"`
	// required: true
	LanguageTag string `gorm:"column:title_language_tag" json:"language_tag"`
	// required: true
	// enum: User,Team
	EntryParticipantType string `json:"entry_participant_type"`
	// required: true
	AllowsMultipleAttempts bool `json:"allows_multiple_attempts"`
	// required: true
	Parents []parentTitle `gorm:"-" json:"parents"`
}

// swagger:operation GET /contests/administered contests contestAdminList
//
//		---
//		summary: List administered contests
//		description:   Get the contests that the user has administration rights on.
//
//
//	               For all explicit-entry items that are timed contests and for that the user is a contest admin
//	               (has `can_view` >= 'content', `can_grant_view` >= 'enter', and `can_watch` >= 'result'),
//	               returns item info (`id`, `title`, `team_only_contest`, parents' `title`-s).
//	               Only parents visible to the user are listed.
//
//
//	               Each title is returned in the user's default language if exists,
//	               otherwise the item's default language is used.
//		parameters:
//			- name: from.id
//				description: Start the page from the contest next to the contest with `id`=`{from.id}`
//				in: query
//				type: integer
//			- name: sort
//				in: query
//				default: [title,id]
//				type: array
//				items:
//					type: string
//					enum: [title,-title,id,-id]
//			- name: limit
//				description: Display the first N contests
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
//						"$ref": "#/definitions/contestAdminListRow"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getAdministeredList(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)
	store := srv.GetStore(r)

	var rows []contestAdminListRow
	query := store.Items().Select(`
			items.id AS item_id,
			items.allows_multiple_attempts,
			items.entry_participant_type,
			COALESCE(user_strings.title, default_strings.title) AS title_translation,
			COALESCE(user_strings.language_tag, default_strings.language_tag) AS title_language_tag`).
		JoinsPermissionsForGroupToItemsWherePermissionAtLeast(user.GroupID, "view", "content").
		WherePermissionIsAtLeast("grant_view", "enter").
		WherePermissionIsAtLeast("watch", "result").
		JoinsUserAndDefaultItemStrings(user).
		Where("items.duration IS NOT NULL").
		Where("items.requires_explicit_entry")

	query, apiError := service.ApplySortingAndPaging(
		r, query,
		&service.SortingAndPagingParameters{
			Fields: service.SortingAndPagingFields{
				"title": {ColumnName: "IFNULL(COALESCE(user_strings.title, default_strings.title), '')"},
				"id":    {ColumnName: "items.id"},
			},
			DefaultRules: "title,id",
			TieBreakers:  service.SortingAndPagingTieBreakers{"id": service.FieldTypeInt64},
		})
	if apiError != service.NoError {
		return apiError
	}
	query = service.NewQueryLimiter().Apply(r, query)

	service.MustNotBeError(query.Scan(&rows).Error())

	if len(rows) > 0 {
		itemIDs := make([]int64, len(rows))
		for index := range rows {
			itemIDs[index] = rows[index].ItemID
		}
		var parents []struct {
			ChildID           int64
			ParentTitle       *string
			ParentLanguageTag string
		}
		service.MustNotBeError(store.Items().
			Joins("JOIN items_items ON items_items.parent_item_id = items.id AND items_items.child_item_id IN (?)", itemIDs).
			WhereItemsAreVisible(user.GroupID).
			JoinsUserAndDefaultItemStrings(user).
			Order("COALESCE(user_strings.title, default_strings.title)").
			Select(`
				items_items.child_item_id as child_id,
				COALESCE(user_strings.title, default_strings.title) AS parent_title,
				COALESCE(user_strings.language_tag, default_strings.language_tag) AS parent_language_tag`).
			Scan(&parents).Error())

		parentTitlesMap := make(map[int64][]parentTitle, len(rows))
		for index := range parents {
			if _, ok := parentTitlesMap[parents[index].ChildID]; !ok {
				parentTitlesMap[parents[index].ChildID] = make([]parentTitle, 0, 1)
			}
			parentTitlesMap[parents[index].ChildID] = append(parentTitlesMap[parents[index].ChildID], parentTitle{
				Title:       parents[index].ParentTitle,
				LanguageTag: parents[index].ParentLanguageTag,
			})
		}
		for index := range rows {
			rows[index].Parents = parentTitlesMap[rows[index].ItemID]
			if rows[index].Parents == nil {
				rows[index].Parents = make([]parentTitle, 0)
			}
		}
	}

	render.Respond(w, r, rows)
	return service.NoError
}
