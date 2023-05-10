package items

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
	"github.com/France-ioi/AlgoreaBackend/app/structures"
)

type itemStringCommon struct {
	// required: true
	LanguageTag string `json:"language_tag"`
	// Nullable
	// required: true
	Title *string `json:"title"`
	// Nullable
	// required: true
	ImageURL *string `json:"image_url"`
	// Nullable; only if `can_view` >= 'content'
	Subtitle *string `json:"subtitle"`
	// Nullable; only if `can_view` >= 'content'
	Description *string `json:"description"`
}

type itemStringRootNodeWithSolutionAccess struct {
	// Nullable; only if the user has access to solutions
	EduComment *string `json:"edu_comment"`
}

// Item-related strings (from `items_strings`) in the user's default language (preferred) or the item's language.
type itemStringRoot struct {
	*itemStringCommon
	*itemStringRootNodeWithSolutionAccess
}

type commonItemFields struct {
	// items

	// required: true
	ID int64 `json:"id,string"`
	// required: true
	// enum: Chapter,Task,Skill
	Type string `json:"type"`
	// required: true
	DisplayDetailsInParent bool `json:"display_details_in_parent"`
	// required: true
	// enum: None,All,AllButOne,Categories,One,Manual
	ValidationType string `json:"validation_type"`
	// required: true
	RequiresExplicitEntry bool `json:"requires_explicit_entry"`
	// required: true
	AllowsMultipleAttempts bool `json:"allows_multiple_attempts"`
	// required: true
	// enum: User,Team
	EntryParticipantType string `json:"entry_participant_type"`
	// Nullable
	// pattern: ^\d{1,3}:[0-5]?\d:[0-5]?\d$
	// example: 838:59:59
	// required: true
	Duration *string `json:"duration"`
	// required: true
	NoScore bool `json:"no_score"`
	// required: true
	DefaultLanguageTag string `json:"default_language_tag"`

	// required: true
	Permissions structures.ItemPermissions `json:"permissions"`
}

type itemRootNodeNotChapterFields struct {
	// Nullable; only if not a chapter
	URL *string `json:"url"`
	// Nullable; only if not a chapter
	Options *string `json:"options"`
	// only if not a chapter
	UsesAPI bool `json:"uses_api"`
	// only if not a chapter
	HintsAllowed bool `json:"hints_allowed"`
}

// only if watched_group_id is given.
type itemResponseWatchedGroupItemInfo struct {
	Permissions *structures.ItemPermissions `json:"permissions,omitempty"`
	// Average score of all "end-members" within the watched group
	// (or of the watched group itself if it is a user or a team).
	// The score of an "end-member" is the max of his `results.score` or 0 if no results.
	// The field is only shown when the current user has 'can_watch' > 'none' permission on the item.
	AverageScore *float32 `json:"average_score,omitempty"`
}

// swagger:model itemResponse
type itemResponse struct {
	*commonItemFields

	// required: true
	// enum: All,Half,One,None
	EntryMinAdmittedMembersRatio string `json:"entry_min_admitted_members_ratio"`
	// required: true
	EntryFrozenTeams bool `json:"entry_frozen_teams"`
	// required: true
	EntryMaxTeamSize int32 `json:"entry_max_team_size"`
	// required: true
	PromptToJoinGroupByCode bool `json:"prompt_to_join_group_by_code"`
	// required: true
	TitleBarVisible bool `json:"title_bar_visible"`
	// Nullable
	// required: true
	TextID *string `json:"text_id"`
	// required: true
	ReadOnly bool `json:"read_only"`
	// required: true
	// enum: forceYes,forceNo,default
	FullScreen string `json:"full_screen"`
	// required: true
	// enum: List,Grid
	ChildrenLayout string `json:"children_layout"`
	// required: true
	ShowUserInfos bool `json:"show_user_infos"`
	// required: true
	EnteringTimeMin time.Time `json:"entering_time_min"`
	// required: true
	EnteringTimeMax time.Time `json:"entering_time_max"`

	// required: true
	SupportedLanguageTags []string `json:"supported_language_tags"`

	// max among all attempts of the user (or of the team given in `{as_team_id}`)
	// required: true
	BestScore float32 `json:"best_score"`

	// required: true
	String itemStringRoot `json:"string"`

	*itemRootNodeNotChapterFields

	WatchedGroup *itemResponseWatchedGroupItemInfo `json:"watched_group,omitempty"`
}

