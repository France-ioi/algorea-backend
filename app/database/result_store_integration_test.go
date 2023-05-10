//go:build !unit

package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/testhelpers"
)

func TestResultStore_GetHintsInfoForActiveAttempt(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString(`
		attempts:
			- {participant_id: 11, id: 1, root_item_id: 112, allows_submissions_until: 3019-05-30 12:00:00}
			- {participant_id: 11, id: 2, root_item_id: 112}
			- {participant_id: 12, id: 2, root_item_id: 114, allows_submissions_until: 2019-05-30 12:00:00}
		results:
			- {participant_id: 11, attempt_id: 1, item_id: 12, started_at: 2019-05-30 11:00:00}
			- {participant_id: 11, attempt_id: 2, item_id: 12, hints_requested: '[0,1,"hint",null]', hints_cached: 4,
				started_at: 2019-07-30 11:00:00}
			- {participant_id: 12, attempt_id: 2, item_id: 14}`)
	defer func() { _ = db.Close() }()

	tests := []struct {
		name          string
		participantID int64
		attemptID     int64
		itemID        int64
		wantHintsInfo *database.HintsInfo
		wantError     error
	}{
		{name: "empty info", participantID: 11, attemptID: 1, itemID: 12, wantHintsInfo: &database.HintsInfo{}},
		{
			name: "with info", participantID: 11, attemptID: 2, itemID: 12,
			wantHintsInfo: &database.HintsInfo{
				HintsRequested: ptrString(`[0,1,"hint",null]`),
				HintsCached:    4,
			},
		},
		{
			name: "finished", participantID: 12, attemptID: 2, itemID: 14, wantHintsInfo: nil,
			wantError: gorm.ErrRecordNotFound,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			assert.NoError(t, database.NewDataStore(db).InTransaction(func(store *database.DataStore) error {
				hintsInfo, err := store.Results().GetHintsInfoForActiveAttempt(test.participantID, test.attemptID, test.itemID)
				assert.Equal(t, test.wantHintsInfo, hintsInfo)
				assert.Equal(t, test.wantError, err)
				return nil
			}))
		})
	}
}

func TestResultStore_Propagate(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "basic", wantErr: false},
	}

	db := testhelpers.SetupDBWithFixture("results_propagation/main")
	defer func() { _ = db.Close() }()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := database.NewDataStore(db).InTransaction(func(s *database.DataStore) error {
				s.ScheduleResultsPropagation()
				return nil
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("ResultStore.propagate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResultStore_Propagate_Concurrent(t *testing.T) {
	db := testhelpers.SetupDBWithFixture("results_propagation/main")
	defer func() { _ = db.Close() }()

	testhelpers.RunConcurrently(func() {
		s := database.NewDataStoreWithContext(context.Background(), db)
		err := s.InTransaction(func(st *database.DataStore) error {
			st.ScheduleResultsPropagation()
			return nil
		})
		assert.NoError(t, err)
	}, 30)
}
