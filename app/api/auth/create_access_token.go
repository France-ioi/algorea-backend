package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/render"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/auth"
	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/domain"
	"github.com/France-ioi/AlgoreaBackend/app/loginmodule"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

type ctxKey int

const parsedRequestData ctxKey = iota

// swagger:operation POST /auth/token auth accessTokenCreate
//
//		---
//		summary: Create or refresh an access token
//		description:
//	    If the `{code}` is given and the "Authorization" header is not given,
//	    the service converts the given OAuth2 authorization code into tokens,
//	    creates or updates the authenticated user in the DB with the data returned by the login module,
//	    and saves new access & refresh tokens into the DB as well.
//	    If OAuth2 authentication has used the PKCE extension, the `{code_verifier}` should be provided
//	    so it can be sent together with the `{code}` to the authentication server.
//
//
//	    If the `{code}` is not given while the "Authorization" header or/and the "access_token" is given
//	    (when both are given, the "Authorization" header is used and the cookie gets deleted),
//	    the service refreshes the access token
//	    (locally for temporary users or via the login module for normal users) and
//	    saves it into the DB keeping only the input token and the new token.
//	    Since the login module responds with both access and refresh tokens, the service updates the user's
//	    refresh token in this case as well. If there is no refresh token for the user in the DB,
//	    the 'not found' error is returned.
//
//
//	    If attributes of the old and the new 'access_token' cookies are different (or the token is returned in the JSON),
//	    the old cookie gets deleted (otherwise, just overwritten).
//
//
//	  * The "Authorization" header is not allowed when the `{code}` is given.
//
//	  * When `{use_cookie}`=1, at least one of `{cookie_secure}` and `{cookie_same_site}` must be true.
//		security: []
//		consumes:
//			- application/json
//			- application/x-www-form-urlencoded
//		parameters:
//			- name: code
//				in: query
//				description: OAuth2 code (can also be given in form data)
//				type: string
//			- name: code_verifier
//				in: query
//				description: OAuth2 PKCE code verifier  (can also be given in form data)
//				type: string
//			- name: redirect_uri
//				in: query
//				description: OAuth2 redirection URI
//				type: string
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
//			- in: body
//				name: parameters
//				description: The optional parameters can be given in the body as well
//				schema:
//					type: object
//					properties:
//						code:
//							type: string
//							description: OAuth2 code
//						code_verifier:
//							type: string
//							description: OAuth2 PKCE code verifier
//						redirect_uri:
//							type: string
//							description: OAuth2 redirection URI
//						use_cookie:
//							type: boolean
//							description: If true, set a cookie instead of returning the OAuth2 code in the data
//						cookie_secure:
//							type: boolean
//							description: If true, set the cookie with the `Secure` attribute
//						cookie_same_site:
//							type: boolean
//							description: If true, set the cookie with the `SameSite`='Strict' attribute value and with `SameSite`='None' otherwise
//		responses:
//			"201":
//				description: "Created. Success response with the new access token"
//				in: body
//				schema:
//					"$ref": "#/definitions/userCreateTmpResponse"
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"404":
//				"$ref": "#/responses/notFoundResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) createAccessToken(w http.ResponseWriter, r *http.Request) service.APIError {
	requestData, apiError := parseRequestParametersForCreateAccessToken(r)
	if apiError != service.NoError {
		return apiError
	}

	cookieAttributes, apiError := srv.resolveCookieAttributes(r, requestData)
	if apiError != service.NoError {
		return apiError
	}

	code, codeGiven := requestData["code"]
	if codeGiven {
		if len(r.Header["Authorization"]) != 0 {
			return service.ErrInvalidRequest(
				errors.New("only one of the 'code' parameter and the 'Authorization' header can be given"))
		}
	} else {
		// The code is not given, requesting a new token from the given token
		auth.UserMiddleware(srv.Base)(service.AppHandler(srv.refreshAccessToken)).
			ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), parsedRequestData, requestData)))
		return service.NoError
	}

	oauthConfig := auth.GetOAuthConfig(srv.AuthConfig)
	oauthOptions := make([]oauth2.AuthCodeOption, 0, 1)
	if codeVerifier, ok := requestData["code_verifier"]; ok {
		oauthOptions = append(oauthOptions, oauth2.SetAuthURLParam("code_verifier", codeVerifier.(string)))
	}
	if redirectURI, ok := requestData["redirect_uri"]; ok {
		oauthOptions = append(oauthOptions, oauth2.SetAuthURLParam("redirect_uri", redirectURI.(string)))
	}

	token, err := oauthConfig.Exchange(r.Context(), code.(string), oauthOptions...)
	service.MustNotBeError(err)

	userProfile, err := loginmodule.NewClient(srv.AuthConfig.GetString("loginModuleURL")).GetUserProfile(r.Context(), token.AccessToken)
	service.MustNotBeError(err)
	userProfile["last_ip"] = strings.SplitN(r.RemoteAddr, ":", 2)[0]

	domainConfig := domain.ConfigFromContext(r.Context())

	service.MustNotBeError(srv.GetStore(r).InTransaction(func(store *database.DataStore) error {
		userID := createOrUpdateUser(store.Users(), userProfile, domainConfig)
		service.MustNotBeError(store.Groups().StoreBadges(userProfile["badges"].([]database.Badge), userID, true))
		service.MustNotBeError(store.Sessions().InsertNewOAuth(userID, token.AccessToken,
			int32(time.Until(token.Expiry)/time.Second), "login-module"))

		service.MustNotBeError(store.Exec(
			"INSERT INTO refresh_tokens (user_id, refresh_token) VALUES (?, ?) ON DUPLICATE KEY UPDATE refresh_token = ?",
			userID, token.RefreshToken, token.RefreshToken).Error())

		return nil
	}))

	srv.respondWithNewAccessToken(r, w, service.CreationSuccess, token.AccessToken, token.Expiry, cookieAttributes)
	return service.NoError
}

