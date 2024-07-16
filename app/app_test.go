package app

import (
	crand "crypto/rand"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bou.ke/monkey"
	"github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus" //nolint:depguard
	"github.com/spf13/viper"
	assertlib "github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/appenv"
	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/logging"
	"github.com/France-ioi/AlgoreaBackend/app/version"
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
	assert.Len(app.HTTPHandler.Middlewares(), 9)
	assert.True(len(app.HTTPHandler.Routes()) > 0)
	assert.Equal("/*", app.HTTPHandler.Routes()[0].Pattern) // test default val
}

func TestNew_SuccessNoCompress(t *testing.T) {
	assert := assertlib.New(t)
	appenv.SetDefaultEnvToTest()
	_ = os.Setenv("ALGOREA_SERVER__COMPRESS", "false")
	defer func() { _ = os.Unsetenv("ALGOREA_SERVER__COMPRESS") }()
	app, _ := New()
	assert.Len(app.HTTPHandler.Middlewares(), 8)
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
	assert.Equal(version.Version, response.Header.Get("Backend-Version"))
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
