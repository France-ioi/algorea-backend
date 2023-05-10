package app

import (
	crand "crypto/rand"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"testing"

	"bou.ke/monkey"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus" //nolint:depguard
	"github.com/spf13/viper"
	assertlib "github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/appenv"
	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/domain"
	"github.com/France-ioi/AlgoreaBackend/app/logging"
)

/* note that the tests of app.New() are very incomplete (even if all exec path are covered) */

func TestNew_Success(t *testing.T) {
	assert := assertlib.New(t)
	appenv.SetDefaultEnvToTest()
	_ = os.Setenv("ALGOREA_SERVER__COMPRESS", "1")
	defer func() { _ = os.Unsetenv("ALGOREA_SERVER__COMPRESS") }()
	app, err := New()
	assert.NotNil(app)
	assert.NoError(err)
	assert.NotNil(app.Config)
	assert.NotNil(app.Database)
	assert.NotNil(app.HTTPHandler)
	assert.NotNil(app.apiCtx)
	assert.Len(app.HTTPHandler.Middlewares(), 7)
	assert.True(len(app.HTTPHandler.Routes()) > 0)
	assert.Equal("/*", app.HTTPHandler.Routes()[0].Pattern) // test default val
}

func TestNew_SuccessNoCompress(t *testing.T) {
	assert := assertlib.New(t)
	appenv.SetDefaultEnvToTest()
	_ = os.Setenv("ALGOREA_SERVER__COMPRESS", "false")
	defer func() { _ = os.Unsetenv("ALGOREA_SERVER__COMPRESS") }()
	app, _ := New()
	assert.Len(app.HTTPHandler.Middlewares(), 6)
}

func TestNew_NotDefaultRootPath(t *testing.T) {
	assert := assertlib.New(t)
	appenv.SetDefaultEnvToTest()
	_ = os.Setenv("ALGOREA_SERVER__ROOTPATH", "/api")
	defer func() { _ = os.Unsetenv("ALGOREA_SERVER__ROOTPATH") }()
	app, err := New()
	assert.NoError(err)
	assert.Equal("/api/*", app.HTTPHandler.Routes()[0].Pattern)
}

func TestNew_DBErr(t *testing.T) {
	assert := assertlib.New(t)
	hook, restoreFct := logging.MockSharedLoggerHook()
	defer restoreFct()
	expectedError := errors.New("db opening error")
	patch := monkey.Patch(database.Open, func(interface{}) (*database.DB, error) {
		return nil, expectedError
	})
	defer patch.Unpatch()
	app, err := New()
	assert.Nil(app)
	assert.Equal(expectedError, err)
	logMsg := hook.LastEntry()
	assert.Equal(logrus.ErrorLevel, logMsg.Level)
	assert.Equal("db opening error", logMsg.Message)
	assert.Equal("database", logMsg.Data["module"])
}

func TestNew_RandSeedingFailed(t *testing.T) {
	assert := assertlib.New(t)
	expectedError := errors.New("some error")
	patch := monkey.Patch(crand.Read, func([]byte) (int, error) {
		return 1, expectedError
	})
	defer patch.Unpatch()
	assert.PanicsWithValue("cannot seed the randomizer", func() { _, _ = New() })
}

func TestNew_DBConfigError(t *testing.T) {
	assert := assertlib.New(t)
	patch := monkey.Patch(DBConfig, func(_ *viper.Viper) (config *mysql.Config, err error) {
		return nil, errors.New("dberror")
	})
	defer patch.Unpatch()
	_, err := New()
	assert.EqualError(err, "unable to load the 'database' configuration: dberror")
}

func TestNew_TokenConfigError(t *testing.T) {
	assert := assertlib.New(t)
	patch := monkey.Patch(LoadConfig, func() *viper.Viper {
		globalConfig := viper.New()
		globalConfig.Set("token.PublicKeyFile", "notafile")
		return globalConfig
	})
	defer patch.Unpatch()
	_, err := New()
	assert.NotNil(err)
	assert.Contains(err.Error(), "no such file or directory")
}

