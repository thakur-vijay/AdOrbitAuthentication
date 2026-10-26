package app

import (
	"database/sql"
	"net/http"

	"github.com/adorbit/authentication/internal/api"
	"github.com/adorbit/authentication/internal/auth"
	"github.com/adorbit/authentication/internal/middleware"
	"github.com/adorbit/authentication/internal/sessions"
)

func BuildRouter(db *sql.DB) http.Handler {

	sessionRepository := sessions.NewRepository(db)
	sessionService := sessions.NewService(sessionRepository)

	// Auth
	authService := auth.NewService(sessionService)
	authHandler := auth.NewHandler(authService)

	authMiddleware := middleware.NewAuthMiddleware(sessionService)

	// Router
	router := api.NewRouter(api.Dependencies{
		AuthHandler:    authHandler,
		AuthMiddleware: authMiddleware,
	})
	return router
	// // Global Middleware
	// return ratelimit.PolicyMiddleware(
	// 	limiters.Default,
	// 	limiters.AuthRefresh,
	// 	limiters.SocialLogin,
	// )(router)
}
