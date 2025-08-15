package config

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/manuelarte/go-web-layout/resources"
)

func Migrate(db *sql.DB, databaseName string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to instantiate postgres driver: %w", err)
	}

	sd, fsErr := iofs.New(resources.MigrationsFolder, "migrations")
	if fsErr != nil {
		return fmt.Errorf("unable to instantiate migration source from filesystem: %w", fsErr)
	}

	migrator, err := migrate.NewWithInstance("iofs", sd, databaseName, driver)
	if err != nil {
		return fmt.Errorf("failed to instantiate migrator: %w", err)
	}

	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}
