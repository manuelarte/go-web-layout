package config

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/manuelarte/go-web-layout/resources"
)

func Migrate() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "test.db?cache=shared&mode=memory")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate sqlit3 driver: %w", err)
	}

	sd, fsErr := iofs.New(resources.MigrationsFolder, "migrations")
	if fsErr != nil {
		return nil, fmt.Errorf("unable to instantiate migration source from filesystem: %w", fsErr)
	}

	migrator, err := migrate.NewWithInstance("iofs", sd, "test.db", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate migrator: %w", err)
	}

	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}
