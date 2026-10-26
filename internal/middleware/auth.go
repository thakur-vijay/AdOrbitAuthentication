package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/adorbit/authentication/internal/response"
	"github.com/adorbit/authentication/internal/sessions"
)

type contextKey string

const (
	userIDKey    contextKey = "user_id"
	sessionIDKey contextKey = "session_id"
)

type Middleware struct {
	sessionService *sessions.Service
}

func NewAuthMiddleware(sessionService *sessions.Service) *Middleware {
	return &Middleware{sessionService: sessionService}
}

func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")

		if header == "" {
			response.Write(w, http.StatusUnauthorized, "authorization header is required")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Write(w, http.StatusUnauthorized, "authorization header is invalid")
			return
		}

		session, err := m.sessionService.ValidateAccessToken(r.Context(), parts[1])
		if err != nil {
			response.Write(w, http.StatusUnauthorized, "authentication failed")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, session.UserID)
		ctx = context.WithValue(ctx, sessionIDKey, session.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}

func SessionID(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(sessionIDKey).(string)
	return sessionID, ok
}
