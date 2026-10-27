package app

import (
	"database/sql"
	"net/http"

	"github.com/adorbit/authentication/internal/api"
	"github.com/adorbit/authentication/internal/auth"
	"github.com/adorbit/authentication/internal/jwt"
)

func BuildRouter(db *sql.DB) (http.Handler, error) {
	// sessionRepository := sessions.NewRepository(db)
	// sessionService := sessions.NewService(sessionRepository)

	jwtRepository := jwt.NewRepository(db)

	privateKey, err := jwt.LoadPrivateKey("keys/private.pem")
	if err != nil {
		return nil, err
	}

	publicKey, err := jwt.LoadPublicKey("keys/public.pem")
	if err != nil {
		return nil, err
	}

	jwtService := jwt.NewService(
		privateKey,
		publicKey,
		jwtRepository,
		"adorbit-authentication",
	)

	// grpcServer, _, err := grpcserver.Start("127.0.0.1:9090", sessionService)
	// if err != nil {
	// 	return nil, nil, err
	// }

	// conn, err := authclient.Dial("127.0.0.1:9090")
	// if err != nil {
	// 	grpcServer.Stop()
	// 	return nil, nil, err
	// }

	// authValidator := authclient.New(conn)
	// authMiddleware := middleware.NewAuthMiddleware(authValidator)

	authService := auth.NewService(jwtService)
	authHandler := auth.NewHandler(authService)

	router := api.NewRouter(api.Dependencies{
		AuthHandler: authHandler,
		// AuthMiddleware: authMiddleware,
	})

	// cleanup := func() {
	// 	_ = conn.Close()
	// 	grpcServer.GracefulStop()
	// }

	return router, nil
}
