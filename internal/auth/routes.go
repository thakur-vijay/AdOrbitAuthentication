package auth

import (
	"net/http"

	"github.com/thakur-vijay/adorbit-go/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	handler *Handler,
	jwtMiddleware *middleware.JWTMiddleware,
) {
	mux.HandleFunc("POST /auth/signup", handler.Signup)
	mux.HandleFunc("POST /auth/login", handler.Login)
	mux.Handle("POST /auth/logout", jwtMiddleware.RequireAuth(http.HandlerFunc(handler.Logout)))
	mux.Handle("POST /auth/refresh", jwtMiddleware.RequireAuth(http.HandlerFunc(handler.Refresh)))
}