func (srv *Service) respondWithNewAccessToken(r *http.Request, w http.ResponseWriter,
	rendererGenerator func(interface{}) render.Renderer, token string, expiresIn time.Time,
	cookieAttributes *auth.SessionCookieAttributes,
) {
	secondsUntilExpiry := int32(time.Until(expiresIn).Round(time.Second) / time.Second)
	response := map[string]interface{}{
		"expires_in": secondsUntilExpiry,
	}
	oldCookieAttributes := auth.SessionCookieAttributesFromContext(r.Context())
	if oldCookieAttributes == nil {
		oldCookieAttributes = &auth.SessionCookieAttributes{}
		_, *oldCookieAttributes = auth.ParseSessionCookie(r)
	}
	if oldCookieAttributes != nil && oldCookieAttributes.UseCookie && *oldCookieAttributes != *cookieAttributes {
		http.SetCookie(w, oldCookieAttributes.SessionCookie("", -1000))
	}
	if cookieAttributes.UseCookie {
		http.SetCookie(w, cookieAttributes.SessionCookie(token, secondsUntilExpiry))
	} else {
		response["access_token"] = token
	}
	service.MustNotBeError(render.Render(w, r, rendererGenerator(response)))
}

func (srv *Service) resolveCookieAttributes(r *http.Request, requestData map[string]interface{}) (
	cookieAttributes *auth.SessionCookieAttributes, apiError service.APIError,
) {
	cookieAttributes = &auth.SessionCookieAttributes{}
	if value, ok := requestData["use_cookie"]; ok && value.(bool) {
		cookieAttributes.UseCookie = true
		cookieAttributes.Domain = domain.CurrentDomainFromContext(r.Context())
		cookieAttributes.Path = srv.ServerConfig.GetString("rootPath")
		if value, ok := requestData["cookie_secure"]; ok && value.(bool) {
			cookieAttributes.Secure = true
		}
		if value, ok := requestData["cookie_same_site"]; ok && value.(bool) {
			cookieAttributes.SameSite = true
		}
		if !cookieAttributes.Secure && !cookieAttributes.SameSite {
			return nil, service.ErrInvalidRequest(errors.New("one of cookie_secure and cookie_same_site must be true when use_cookie is true"))
		}
	}
	return cookieAttributes, service.NoError
}

func parseRequestParametersForCreateAccessToken(r *http.Request) (map[string]interface{}, service.APIError) {
	allowedParameters := []string{
		"code", "code_verifier", "redirect_uri",
		"use_cookie", "cookie_secure", "cookie_same_site",
	}
	requestData := make(map[string]interface{}, 2)
	query := r.URL.Query()
	for _, parameterName := range allowedParameters {
		extractOptionalParameter(query, parameterName, requestData)
	}

	contentType := strings.ToLower(strings.TrimSpace(strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0]))
	switch contentType {
	case "application/json":
		if apiError := parseJSONParams(r, requestData); apiError != service.NoError {
			return nil, apiError
		}
	case "application/x-www-form-urlencoded":
		err := r.ParseForm()
		if err != nil {
			return nil, service.ErrInvalidRequest(err)
		}
		for _, parameterName := range allowedParameters {
			extractOptionalParameter(r.PostForm, parameterName, requestData)
		}
	}
	return preprocessBooleanCookieAttributes(requestData)
}

