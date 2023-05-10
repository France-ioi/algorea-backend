package contests

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/render"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation PUT /contests/{item_id}/groups/{group_id}/additional-times contests contestSetAdditionalTime
//
//		---
//		summary: Set additional time for a contest
//		description: >
//	               For the input group and item, sets the `groups_contest_items.additional_time` to the `time` value.
//	               If there is no `groups_contest_items` for the given `group_id`, `item_id` and the `seconds` != 0, creates it
//	               (with default values in other columns).
//	               If no `groups_contest_items` and `seconds` == 0, succeed without doing any change.
//
//
//	               `groups_groups.expires_at` & `attempts.allows_submissions_until` (for the latest attempt) of affected
//	               `items.participants_group_id` members is set to
//	               `results.started_at` + `items.duration` + total additional time.
//
//
//	               Restrictions:
//	                 * `item_id` should be a timed contest;
//	                 * the authenticated user should have `can_view` >= 'content' on the input item;
//	                 * the authenticated user should have `can_grant_view` >= 'enter' on the input item;
//	                 * the authenticated user should have `can_watch` >= 'result' on the input item;
//	                 * the authenticated user should be a manager of the `group_id`
//	                   with `can_grant_group_access` and `can_watch_members` permissions;
//	                 * if the contest is team-only (`items.entry_participant_type` = 'Team'), then the group should not be a user.
//
//	               Otherwise, the "Forbidden" response is returned.
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
//			- name: seconds
//				description: additional time in seconds (can be negative)
//				in: query
//				type: integer
//				minimum: -3020399
//				maximum: 3020399
//				required: true
//		responses:
//			"200":
//				"$ref": "#/responses/updatedResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) setAdditionalTime(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)
	store := srv.GetStore(r)

	itemID, groupID, seconds, apiError := srv.getParametersForSetAdditionalTime(r)
	if apiError != service.NoError {
		return apiError
	}

	var groupType string
	err := store.Groups().ManagedBy(user).Where("groups.id = ?", groupID).
		Having("MAX(can_grant_group_access) AND MAX(can_watch_members)").
		Group("groups.id").
		PluckFirst("groups.type", &groupType).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return service.InsufficientAccessRightsError
	}
	service.MustNotBeError(err)

	var contestInfo struct {
		DurationInSeconds   int64
		IsTeamOnlyContest   bool
		ParticipantsGroupID int64
	}

	err = store.InTransaction(func(store *database.DataStore) error {
		err = store.Items().ContestManagedByUser(itemID, user).WithWriteLock().
			Select(`
				TIME_TO_SEC(items.duration) AS duration_in_seconds,
				items.entry_participant_type = 'Team' AS is_team_only_contest,
				items.participants_group_id`).
			Take(&contestInfo).Error()
		if errors.Is(err, gorm.ErrRecordNotFound) || (contestInfo.IsTeamOnlyContest && groupType == "User") {
			apiError = service.InsufficientAccessRightsError
			return apiError.Error
		}
		service.MustNotBeError(err)

		setAdditionalTimeForGroupInContest(store, groupID, itemID, contestInfo.ParticipantsGroupID,
			contestInfo.DurationInSeconds, seconds)
		return nil
	})
	if apiError != service.NoError {
		return apiError
	}
	service.MustNotBeError(err)

	render.Respond(w, r, service.UpdateSuccess(nil))
	return service.NoError
}

func (srv *Service) getParametersForSetAdditionalTime(r *http.Request) (itemID, groupID, seconds int64, apiError service.APIError) {
	itemID, err := service.ResolveURLQueryPathInt64Field(r, "item_id")
	if err != nil {
		return 0, 0, 0, service.ErrInvalidRequest(err)
	}
	groupID, err = service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return 0, 0, 0, service.ErrInvalidRequest(err)
	}
	seconds, err = service.ResolveURLQueryGetInt64Field(r, "seconds")
	if err != nil {
		return 0, 0, 0, service.ErrInvalidRequest(err)
	}
	const maxSeconds = 838*3600 + 59*60 + 59
	// 838:59:59 is the maximum possible TIME value in MySQL
	if seconds < -maxSeconds || maxSeconds < seconds {
		return 0, 0, 0, service.ErrInvalidRequest(fmt.Errorf("'seconds' should be between %d and %d", -maxSeconds, maxSeconds))
	}
	return itemID, groupID, seconds, service.NoError
}