// swagger:operation GET /items/{item_id} items itemView
//
//		---
//		summary: Get an item
//		description: Returns data related to the specified item,
//	             and the current user's (or the team's given in `{as_team_id}`) permissions on it
//	             (from tables `items`, `items_string`, `permissions_generated`).
//
//
//	             * If the specified item is not visible by the current user (or the team given in `as_team_id`),
//	               the 'not found' response is returned.
//
//	             * If `{language_tag}` is given, but there is no items_strings row for the `{item_id}` and `{language_tag}`,
//	               the 'not found' response is returned as well.
//
//	             * If `as_team_id` is given, it should be a user's parent team group,
//	               otherwise the "forbidden" error is returned.
//
//	             * If `{watched_group_id}` is given, the user should ba a manager of the group with the 'can_watch_members' permission,
//	               otherwise the "forbidden" error is returned. Permissions of the watched group are only shown if the current user
//	               can watch the item or grant permissions to both the watched group and the item.
//		parameters:
//			- name: item_id
//				in: path
//				type: integer
//				format: int64
//				required: true
//			- name: as_team_id
//				in: query
//				type: integer
//				format: int64
//			- name: watched_group_id
//				in: query
//				type: integer
//				format: int64
//			- name: language_tag
//				in: query
//				type: string
//		responses:
//			"200":
//				description: OK. Success response with item data
//				schema:
//					"$ref": "#/definitions/itemResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"404":
//				"$ref": "#/responses/notFoundResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getItem(rw http.ResponseWriter, httpReq *http.Request) service.APIError {
	itemID, err := service.ResolveURLQueryPathInt64Field(httpReq, "item_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	user := srv.GetUser(httpReq)
	participantID := service.ParticipantIDFromContext(httpReq.Context())

	watchedGroupID, watchedGroupIDSet, apiError := srv.ResolveWatchedGroupID(httpReq)
	if apiError != service.NoError {
		return apiError
	}

	var languageTag string
	var languageTagSet bool
	if len(httpReq.URL.Query()["language_tag"]) != 0 {
		languageTag = httpReq.URL.Query().Get("language_tag")
		languageTagSet = true
	}

	store := srv.GetStore(httpReq)
	rawData := getRawItemData(store.Items(), itemID, participantID, languageTag, languageTagSet, user, watchedGroupID, watchedGroupIDSet)
	if rawData == nil {
		return service.ErrNotFound(errors.New("insufficient access rights on the given item id or the item doesn't exist"))
	}

	permissionGrantedStore := store.PermissionsGranted()
	response := constructItemResponseFromDBData(rawData, permissionGrantedStore, watchedGroupIDSet)

	render.Respond(rw, httpReq, response)
	return service.NoError
}

// rawItem represents the getItem service data returned from the DB.
type rawItem struct {
	*RawCommonItemFields

	// items
	TitleBarVisible              bool
	ReadOnly                     bool
	FullScreen                   string
	ChildrenLayout               string
	ShowUserInfos                bool
	EntryMinAdmittedMembersRatio string
	EntryFrozenTeams             bool
	EntryMaxTeamSize             int32
	PromptToJoinGroupByCode      bool
	TextID                       *string
	URL                          *string // only if not a chapter
	Options                      *string // only if not a chapter
	UsesAPI                      bool    // only if not a chapter
	HintsAllowed                 bool    // only if not a chapter
	BestScore                    float32

	// items_strings
	SupportedLanguageTags string

	// from items_strings: in the user’s default language or (if not available) default language of the item
	StringLanguageTag string  `gorm:"column:language_tag"`
	StringTitle       *string `gorm:"column:title"`
	StringImageURL    *string `gorm:"column:image_url"`
	StringSubtitle    *string `gorm:"column:subtitle"`
	StringDescription *string `gorm:"column:description"`
	StringEduComment  *string `gorm:"column:edu_comment"`

	WatchedGroupPermissions        *database.RawGeneratedPermissionFields `gorm:"embedded;embeddedPrefix:watched_group_permissions_"`
	CanViewWatchedGroupPermissions bool
	WatchedGroupAverageScore       float32
	CanWatchForGroupResults        bool
}

