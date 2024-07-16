package database_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/France-ioi/AlgoreaBackend/app/database"

	"github.com/stretchr/testify/assert"
	"github.com/thingful/httpmock"

	"github.com/France-ioi/AlgoreaBackend/app/logging"
	"github.com/France-ioi/AlgoreaBackend/app/loggingtest"
	"github.com/France-ioi/AlgoreaBackend/testhelpers"
)

func TestStartAsyncPropagation(t *testing.T) {
	type args struct {
		endpoint string
	}
	tests := []struct {
		name                 string
		args                 args
		endpointCallErr      error
		endpointResponseCode int
		loggedError          string
		propagated           bool
	}{
		{
			name: "should propagate sync when endpoint is defined",
			args: args{
				endpoint: "",
			},
			propagated: true,
		},
		{
			name: "should propagate sync when the endpoint call returns an error",
			args: args{
				endpoint: "https://example.com",
			},
			loggedError:     "Propagation endpoint error: Get \"https://example.com?types=permissions\": error",
			endpointCallErr: fmt.Errorf("error"),
			propagated:      true,
		},
		{
			name: "should not propagate now (async) when endpoint is defined, and endpoint must be called",
			args: args{
				endpoint: "https://example.com",
			},
			endpointResponseCode: http.StatusOK,
			propagated:           false,
		},
		{
			name: "should do propagation when the endpoint is defined but the call fails",
			args: args{
				endpoint: "https://example.com",
			},
			loggedError:          "Propagation endpoint error: status=500",
			endpointResponseCode: http.StatusInternalServerError,
			propagated:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testhelpers.SetupDBWithFixtureString(`
				groups:
					- {id: 1, type: Class}
				items:
					- {id: 1, type: Task, default_language_tag: fr}
				permissions_granted:
					- {group_id: 1, item_id: 1, source_group_id: 1, origin: "group_membership", can_view: "content"}
			`)
			defer func() { _ = db.Close() }()
			store := database.NewDataStore(db)

			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			logHook, restoreFunc := logging.MockSharedLoggerHook()
			defer restoreFunc()

			if tt.args.endpoint != "" {
				if tt.endpointCallErr == nil {
					httpmock.RegisterStubRequest(
						httpmock.NewStubRequest(
							"GET",
							tt.args.endpoint+"?types=permissions",
							httpmock.NewStringResponder(tt.endpointResponseCode, ""),
						),
					)
				} else {
					httpmock.RegisterStubRequest(
						httpmock.NewStubRequest(
							"GET",
							tt.args.endpoint+"?types=permissions",
							func(*http.Request) (*http.Response, error) {
								return nil, tt.endpointCallErr
							},
						),
					)
				}
			}

			database.StartAsyncPropagation(store, tt.args.endpoint, []string{"permissions"})

			exists, err := store.Permissions().Where("item_id = 1").HasRows()
			assert.NoError(t, err)
			assert.Equal(t, tt.propagated, exists)

			// Verify that all stubs were called.
			if err := httpmock.AllStubsCalled(); err != nil {
				t.Errorf("Not all stubs were called: %s", err)
			}

			// Verify logs.
			if tt.loggedError != "" {
				logs := (&loggingtest.Hook{Hook: logHook}).GetAllStructuredLogs()
				assert.Contains(t, logs, fmt.Sprintf("level=error msg=%q", tt.loggedError))
			}
		})
	}
}

func TestStartAsyncPropagation_ShouldPanicWhenCalledInsideTransaction(t *testing.T) {
	db := testhelpers.SetupDBWithFixtureString("")
	defer func() { _ = db.Close() }()
	store := database.NewDataStore(db)

	assert.Panics(t, func() {
		err := store.InTransaction(func(store *database.DataStore) error {
			database.StartAsyncPropagation(store, "", []string{"permissions"})
			return nil
		})
		assert.NoError(t, err)
	})
}