func setAdditionalTimeForGroupInContest(
	store *database.DataStore, groupID, itemID, participantsGroupID, durationInSeconds, additionalTimeInSeconds int64,
) {
	groupContestItemStore := store.GroupContestItems()
	scope := groupContestItemStore.Where("group_id = ?", groupID).Where("item_id = ?", itemID)
	found, err := scope.WithWriteLock().HasRows()
	service.MustNotBeError(err)
	if found {
		service.MustNotBeError(scope.UpdateColumn("additional_time",
			gorm.Expr("SEC_TO_TIME(?)", additionalTimeInSeconds)).Error())
	} else if additionalTimeInSeconds != 0 {
		service.MustNotBeError(groupContestItemStore.Exec(
			"INSERT INTO groups_contest_items (group_id, item_id, additional_time) VALUES(?, ?, SEC_TO_TIME(?))",
			groupID, itemID, additionalTimeInSeconds).Error())
	}

	service.MustNotBeError(store.Exec("DROP TEMPORARY TABLE IF EXISTS new_expires_at").Error())
	service.MustNotBeError(store.Exec(`
		CREATE TEMPORARY TABLE new_expires_at (
			child_group_id BIGINT(20) NOT NULL,
			expires_at DATETIME NOT NULL,
			PRIMARY KEY child_group_id (child_group_id)
		)`).Error())
	service.MustNotBeError(store.Exec(`
		INSERT INTO new_expires_at ?`,
		// For each of groups participating/participated in the contest ...
		store.GroupGroups().
			Where("groups_groups.parent_group_id = ?", participantsGroupID).
			// ... that are descendants of `groupID` (so affected by the change) ...
			Joins(`
				JOIN groups_ancestors_active AS changed_group_descendants
					ON changed_group_descendants.child_group_id = groups_groups.child_group_id AND
						changed_group_descendants.ancestor_group_id = ?`, groupID).
			// ... and have entered the contest ...
			Joins(`
				JOIN results AS contest_participations
					ON contest_participations.participant_id = groups_groups.child_group_id AND
						contest_participations.started_at IS NOT NULL AND
						contest_participations.item_id = ?`, itemID).
			// ... and the attempt is not ended, ...
			Joins(`
				JOIN attempts
					ON attempts.participant_id = contest_participations.participant_id AND
					   attempts.id = contest_participations.attempt_id AND
					   attempts.ended_at IS NULL`).
			// ... we get all the ancestors to calculate the total additional time
			Joins("JOIN groups_ancestors_active ON groups_ancestors_active.child_group_id = groups_groups.child_group_id").
			Joins(`
				JOIN groups_contest_items
					ON groups_contest_items.group_id = groups_ancestors_active.ancestor_group_id AND
						groups_contest_items.item_id = contest_participations.item_id`).
			Group("groups_groups.child_group_id").
			Select(`
				groups_groups.child_group_id,
				DATE_ADD(
					MIN(contest_participations.started_at),
					INTERVAL (? + IFNULL(SUM(TIME_TO_SEC(groups_contest_items.additional_time)), 0)) SECOND
				) AS expires_at`, durationInSeconds).
			WithWriteLock().QueryExpr()).Error())

	// we always modify groups_groups.expires_at, no matter if it has been expired or not
	result := store.Exec(`
		UPDATE groups_groups
		JOIN new_expires_at
			ON new_expires_at.child_group_id = groups_groups.child_group_id
		SET groups_groups.expires_at = new_expires_at.expires_at
		WHERE groups_groups.parent_group_id = ?`, participantsGroupID)
	service.MustNotBeError(result.Error())

	groupsGroupsModified := result.RowsAffected() > 0

	// We are assuming here that a participant has only at most one ongoing participation at a moment.
	// This assumption impacts, for instance, this scenario:
	//
	//   * a user starts a first attempt at 2:00 which ends at 3:00,
	//   * the user starts a second attempt at 3:01 which will end at 4:01,
	//   * at 3:05, an admin adds 15min to the contest -> only the second attempt gets the 15m extra.
	//
	// We only update attempts.allows_submission_until if the participation is active or
	// if the change makes it active.
	service.MustNotBeError(store.Exec(`
		UPDATE attempts
		JOIN new_expires_at
			ON new_expires_at.child_group_id = attempts.participant_id
		SET attempts.allows_submissions_until = new_expires_at.expires_at
		WHERE
			attempts.root_item_id = ? AND
			attempts.id =
				(SELECT id FROM (
					SELECT MAX(id) AS id FROM attempts WHERE participant_id = new_expires_at.child_group_id AND root_item_id = ? FOR UPDATE
				) AS latest_attempt) AND
			(NOW() < new_expires_at.expires_at OR NOW() < attempts.allows_submissions_until)
	`, itemID, itemID).Error())
	service.MustNotBeError(store.Exec("DROP TEMPORARY TABLE new_expires_at").Error())
	if groupsGroupsModified {
		service.MustNotBeError(store.GroupGroups().After())
		store.ScheduleResultsPropagation()
	}
}
