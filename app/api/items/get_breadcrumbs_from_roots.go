package items

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:model breadcrumbElement
type breadcrumbElement struct {
	// required: true
	ID int64 `json:"id,string"`
	// Nullable
	// required: true
	Title *string `json:"title"`
	// required: true
	// enum: Chapter,Task,Course,Skill
	Type *string `json:"type"`
	// required: true
	LanguageTag string `json:"language_tag"`
}

// swagger:operation GET /items/{item_id}/breadcrumbs-from-roots items itemBreadcrumbsFromRootsGet
//
//	---
//	summary: List all possible breadcrumbs for a started item using `item_id`
//	description: >
//		Lists all paths from a root (`root_activity_id`|`root_skill_id` of groups the participant is descendant of or manages)
//		to the given item that the participant may have used to access this item,
//		so path for which the participant has a started attempt (possibly ended/not-allowing-submissions) on every item.
//
//
//		The participant is `participant_id` (if given) or the current user (otherwise).
//
//
//		Paths can contain only items visible to the current user
//		(`can_view`>='content' on every item on the path but the last one and `can_view`>='info' for the last one).
//		The item info (`title` and `language_tag`) in the paths is in the current user's language,
//		or the item's default language (if not available).
//
//
//		Restrictions:
//
//			* if `participant_id` is given, it should be a descendant of a group the current user can manage with `can_watch_members`,
//			* at least one path should exist,
//
//		otherwise the 'forbidden' error is returned.
//	parameters:
//		- name: item_id
//			in: path
//			type: integer
//			format: int64
//			required: true
//		- name: participant_id
//			in: query
//			type: integer
//			format: int64
//	responses:
//		"200":
//			description: OK. Success response with the found item path
//			schema:
//				type: array
//				items:
//					type: array
//					items:
//						"$ref": "#/definitions/breadcrumbElement"
//		"400":
//			"$ref": "#/responses/badRequestResponse"
//		"401":
//			"$ref": "#/responses/unauthorizedResponse"
//		"403":
//			"$ref": "#/responses/forbiddenResponse"
//		"500":
//			"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getBreadcrumbsFromRootsByItemID(w http.ResponseWriter, r *http.Request) service.APIError {
	itemID, err := service.ResolveURLQueryPathInt64Field(r, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	return srv.getBreadcrumbsFromRoots(w, r, itemID)
}

// swagger:operation GET /items/by-text-id/{text_id}/breadcrumbs-from-roots items itemBreadcrumbsFromRootsByTextIdGet
//
//	---
//	summary: List all possible breadcrumbs for a started item using `text_id`
//	description: >
//
//		Same as [/items/{item_id}/breadcrumbs-from-roots](#tag/items/operation/itemBreadcrumbsFromRootsGet)
//		but using `text_id`.
//
//		* `text_id` must be URL-encoded.
//
//	parameters:
//		- name: text_id
//			in: path
//			type: string
//			required: true
//		- name: participant_id
//			in: query
//			type: integer
//			format: int64
//
//	responses:
//		"200":
//			description: OK. Success response with the found item path
//			schema:
//				type: array
//				items:
//					type: array
//					items:
//						"$ref": "#/definitions/breadcrumbElement"
//		"400":
//	  	"$ref": "#/responses/badRequestResponse"
//		"401":
//	  	"$ref": "#/responses/unauthorizedResponse"
//		"403":
//	  	"$ref": "#/responses/forbiddenResponse"
//		"500":
//	  	"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getBreadcrumbsFromRootsByTextID(w http.ResponseWriter, r *http.Request) service.APIError {
	textID := chi.URLParam(r, "text_id")

	// we wouldn't be here if the url weren't valid.
	decodedTextID, _ := url.QueryUnescape(textID)

	store := srv.GetStore(r)
	itemID, err := store.Items().GetItemIDFromTextID(decodedTextID)
	if err != nil {
		return service.ErrInvalidRequest(errors.New("no item found with text_id"))
	}

	return srv.getBreadcrumbsFromRoots(w, r, itemID)
}

func (srv *Service) getBreadcrumbsFromRoots(w http.ResponseWriter, r *http.Request, itemID int64) service.APIError {
	store := srv.GetStore(r)
	user := srv.GetUser(r)

	participantID := user.GroupID
	if len(r.URL.Query()["participant_id"]) != 0 {
		var err error
		participantID, err = service.ResolveURLQueryGetInt64Field(r, "participant_id")
		if err != nil {
			return service.ErrInvalidRequest(err)
		}

		if !user.CanWatchGroupMembers(store, participantID) {
			return service.InsufficientAccessRightsError
		}
	}

	breadcrumbs := findItemBreadcrumbs(store, participantID, user, itemID)
	if breadcrumbs == nil {
		return service.InsufficientAccessRightsError
	}
	render.Respond(w, r, breadcrumbs)
	return service.NoError
}

