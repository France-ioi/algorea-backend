package auth

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/auth"
	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/logging"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

type userIDsInProgressMap sync.Map

func (m *userIDsInProgressMap) withLock(userID int64, r *http.Request, f func() error) error {
	userMutex := make(chan bool)
	defer close(userMutex)
	userMutexInterface, loaded := (*sync.Map)(m).LoadOrStore(userID, userMutex)
	// retry storing our mutex into the map
	for ; loaded; userMutexInterface, loaded = (*sync.Map)(m).LoadOrStore(userID, userMutex) {
		select { // like mutex.Lock(), but with cancel/deadline
		case <-userMutexInterface.(chan bool): // it is much better than <-time.After(...)
		case <-r.Context().Done():
			logging.GetLogEntry(r).Warnf("The request is canceled: %s", r.Context().Err())
			return r.Context().Err()
		}
	}
	defer (*sync.Map)(m).Delete(userID)

	return f()
}

var userIDsInProgress userIDsInProgressMap

func (srv *Service) refreshAccessToken(w http.ResponseWriter, r *http.Request) service.APIError {
	requestData := r.Context().Value(parsedRequestData).(map[string]interface{})
	cookieAttributes, _ := srv.resolveCookieAttributes(r, requestData) // the error has been checked in createAccessToken()

	user := srv.GetUser(r)
	oldAccessToken := auth.BearerTokenFromContext(r.Context())

	var newToken string
	var expiresIn int32
	apiError := service.NoError

	if user.IsTempUser {
		service.MustNotBeError(srv.GetStore(r).InTransaction(func(store *database.DataStore) error {
			sessionStore := store.Sessions()
			// delete all the user's access tokens keeping the input token only
			service.MustNotBeError(sessionStore.Delete("user_id = ? AND access_token != ?",
				user.GroupID, oldAccessToken).Error())
			var err error
			newToken, expiresIn, err = auth.CreateNewTempSession(sessionStore, user.GroupID)
			return err
		}))
	} else {
		// We should not allow concurrency in this part because the login module generates not only
		// a new access token, but also a new refresh token and revokes the old one. We want to prevent
		// usage of the old refresh token for that reason.
		service.MustNotBeError(userIDsInProgress.withLock(user.GroupID, r, func() error {
			newToken, expiresIn, apiError = srv.refreshTokens(r.Context(), srv.GetStore(r), user, oldAccessToken)
			return nil
		}))
	}

	if apiError != service.NoError {
		return apiError
	}
	srv.respondWithNewAccessToken(r, w, service.CreationSuccess, newToken, time.Now().Add(time.Duration(expiresIn)*time.Second),
		cookieAttributes)
	return service.NoError
}

func (srv *Service) refreshTokens(ctx context.Context, store *database.DataStore, user *database.User, oldAccessToken string) (
	newToken string, expiresIn int32, apiError service.APIError,
) {
	var refreshToken string
	err := store.RefreshTokens().Where("user_id = ?", user.GroupID).
		PluckFirst("refresh_token", &refreshToken).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logging.Warnf("No refresh token found in the DB for user %d", user.GroupID)
		return "", 0, service.ErrNotFound(errors.New("no refresh token found in the DB for the authenticated user"))
	}
	service.MustNotBeError(err)
	// oldToken is invalid since its AccessToken is empty, so the lib will refresh it
	oldToken := &oauth2.Token{RefreshToken: refreshToken}
	oauthConfig := auth.GetOAuthConfig(srv.AuthConfig)
	token, err := oauthConfig.TokenSource(ctx, oldToken).Token()
	service.MustNotBeError(err)
	service.MustNotBeError(store.InTransaction(func(store *database.DataStore) error {
		sessionStore := store.Sessions()
		// delete all the user's access tokens keeping the input token only
		service.MustNotBeError(sessionStore.Delete("user_id = ? AND access_token != ?",
			user.GroupID, oldAccessToken).Error())
		// insert the new access token
		service.MustNotBeError(sessionStore.InsertNewOAuth(user.GroupID, token.AccessToken,
			int32(time.Until(token.Expiry)/time.Second), "login-module"))
		if refreshToken != token.RefreshToken {
			service.MustNotBeError(store.RefreshTokens().Where("user_id = ?", user.GroupID).
				UpdateColumn("refresh_token", token.RefreshToken).Error())
		}
		newToken = token.AccessToken
		expiresIn = int32(time.Until(token.Expiry).Round(time.Second) / time.Second)
		return nil
	}))
	return newToken, expiresIn, service.NoError
}