func TestNew_DomainsConfigError(t *testing.T) {
	assert := assertlib.New(t)
	patch := monkey.Patch(LoadConfig, func() *viper.Viper {
		globalConfig := viper.New()
		globalConfig.Set("domains", []int{1, 2})
		return globalConfig
	})
	defer patch.Unpatch()
	_, err := New()
	assert.NotNil(err)
	assert.Contains(err.Error(), "unable to load the 'domain' configuration: 2 error(s) decoding")
}

// The goal of the following `TestMiddlewares*` tests are not to test the middleware themselves
// but their interaction (impacted by the order of definition)

func TestMiddlewares_OnPanic(t *testing.T) {
	assert := assertlib.New(t)
	hook, restoreFct := logging.MockSharedLoggerHook()
	defer restoreFct()
	app, _ := New()
	router := app.HTTPHandler
	router.Get("/dummy", func(http.ResponseWriter, *http.Request) {
		panic("error in service")
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	nbLogsBeforeRequest := len(hook.AllEntries())
	request, _ := http.NewRequest("GET", srv.URL+"/dummy", http.NoBody)
	request.Header.Set("X-Forwarded-For", "1.1.1.1")
	response, err := http.DefaultClient.Do(request)
	assert.NoError(err)
	if err != nil {
		return
	}
	respBody, _ := ioutil.ReadAll(response.Body)
	_ = response.Body.Close()

	// check that the error has been handled by the recover
	assert.Equal(http.StatusInternalServerError, response.StatusCode)
	assert.Equal("Internal Server Error\n", string(respBody))
	assert.Equal("text/plain; charset=utf-8", response.Header.Get("Content-type"))
	allLogs := hook.AllEntries()
	assert.Equal(2, len(allLogs)-nbLogsBeforeRequest)
	// check that the req id is correct
	assert.Equal(allLogs[len(allLogs)-1].Data["req_id"], allLogs[len(allLogs)-2].Data["req_id"])
	// check that the recovere put the error info in the logs
	assert.Equal("error in service", hook.LastEntry().Data["panic"])
	assert.NotNil(hook.LastEntry().Data["stack"])
	// check that the real IP is used in the logs
	assert.Equal("1.1.1.1", allLogs[len(allLogs)-1].Data["remote_addr"])
	assert.Equal("1.1.1.1", allLogs[len(allLogs)-2].Data["remote_addr"])
}

func TestMiddlewares_OnSuccess(t *testing.T) {
	assert := assertlib.New(t)
	_ = os.Setenv("ALGOREA_SERVER__COMPRESS", "1")
	defer func() { _ = os.Unsetenv("ALGOREA_SERVER__COMPRESS") }()
	hook, restoreFct := logging.MockSharedLoggerHook()
	defer restoreFct()
	app, _ := New()
	router := app.HTTPHandler
	router.Get("/dummy", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"data\":\"datadatadata\"}"))
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	nbLogsBeforeRequest := len(hook.AllEntries())
	request, _ := http.NewRequest("GET", srv.URL+"/dummy", http.NoBody)
	request.Header.Set("X-Real-IP", "1.1.1.1")
	request.Header.Set("Accept-Encoding", "gzip, deflate")
	response, err := http.DefaultClient.Do(request)
	assert.NoError(err)
	if err != nil {
		return
	}
	defer func() { _ = response.Body.Close() }()
	assert.NotNil(response.Header.Get("Content-type"))
	assert.Equal("application/json", response.Header.Get("Content-Type"))
	allLogs := hook.AllEntries()
	assert.Equal(2, len(allLogs)-nbLogsBeforeRequest)
	// check that the req id is correct
	assert.Equal(allLogs[len(allLogs)-1].Data["req_id"], allLogs[len(allLogs)-2].Data["req_id"])
	// check that the real IP is used in the logs
	assert.Equal("1.1.1.1", allLogs[len(allLogs)-1].Data["remote_addr"])
	assert.Equal("1.1.1.1", allLogs[len(allLogs)-2].Data["remote_addr"])
	// check that the compression has been applied but the length in the logs is not altered by compression i
	assert.Equal(23, hook.LastEntry().Data["resp_bytes_length"])
	assert.Equal("gzip", response.Header.Get("Content-Encoding"))
}

func TestNew_MountsPprofInDev(t *testing.T) {
	assert := assertlib.New(t)

	appenv.SetDefaultEnvToTest()
	monkey.Patch(appenv.IsEnvDev, func() bool { return true })
	defer monkey.UnpatchAll()

	app, err := New()
	assert.NotNil(app)
	assert.NoError(err)

	srv := httptest.NewServer(app.HTTPHandler)
	defer srv.Close()

	request, _ := http.NewRequest("GET", srv.URL+"/debug", http.NoBody)
	response, err := http.DefaultClient.Do(request)
	assert.NoError(err)
	if err != nil {
		return
	}
	defer func() { _ = response.Body.Close() }()
	body, err := ioutil.ReadAll(response.Body)
	assert.NoError(err)
	assert.Contains(string(body), "Types of profiles available:")
}

func TestNew_DoesNotMountPprofInEnvironmentsOtherThanDev(t *testing.T) {
	assert := assertlib.New(t)

	monkey.Patch(appenv.IsEnvDev, func() bool { return false })
	defer monkey.UnpatchAll()

	app, err := New()
	assert.NotNil(app)
	assert.NoError(err)

	srv := httptest.NewServer(app.HTTPHandler)
	defer srv.Close()

	request, _ := http.NewRequest("GET", srv.URL+"/debug", http.NoBody)
	response, err := http.DefaultClient.Do(request)
	assert.NoError(err)
	if err != nil {
		return
	}
	defer func() { _ = response.Body.Close() }()
	assert.Equal(404, response.StatusCode)
}

type relationSpec struct {
	database.ParentChild
	exists bool
	error  bool
}

func TestApplication_CheckConfig_UnmarshalError(t *testing.T) {
	assert := assertlib.New(t)
	db, _ := database.NewDBMock()
	defer func() { _ = db.Close() }()
	app := &Application{Config: viper.New(), Database: db}
	app.Config.Set("domains", []int{1, 2})
	assert.Contains(app.CheckConfig().Error(), "unable to unmarshal domains config: 2 error(s) decoding")
}

func TestApplication_CheckConfig(t *testing.T) { //nolint:gocognit Should be refactored.
	type groupSpec struct {
		id     int64
		exists bool
		error  bool
	}

	tests := []struct {
		name                     string
		config                   []domain.ConfigItem
		expectedGroupsToCheck    []groupSpec
		expectedRelationsToCheck []relationSpec
		expectedError            error
	}{
		{
			name: "everything is okay",
			config: []domain.ConfigItem{
				{
					Domains:       []string{"127.0.0.1", "192.168.0.1"},
					AllUsersGroup: 2, TempUsersGroup: 4,
				},
				{
					Domains:       []string{"www.france-ioi.org"},
					AllUsersGroup: 6, TempUsersGroup: 8,
				},
			},
			expectedGroupsToCheck: []groupSpec{
				{id: 2, exists: true},
				{id: 4, exists: true},
				{id: 6, exists: true},
				{id: 8, exists: true},
			},
			expectedRelationsToCheck: []relationSpec{
				{ParentChild: database.ParentChild{ParentID: 2, ChildID: 4}, exists: true},
				{ParentChild: database.ParentChild{ParentID: 6, ChildID: 8}, exists: true},
			},
		},
		{
			name: "AllUsers is missing",
			config: []domain.ConfigItem{
				{
					Domains:       []string{"192.168.0.1"},
					AllUsersGroup: 2, TempUsersGroup: 4,
				},
			},
			expectedGroupsToCheck: []groupSpec{
				{id: 2},
			},
			expectedError: errors.New("no AllUsers group for domain \"192.168.0.1\""),
		},
		{
			name: "TempUsers is missing",
			config: []domain.ConfigItem{
				{
					Domains:       []string{"127.0.0.1"},
					AllUsersGroup: 2, TempUsersGroup: 4,
				},
			},
			expectedGroupsToCheck: []groupSpec{
				{id: 2, exists: true},
				{id: 4},
			},
			expectedError: errors.New("no TempUsers group for domain \"127.0.0.1\""),
		},
		{
			name: "AllUsers -> TempUsers relation is missing",
			config: []domain.ConfigItem{
				{
					Domains:       []string{"127.0.0.1"},
					AllUsersGroup: 2, TempUsersGroup: 4,
				},
			},
			expectedGroupsToCheck: []groupSpec{
				{id: 2, exists: true},
				{id: 4, exists: true},
			},
			expectedRelationsToCheck: []relationSpec{
				{ParentChild: database.ParentChild{ParentID: 2, ChildID: 4}},
			},
			expectedError: errors.New("no AllUsers -> TempUsers link in groups_groups for domain \"127.0.0.1\""),
		},
		{
			name: "error on group checking",
			config: []domain.ConfigItem{
				{
					Domains:       []string{"127.0.0.1"},
					AllUsersGroup: 2, TempUsersGroup: 4,
				},
			},
			expectedGroupsToCheck: []groupSpec{
				{id: 2, error: true},
			},
			expectedError: errors.New("some error"),
		},
		{
			name: "error on relation checking",
			config: []domain.ConfigItem{
				{
					Domains:       []string{"127.0.0.1"},
					AllUsersGroup: 2, TempUsersGroup: 4,
				},
			},
			expectedGroupsToCheck: []groupSpec{
				{id: 2, exists: true},
				{id: 4, exists: true},
			},
			expectedRelationsToCheck: []relationSpec{
				{ParentChild: database.ParentChild{ParentID: 2, ChildID: 4}, error: true},
			},
			expectedError: errors.New("some error"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			db, mock := database.NewDBMock()
			defer func() { _ = db.Close() }()
			mock.MatchExpectationsInOrder(false)

			var expectedError error

			for _, expectedGroupToCheck := range tt.expectedGroupsToCheck {
				queryMock := mock.ExpectQuery("^" + regexp.QuoteMeta(
					"SELECT 1 FROM `groups` WHERE groups.id = ? LIMIT 1",
				) + "$").WithArgs(expectedGroupToCheck.id)
				if !expectedGroupToCheck.error {
					rowsToReturn := mock.NewRows([]string{"1"})
					if expectedGroupToCheck.exists {
						rowsToReturn.AddRow(1)
					}
					queryMock.WillReturnRows(rowsToReturn)
				} else {
					expectedError = errors.New("some error")
					queryMock.WillReturnError(expectedError)
				}
			}
			if expectedError == nil {
				for _, expectedRelationToCheck := range tt.expectedRelationsToCheck {
					rowsToReturn := mock.NewRows([]string{"1"})
					if expectedRelationToCheck.exists {
						rowsToReturn.AddRow(1)
					}
					queryMock := mock.ExpectQuery("^"+regexp.QuoteMeta(
						"SELECT 1 FROM `groups_groups_active` WHERE parent_group_id = ? AND child_group_id = ? LIMIT 1",
					)+"$").
						WithArgs(expectedRelationToCheck.ParentID, expectedRelationToCheck.ChildID)
					if !expectedRelationToCheck.error {
						queryMock.WillReturnRows(rowsToReturn)
					} else {
						expectedError = errors.New("some error")
						queryMock.WillReturnError(expectedError)
					}
				}
			}
			config := viper.New()
			config.Set(domainsConfigKey, tt.config)

			app := &Application{
				Config:   config,
				Database: db,
			}
			err := app.CheckConfig()
			assertlib.Equal(t, tt.expectedError, err)
			assertlib.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

type groupSpec struct {
	name          string
	id            int64
	exists        bool
	error         bool
	errorOnInsert bool
}

type createMissingDataTestCase struct {
	name                      string
	config                    []domain.ConfigItem
	expectedGroupsToInsert    []groupSpec
	expectedRelationsToCheck  []relationSpec
	expectedRelationsToInsert []map[string]interface{}
	relationsError            bool
	skipRelations             bool
}

func TestApplication_CreateMissingData(t *testing.T) {
	tests := []createMissingDataTestCase{
		{
			name: "create all",
			config: []domain.ConfigItem{
				{AllUsersGroup: 2, TempUsersGroup: 4},
			},
			expectedGroupsToInsert: []groupSpec{
				{name: "AllUsers", id: 2}, {name: "TempUsers", id: 4},
			},
			expectedRelationsToCheck: []relationSpec{
				{ParentChild: database.ParentChild{ParentID: 2, ChildID: 4}},
			},
			expectedRelationsToInsert: []map[string]interface{}{
				{"parent_group_id": int64(2), "child_group_id": int64(4)},
			},
		},
		{
			name: "create some",
			config: []domain.ConfigItem{
				{AllUsersGroup: 6, TempUsersGroup: 8},
			},
			expectedGroupsToInsert: []groupSpec{
				{name: "AllUsers", id: 6, exists: true},
				{name: "TempUsers", id: 8},
			},
			expectedRelationsToCheck: []relationSpec{
				{ParentChild: database.ParentChild{ParentID: 6, ChildID: 8}},
			},
			expectedRelationsToInsert: []map[string]interface{}{
				{"parent_group_id": int64(6), "child_group_id": int64(8)},
			},
		},
		{
			name: "error on group checking",
			config: []domain.ConfigItem{
				{AllUsersGroup: 2, TempUsersGroup: 4},
			},
			expectedGroupsToInsert: []groupSpec{
				{name: "AllUsers", id: 2, error: true},
			},
		},
		{
			name: "error on group insertion",
			config: []domain.ConfigItem{
				{AllUsersGroup: 2, TempUsersGroup: 4},
			},
			expectedGroupsToInsert: []groupSpec{
				{name: "AllUsers", id: 2, errorOnInsert: true},
			},
		},
		{
			name: "error on relation checking",
			config: []domain.ConfigItem{
				{AllUsersGroup: 6, TempUsersGroup: 8},
			},
			expectedGroupsToInsert: []groupSpec{
				{name: "AllUsers", id: 6},
				{name: "TempUsers", id: 8},
			},
			expectedRelationsToCheck: []relationSpec{
				{ParentChild: database.ParentChild{ParentID: 6, ChildID: 8}, error: true},
			},
		},
		{
			name: "error while creating relations",
			config: []domain.ConfigItem{
				{AllUsersGroup: 2, TempUsersGroup: 4},
			},
			expectedGroupsToInsert: []groupSpec{
				{name: "AllUsers", id: 2}, {name: "TempUsers", id: 4},
			},
			expectedRelationsToCheck: []relationSpec{
				{ParentChild: database.ParentChild{ParentID: 2, ChildID: 4}},
			},
			expectedRelationsToInsert: []map[string]interface{}{
				{"parent_group_id": int64(2), "child_group_id": int64(4)},
			},
			relationsError: true,
		},
		{
			name: "no relations to insert",
			config: []domain.ConfigItem{
				{AllUsersGroup: 2, TempUsersGroup: 4},
			},
			expectedGroupsToInsert: []groupSpec{
				{name: "AllUsers", id: 2, exists: true},
				{name: "TempUsers", id: 4, exists: true},
			},
			expectedRelationsToCheck: []relationSpec{
				{ParentChild: database.ParentChild{ParentID: 2, ChildID: 4}, exists: true},
			},
			skipRelations: true,
		},
		{
			name: "only one relation to insert",
			config: []domain.ConfigItem{
				{AllUsersGroup: 2, TempUsersGroup: 4},
			},
			expectedGroupsToInsert: []groupSpec{
				{name: "AllUsers", id: 2, exists: true},
				{name: "TempUsers", id: 4, exists: true},
			},
			expectedRelationsToCheck: []relationSpec{
				{ParentChild: database.ParentChild{ParentID: 2, ChildID: 4}},
			},
			expectedRelationsToInsert: []map[string]interface{}{
				{"parent_group_id": int64(2), "child_group_id": int64(4)},
			},
		},
		{
			name: "only one group to insert",
			config: []domain.ConfigItem{
				{AllUsersGroup: 2, TempUsersGroup: 4},
			},
			expectedGroupsToInsert: []groupSpec{
				{name: "AllUsers", id: 2, exists: true},
				{name: "TempUsers", id: 4},
			},
			expectedRelationsToCheck: []relationSpec{
				{ParentChild: database.ParentChild{ParentID: 2, ChildID: 4}, exists: true},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			db, mock := database.NewDBMock()
			defer func() { _ = db.Close() }()

			var expectedError error
			var createdRelations bool
			monkey.PatchInstanceMethod(reflect.TypeOf(&database.GroupGroupStore{}),
				"CreateRelationsWithoutChecking",
				func(store *database.GroupGroupStore, relations []map[string]interface{}) error {
					assertlib.Equal(t, tt.expectedRelationsToInsert, relations)
					createdRelations = true
					if tt.relationsError {
						expectedError = errors.New("some error")
						return expectedError
					}
					return nil
				})
			defer monkey.UnpatchAll()

			expectedError = setDBExpectationsForCreateMissingData(mock, &tt, expectedError)

			config := viper.New()
			config.Set(domainsConfigKey, tt.config)

			app := &Application{
				Config:   config,
				Database: db,
			}
			err := app.CreateMissingData()
			assertlib.Equal(t, expectedError, err)
			assertlib.Equal(t, (expectedError == nil || tt.relationsError) && !tt.skipRelations, createdRelations)
			assertlib.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func setDBExpectationsForCreateMissingData(mock sqlmock.Sqlmock, tt *createMissingDataTestCase, expectedError error) error {
	mock.ExpectBegin()
	for _, expectedGroupToInsert := range tt.expectedGroupsToInsert {
		expectedError = setDBExpectationsForGroupInCreateMissingData(mock, expectedGroupToInsert, expectedError)
	}
	if expectedError == nil {
		for _, expectedRelationToCheck := range tt.expectedRelationsToCheck {
			rowsToReturn := mock.NewRows([]string{"1"})
			if expectedRelationToCheck.exists {
				rowsToReturn.AddRow(1)
			}
			queryMock := mock.ExpectQuery("^"+regexp.QuoteMeta(
				"SELECT 1 FROM `groups_groups`  WHERE parent_group_id = ? AND child_group_id = ? LIMIT 1",
			)+"$").
				WithArgs(expectedRelationToCheck.ParentID, expectedRelationToCheck.ChildID)
			if !expectedRelationToCheck.error {
				queryMock.WillReturnRows(rowsToReturn)
			} else {
				expectedError = errors.New("some error")
				queryMock.WillReturnError(expectedError)
			}
		}
	}
	if expectedError == nil && !tt.relationsError {
		mock.ExpectCommit()
	} else {
		mock.ExpectRollback()
	}
	return expectedError
}

func setDBExpectationsForGroupInCreateMissingData(mock sqlmock.Sqlmock, expectedGroupToInsert groupSpec, expectedError error) error {
	queryMock := mock.ExpectQuery("^"+regexp.QuoteMeta(
		"SELECT 1 FROM `groups`  WHERE groups.id = ? AND type = 'Base' AND name = ? AND text_id = ? LIMIT 1",
	)+"$").
		WithArgs(expectedGroupToInsert.id, expectedGroupToInsert.name, expectedGroupToInsert.name)
	if !expectedGroupToInsert.error {
		rowsToReturn := mock.NewRows([]string{"1"})
		if expectedGroupToInsert.exists {
			rowsToReturn.AddRow(1)
		}
		queryMock.WillReturnRows(rowsToReturn)
	} else {
		expectedError = errors.New("some error")
		queryMock.WillReturnError(expectedError)
	}
	if !expectedGroupToInsert.exists && !expectedGroupToInsert.error {
		insertMock := mock.ExpectExec("^"+regexp.QuoteMeta(
			"INSERT INTO `groups` (`id`, `name`, `text_id`, `type`) VALUES (?, ?, ?, ?)",
		)+"$").WithArgs(expectedGroupToInsert.id, expectedGroupToInsert.name, expectedGroupToInsert.name, "Base")
		if !expectedGroupToInsert.errorOnInsert {
			insertMock.WillReturnResult(sqlmock.NewResult(expectedGroupToInsert.id, 1))
		} else {
			expectedError = errors.New("some error")
			insertMock.WillReturnError(expectedError)
		}
	}
	return expectedError
}

func TestApplication_insertRootGroupsAndRelations_UnmarshalError(t *testing.T) {
	assert := assertlib.New(t)
	db, _ := database.NewDBMock()
	defer func() { _ = db.Close() }()
	app := &Application{Config: viper.New(), Database: db}
	app.Config.Set("domains", []int{1, 2})
	assert.Contains(app.insertRootGroupsAndRelations(&database.DataStore{DB: db}).Error(), "unable to unmarshal domains config: 2")
}
