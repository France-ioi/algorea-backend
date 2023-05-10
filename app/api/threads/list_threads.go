package threads

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:model thread
type thread struct {
	Item        item        `json:"item" gorm:"embedded;embeddedPrefix:item__"`
	Participant participant `json:"participant" gorm:"embedded;embeddedPrefix:participant__"`

	// enum: not_started,waiting_for_participant,waiting_for_trainer,closed
	Status         string         `json:"status"`
	LatestUpdateAt *database.Time `json:"latest_update_at"`
	MessageCount   int            `json:"message_count"`
}

type item struct {
	ID          int64  `json:"id,string"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	LanguageTag string `json:"language_tag"`
}

type participant struct {
	ID        int64  `json:"id,string"`
	Login     string `json:"login"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type listThreadParameters struct {
	WatchedGroupID int64
	IsMine         bool
	ItemID         int64
	Status         string
	LatestUpdateGt *time.Time
}

// swagger:operation GET /thread threads listThreads
//
//	---
//	summary: Service to list the visible threads for a user.
//	description: >
//
//		Service to list the visible threads for a user.
//
//		Exactly one of [`watched_group_id`, `is_mine`] is given.
//
//		Only thread for which the item has can_view>=content for the current user are returned.
//
//		* If `is_mine` = 1, only threads for which the participant is the current-user and for which the current-user has
//			`can_view >= content` on the item are returned.
//		* If `is_mine` = 0, only threads that the current-user
//			[https://france-ioi.github.io/algorea-devdoc/forum/forum-perm/#listing--reading-a-thread](can list) and for which
//			the current-user is NOT the participant are returned.
//		* If `watched_group_id` is given, only threads in which the participant is descendant (including self)
//			of the `watched_group_id` are returned.
//		* If `item_id` is given, only threads for which the `item_id` is or is descendant of the given `item_id` are returned.
//		* `first_name` and `last_name` are only returned for the current user, or if the user approved access to their personal
//			info for some group managed by the current user.
//
//		Validations:
//			* if `watched_group_id` is given: the current-user must be (implicitly or explicitly) a manager
//				with `can_watch_members` on `watched_group_id`.
//
//		Extra:
//			* By default, ordering is by `latest_update_at` DESC.
//			* Filter by `status` by providing the `status` parameter with the filter value.
//			* Filter by greater than `latest_update_at` by providing the `latest_update_gt` with a datetime in UTC8601 format.
//
//	parameters:
//		- name: watched_group_id
//			in: query
//			type: integer
//			format: int64
//		- name: is_mine
//			in: query
//			type: boolean
//		- name: status
//			description: Filter by status
//			in: query
//			type: string
//			enum: [waiting_for_participant,waiting_for_trainer,closed]
//		- name: latest_update_gt
//			description: Only threads where `latest_update_at`>`latest_update_gt`.
//			in: query
//			type: string
//			format: date-time
//		- name: sort
//			in: query
//			default: [-latest_update_at,item_id,participant_id]
//			type: array
//			items:
//				type: string
//				enum: [latest_update_at,-latest_update_at,item_id,-item_id,participant_id,-participant_id]
//		- name: from.item_id
//			description: >
//				Start the page from the thread next to the thread with `threads.item.id`=`{from.item_id}`.
//				When provided, from.participant_id should be provided too.
//			in: query
//			type: integer
//		- name: from.participant_id
//			description: >
//				Start the page from the thread next to the thread with `threads.participant.id`=`{from.participant_id}`.
//				When provided, from.item_id should be provided too.
//			in: query
//			type: integer
//		- name: limit
//			description: Display the first N threads
//			in: query
//			type: integer
//			maximum: 1000
//			default: 500
//	responses:
//
//		"200":
//			description: OK. Threads data
//			schema:
//				type: array
//				items:
//					"$ref": "#/definitions/thread"
//		"400":
//			"$ref": "#/responses/badRequestResponse"
//		"401":
//			"$ref": "#/responses/unauthorizedResponse"
//		"403":
//			"$ref": "#/responses/forbiddenResponse"
//		"500":
//			"$ref": "#/responses/internalErrorResponse"
func (srv *Service) listThreads(rw http.ResponseWriter, r *http.Request) service.APIError {
	params, apiError := srv.resolveListThreadParameters(r)
	if apiError != service.NoError {
		return apiError
	}

	var queryDB *database.DB
	queryDB, apiError = srv.constructListThreadsQuery(r, params)
	if apiError != service.NoError {
		return apiError
	}

	var threads []thread
	err := queryDB.Scan(&threads).Error()
	service.MustNotBeError(err)

	render.Respond(rw, r, threads)
	return service.NoError
}

func (srv *Service) constructListThreadsQuery(r *http.Request, params listThreadParameters) (*database.DB, service.APIError) {
	user := srv.GetUser(r)
	store := srv.GetStore(r)

	query := store.Threads().
		JoinsItem().
		JoinsUserParticipant()

	if params.ItemID != 0 {
		query = query.NewThreadStore(query.
			WhereItemsAreSelfOrDescendantsOf(params.ItemID),
		)
	}

	if params.Status != "" {
		query = query.NewThreadStore(query.
			Where("threads.status = ?", params.Status),
		)
	}

	if params.LatestUpdateGt != nil {
		query = query.NewThreadStore(query.
			Where("threads.latest_update_at > ?", params.LatestUpdateGt),
		)
	}

	switch {
	case params.WatchedGroupID != 0:
		query = query.WhereParticipantIsInGroup(params.WatchedGroupID)

	case params.IsMine:
		query = query.NewThreadStore(query.
			Where("threads.participant_id = ?", user.GroupID),
		)

	case !params.IsMine:
		// The user needs to:
		// - be allowed to view the item, AND
		// - not be the participant of the thread, AND
		// - one of the following conditions:
		//		* [canWatchAnswerQuery] have can_watch>=answer permission on the item, OR:
		//		* [userCanHelpQuery] The conditions of "WhereUserCanHelp"

		// It doesn't seem to be very efficient to do this. We could try to leverage the fact that MatchingGroupAncestors
		// is used in both canWatchAnswerQuery and WhereItemsAreVisible if we measure perf issues.
		canWatchAnswerQuery := store.Threads().
			Select("items.id").
			WhereUserHasPermissionOnItems(user, "watch", "answer").
			SubQuery()

		userCanHelpQuery := store.Threads().
			WhereUserCanHelp(user).
			Select("threads.item_id, threads.participant_id").
			SubQuery()

		query = query.NewThreadStore(query.
			Where("threads.participant_id != ?", user.GroupID).
			Where("threads.item_id IN (?) OR (threads.item_id, threads.participant_id) IN (?)", canWatchAnswerQuery, userCanHelpQuery),
		)
	}

	queryDB := query.
		WhereItemsContentAreVisible(user.GroupID).
		JoinsUserAndDefaultItemStrings(user).
		WithPersonalInfoViewApprovals(user).
		Select(`
			items.id AS item__id,
			items.type AS item__type,
			COALESCE(user_strings.language_tag, default_strings.language_tag) AS item__language_tag,
			COALESCE(user_strings.title, default_strings.title) AS item__title,
			threads.participant_id AS participant__id,
			IF(threads.participant_id = ? OR personal_info_view_approvals.approved, users.first_name, NULL) AS participant__first_name,
			IF(threads.participant_id = ? OR personal_info_view_approvals.approved, users.last_name, NULL) AS participant__last_name,
			users.login AS participant__login,
			threads.status AS status,
			threads.message_count AS message_count,
			threads.latest_update_at AS latest_update_at
		`, user.GroupID, user.GroupID)

	var apiError service.APIError
	queryDB, apiError = applySortingAndPaging(r, queryDB)
	if apiError != service.NoError {
		return queryDB, apiError
	}

	return queryDB, service.NoError
}

func applySortingAndPaging(r *http.Request, queryDB *database.DB) (*database.DB, service.APIError) {
	queryDB = service.NewQueryLimiter().Apply(r, queryDB)

	var apiError service.APIError
	queryDB, apiError = service.ApplySortingAndPaging(r, queryDB, &service.SortingAndPagingParameters{
		Fields: service.SortingAndPagingFields{
			"latest_update_at": {ColumnName: "threads.latest_update_at"},
			"item_id":          {ColumnName: "items.id"},
			"participant_id":   {ColumnName: "threads.participant_id"},
		},
		DefaultRules: "-latest_update_at",
		TieBreakers: service.SortingAndPagingTieBreakers{
			"item_id":        service.FieldTypeInt64,
			"participant_id": service.FieldTypeInt64,
		},
	})

	return queryDB, apiError
}

func (srv *Service) resolveListThreadParameters(r *http.Request) (params listThreadParameters, apiError service.APIError) {
	var watchedGroupOK bool
	params.WatchedGroupID, watchedGroupOK, apiError = srv.ResolveWatchedGroupID(r)
	if apiError != service.NoError {
		return params, apiError
	}

	var isMineError error
	params.IsMine, isMineError = service.ResolveURLQueryGetBoolField(r, "is_mine")

	if watchedGroupOK && isMineError == nil {
		return params, service.ErrInvalidRequest(errors.New("must not provide watched_group_id and is_mine at the same time"))
	}
	if !watchedGroupOK && isMineError != nil {
		return params, service.ErrInvalidRequest(errors.New("one of watched_group_id or is_mine must be given"))
	}

	var err error

	if service.URLQueryPathHasField(r, "item_id") {
		params.ItemID, err = service.ResolveURLQueryGetInt64Field(r, "item_id")
		if err != nil {
			return params, service.ErrInvalidRequest(err)
		}
	}

	params, apiError = resolveFilterParameters(r, params)
	if apiError != service.NoError {
		return params, apiError
	}

	return params, service.NoError
}

func resolveFilterParameters(r *http.Request, params listThreadParameters) (listThreadParameters, service.APIError) {
	var err error

	params.Status, err = service.ResolveURLQueryGetStringField(r, "status")
	if err != nil {
		params.Status = ""
	}

	if service.URLQueryPathHasField(r, "latest_update_gt") {
		latestUpdateGt, err := service.ResolveURLQueryGetTimeField(r, "latest_update_gt")
		if err != nil {
			return params, service.ErrInvalidRequest(err)
		}

		params.LatestUpdateGt = &latestUpdateGt
	}

	return params, service.NoError
}
