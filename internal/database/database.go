package database

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/thakur-vijay/adorbit-go/apperrors"
)

func Connect(ctx context.Context, databaseURL string) (*sql.DB, error) {
	if databaseURL == "" {
		return nil, apperrors.NewConfigError("database url is not set.", nil)
	}

	db, err := sql.Open("pgx", databaseURL)

	if err != nil {
		return nil, apperrors.NewServiceUnavailable("database connection failed.", err)
	}

	if ctx != nil {
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, apperrors.NewServiceUnavailable("database ping failed.", err)
		}
	}

	return db, nil
}
