package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/logging"
)

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxBearer
	ctxSessionCookieAttributes
)

// GetStorer is an interface allowing to get a data store bound to the context of the given request.
type GetStorer interface {
	GetStore(r *http.Request) *database.DataStore
}

// UserMiddleware is a middleware retrieving a user from the request content.
// It takes the access token from the 'Authorization' header and loads the user info from the DB.
func UserMiddleware(service GetStorer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var user database.User
			var authorized bool

			accessToken, cookieAttributes := ParseSessionCookie(r)

			for _, authValue := range r.Header["Authorization"] {
				parsedAuthValue := strings.SplitN(authValue, " ", 3)
				// credentials = "Bearer" 1*SP b64token (see https://tools.ietf.org/html/rfc6750#section-2.1)
				if len(parsedAuthValue) == 2 && parsedAuthValue[0] == "Bearer" {
					accessToken = parsedAuthValue[1]
					// Delete the cookie since the Authorization header is given
					deleteSessionCookie(w, &cookieAttributes)
					break
				}
			}

			if accessToken == "" {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"message":"Unauthorized","error_text":"No access token provided"}` + "\n"))
				return
			}

			if len(accessToken) <= 2000 {
				err := service.GetStore(r).Sessions().
					Select(`
						users.login, users.login_id, users.is_admin, users.group_id, users.access_group_id,
						users.temp_user, users.notifications_read_at, users.default_language`).
					Joins("JOIN users ON users.group_id = sessions.user_id").
					Where("access_token = ?", accessToken).
					Where("expires_at > NOW()").Take(&user).
					Error()
				authorized = err == nil
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					logging.Errorf("Can't validate an access token: %s", err)
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"success":false,"message":"Internal server error","error_text":"Can't validate the access token"}` + "\n"))
					return
				}
			}

			if !authorized {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"message":"Unauthorized","error_text":"Invalid access token"}` + "\n"))
				return
			}

			ctx := context.WithValue(r.Context(), ctxBearer, accessToken)
			ctx = context.WithValue(ctx, ctxSessionCookieAttributes, &cookieAttributes)
			ctx = context.WithValue(ctx, ctxUser, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ParseSessionCookie parses the 'access_token' cookie (if given) and returns the access token among with cookie attributes.
func ParseSessionCookie(r *http.Request) (accessToken string, cookieAttributes SessionCookieAttributes) {
	if cookie, cookieErr := r.Cookie("access_token"); cookieErr == nil {
		accessToken, cookieAttributes = unmarshalSessionCookieValue(cookie.Value)
	}
	return accessToken, cookieAttributes
}

func deleteSessionCookie(w http.ResponseWriter, cookieAttributes *SessionCookieAttributes) {
	if cookieAttributes.UseCookie {
		http.SetCookie(w, cookieAttributes.SessionCookie("", -1000))
		*cookieAttributes = SessionCookieAttributes{}
	}
}
