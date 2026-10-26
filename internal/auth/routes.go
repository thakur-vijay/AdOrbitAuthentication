package auth

import (
	"net/http"

	"github.com/adorbit/authentication/internal/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	handler *Handler,
	authMiddleware *middleware.Middleware,
) {
	mux.HandleFunc("POST /auth/signup", handler.Signup)
	mux.HandleFunc("POST /auth/login", handler.Login)
	mux.Handle("POST /auth/logout", authMiddleware.RequireAuth(http.HandlerFunc(handler.Logout)))
	mux.Handle("POST /auth/refresh", authMiddleware.RequireAuth(http.HandlerFunc(handler.Refresh)))
}
