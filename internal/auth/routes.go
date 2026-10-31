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
	signupHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := idempotency.WithIdempotencyScope(
			r.Context(),
			"public:signup",
		)

		r = r.WithContext(ctx)
		handler.Signup(w, r)
	})

	mux.Handle(
		"POST /auth/signup",
		idempotencyMiddleware.RequireIdempotency("auth.signup")(
			idempotencyMiddleware.RequireIdempotencyKey(
				signupHandler,
			),
		),
	)

	mux.HandleFunc("POST /auth/login", handler.Login)

	mux.Handle("POST /auth/logout",
		jwtMiddleware.RequireAuth(http.HandlerFunc(handler.Logout)))

	mux.Handle("POST /auth/refresh",
		jwtMiddleware.RequireAuth(http.HandlerFunc(handler.Refresh)))
}
