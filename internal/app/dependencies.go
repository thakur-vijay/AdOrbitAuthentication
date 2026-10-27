package app

import (
	"database/sql"
	"net/http"

	"github.com/adorbit/authentication/internal/api"
	"github.com/adorbit/authentication/internal/auth"
	"github.com/adorbit/authentication/internal/grpcserver"
	"github.com/adorbit/authentication/internal/sessions"
	"github.com/thakur-vijay/adorbit-go/authclient"
	"github.com/thakur-vijay/adorbit-go/middleware"
)

func BuildRouter(db *sql.DB) (http.Handler, func(), error) {
	sessionRepository := sessions.NewRepository(db)
	sessionService := sessions.NewService(sessionRepository)

	grpcServer, _, err := grpcserver.Start("127.0.0.1:9090", sessionService)
	if err != nil {
		return nil, nil, err
	}

	conn, err := authclient.Dial("127.0.0.1:9090")
	if err != nil {
		grpcServer.Stop()
		return nil, nil, err
	}

	authValidator := authclient.New(conn)
	authMiddleware := middleware.NewAuthMiddleware(authValidator)

	authService := auth.NewService(sessionService)
	authHandler := auth.NewHandler(authService)

	router := api.NewRouter(api.Dependencies{
		AuthHandler:    authHandler,
		AuthMiddleware: authMiddleware,
	})

	cleanup := func() {
		_ = conn.Close()
		grpcServer.GracefulStop()
	}

	return router, cleanup, nil
}
