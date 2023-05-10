package database

// AnswerStore implements database operations on `answers`.
type AnswerStore struct {
	*DataStore
}

// WithUsers creates a composable query for getting answers joined with users (via author_id).
func (s *AnswerStore) WithUsers() *AnswerStore {
	return &AnswerStore{
		NewDataStoreWithTable(
			s.Joins("JOIN users ON users.group_id = answers.author_id"), s.tableName,
		),
	}
}

// WithResults creates a composable query for getting answers joined with results.
func (s *AnswerStore) WithResults() *AnswerStore {
	return &AnswerStore{
		NewDataStoreWithTable(
			s.Joins(`
				JOIN results ON results.participant_id = answers.participant_id AND
					results.attempt_id = answers.attempt_id AND results.item_id = answers.item_id`), s.tableName,
		),
	}
}

// WithItems joins `items`.
func (s *AnswerStore) WithItems() *AnswerStore {
	return &AnswerStore{
		NewDataStoreWithTable(
			s.Joins("JOIN items ON items.id = answers.item_id"), s.tableName,
		),
	}
}

// SubmitNewAnswer inserts a new row with type='Submission', created_at=NOW()
// into the `answers` table.
func (s *AnswerStore) SubmitNewAnswer(authorID, participantID, attemptID, itemID int64, answer string) (int64, error) {
	var answerID int64
	err := s.retryOnDuplicatePrimaryKeyError(func(db *DB) error {
		store := NewDataStore(db)
		answerID = store.NewID()
		return db.db.Exec(`
				INSERT INTO answers (id, author_id, participant_id, attempt_id, item_id, answer, created_at, type)
				VALUES (?, ?, ?, ?, ?, ?, NOW(), 'Submission')`,
			answerID, authorID, participantID, attemptID, itemID, answer).Error
	})
	return answerID, err
}

// Visible returns a composable query for getting answers with the following access rights
// restrictions:
//  1. the user should have at least 'content' access rights to the answers.item_id item,
//  2. the user is able to see answers related to his group's attempts, so
//     the user should be a member of the answers.participant_id team or
//     answers.participant_id should be equal to the user's self group
func (s *AnswerStore) Visible(user *User) *DB {
	usersGroupsQuery := s.GroupGroups().WhereUserIsMember(user).Select("parent_group_id")

	// the user should have at least 'content' access to the answers.item_id
	permsQuery := s.Permissions().MatchingUserAncestors(user).WherePermissionIsAtLeast("view", "content").
		Select("DISTINCT item_id")

	return s.
		// the user should have at least 'content' access to the answers.item_id
		Joins("JOIN (?) AS permissions USING(item_id)", permsQuery.SubQuery()).
		// attempts.group_id should be one of the authorized user's groups or the user's self group
		Where("answers.participant_id = ? OR answers.participant_id IN (?)",
			user.GroupID, usersGroupsQuery.SubQuery())
}