// getRawItemData reads data needed by the getItem service from the DB and returns an array of rawItem's.
func getRawItemData(s *database.ItemStore, rootID, groupID int64, languageTag string, languageTagSet bool, user *database.User,
	watchedGroupID int64, watchedGroupIDSet bool,
) *rawItem {
	var result rawItem

	columnsBuffer := bytes.NewBufferString(`
		items.id AS id,
		items.type,
		items.display_details_in_parent,
		items.validation_type,
		items.entry_min_admitted_members_ratio,
		items.entry_frozen_teams,
		items.entry_max_team_size,
		items.entering_time_min,
		items.entering_time_max,
		items.allows_multiple_attempts,
		items.entry_participant_type,
		items.duration,
		items.no_score,
		items.text_id,
		items.default_language_tag,
		IFNULL((SELECT GROUP_CONCAT(language_tag ORDER BY language_tag)
		        FROM items_strings WHERE item_id = items.id), '') AS supported_language_tags,
		items.prompt_to_join_group_by_code,
		items.title_bar_visible,
		items.read_only,
		items.full_screen,
		items.children_layout,
		items.show_user_infos,
		items.url,
		items.options,
		items.requires_explicit_entry,
		IF(items.type <> 'Chapter', items.uses_api, NULL) AS uses_api,
		IF(items.type <> 'Chapter', items.hints_allowed, NULL) AS hints_allowed,
		permissions.can_view_generated_value, permissions.can_grant_view_generated_value, permissions.can_watch_generated_value,
		permissions.can_edit_generated_value, permissions.is_owner_generated,
		IFNULL(
			(SELECT MAX(results.score_computed) AS best_score
			 FROM results
			 WHERE results.item_id = items.id AND results.participant_id = ?), 0) AS best_score`)

	columnValues := []interface{}{groupID}
	query := s.ByID(rootID).
		JoinsPermissionsForGroupToItemsWherePermissionAtLeast(groupID, "view", "info")

	if watchedGroupIDSet {
		watchedGroupPermissionsQuery := database.NewDataStore(s.New()).Permissions().
			AggregatedPermissionsForItems(watchedGroupID).
			Where("permissions.item_id = items.id")
		query = query.Joins(
			"LEFT JOIN LATERAL ? AS watched_group_permissions ON watched_group_permissions.item_id = items.id",
			watchedGroupPermissionsQuery.SubQuery())

		currentUserCanGrantAccessToTheWatchedGroupQuery := s.
			GroupAncestors().
			ManagedByUser(user).
			Where("groups_ancestors.child_group_id = ?", watchedGroupID).
			Where("can_grant_group_access").Select("1").Limit(1)

		_, err := columnsBuffer.WriteString(`,
			IFNULL(watched_group_permissions.can_view_generated_value, 1) AS watched_group_permissions_can_view_generated_value,
			IFNULL(watched_group_permissions.can_grant_view_generated_value, 1) AS watched_group_permissions_can_grant_view_generated_value,
			IFNULL(watched_group_permissions.can_watch_generated_value, 1) AS watched_group_permissions_can_watch_generated_value,
			IFNULL(watched_group_permissions.can_edit_generated_value, 1) AS watched_group_permissions_can_edit_generated_value,
			IFNULL(watched_group_permissions.is_owner_generated, 0) watched_group_permissions_is_owner_generated`)
		service.MustNotBeError(err)

		if user.GroupID != groupID {
			// as_team_id is given, so `permissions` are related to the team,
			// and we need to join permissions of the current user explicitly to determine
			// if the current user is able to view the average score and permissions of the watched group
			currentUserPermissionsQuery := database.NewDataStore(s.New()).Permissions().
				AggregatedPermissionsForItems(user.GroupID).
				Where("permissions.item_id = items.id")
			query = query.Joins(
				"LEFT JOIN LATERAL ? AS user_permissions ON user_permissions.item_id = items.id",
				currentUserPermissionsQuery.SubQuery())
			_, err = columnsBuffer.WriteString(`,
				user_permissions.can_watch_generated_value > ? OR (
					user_permissions.can_grant_view_generated_value > ? AND ?
				) AS can_view_watched_group_permissions,
				user_permissions.can_watch_generated_value > ? AS can_watch_for_group_results`)
			service.MustNotBeError(err)
		} else {
			_, err = columnsBuffer.WriteString(`,
				permissions.can_watch_generated_value > ? OR (
					permissions.can_grant_view_generated_value > ? AND ?
				) AS can_view_watched_group_permissions,
				permissions.can_watch_generated_value > ? AS can_watch_for_group_results`)
			service.MustNotBeError(err)
		}
		permissionsGrantedStore := s.PermissionsGranted()
		columnValues = append(columnValues,
			permissionsGrantedStore.WatchIndexByName("none"),
			permissionsGrantedStore.GrantViewIndexByName("none"),
			currentUserCanGrantAccessToTheWatchedGroupQuery.SubQuery(),
			permissionsGrantedStore.WatchIndexByName("none"))

		_, err = columnsBuffer.WriteString(`,
			(SELECT IFNULL(AVG(score), 0) AS avg_score FROM ? AS stats) AS watched_group_average_score`)
		service.MustNotBeError(err)
		columnValues = append(columnValues,
			s.ActiveGroupAncestors().
				Select("participant.id").
				Joins(`
					JOIN `+"`groups`"+` AS participant
						ON participant.id = groups_ancestors_active.child_group_id AND participant.type IN ('User', 'Team')`).
				Where("groups_ancestors_active.ancestor_group_id = ?", watchedGroupID).
				Joins(`
					LEFT JOIN (
						SELECT participant_id, score_computed FROM results
						WHERE results.item_id = items.id
					) AS results ON results.participant_id = participant.id`).
				Select("MAX(IFNULL(results.score_computed, 0)) AS score").
				Group("participant.id").SubQuery())
	}

	if languageTagSet {
		query = query.Joins("JOIN items_strings ON items_strings.item_id = items.id AND items_strings.language_tag = ?", languageTag)
		_, err := columnsBuffer.WriteString(`,
			items_strings.language_tag, items_strings.title, items_strings.image_url, items_strings.subtitle,
			items_strings.description, items_strings.edu_comment`)
		service.MustNotBeError(err)
	} else {
		query = query.JoinsUserAndDefaultItemStrings(user)
		_, err := columnsBuffer.WriteString(`,
			COALESCE(user_strings.language_tag, default_strings.language_tag) AS language_tag,
			IF(user_strings.language_tag IS NULL, default_strings.title, user_strings.title) AS title,
			IF(user_strings.language_tag IS NULL, default_strings.image_url, user_strings.image_url) AS image_url,
			IF(user_strings.language_tag IS NULL, default_strings.subtitle, user_strings.subtitle) AS subtitle,
			IF(user_strings.language_tag IS NULL, default_strings.description, user_strings.description) AS description,
			IF(user_strings.language_tag IS NULL, default_strings.edu_comment, user_strings.edu_comment) AS edu_comment`)
		service.MustNotBeError(err)
	}

	// nolint:gosec
	query = query.Select(columnsBuffer.String(), columnValues...)

	err := query.Take(&result).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	service.MustNotBeError(err)
	return &result
}

