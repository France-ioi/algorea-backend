package cmd

import (
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql" // use to force database/sql to use mysql
	"github.com/spf13/cobra"

	"github.com/France-ioi/AlgoreaBackend/app"
	"github.com/France-ioi/AlgoreaBackend/app/appenv"
	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/database/configdb"
)

// nolint:gosec
func init() { //nolint:gochecknoinits,gocyclo
	installCmd := &cobra.Command{
		Use:   "install [environment]",
		Short: "fill the database with required data",
		Long: `If the root group IDs specified in the config file
do not exist or have missing relations, creates them all
(groups, groups_groups, and groups_ancestors)`,
		Run: func(cmd *cobra.Command, args []string) {
			// if arg given, replace the env
			if len(args) > 0 {
				appenv.SetEnv(args[0])
			}

			appenv.SetDefaultEnv("dev")

			application, err := app.New()
			if err != nil {
				log.Fatal(err)
			}

			domainsConfig, err := app.DomainsConfig(application.Config)
			if err != nil {
				log.Fatal(err)
			}

			err = configdb.CreateMissingData(database.NewDataStore(application.Database), domainsConfig)
			if err != nil {
				log.Fatal(err)
			}

			// Success
			fmt.Println("DONE")
		},
	}

	rootCmd.AddCommand(installCmd)
}
