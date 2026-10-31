package database

import (
	"database/sql"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/thakur-vijay/adorbit-go/apperrors"
	"github.com/thakur-vijay/adorbit-go/idempotency"
)

func Migrate(db *sql.DB, databaseURL string) error {
	if databaseURL == "" {
		return apperrors.NewConfigError("Database URL is not set.", nil)
	}

	// 1. Run existing Authentication migrations.
	authMigrations, err := migrate.New(
		"file://internal/database/migrations",
		databaseURL,
	)
	if err != nil {
		return apperrors.NewServiceUnavailable(
			"authentication migration setup failed",
			err,
		)
	}

	err = authMigrations.Up()
	authMigrations.Close()

	if err != nil && err != migrate.ErrNoChange {
		return apperrors.NewServiceUnavailable(
			"authentication migration failed",
			err,
		)
	}

	// 2. Create a separate migration source for Idempotency.
	source, err := iofs.New(idempotency.Migrations, "migrations")
	if err != nil {
		return apperrors.NewServiceUnavailable(
			"idempotency migration source setup failed",
			err,
		)
	}

	// 3. Configure a separate migration tracking table.
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		source.Close()
		return apperrors.NewConfigError(
			"invalid database URL",
			err,
		)
	}

	query := parsedURL.Query()
	query.Set("x-migrations-table", "idempotency_schema_migrations")
	parsedURL.RawQuery = query.Encode()

	// 3. Configure a separate migration tracking table.
	databaseDriver, err := migratepostgres.WithInstance(
		db,
		&migratepostgres.Config{
			MigrationsTable: "idempotency_schema_migrations",
		},
	)
	if err != nil {
		source.Close()
		return apperrors.NewServiceUnavailable(
			"idempotency database migration setup failed",
			err,
		)
	}

	// 4. Run Idempotency migrations.
	idempotencyMigrations, err := migrate.NewWithInstance(
		"iofs",
		source,
		"adorbit_auth",
		databaseDriver,
	)

	if err != nil {
		source.Close()
		databaseDriver.Close()
		return apperrors.NewServiceUnavailable(
			"idempotency migration setup failed",
			err,
		)
	}

	defer idempotencyMigrations.Close()

	err = idempotencyMigrations.Up()
	if err != nil && err != migrate.ErrNoChange {
		return apperrors.NewServiceUnavailable(
			"idempotency migration failed",
			err,
		)
	}

	return nil
}
