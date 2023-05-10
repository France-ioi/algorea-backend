package items

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:model prerequisiteOrDependencyItem
type prerequisiteOrDependencyItem struct {
	*commonItemFields

	// item_dependencies.score
	// required: true
	DependencyRequiredScore int `json:"dependency_required_score"`
	// item_dependencies.grant_content_view
	// required: true
	DependencyGrantContentView bool `json:"dependency_grant_content_view"`

	// required: true
	String listItemString `json:"string"`

	// max among all attempts of the user (or of the team given in `{as_team_id}`)
	// required: true
	BestScore float32 `json:"best_score"`

	WatchedGroup *itemWatchedGroupStat `json:"watched_group,omitempty"`
}

type rawPrerequisiteOrDependencyItem struct {
	*RawCommonItemFields

	// from items_strings: in the user’s default language or (if not available) default language of the item
	StringLanguageTag string  `gorm:"column:language_tag"`
	StringTitle       *string `gorm:"column:title"`
	StringSubtitle    *string `gorm:"column:subtitle"`
	StringImageURL    *string `gorm:"column:image_url"`

	// max from results of the current participant
	BestScore float32

	// from item_dependencies
	DependencyRequiredScore    int
	DependencyGrantContentView bool

	*RawWatchedGroupStatFields
}

// swagger:operation GET /items/{item_id}/prerequisites items itemPrerequisitesView
//
//		---
//		summary: Get prerequisites for an item
//		description: Lists prerequisite items for the specified item
//	             and the current user's (or the team's given in `as_team_id`) interactions with them
//	             (from tables `items`, `item_dependencies`, `items_string`, `results`, `permissions_generated`).
//	             Only items visible to the current user (or to the `{as_team_id}` team) are shown.
//	             If `{watched_group_id}` is given, some additional info about the given group's results on the items is shown.
//
//
//	             * The current user (or the team given in `as_team_id`) should have at least 'info' permissions on the specified item,
//	               otherwise the 'forbidden' response is returned.
//
//	             * If `as_team_id` is given, it should be a user's parent team group,
//	               otherwise the "forbidden" error is returned.
//
//	             * If `{watched_group_id}` is given, the user should ba a manager of the group with the 'can_watch_members' permission,
//	               otherwise the "forbidden" error is returned.
//		parameters:
//			- name: item_id
//				in: path
//				type: integer
//				format: int64
//				required: true
//			- name: as_team_id
//				in: query
//				type: integer
//			- name: watched_group_id
//				in: query
//				type: integer
//		responses:
//			"200":
//				description: OK. Success response with prerequisite items
//				schema:
//					type: array
//					items:
//						"$ref": "#/definitions/prerequisiteOrDependencyItem"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getItemPrerequisites(rw http.ResponseWriter, httpReq *http.Request) service.APIError {
	return srv.getItemPrerequisitesOrDependencies(rw, httpReq, "dependent_item_id", "item_id")
}

func (srv *Service) getItemPrerequisitesOrDependencies(
	rw http.ResponseWriter, httpReq *http.Request,
	givenColumn, joinToColumn string,
) service.APIError {
	itemID, err := service.ResolveURLQueryPathInt64Field(httpReq, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(httpReq)
	participantID := service.ParticipantIDFromContext(httpReq.Context())
	store := srv.GetStore(httpReq)

	watchedGroupID, watchedGroupIDSet, apiError := srv.ResolveWatchedGroupID(httpReq)
	if apiError != service.NoError {
		return apiError
	}

	found, err := store.Permissions().
		MatchingGroupAncestors(participantID).
		WherePermissionIsAtLeast("view", "info").
		Where("permissions.item_id = ?", itemID).
		HasRows()
	service.MustNotBeError(err)
	if !found {
		return service.InsufficientAccessRightsError
	}

	var rawData []rawPrerequisiteOrDependencyItem
	service.MustNotBeError(
		constructItemListWithoutResultsQuery(
			store, participantID, "info", watchedGroupIDSet, watchedGroupID,
			`items.allows_multiple_attempts, items.id, items.type, items.default_language_tag,
				validation_type, display_details_in_parent, duration, entry_participant_type, no_score,
				can_view_generated_value, can_grant_view_generated_value, can_watch_generated_value, can_edit_generated_value, is_owner_generated,
				score AS dependency_required_score, grant_content_view AS dependency_grant_content_view,
				IFNULL(
					(SELECT MAX(results.score_computed) AS best_score
					FROM results
					WHERE results.item_id = items.id AND results.participant_id = ?), 0) AS best_score,
				COALESCE(user_strings.language_tag, default_strings.language_tag) AS language_tag,
				IF(user_strings.language_tag IS NULL, default_strings.title, user_strings.title) AS title,
				IF(user_strings.image_url IS NULL, default_strings.image_url, user_strings.image_url) AS image_url,
				IF(user_strings.language_tag IS NULL, default_strings.subtitle, user_strings.subtitle) AS subtitle`,
			[]interface{}{participantID},
			func(db *database.DB) *database.DB {
				return db.Joins(
					"JOIN item_dependencies ON item_dependencies."+givenColumn+" = ? AND item_dependencies."+joinToColumn+" = items.id", itemID).
					JoinsUserAndDefaultItemStrings(user)
			},
			func(db *database.DB) *database.DB {
				return db.Joins("JOIN item_dependencies ON item_dependencies."+joinToColumn+" = permissions.item_id").
					Where("item_dependencies."+givenColumn+"= ?", itemID)
			}).
			Order("title, subtitle, id").
			Scan(&rawData).Error())

	response := prerequisiteOrDependencyItemsFromRawData(rawData, watchedGroupIDSet, store.PermissionsGranted())

	render.Respond(rw, httpReq, response)
	return service.NoError
}

func prerequisiteOrDependencyItemsFromRawData(
	rawData []rawPrerequisiteOrDependencyItem, watchedGroupIDSet bool,
	permissionGrantedStore *database.PermissionGrantedStore,
) []prerequisiteOrDependencyItem {
	result := make([]prerequisiteOrDependencyItem, 0, len(rawData))
	for index := range rawData {
		item := prerequisiteOrDependencyItem{
			commonItemFields: rawData[index].RawCommonItemFields.asItemCommonFields(permissionGrantedStore),
			BestScore:        rawData[index].BestScore,
			String: listItemString{
				LanguageTag: rawData[index].StringLanguageTag,
				ImageURL:    rawData[index].StringImageURL,
				Title:       rawData[index].StringTitle,
			},
			DependencyRequiredScore:    rawData[index].DependencyRequiredScore,
			DependencyGrantContentView: rawData[index].DependencyGrantContentView,
		}
		if rawData[index].CanViewGeneratedValue >= permissionGrantedStore.ViewIndexByName("content") {
			item.String.listItemStringNotInfo = &listItemStringNotInfo{Subtitle: rawData[index].StringSubtitle}
		}
		item.WatchedGroup = rawData[index].RawWatchedGroupStatFields.asItemWatchedGroupStat(watchedGroupIDSet, permissionGrantedStore)
		result = append(result, item)
	}
	return result
}
