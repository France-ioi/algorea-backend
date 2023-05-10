package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/France-ioi/AlgoreaBackend/app/auth"
	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/domain"
	"github.com/France-ioi/AlgoreaBackend/app/rand"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation POST /auth/temp-user auth tempUserCreate
//
//		---
//		summary: Create a temporary user
//		description: Creates a temporary user and generates an access token valid for 2 hours.
//
//
//	    If attributes of the old and the new 'access_token' cookies are different (or the token is returned in the JSON),
//	    the old cookie gets deleted (otherwise, just overwritten).
//
//	  * The "Authorization" header must not be given.
//
//	  * When `{use_cookie}`=1, at least one of `{cookie_secure}` and `{cookie_same_site}` must be true.
//		parameters:
//			- name: default_language
//				in: query
//				type: string
//				maxLength: 3
//			- name: use_cookie
//				in: query
//				description: If 1, set a cookie instead of returning the OAuth2 code in the data
//				type: integer
//				enum: [0,1]
//				default: 0
//			- name: cookie_secure
//				in: query
//				description: If 1, set the cookie with the `Secure` attribute
//				type: integer
//				enum: [0,1]
//				default: 0
//			- name: cookie_same_site
//				in: query
//				description: If 1, set the cookie with the `SameSite`='Strict' attribute value and with `SameSite`='None' otherwise
//				type: integer
//				enum: [0,1]
//				default: 0
//		responses:
//			"201":
//				description: "Created. Success response with the new access token"
//				in: body
//				schema:
//					"$ref": "#/definitions/userCreateTmpResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) createTempUser(w http.ResponseWriter, r *http.Request) service.APIError {
	cookieAttributes, apiError := srv.resolveCookieAttributesFromRequest(r)
	if apiError != service.NoError {
		return apiError
	}

	if len(r.Header["Authorization"]) != 0 {
		return service.ErrInvalidRequest(errors.New("the 'Authorization' header must not be present"))
	}

	defaultLanguage := database.Default()
	if len(r.URL.Query()["default_language"]) != 0 {
		defaultLanguage = r.URL.Query().Get("default_language")
		if utf8.RuneCountInString(defaultLanguage.(string)) > 3 {
			return service.ErrInvalidRequest(errors.New("the length of default_language should be no more than 3 characters"))
		}
	}

	var token string
	var expiresIn int32

	service.MustNotBeError(srv.GetStore(r).InTransaction(func(store *database.DataStore) error {
		var login string
		var userID int64
		service.MustNotBeError(store.RetryOnDuplicatePrimaryKeyError(func(retryIDStore *database.DataStore) error {
			userID = retryIDStore.NewID()
			return retryIDStore.Groups().InsertMap(map[string]interface{}{
				"id":          userID,
				"type":        "User",
				"created_at":  database.Now(),
				"is_open":     false,
				"send_emails": false,
			})
		}))
		service.MustNotBeError(store.RetryOnDuplicateKeyError("login", "login", func(retryLoginStore *database.DataStore) error {
			login = fmt.Sprintf("tmp-%d", rand.Int31n(99999999-10000000+1)+10000000)
			return retryLoginStore.Users().InsertMap(map[string]interface{}{
				"login_id":         0,
				"login":            login,
				"temp_user":        true,
				"registered_at":    database.Now(),
				"group_id":         userID,
				"default_language": defaultLanguage,
				"last_ip":          strings.SplitN(r.RemoteAddr, ":", 2)[0],
			})
		}))

		service.MustNotBeError(store.Groups().ByID(userID).UpdateColumns(map[string]interface{}{
			"name":        login,
			"description": login,
		}).Error())

		service.MustNotBeError(store.Attempts().InsertMap(map[string]interface{}{
			"participant_id": userID,
			"id":             0,
			"creator_id":     userID,
			"created_at":     database.Now(),
		}))

		domainConfig := domain.ConfigFromContext(r.Context())
		service.MustNotBeError(store.GroupGroups().CreateRelationsWithoutChecking(
			[]map[string]interface{}{{"parent_group_id": domainConfig.TempUsersGroupID, "child_group_id": userID}}))

		var err error
		token, expiresIn, err = auth.CreateNewTempSession(store.Sessions(), userID)
		return err
	}))

	srv.respondWithNewAccessToken(r, w, service.CreationSuccess,
		token, time.Now().Add(time.Duration(expiresIn)*time.Second), cookieAttributes)
	return service.NoError
}

func (srv *Service) resolveCookieAttributesFromRequest(r *http.Request) (*auth.SessionCookieAttributes, service.APIError) {
	requestData, apiError := parseCookieAttributesForCreateTempUser(r)
	if apiError != service.NoError {
		return nil, apiError
	}
	cookieAttributes, apiError := srv.resolveCookieAttributes(r, requestData)
	if apiError != service.NoError {
		return nil, apiError
	}
	return cookieAttributes, service.NoError
}

func parseCookieAttributesForCreateTempUser(r *http.Request) (map[string]interface{}, service.APIError) {
	allowedParameters := []string{"use_cookie", "cookie_secure", "cookie_same_site"}
	requestData := make(map[string]interface{}, 2)
	query := r.URL.Query()
	for _, parameterName := range allowedParameters {
		extractOptionalParameter(query, parameterName, requestData)
	}

	return preprocessBooleanCookieAttributes(requestData)
}
