package service

import (
	"net/http"

	"github.com/France-ioi/AlgoreaBackend/app/auth"
	"github.com/France-ioi/AlgoreaBackend/app/config"
	"github.com/France-ioi/AlgoreaBackend/app/database"
)

// Base is the common service context data
type Base struct {
	Store  *database.DataStore
	Config *config.Root
}

// GetUser returns the authenticated user data from context
func (srv *Base) GetUser(r *http.Request) *auth.User {
	return auth.UserFromContext(r.Context(), srv.Store.Users())
}
