// Package auth provides API services related to authentication
package auth

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"golang.org/x/oauth2"

	"github.com/France-ioi/AlgoreaBackend/app/auth"
	"github.com/France-ioi/AlgoreaBackend/app/config"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// Service is the mount point for services related to `auth`
type Service struct {
	service.Base
}

// SetRoutes defines the routes for this package in a route group
func (srv *Service) SetRoutes(router chi.Router) {
	router.Use(render.SetContentType(render.ContentTypeJSON))
	router.Post("/auth/temp-user", service.AppHandler(srv.createTempUser).ServeHTTP)

	router.Post("/auth/token", service.AppHandler(srv.createAccessToken).ServeHTTP)
	router.With(auth.UserMiddleware(srv.Store.Sessions())).
		Post("/auth/logout", service.AppHandler(srv.logout).ServeHTTP)
}

func getOAuthConfig(conf *config.Auth) *oauth2.Config {
	oauthConfig := oauth2.Config{
		ClientID:     conf.ClientID,
		ClientSecret: conf.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  conf.LoginModuleURL + "/oauth/authorize",
			TokenURL: conf.LoginModuleURL + "/oauth/token",

			// AuthStyle optionally specifies how the endpoint wants the
			// client id & client secret sent. The zero value means to
			// auto-detect.
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: conf.CallbackURL,
		Scopes:      []string{"account"},
	}
	return &oauthConfig
}
