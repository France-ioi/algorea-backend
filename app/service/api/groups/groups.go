package groups

import (
	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/go-chi/chi"
)

type GroupsService struct {
	Store *database.DataStore
}

// New creates a service context
func New(store *database.DataStore) *GroupsService {
	return &GroupsService{store}
}

// Router returns the router to the services
func (srv *GroupsService) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", srv.getAll)
	return r
}
