//go:build !unit

package database_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/testhelpers"
)

type validationDateResultRow struct {
	ParticipantID int64
	AttemptID     int64
	ItemID        int64
	ValidatedAt   *database.Time
	State         string
}

func constructExpectedResultsForValidatedAtTests(t11, t12, t13, t14, t23, t24 *time.Time) []validationDateResultRow {
	return []validationDateResultRow{
		{ParticipantID: 101, AttemptID: 1, ItemID: 1, ValidatedAt: (*database.Time)(t11), State: "done"},
		{ParticipantID: 101, AttemptID: 1, ItemID: 2, ValidatedAt: (*database.Time)(t12), State: "done"},
		{ParticipantID: 101, AttemptID: 1, ItemID: 3, ValidatedAt: (*database.Time)(t13), State: "done"},
		{ParticipantID: 101, AttemptID: 1, ItemID: 4, ValidatedAt: (*database.Time)(t14), State: "done"},
		{ParticipantID: 101, AttemptID: 2, ItemID: 3, ValidatedAt: (*database.Time)(t23), State: "done"},
		{ParticipantID: 101, AttemptID: 2, ItemID: 4, ValidatedAt: (*database.Time)(t24), State: "done"},
		// another user
		{ParticipantID: 102, AttemptID: 1, ItemID: 2, ValidatedAt: nil, State: "done"},
	}
}

func TestResultStore_Propagate_NonCategories_SetsValidatedAtToMaxOfChildrenValidatedAts(t *testing.T) {
	db := testhelpers.SetupDBWithFixture("results_propagation/_common", "results_propagation/validated_at")
	defer func() { _ = db.Close() }()

	resultStore := database.NewDataStore(db).Results()

	baseDate := time.Now().Round(time.Second).UTC()
	skippedDate := baseDate.AddDate(-2, -1, -1)
	oldestForItem3 := baseDate.AddDate(-1, -1, -1)
	skippedInItem3 := oldestForItem3.Add(24 * time.Hour)
	oldestForItem4AndWinner := baseDate.AddDate(0, -1, -1)
	skippedInItem4 := oldestForItem4AndWinner.Add(24 * time.Hour)

	assert.NoError(t, resultStore.Where("item_id = 3 AND participant_id = 101 AND attempt_id = 1").
		UpdateColumn("validated_at", oldestForItem3).Error())
	assert.NoError(t, resultStore.Where("item_id = 3 AND participant_id = 101 AND attempt_id = 2").
		UpdateColumn("validated_at", skippedInItem3).Error())

	assert.NoError(t, resultStore.Where("item_id = 4 AND participant_id = 101 AND attempt_id = 1").
		UpdateColumn("validated_at", oldestForItem4AndWinner).Error())
	assert.NoError(t, resultStore.Where("item_id = 4 AND participant_id = 101 AND attempt_id = 2").
		UpdateColumn("validated_at", skippedInItem4).Error())

	assert.NoError(t, resultStore.Where("attempt_id = 1 AND item_id = 1 AND participant_id = 101").
		UpdateColumn("validated_at", skippedDate).Error())

	err := resultStore.InTransaction(func(s *database.DataStore) error {
		s.ScheduleResultsPropagation()
		return nil
	})
	assert.NoError(t, err)

	var result []validationDateResultRow
	queryResultsAndStatesForTests(t, resultStore, &result, "validated_at")
	assert.Equal(t,
		constructExpectedResultsForValidatedAtTests(&skippedDate, &oldestForItem4AndWinner, &oldestForItem3,
			&oldestForItem4AndWinner, &skippedInItem3, &skippedInItem4), result)
}

func TestResultStore_Propagate_Categories_SetsValidatedAtToMaxOfValidatedAtsOfChildrenWithCategoryValidation_NoSuitableChildren(
	t *testing.T,
) {
	db := testhelpers.SetupDBWithFixture("results_propagation/_common", "results_propagation/validated_at")
	defer func() { _ = db.Close() }()

	resultStore := database.NewDataStore(db).Results()

	expectedDate := time.Now().Round(time.Second).UTC()
	oldDate := expectedDate.AddDate(-1, -1, -1)

	assert.NoError(t, resultStore.Where("item_id = 3 AND participant_id = 101 AND attempt_id = 1").
		UpdateColumn("validated_at", oldDate).Error())
	assert.NoError(t, resultStore.Where("attempt_id = 1 AND item_id = 1 AND participant_id = 101").
		UpdateColumn("validated_at", expectedDate).Error())
	assert.NoError(
		t, database.NewDataStore(db).Items().Where("id=2").UpdateColumn("validation_type", "Categories").
			Error())

	err := resultStore.InTransaction(func(s *database.DataStore) error {
		s.ScheduleResultsPropagation()
		return nil
	})
	assert.NoError(t, err)

	var result []validationDateResultRow
	queryResultsAndStatesForTests(t, resultStore, &result, "validated_at")
	assert.Equal(t,
		constructExpectedResultsForValidatedAtTests(&expectedDate, nil, &oldDate, nil, nil, nil), result)
}

