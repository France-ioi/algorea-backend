package database

import (
	"gorm.io/gorm"
)

// AttemptStore implements database operations on `attempts`.
type AttemptStore struct {
	*DataStore
}

// CreateNew creates a new attempt (with id > 0) with parent_attempt_id = parentAttemptID and a new result.
// It also sets attempts.created_at, results.started_at, results.latest_activity_at, so the result should be propagated.
func (s *AttemptStore) CreateNew(participantID, parentAttemptID, itemID, creatorID int64) (attemptID int64, err error) {
	s.mustBeInTransaction()
	recoverPanics(&err)

	mustNotBeError(s.InsertMap(map[string]interface{}{
		"id": gorm.Expr("(SELECT * FROM (?) AS max_attempt)", s.Attempts().Select("IFNULL(MAX(id)+1, 0)").
			Where("participant_id = ?", participantID).WithWriteLock().SubQuery()),
		"participant_id": participantID, "creator_id": creatorID,
		"parent_attempt_id": parentAttemptID, "root_item_id": itemID, "created_at": Now(),
	}))
	mustNotBeError(s.Where("participant_id = ?", participantID).PluckFirst("MAX(id)", &attemptID).Error())

	mustNotBeError(s.Results().InsertMap(map[string]interface{}{
		"participant_id": participantID, "attempt_id": attemptID, "item_id": itemID,
		"started_at": Now(), "latest_activity_at": Now(),
	}))
	mustNotBeError(s.Results().MarkAsToBePropagated(participantID, attemptID, itemID))
	return attemptID, nil
}
