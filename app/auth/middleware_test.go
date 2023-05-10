package auth

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus/hooks/test" // nolint:depguard
	assertlib "github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/logging"
	"github.com/France-ioi/AlgoreaBackend/app/loggingtest"
)

func TestUserMiddleware(t *testing.T) {
	now := time.Now()
	patch := monkey.Patch(time.Now, func() time.Time { return now })
	defer patch.Unpatch()

	tests := []struct {
		name                     string
		authHeaders              []string
		cookieHeaders            []string
		expectedAccessToken      string
		userIDReturnedByDB       int64
		dbError                  error
		expectedStatusCode       int
		expectedServiceWasCalled bool
		expectedCookie           *http.Cookie
		expectedBody             string
		expectedLogs             string
	}{
		{
			name:                     "valid access token",
			authHeaders:              []string{"Bearer 1234567"},
			expectedAccessToken:      "1234567",
			userIDReturnedByDB:       890123,
			expectedStatusCode:       200,
			expectedServiceWasCalled: true,
			expectedBody:             "user_id:890123\nBearer:1234567",
		},
		{
			name:                     "missing access token",
			expectedStatusCode:       401,
			expectedServiceWasCalled: false,
			expectedBody:             "No access token provided",
		},
		{
			name:                     "database error",
			authHeaders:              []string{"Bearer 123"},
			expectedAccessToken:      "123",
			dbError:                  errors.New("some error"),
			expectedStatusCode:       500,
			expectedServiceWasCalled: false,
			expectedBody:             `{"success":false,"message":"Internal server error","error_text":"Can't validate the access token"}` + "\n",
			expectedLogs:             `level=error msg="Can't validate an access token: some error"`,
		},
		{
			name:                     "expired token",
			authHeaders:              []string{"Bearer abcdefgh"},
			expectedAccessToken:      "abcdefgh",
			expectedStatusCode:       401,
			expectedServiceWasCalled: false,
			expectedBody:             `{"success":false,"message":"Unauthorized","error_text":"Invalid access token"}` + "\n",
		},
		{
			name:                     "spaces before the access token",
			authHeaders:              []string{"Bearer   1234567"},
			expectedStatusCode:       401,
			expectedServiceWasCalled: false,
			expectedBody:             `{"success":false,"message":"Unauthorized","error_text":"No access token provided"}` + "\n",
		},
		{
			name:                     "spaces in access token",
			authHeaders:              []string{"Bearer 123 456 7"},
			expectedStatusCode:       401,
			expectedServiceWasCalled: false,
			expectedBody:             "No access token provided",
		},
		{
			name:                     "ignores other kinds of authorization headers",
			authHeaders:              []string{"Basic aladdin:opensesame"},
			expectedStatusCode:       401,
			expectedServiceWasCalled: false,
			expectedBody:             "No access token provided",
		},
		{
			name:                     "token is too long (should not query the DB)",
			authHeaders:              []string{"Bearer " + strings.Repeat("1", 2001)},
			expectedStatusCode:       401,
			expectedServiceWasCalled: false,
			expectedBody:             "Invalid access token",
		},
		{
			name:                     "works fine with long tokens",
			authHeaders:              []string{"Bearer " + strings.Repeat("1", 2000)},
			expectedStatusCode:       200,
			expectedAccessToken:      strings.Repeat("1", 2000),
			userIDReturnedByDB:       78234,
			expectedServiceWasCalled: true,
			expectedBody:             "user_id:78234\nBearer:" + strings.Repeat("1", 2000),
		},
		{
			name:                     "takes the first access token from headers",
			authHeaders:              []string{"Basic admin:password", "Bearer 1234567", "Bearer abcdefg"},
			expectedAccessToken:      "1234567",
			userIDReturnedByDB:       890123,
			expectedStatusCode:       200,
			expectedServiceWasCalled: true,
			expectedBody:             "user_id:890123",
		},
		{
			name:                     "accepts access token from cookies",
			cookieHeaders:            []string{"cookie=something;access_token=3!1234567!example.org!/api/;key=value"},
			expectedAccessToken:      "1234567",
			userIDReturnedByDB:       890123,
			expectedStatusCode:       200,
			expectedServiceWasCalled: true,
			expectedBody:             "user_id:890123",
		},
		{
			name: "takes the first access token from cookies",
			cookieHeaders: []string{
				"cookie=something;access_token=2!1234567!127.0.0.1!/;key=value;access_token=3!2489101!!",
				"cookie=5678901234",
			},
			expectedAccessToken:      "1234567",
			userIDReturnedByDB:       890123,
			expectedStatusCode:       200,
			expectedServiceWasCalled: true,
			expectedBody:             "user_id:890123",
		},
		{
			name:                     "prefers an access token from the Authorization header if both cookie and the Authorization header are given",
			authHeaders:              []string{"Bearer 1234567"},
			cookieHeaders:            []string{"cookie=5678901234"},
			expectedAccessToken:      "1234567",
			userIDReturnedByDB:       890123,
			expectedStatusCode:       200,
			expectedServiceWasCalled: true,
			expectedBody:             "user_id:890123",
		},
		{
			name:                     "sets user attributes",
			authHeaders:              []string{"Bearer 1234567"},
			expectedAccessToken:      "1234567",
			userIDReturnedByDB:       890123,
			expectedStatusCode:       200,
			expectedServiceWasCalled: true,
			expectedBody: `User:{"GroupID":890123,"Login":"login","LoginID":12345,"DefaultLanguage":"fr",` +
				`"IsAdmin":true,"IsTempUser":true,"AccessGroupID":23456,"NotificationsReadAt":"2019-05-30T11:00:00Z"}`,
		},
		{
			name:                     "sets cookie attributes when there is no cookie",
			authHeaders:              []string{"Bearer 1234567"},
			expectedAccessToken:      "1234567",
			userIDReturnedByDB:       890123,
			expectedStatusCode:       200,
			expectedServiceWasCalled: true,
			expectedBody:             `CookieAttributes:{"UseCookie":false,"Secure":false,"SameSite":false,"Domain":"","Path":""}`,
		},
		{
			name:                     "sets cookie attributes when there is a cookie",
			cookieHeaders:            []string{"access_token=3!1234567!example.org!/api/"},
			expectedAccessToken:      "1234567",
			userIDReturnedByDB:       890123,
			expectedStatusCode:       200,
			expectedServiceWasCalled: true,
			expectedBody:             `CookieAttributes:{"UseCookie":true,"Secure":true,"SameSite":true,"Domain":"example.org","Path":"/api/"}`,
		},
		{
			name:                     "deletes the cookie when both the cookie and the header are given",
			cookieHeaders:            []string{"access_token=3!1234567!example.org!/api/"},
			authHeaders:              []string{"Bearer 2345678"},
			expectedAccessToken:      "2345678",
			userIDReturnedByDB:       890123,
			expectedStatusCode:       200,
			expectedServiceWasCalled: true,
			expectedCookie: &http.Cookie{
				Name:     "access_token",
				Path:     "/api/",
				Domain:   "example.org",
				Expires:  time.Now().UTC().Add(-1000 * time.Second),
				MaxAge:   -1,
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			},
			expectedBody: `CookieAttributes:{"UseCookie":false,"Secure":false,"SameSite":false,"Domain":"","Path":""}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert := assertlib.New(t)

			serviceWasCalled, resp, mock, logHook := callAuthThroughMiddleware(tt.expectedAccessToken, tt.authHeaders, tt.cookieHeaders,
				tt.userIDReturnedByDB, tt.dbError)
			defer func() { _ = resp.Body.Close() }()
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			assert.Equal(tt.expectedStatusCode, resp.StatusCode)
			assert.Equal("application/json; charset=utf-8", resp.Header.Get("Content-Type"))
			assert.Equal(tt.expectedServiceWasCalled, serviceWasCalled)
			if tt.expectedCookie != nil {
				assert.Equal(resp.Cookies()[0].String(), tt.expectedCookie.String())
			}
			assert.Contains(string(bodyBytes), tt.expectedBody)
			logs := (&loggingtest.Hook{Hook: logHook}).GetAllStructuredLogs()
			if tt.expectedLogs == "" {
				assert.Empty(logs)
			} else {
				assert.Contains(logs, tt.expectedLogs)
			}
			assert.NoError(mock.ExpectationsWereMet())
		})
	}
}

type storeProvider struct {
	store *database.DataStore
}

func (sp *storeProvider) GetStore(*http.Request) *database.DataStore { return sp.store }

var _ GetStorer = &storeProvider{}

func callAuthThroughMiddleware(expectedSessionID string, authorizationHeaders, cookieHeaders []string,
	userID int64, dbError error,
) (bool, *http.Response, sqlmock.Sqlmock, *test.Hook) {
	dbmock, mock := database.NewDBMock()
	logHook, restoreFunc := logging.MockSharedLoggerHook()
	defer restoreFunc()
	defer func() { _ = dbmock.Close() }()
	if expectedSessionID != "" {
		expectation := mock.ExpectQuery("^" +
			regexp.QuoteMeta(
				"SELECT users.login, users.login_id, users.is_admin, users.group_id, users.access_group_id, "+
					"users.temp_user, users.notifications_read_at, users.default_language "+
					"FROM `sessions` "+
					"JOIN users ON users.group_id = sessions.user_id "+
					"WHERE access_token = ? AND expires_at > NOW() LIMIT 1") +
			"$").WithArgs(expectedSessionID)
		if dbError != nil {
			expectation.WillReturnError(dbError)
		} else {
			neededRows := mock.NewRows([]string{
				"group_id", "login", "login_id", "is_admin", "access_group_id", "temp_user",
				"notifications_read_at", "default_language",
			})
			if userID != 0 {
				neededRows = neededRows.AddRow(userID, "login", "12345", int64(1), int64(23456), int64(1),
					[]byte("2019-05-30 11:00:00"), "fr")
			}
			expectation.WillReturnRows(neededRows)
		}
	}

	// dummy server using the middleware
	middleware := UserMiddleware(&storeProvider{database.NewDataStore(dbmock)})
	enteredService := false // used to log if the service has been reached
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enteredService = true // has passed into the service
		user := r.Context().Value(ctxUser).(*database.User)
		cookieAttributes, _ := json.Marshal(r.Context().Value(ctxSessionCookieAttributes))
		userAttributes, _ := json.Marshal(r.Context().Value(ctxUser))
		body := "user_id:" + strconv.FormatInt(user.GroupID, 10) + "\nBearer:" + r.Context().Value(ctxBearer).(string) +
			"\nCookieAttributes:" + string(cookieAttributes) + "\nUser:" + string(userAttributes)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
	mainSrv := httptest.NewServer(middleware(handler))
	defer mainSrv.Close()

	// calling web server
	mainRequest, _ := http.NewRequest("GET", mainSrv.URL, http.NoBody)
	for _, header := range authorizationHeaders {
		mainRequest.Header.Add("Authorization", header)
	}
	for _, header := range cookieHeaders {
		mainRequest.Header.Add("Cookie", header)
	}
	client := &http.Client{}
	resp, _ := client.Do(mainRequest)

	return enteredService, resp, mock, logHook
}
