package config

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func Migrate(resourcesFolder embed.FS, name ...string) (*sql.DB, error) {
	dbName := "test.db"
	if len(name) > 0 {
		dbName = name[0]
	}

	dsn := fmt.Sprintf("file:%s?cache=shared&mode=memory", dbName)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate sqlit3 driver: %w", err)
	}

	sd, fsErr := iofs.New(resourcesFolder, "resources/migrations")
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
