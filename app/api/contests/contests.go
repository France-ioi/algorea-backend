// Package contests provides API services for contests managing
package contests

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/auth"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// Service is the mount point for services related to `contests`
type Service struct {
	service.Base
}

// SetRoutes defines the routes for this package in a route contests
func (srv *Service) SetRoutes(router chi.Router) {
	router.Use(render.SetContentType(render.ContentTypeJSON))
	router.Use(auth.UserMiddleware(srv.Store.Sessions()))

	router.Get("/contests/{item_id}/group-by-name", service.AppHandler(srv.getGroupByName).ServeHTTP)
}
