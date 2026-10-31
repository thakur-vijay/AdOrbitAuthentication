package auth

import (
	"net/http"

	"github.com/thakur-vijay/adorbit-go/idempotency"
	"github.com/thakur-vijay/adorbit-go/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	handler *Handler,
	jwtMiddleware *middleware.JWTMiddleware,
	idempotencyMiddleware *idempotency.Middleware,
) {
	signupHandler := idempotencyMiddleware.RequireIdempotencyKey(
		idempotencyMiddleware.RequireIdempotency("auth.signup")(
			http.HandlerFunc(handler.Signup),
		),
	)

	mux.Handle("POST /auth/signup", http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		ctx := idempotency.WithIdempotencyScope(
			r.Context(),
			"public:signup",
		)

		signupHandler.ServeHTTP(w, r.WithContext(ctx))
	}))

	mux.HandleFunc("POST /auth/login", handler.Login)

	mux.Handle(
		"POST /auth/logout",
		jwtMiddleware.RequireAuth(http.HandlerFunc(handler.Logout)),
	)

	mux.Handle(
		"POST /auth/refresh",
		jwtMiddleware.RequireAuth(http.HandlerFunc(handler.Refresh)),
	)
}
