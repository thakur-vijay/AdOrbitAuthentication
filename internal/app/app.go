package app

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/adorbit/authentication/internal/config"
	"github.com/adorbit/authentication/internal/database"
	"github.com/adorbit/authentication/internal/usersclient"
	"github.com/thakur-vijay/adorbit-go/apperrors"
	"google.golang.org/grpc"
)

type App struct {
	DB            *sql.DB
	Router        http.Handler
	UsersClient   *usersclient.Client
	usersGRPCConn *grpc.ClientConn
	cleanup       func()
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

	usersClient, usersConn, err := usersclient.Dial(cfg.UsersGRPCAddress)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	router, err := BuildRouter(db, cfg.JWTPrivateKey, cfg.JWTPublicKey, usersClient)
	if err != nil {
		_ = usersConn.Close()
		_ = db.Close()
		return nil, err
	}

	return &App{
		DB:            db,
		Router:        router,
		UsersClient:   usersClient,
		usersGRPCConn: usersConn,
	}, nil
}
