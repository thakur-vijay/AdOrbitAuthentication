package database

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/thakur-vijay/adorbit-go/apperrors"
)

func Migrate(db *sql.DB, databseURL string) error {
	if databseURL == "" {
		return apperrors.NewConfigError("Database URL is not set.", nil)
	}

	m, err := migrate.New("file://internal/database/migrations", databseURL)

	if err != nil {
		return apperrors.NewServiceUnavailable("database migration setup failed", err)
	}

	defer m.Close()

	err = m.Up()

	if err != nil && err != migrate.ErrNoChange {
		return apperrors.NewServiceUnavailable("database migration failed.", err)
	}

	return nil
}
