package app

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	"github.com/France-ioi/AlgoreaBackend/app/api"
	"github.com/France-ioi/AlgoreaBackend/app/appenv"
	"github.com/France-ioi/AlgoreaBackend/app/config"
	"github.com/France-ioi/AlgoreaBackend/app/database"
	_ "github.com/France-ioi/AlgoreaBackend/app/doc" // for doc generation
	"github.com/France-ioi/AlgoreaBackend/app/domain"
	"github.com/France-ioi/AlgoreaBackend/app/logging"
	"github.com/France-ioi/AlgoreaBackend/app/token"
)

// Application is the core state of the app
type Application struct {
	HTTPHandler *chi.Mux
	Config      *config.Root
	Database    *database.DB
	TokenConfig *token.Config
}

// New configures application resources and routes.
func New() (*Application, error) {
	var err error

	conf := config.Load() // exits on errors

	// Apply the config to the global logger
	logging.SharedLogger.Configure(conf.Logging)

	// Init the PRNG with current time
	rand.Seed(time.Now().UTC().UnixNano())

	var db *database.DB
	dbConfig := conf.Database.Connection.FormatDSN()
	if db, err = database.Open(dbConfig); err != nil {
		logging.WithField("module", "database").Error(err)
		return nil, err
	}

	tokenConfig, err := token.Initialize(&conf.Token)
	if err != nil {
		logging.Error(err)
		return nil, err
	}

	var apiCtx *api.Ctx
	if apiCtx, err = api.NewCtx(conf, db, tokenConfig); err != nil {
		logging.Error(err)
		return nil, err
	}

	// Set up middlewares
	router := chi.NewRouter()

	router.Use(middleware.RealIP)             // must be before logger or any middleware using remote IP
	router.Use(middleware.DefaultCompress)    // apply last on response
	router.Use(middleware.RequestID)          // must be before any middleware using the request id (the logger and the recoverer do)
	router.Use(logging.NewStructuredLogger()) //
	router.Use(middleware.Recoverer)          // must be before logger so that it an log panics

	router.Use(corsConfig().Handler) // no need for CORS if served through the same domain
	router.Use(domain.Middleware(conf.Domains))

	if appenv.IsEnvDev() {
		router.Mount("/debug", middleware.Profiler())
	}
	router.Mount(conf.Server.RootPath, apiCtx.Router())

	return &Application{
		HTTPHandler: router,
		Config:      conf,
		Database:    db,
		TokenConfig: tokenConfig,
	}, nil
}

// SelfCheck checks that the database contains all the required data
func (app *Application) SelfCheck() error {
	groupStore := database.NewDataStore(app.Database).Groups()
	groupGroupStore := groupStore.GroupGroups()
	for _, domainConfig := range app.Config.Domains {
		for _, spec := range []struct {
			name string
			id   int64
		}{
			{name: "Root", id: domainConfig.RootGroup},
			{name: "RootSelf", id: domainConfig.RootSelfGroup},
			{name: "RootAdmin", id: domainConfig.RootAdminGroup},
			{name: "RootTemp", id: domainConfig.RootTempGroup},
		} {
			hasRows, err := groupStore.ByID(spec.id).
				Where("sType = 'Base'").
				Where("sName = ?", spec.name).
				Where("sTextId = ?", spec.name).Limit(1).HasRows()
			if err != nil {
				return err
			}
			if !hasRows {
				return fmt.Errorf("no %s group for domain %q", spec.name, domainConfig.Domains[0])
			}
		}

		for _, spec := range []struct {
			parentName string
			childName  string
			parentID   int64
			childID    int64
		}{
			{parentName: "Root", childName: "RootSelf", parentID: domainConfig.RootGroup, childID: domainConfig.RootSelfGroup},
			{parentName: "Root", childName: "RootAdmin", parentID: domainConfig.RootGroup, childID: domainConfig.RootAdminGroup},
			{parentName: "RootSelf", childName: "RootTemp", parentID: domainConfig.RootSelfGroup, childID: domainConfig.RootTempGroup},
		} {
			hasRows, err := groupGroupStore.Where("sType = 'direct'").
				Where("idGroupParent = ?", spec.parentID).
				Where("idGroupChild = ?", spec.childID).Select("1").Limit(1).HasRows()
			if err != nil {
				return err
			}
			if !hasRows {
				return fmt.Errorf("no %s -> %s link in groups_groups for domain %q",
					spec.parentName, spec.childName, domainConfig.Domains[0])
			}
		}
	}
	return nil
}
