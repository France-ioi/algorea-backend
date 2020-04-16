package service

import (
	"net/http"

	"github.com/France-ioi/AlgoreaBackend/app/auth"
	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/token"
	"github.com/spf13/viper"
)

// Auth is the part of the config related to the user authentication
type AuthConfig struct {
	ClientID       string
	ClientSecret   string
	LoginModuleURL string
	CallbackURL    string
}

// Base is the common service context data
type Base struct {
	Store        *database.DataStore
	ServerConfig *viper.Viper
	DomainConfig *viper.Viper
	AuthConfig   *viper.Viper
	TokenConfig  *token.Config
}

// GetUser returns the authenticated user data from context
func (srv *Base) GetUser(r *http.Request) *database.User {
	return auth.UserFromContext(r.Context())
}
