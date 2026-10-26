package app

import (
	"context"
	"database/sql"

	"github.com/adorbit/authentication/internal/apperrors"
	"github.com/adorbit/authentication/internal/config"
	"github.com/adorbit/authentication/internal/database"
)

type App struct {
	DB *sql.DB
	// Router http.Handler
}

func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, apperrors.NewConfigError("app configuration required", nil)
	}

	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	if err := database.Migrate(db, cfg.DatabaseURL); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &App{DB: db}, nil
}
