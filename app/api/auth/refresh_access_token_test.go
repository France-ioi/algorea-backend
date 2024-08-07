package auth

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/France-ioi/AlgoreaBackend/v2/app/auth"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/v2/app/database"
	"github.com/France-ioi/AlgoreaBackend/v2/app/service"
	"github.com/France-ioi/AlgoreaBackend/v2/app/servicetest"
)

func TestService_refreshAccessToken_NotAllowRefreshTokenRaces(t *testing.T) {
	expectedClientID := "1234"
	expectedClientSecret := "secret"
	loginModuleStubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.URL.Path == "/oauth/token" &&
			r.Form.Get("grant_type") == "refresh_token" &&
			r.Form.Get("client_id") == expectedClientID &&
			r.Form.Get("client_secret") == expectedClientSecret {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(
				`{"access_token": "newaccesstoken", "refresh_token": "new` + r.Form.Get("refresh_token") +
					`", "expires_in": 78901234}`))
		}
	}))
	done := make(chan bool)
	doRequest := func(timeout bool) {
		response, mock, logs, err := servicetest.GetResponseForRouteWithMockedDBAndUser(
			"POST", "/auth/token", "", &database.User{GroupID: 2},
			func(mock sqlmock.Sqlmock) {
				if !timeout {
					mock.ExpectQuery("^" +
						regexp.QuoteMeta(
							"SELECT "+
								"token, "+
								"TIMESTAMPDIFF(SECOND, NOW(), expires_at) AS seconds_until_expiry, "+
								"issued_at > (NOW() - INTERVAL 5 MINUTE) AS too_new_to_refresh "+
								"FROM `access_tokens`  WHERE (session_id = ?) ORDER BY expires_at DESC LIMIT 1") + "$").
						WithArgs(sqlmock.AnyArg()).
						WillReturnRows(mock.NewRows([]string{"token", "seconds_until_expiry", "too_new_to_refresh"}).
							AddRow("accesstoken", 600, false))
					mock.ExpectQuery("^" +
						regexp.QuoteMeta("SELECT refresh_token FROM `sessions` WHERE (session_id = ?) LIMIT 1") + "$").
						WithArgs(sqlmock.AnyArg()).
						WillReturnRows(mock.NewRows([]string{"refresh_token"}).AddRow("firstrefreshtoken"))
					mock.ExpectBegin()
					mock.ExpectExec("^"+regexp.QuoteMeta(
						"INSERT INTO `access_tokens` (`expires_at`, `issued_at`, `session_id`, `token`) "+
							"VALUES (NOW() + INTERVAL ? SECOND, NOW(), ?, ?)")+"$").
						WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "newaccesstoken").
						WillReturnResult(sqlmock.NewResult(123, 1))
					mock.ExpectExec("^"+regexp.QuoteMeta("UPDATE `sessions` SET `refresh_token` = ? WHERE (session_id = ?)")+"$").
						WithArgs("newfirstrefreshtoken", sqlmock.AnyArg()).
						WillReturnResult(sqlmock.NewResult(-1, 1))
					mock.ExpectCommit()
					mock.ExpectBegin()
					mock.ExpectExec("^" + regexp.QuoteMeta(
						"DELETE FROM `access_tokens`  WHERE "+
							"(session_id IN ((SELECT session_id FROM `sessions`  WHERE (user_id = ?)))) AND (expires_at < NOW())",
					) + "$").
						WithArgs(sqlmock.AnyArg()).
						WillReturnResult(sqlmock.NewResult(-1, 1))
					mock.ExpectCommit()
				}
			},
			func(router *chi.Mux, baseService *service.Base) {
				srv := &Service{Base: baseService}
				srv.AuthConfig = viper.New()
				srv.AuthConfig.Set("loginModuleURL", loginModuleStubServer.URL)
				srv.AuthConfig.Set("clientID", expectedClientID)
				srv.AuthConfig.Set("clientSecret", expectedClientSecret)
				if timeout {
					router.With(middleware.Timeout(0)).
						With(middleware.WithValue(parsedRequestData, map[string]interface{}{})).
						Post("/auth/token", service.AppHandler(srv.refreshAccessToken).ServeHTTP)
				} else {
					router.
						With(middleware.WithValue(parsedRequestData, map[string]interface{}{})).
						Post("/auth/token", service.AppHandler(srv.refreshAccessToken).ServeHTTP)
				}
			})
		assert.NoError(t, err)
		if err == nil {
			defer func() { _ = response.Body.Close() }()
		}
		if timeout {
			assert.Equal(t, 500, response.StatusCode)
			assert.Contains(t, logs, "The request is canceled: context deadline exceeded")
		} else {
			assert.Equal(t, 201, response.StatusCode)
			body, _ := ioutil.ReadAll(response.Body)
			assert.Equal(t,
				`{"success":true,"message":"created","data":{"access_token":"newaccesstoken","expires_in":78901234}}`+"\n",
				string(body))
		}
		assert.NoError(t, mock.ExpectationsWereMet())
		done <- true
	}

	// check that the service waits while the user is locked
	mutexChannel := make(chan bool, 1)
	(*sync.Map)(&sessionIDsInProgress).Store(auth.MockCtxSessionID, mutexChannel) // lock the session
	mutexChannel <- true
	go doRequest(false)
	mutexChannel <- true // wait until refreshAccessToken() reads from the channel (meaning the service is inside the for loop)
	close(mutexChannel)
	(*sync.Map)(&sessionIDsInProgress).Delete(auth.MockCtxSessionID) // here the service gets unlocked
	<-done                                                           // wait until the service finishes

	// check that the service timeouts if the user is locked for too long
	mutexChannel = make(chan bool, 1)
	(*sync.Map)(&sessionIDsInProgress).Store(auth.MockCtxSessionID, mutexChannel) // lock the session
	// Remove the mutex once we're finished, otherwise it makes further tests block if they use the same session ID.
	defer (*sync.Map)(&sessionIDsInProgress).Delete(auth.MockCtxSessionID)
	mutexChannel <- true
	go doRequest(true)
	<-done // wait until the service finishes
}
