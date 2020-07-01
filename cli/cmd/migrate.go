package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
)

type direction string

const (
	up   = direction("up")
	down = direction("down")
)

var (
	migrations     string
	connection     string
	both           bool
	testConnection string
	onlyTest       bool
)

func init() {
	migrateCommand.Flags().StringVarP(&migrations, "migrations", "m", config.Database.Migrations, "migrations source")
	migrateCommand.Flags().StringVarP(&connection, "connection", "c", config.Database.Dev.ConnectionString(), "database connection")
	migrateCommand.Flags().BoolVarP(&both, "both", "", false, "migrate using the database connection and test database connection")
	migrateCommand.Flags().StringVarP(&testConnection, "test-connection", "t", config.Database.Test.ConnectionString(), "test database connection")
	migrateCommand.Flags().BoolVarP(&onlyTest, "only-test", "", false, "only migrate using the test database connection")
	migrateCommand.AddCommand(upCommand, downCommand, createCommand)
}

var migrateCommand = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate database",
}

var upCommand = &cobra.Command{
	Use:   "up [N]",
	Short: "Migrate database up",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		if len(args) > 1 {
			return cobra.MaximumNArgs(1)(cmd, args)
		}
		_, err := strconv.Atoi(args[0])
		return errors.WithMessage(err, "argument must be an integer")
	},
	Run: func(cmd *cobra.Command, args []string) {
		_migrate(up, cmd, args)
	},
}

var downCommand = &cobra.Command{
	Use:   "down [N]",
	Short: "Migrate database down",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		if len(args) > 1 {
			return cobra.MaximumNArgs(1)(cmd, args)
		}
		_, err := strconv.Atoi(args[0])
		return errors.WithMessage(err, "argument must be an integer")
	},
	Run: func(cmd *cobra.Command, args []string) {
		_migrate(down, cmd, args)
	},
}

func _migrate(dir direction, cmd *cobra.Command, args []string) {
	impl := func(m *migrate.Migrate, dir direction, args []string) {
		var err error

		if len(args) == 0 {
			if dir == up {
				err = m.Up()
			} else {
				err = m.Down()
			}

			if err != nil {
				if err.Error() == "no change" {
					fmt.Println("no change")
				} else {
					fatal(errors.WithMessagef(err, "failed to migrate database %s", dir))
				}
			}
			return
		}

		// this has already been validated
		steps, _ := strconv.Atoi(args[0])
		if dir == up {
			err = m.Steps(steps)
		} else {
			err = m.Steps(-steps)
		}

		if err != nil {
			if err.Error() == "no change" {
				fmt.Println("no change")
			} else {
				fatal(errors.WithMessagef(err, "failed to migrate database %s", dir))
			}
		}
	}

	l := logger{cmd}
	if !onlyTest {
		m, err := migrate.New(migrations, connection)
		fatalIf(errors.WithMessage(err, "failed to connect to database"))
		m.Log = l

		if both {
			fmt.Println("migrating dev database")
		}
		impl(m, dir, args)
	}

	if onlyTest || both {
		if both {
			fmt.Println("migrating test database")
		}
		m, err := migrate.New(migrations, testConnection)
		fatalIf(errors.WithMessage(err, "failed to connect to test database"))
		m.Log = l
		impl(m, dir, args)
	}
}

var createCommand = &cobra.Command{
	Use:   "create [NAME]",
	Short: "Create database migration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if strings.HasPrefix(migrations, "file://") {
			migrations = migrations[7:]
		}
		_, err := os.Stat(migrations)
		fatalIf(err)

		filename := time.Now().Format("20060102150405_") + args[0]
		f, err := os.Create(filepath.Join(migrations, filename+".up.sql"))
		fatalIf(err)
		_ = f.Close()
		f, err = os.Create(filepath.Join(migrations, filename+".down.sql"))
		fatalIf(err)
		_ = f.Close()
	},
}

type logger struct {
	*cobra.Command
}

func (l logger) Verbose() bool {
	return false
}
