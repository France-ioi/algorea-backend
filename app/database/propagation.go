package database

import (
	"net/http"
	"strings"
	"time"

	"github.com/France-ioi/AlgoreaBackend/app/logging"
)

const (
	endpointTimeout                   = 3 * time.Second
	propagationTransactionMinDuration = 100 * time.Millisecond
)

// StartAsyncPropagation schedules asynchronous propagation of the given types.
// If endpoint is an empty string, it will be done synchronously.
func StartAsyncPropagation(store *DataStore, endpoint string, types []string) {
	// Must not be called in a transaction because it calls an endpoint, which can take a long time.
	store.MustNotBeInTransaction()

	endpointFailed := false
	if endpoint != "" {
		// Async.
		client := http.Client{
			Timeout: endpointTimeout,
		}

		callTime := time.Now()
		response, err := client.Get(endpoint + "?types=" + strings.Join(types, ","))
		logging.Infof("Propagation endpoint called: %v, types=%v, duration=%v", endpoint, types, time.Since(callTime))

		if err != nil {
			logging.Errorf("Propagation endpoint error: %v", err)

			endpointFailed = true
		} else {
			defer func(response *http.Response) {
				_ = response.Body.Close()
			}(response)

			if response.StatusCode != http.StatusOK {
				logging.Errorf("Propagation endpoint error: status=%v", response.StatusCode)

				endpointFailed = true
			}
		}
	}

	if endpoint == "" || endpointFailed {
		err := store.InTransaction(func(store *DataStore) error {
			store.ScheduleItemsAncestorsPropagation()
			store.SchedulePermissionsPropagation()
			store.ScheduleResultsPropagation()

			return nil
		})
		mustNotBeError(err)
	}
}

// PropagationStepTransaction runs a step function until it returns true.
// The steps are run inside a transaction, and the transaction is committed every propagationTransactionMinDuration.
// This is to avoid overhead from having many small transactions, if we simply do one transaction per step.
// Typically, small steps will be run in a single larger transaction, and bigger steps in their own transactions.
func (s *DataStore) PropagationStepTransaction(step func(store *DataStore) bool) {
	done := false
	for !done {
		endTransactionMinimumTime := time.Now().Add(propagationTransactionMinDuration)
		mustNotBeError(s.InTransaction(func(s *DataStore) error {
			for !done && time.Now().Before(endTransactionMinimumTime) {
				done = step(s)
			}

			return nil
		}))
	}
}
