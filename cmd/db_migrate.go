package cmd

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql" // use to force database/sql to use mysql
	"github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"

	"github.com/France-ioi/AlgoreaBackend/app/config"
)

func init() {

	var migrateCmd = &cobra.Command{
		Use:   "db-migrate",
		Short: "apply schema-change migrations to the database",
		Long:  `migrate uses go-pg migration tool under the hood supporting the same commands and an additional reset command`,
		Run: func(cmd *cobra.Command, args []string) {
			var err error

			// load config
			var conf *config.Root
			conf, err = config.Load()
			if err != nil {
				fmt.Println("Unable to load config: ", err)
				os.Exit(1)
			}

			// open DB
			migrations := &migrate.FileMigrationSource{Dir: "db/migrations"}
			var db *sql.DB
			db, err = sql.Open("mysql", conf.Database.Connection.FormatDSN())
			if err != nil {
				fmt.Println("Unable to connect to the database: ", err)
				os.Exit(1)
			}

			// migrate
			var n int
			n, err = migrate.Exec(db, "mysql", migrations, migrate.Up)
			if err != nil {
				fmt.Println("Unable to apply migration:", err)
			} else if n == 0 {
				fmt.Println("No migrations to apply!")
			} else {
				fmt.Printf("%d migration(s) applied successfully!\n", n)
			}

		},
	}

	rootCmd.AddCommand(migrateCmd)
}