func findItemBreadcrumbs(store *database.DataStore, participantID int64, user *database.User, itemID int64) [][]breadcrumbElement {
	participantAncestors := store.ActiveGroupAncestors().Where("child_group_id = ?", participantID).
		Joins("JOIN `groups` ON groups.id = groups_ancestors_active.ancestor_group_id").
		Select("groups.id, root_activity_id, root_skill_id")
	groupsManagedByParticipant := store.ActiveGroupAncestors().ManagedByUser(user).
		Joins("JOIN `groups` ON groups.id = groups_ancestors_active.child_group_id").
		Select("groups.id, root_activity_id, root_skill_id")
	groupsWithRootItems := participantAncestors.Union(groupsManagedByParticipant.SubQuery())

	visibleItems := store.Permissions().MatchingUserAncestors(user).
		Where("permissions.can_view_generated_value >= ?", store.PermissionsGranted().ViewIndexByName("info")).
		Joins("JOIN items ON items.id = permissions.item_id").
		Select("items.id, requires_explicit_entry, MAX(can_view_generated_value) AS can_view_generated_value").
		Group("items.id")

	canViewContentIndex := store.PermissionsGranted().ViewIndexByName("content")

	var pathStrings []string
	service.MustNotBeError(store.Raw(`
			WITH RECURSIVE paths (path, last_item_id, last_attempt_id) AS (
				WITH groups_with_root_items AS (?),
					visible_items AS (?),
					root_items AS (
						SELECT visible_items.id AS id FROM groups_with_root_items JOIN visible_items ON visible_items.id = root_activity_id
						UNION
						SELECT visible_items.id FROM groups_with_root_items JOIN visible_items ON visible_items.id = root_skill_id),
					item_ancestors AS (
						SELECT visible_items.id, requires_explicit_entry, can_view_generated_value
						FROM items_ancestors
						JOIN visible_items ON visible_items.id = items_ancestors.ancestor_item_id WHERE child_item_id = ?
						UNION
						SELECT id, requires_explicit_entry, can_view_generated_value FROM visible_items WHERE id = ?),
					root_ancestors AS (
						SELECT item_ancestors.id, requires_explicit_entry, can_view_generated_value
						FROM item_ancestors
						JOIN root_items ON root_items.id = item_ancestors.id)
				(SELECT CAST(root_ancestors.id AS CHAR(1024)), root_ancestors.id, attempts.id
				FROM root_ancestors
				JOIN attempts ON attempts.participant_id = ? AND
					(NOT root_ancestors.requires_explicit_entry OR attempts.root_item_id = root_ancestors.id)
				JOIN results ON results.participant_id = attempts.participant_id AND
					attempts.id = results.attempt_id AND results.item_id = root_ancestors.id
				WHERE (root_ancestors.id = ? OR root_ancestors.can_view_generated_value >= ?) AND
					(results.started_at IS NOT NULL))
				UNION
				(SELECT CONCAT(paths.path, '/', item_ancestors.id), item_ancestors.id, attempts.id
				FROM paths
				JOIN items_items ON items_items.parent_item_id = paths.last_item_id
				JOIN item_ancestors ON item_ancestors.id = items_items.child_item_id
				JOIN attempts ON attempts.participant_id = ? AND
					(NOT item_ancestors.requires_explicit_entry OR attempts.root_item_id = item_ancestors.id) AND
					IF(attempts.root_item_id = item_ancestors.id, attempts.parent_attempt_id, attempts.id) = paths.last_attempt_id
				JOIN results ON results.participant_id = attempts.participant_id AND
						attempts.id = results.attempt_id AND results.item_id = item_ancestors.id
				WHERE paths.last_item_id <> ? AND (item_ancestors.id = ? OR item_ancestors.can_view_generated_value >= ?) AND
					(results.started_at IS NOT NULL)))
			SELECT path FROM paths WHERE paths.last_item_id = ? GROUP BY path ORDER BY path`,
		groupsWithRootItems.SubQuery(), visibleItems.SubQuery(), itemID, itemID, participantID, itemID, canViewContentIndex,
		participantID, itemID, itemID, canViewContentIndex, itemID).
		ScanIntoSlices(&pathStrings).Error())

	if len(pathStrings) == 0 {
		return nil
	}

	itemIDsMap := make(map[int64]bool, len(pathStrings))
	breadcrumbs := make([][]breadcrumbElement, 0, len(pathStrings))
	for _, path := range pathStrings {
		ids := strings.Split(path, "/")
		breadcrumb := make([]breadcrumbElement, 0, len(ids))
		for _, id := range ids {
			idInt64, _ := strconv.ParseInt(id, 10, 64)
			itemIDsMap[idInt64] = true
			breadcrumb = append(breadcrumb, breadcrumbElement{ID: idInt64})
		}
		breadcrumbs = append(breadcrumbs, breadcrumb)
	}
	idsList := make([]int64, 0, len(itemIDsMap))
	for id := range itemIDsMap {
		idsList = append(idsList, id)
	}

	var itemsInfo []struct {
		ID          int64
		Title       *string
		Type        *string
		LanguageTag string
	}
	service.MustNotBeError(store.Items().Where("id IN(?)", idsList).
		JoinsUserAndDefaultItemStrings(user).
		Select(`
			id,
			COALESCE(user_strings.title, default_strings.title) AS title,
			COALESCE(user_strings.language_tag, default_strings.language_tag) AS language_tag,
			type
		`).
		Scan(&itemsInfo).Error())

	itemTitles := make(map[int64]*string, len(itemsInfo))
	itemLanguageTags := make(map[int64]string, len(itemsInfo))
	itemType := make(map[int64]*string, len(itemsInfo))
	for _, itemInfo := range itemsInfo {
		itemTitles[itemInfo.ID] = itemInfo.Title
		itemLanguageTags[itemInfo.ID] = itemInfo.LanguageTag
		itemType[itemInfo.ID] = itemInfo.Type
	}

	for breadcrumbsIndex := range breadcrumbs {
		for pathIndex := range breadcrumbs[breadcrumbsIndex] {
			id := breadcrumbs[breadcrumbsIndex][pathIndex].ID
			breadcrumbs[breadcrumbsIndex][pathIndex].Title = itemTitles[id]
			breadcrumbs[breadcrumbsIndex][pathIndex].LanguageTag = itemLanguageTags[id]
			breadcrumbs[breadcrumbsIndex][pathIndex].Type = itemType[id]
		}
	}
	return breadcrumbs
}