func TestResultStore_Propagate_Categories_SetsValidatedAtToNull_IfSomeCategoriesAreNotValidated(
	t *testing.T,
) {
	db := testhelpers.SetupDBWithFixture("results_propagation/_common", "results_propagation/validated_at")
	defer func() { _ = db.Close() }()

	resultStore := database.NewDataStore(db).Results()

	expectedDate := time.Now().Round(time.Second).UTC()
	oldDate := expectedDate.AddDate(-1, -1, -1)

	assert.NoError(t, resultStore.Where("item_id = 3 AND participant_id = 101 AND attempt_id = 1").
		UpdateColumn("validated_at", oldDate).Error())
	assert.NoError(t, resultStore.Where("attempt_id = 1 AND item_id = 1 AND participant_id = 101").
		UpdateColumn("validated_at", expectedDate).Error())
	assert.NoError(
		t, database.NewDataStore(db).Items().Where("id=2").UpdateColumn("validation_type", "Categories").
			Error())
	assert.NoError(t, database.NewDataStore(db).ItemItems().Where("parent_item_id = 2 AND child_item_id IN (3, 4)").
		UpdateColumn("category", "Validation").Error())

	err := resultStore.InTransaction(func(s *database.DataStore) error {
		s.ScheduleResultsPropagation()
		return nil
	})
	assert.NoError(t, err)

	var result []validationDateResultRow
	queryResultsAndStatesForTests(t, resultStore, &result, "validated_at")
	assert.Equal(t,
		constructExpectedResultsForValidatedAtTests(&expectedDate, nil, &oldDate, nil, nil, nil), result)
}

func TestResultStore_Propagate_Categories_ValidatedAtShouldBeMaxOfChildrensWithCategoryValidation_IfAllAreValidated(
	t *testing.T,
) {
	db := testhelpers.SetupDBWithFixture("results_propagation/_common", "results_propagation/validated_at")
	defer func() { _ = db.Close() }()

	resultStore := database.NewDataStore(db).Results()

	expectedDate := time.Now().Round(time.Second).UTC()
	oldDate := expectedDate.AddDate(-1, -1, -1)

	assert.NoError(t, resultStore.Where("item_id = 3 AND participant_id = 101 AND attempt_id = 1").
		UpdateColumn("validated_at", oldDate).Error())
	assert.NoError(t, resultStore.Where("attempt_id = 1 AND item_id = 1 AND participant_id = 101").
		UpdateColumn("validated_at", oldDate).Error())
	assert.NoError(t, resultStore.Where("item_id = 4 AND participant_id = 101 AND attempt_id = 2").
		UpdateColumn("validated_at", expectedDate).Error())
	assert.NoError(t, resultStore.Attempts().Where("participant_id = 101 AND id = 2").UpdateColumns(map[string]interface{}{
		"root_item_id": 4, "parent_attempt_id": 1,
	}).Error())
	assert.NoError(
		t, database.NewDataStore(db).Items().Where("id=2").UpdateColumn("validation_type", "Categories").
			Error())
	assert.NoError(t, database.NewDataStore(db).ItemItems().Where("parent_item_id = 2 AND child_item_id IN (3, 4)").
		UpdateColumn("category", "Validation").Error())

	err := resultStore.InTransaction(func(s *database.DataStore) error {
		s.ScheduleResultsPropagation()
		return nil
	})
	assert.NoError(t, err)

	var result []validationDateResultRow
	queryResultsAndStatesForTests(t, resultStore, &result, "validated_at")
	assert.Equal(t,
		constructExpectedResultsForValidatedAtTests(&oldDate, &expectedDate, &oldDate, nil, nil, &expectedDate), result)
}

func TestResultStore_Propagate_Categories_SetsValidatedAtToMaxOfValidatedAtsOfChildrenWithCategoryValidation_IgnoresNoScoreItems(
	t *testing.T,
) {
	db := testhelpers.SetupDBWithFixture("results_propagation/_common", "results_propagation/validated_at")
	defer func() { _ = db.Close() }()

	resultStore := database.NewDataStore(db).Results()

	expectedDate := time.Now().Round(time.Second).UTC()
	oldDate := expectedDate.AddDate(-1, -1, -1)
	oldDatePlusOneDay := oldDate.Add(24 * time.Hour)

	itemStore := database.NewDataStore(db).Items()
	assert.NoError(t, resultStore.Where("item_id = 3 AND participant_id = 101 AND attempt_id = 1").
		UpdateColumn("validated_at", oldDate).Error())
	assert.NoError(t, resultStore.Where("item_id = 4 AND participant_id = 101 AND attempt_id = 1").
		UpdateColumn("validated_at", oldDate).Error())
	assert.NoError(t, resultStore.Where("item_id = 3 AND participant_id = 101 AND attempt_id = 2").
		UpdateColumn("validated_at", oldDatePlusOneDay).Error()) // should be ignored
	assert.NoError(t, resultStore.Where("attempt_id = 1 AND item_id = 1 AND participant_id = 101").
		UpdateColumn("validated_at", expectedDate).Error())
	assert.NoError(t, itemStore.Where("id=2").UpdateColumn("validation_type", "Categories").Error())
	assert.NoError(t, database.NewDataStore(db).ItemItems().Where("parent_item_id = 2 AND child_item_id IN (1, 3, 4)").
		UpdateColumn("category", "Validation").Error())

	assert.NoError(t, itemStore.Where("id=1").Updates(map[string]interface{}{
		"type": "Task",
	}).Error())
	assert.NoError(t, itemStore.Where("id=3").Updates(map[string]interface{}{
		"no_score": true,
	}).Error())

	err := resultStore.InTransaction(func(s *database.DataStore) error {
		s.ScheduleResultsPropagation()
		return nil
	})
	assert.NoError(t, err)

	var result []validationDateResultRow
	queryResultsAndStatesForTests(t, resultStore, &result, "validated_at")
	assert.Equal(t,
		constructExpectedResultsForValidatedAtTests(&expectedDate, &expectedDate, &oldDate, &oldDate, &oldDatePlusOneDay, nil), result)
}