func constructItemResponseFromDBData(
	rawData *rawItem, permissionGrantedStore *database.PermissionGrantedStore, watchedGroupIDSet bool,
) *itemResponse {
	result := &itemResponse{
		commonItemFields: rawData.asItemCommonFields(permissionGrantedStore),
		String: itemStringRoot{
			itemStringCommon: constructItemStringCommon(rawData),
		},
		EntryMinAdmittedMembersRatio: rawData.EntryMinAdmittedMembersRatio,
		EntryFrozenTeams:             rawData.EntryFrozenTeams,
		EntryMaxTeamSize:             rawData.EntryMaxTeamSize,
		PromptToJoinGroupByCode:      rawData.PromptToJoinGroupByCode,
		TitleBarVisible:              rawData.TitleBarVisible,
		TextID:                       rawData.TextID,
		ReadOnly:                     rawData.ReadOnly,
		FullScreen:                   rawData.FullScreen,
		ChildrenLayout:               rawData.ChildrenLayout,
		ShowUserInfos:                rawData.ShowUserInfos,
		EnteringTimeMin:              time.Time(rawData.EnteringTimeMin),
		EnteringTimeMax:              time.Time(rawData.EnteringTimeMax),
		BestScore:                    rawData.BestScore,
		SupportedLanguageTags:        strings.Split(rawData.SupportedLanguageTags, ","),
	}

	if rawData.CanViewGeneratedValue == permissionGrantedStore.ViewIndexByName("solution") {
		result.String.itemStringRootNodeWithSolutionAccess = &itemStringRootNodeWithSolutionAccess{
			EduComment: rawData.StringEduComment,
		}
	}
	if rawData.Type != "Chapter" {
		result.itemRootNodeNotChapterFields = &itemRootNodeNotChapterFields{
			URL:          rawData.URL,
			Options:      rawData.Options,
			UsesAPI:      rawData.UsesAPI,
			HintsAllowed: rawData.HintsAllowed,
		}
	}
	if watchedGroupIDSet {
		result.WatchedGroup = &itemResponseWatchedGroupItemInfo{}
		if rawData.CanWatchForGroupResults {
			result.WatchedGroup.AverageScore = &rawData.WatchedGroupAverageScore
		}
		if rawData.CanViewWatchedGroupPermissions {
			result.WatchedGroup.Permissions = rawData.WatchedGroupPermissions.AsItemPermissions(permissionGrantedStore)
		}
	}

	return result
}

func constructItemStringCommon(rawData *rawItem) *itemStringCommon {
	return &itemStringCommon{
		LanguageTag: rawData.StringLanguageTag,
		Title:       rawData.StringTitle,
		ImageURL:    rawData.StringImageURL,
		Subtitle:    rawData.StringSubtitle,
		Description: rawData.StringDescription,
	}
}