func parseJSONParams(r *http.Request, requestData map[string]interface{}) service.APIError {
	var jsonPayload struct {
		Code           *string `json:"code"`
		CodeVerifier   *string `json:"code_verifier"`
		RedirectURI    *string `json:"redirect_uri"`
		UseCookie      *bool   `json:"use_cookie"`
		CookieSecure   *bool   `json:"cookie_secure"`
		CookieSameSite *bool   `json:"cookie_same_site"`
	}
	defer func() { _, _ = io.Copy(ioutil.Discard, r.Body) }()
	err := json.NewDecoder(r.Body).Decode(&jsonPayload)
	if err != nil {
		return service.ErrInvalidRequest(err)
	}
	if jsonPayload.Code != nil {
		requestData["code"] = *jsonPayload.Code
	}
	if jsonPayload.CodeVerifier != nil {
		requestData["code_verifier"] = *jsonPayload.CodeVerifier
	}
	if jsonPayload.RedirectURI != nil {
		requestData["redirect_uri"] = *jsonPayload.RedirectURI
	}
	bool2String := map[bool]string{false: "0", true: "1"}
	if jsonPayload.UseCookie != nil {
		requestData["use_cookie"] = bool2String[*jsonPayload.UseCookie]
	}
	if jsonPayload.CookieSecure != nil {
		requestData["cookie_secure"] = bool2String[*jsonPayload.CookieSecure]
	}
	if jsonPayload.CookieSameSite != nil {
		requestData["cookie_same_site"] = bool2String[*jsonPayload.CookieSameSite]
	}
	return service.NoError
}

func preprocessBooleanCookieAttributes(requestData map[string]interface{}) (map[string]interface{}, service.APIError) {
	for _, flagName := range []string{"use_cookie", "cookie_secure", "cookie_same_site"} {
		if stringValue, ok := requestData[flagName]; ok {
			if _, ok = map[string]bool{"0": false, "1": true}[stringValue.(string)]; !ok {
				return nil, service.ErrInvalidRequest(fmt.Errorf("wrong value for %s (should have a boolean value (0 or 1))", flagName))
			}
			delete(requestData, flagName)
			if stringValue == "1" {
				requestData[flagName] = true
			}
		}
	}
	return requestData, service.NoError
}

func extractOptionalParameter(query url.Values, paramName string, requestData map[string]interface{}) {
	if len(query[paramName]) != 0 {
		requestData[paramName] = query.Get(paramName)
	}
}

func createOrUpdateUser(s *database.UserStore, userData map[string]interface{}, domainConfig *domain.CtxConfig) int64 {
	var groupID int64
	err := s.WithWriteLock().
		Where("login_id = ?", userData["login_id"]).PluckFirst("group_id", &groupID).Error()

	userData["latest_login_at"] = database.Now()
	userData["latest_activity_at"] = database.Now()
	userData["latest_profile_sync_at"] = database.Now()

	if defaultLanguage, ok := userData["default_language"]; ok && defaultLanguage == nil {
		userData["default_language"] = database.Default()
	}

	badges := userData["badges"]
	delete(userData, "badges")
	defer func() { userData["badges"] = badges }()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		selfGroupID := createGroupsFromLogin(s.Groups(), userData["login"].(string), domainConfig)
		userData["temp_user"] = 0
		userData["registered_at"] = database.Now()
		userData["group_id"] = selfGroupID

		service.MustNotBeError(s.Users().InsertMap(userData))
		service.MustNotBeError(s.Attempts().InsertMap(map[string]interface{}{
			"participant_id": selfGroupID,
			"id":             0,
			"creator_id":     selfGroupID,
			"created_at":     database.Now(),
		}))

		return selfGroupID
	}
	service.MustNotBeError(err)

	found, err := s.GroupGroups().WithWriteLock().Where("parent_group_id = ?", domainConfig.AllUsersGroupID).
		Where("child_group_id = ?", groupID).HasRows()
	service.MustNotBeError(err)
	groupsToCreate := make([]map[string]interface{}, 0, 2)
	if !found {
		groupsToCreate = append(groupsToCreate,
			map[string]interface{}{"parent_group_id": domainConfig.AllUsersGroupID, "child_group_id": groupID})
	}

	service.MustNotBeError(s.GroupGroups().CreateRelationsWithoutChecking(groupsToCreate))
	delete(userData, "default_language")
	service.MustNotBeError(s.ByID(groupID).UpdateColumns(userData).Error())
	return groupID
}

func createGroupsFromLogin(store *database.GroupStore, login string, domainConfig *domain.CtxConfig) (selfGroupID int64) {
	service.MustNotBeError(store.RetryOnDuplicatePrimaryKeyError(func(retryIDStore *database.DataStore) error {
		selfGroupID = retryIDStore.NewID()
		return retryIDStore.Groups().InsertMap(map[string]interface{}{
			"id":          selfGroupID,
			"name":        login,
			"type":        "User",
			"description": login,
			"created_at":  database.Now(),
			"is_open":     false,
			"send_emails": false,
		})
	}))

	service.MustNotBeError(store.GroupGroups().CreateRelationsWithoutChecking([]map[string]interface{}{
		{"parent_group_id": domainConfig.AllUsersGroupID, "child_group_id": selfGroupID},
	}))

	return selfGroupID
}
