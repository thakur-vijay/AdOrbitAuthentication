package database

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
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
	_, closeErr := authMigrations.Close()

	if err != nil && err != migrate.ErrNoChange {
		return apperrors.NewServiceUnavailable(
			"authentication migration failed",
			err,
		)
	}

	if closeErr != nil {
		return apperrors.NewServiceUnavailable(
			"authentication migration close failed",
			closeErr,
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

	// 3. Open a dedicated database connection for Idempotency migrations.
	migrationDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		source.Close()
		return apperrors.NewServiceUnavailable(
			"idempotency database connection failed",
			err,
		)
	}

	if err := migrationDB.Ping(); err != nil {
		migrationDB.Close()
		source.Close()
		return apperrors.NewServiceUnavailable(
			"idempotency database ping failed",
			err,
		)
	}

	// 4. Configure a separate migration tracking table.
	databaseDriver, err := migratepostgres.WithInstance(
		migrationDB,
		&migratepostgres.Config{
			MigrationsTable: "idempotency_schema_migrations",
		},
	)
	if err != nil {
		migrationDB.Close()
		source.Close()
		return apperrors.NewServiceUnavailable(
			"idempotency database migration setup failed",
			err,
		)
	}

	// 5. Run Idempotency migrations.
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
