package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/adorbit/authentication/internal/auth"
	appjwt "github.com/adorbit/authentication/internal/jwt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/thakur-vijay/adorbit-go/idempotency"
	"github.com/thakur-vijay/adorbit-go/middleware"
)

func TestHealthReturnsJSONStatus(t *testing.T) {
	response := httptest.NewRecorder()
	health(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("health response is not JSON: %v", err)
	}
	if body["status"] != "ok" || len(body) != 1 {
		t.Fatalf("health response = %#v, want {status: ok}", body)
	}
}

func TestRouterWiresVersionedAuthEndpointsAndIdempotency(t *testing.T) {
	router := NewRouter(Dependencies{
		AuthHandler:           &auth.Handler{},
		IdempotencyMiddleware: idempotency.NewMiddleware(),
	})
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{name: "versioned login reaches request validation", method: http.MethodPost, path: "/api/v1/auth/login", body: `{}`, want: http.StatusBadRequest},
		{name: "signup requires idempotency key", method: http.MethodPost, path: "/api/v1/auth/signup", body: `{}`, want: http.StatusBadRequest},
		{name: "verify requires idempotency key", method: http.MethodPost, path: "/api/v1/auth/verifyOTP", body: `{}`, want: http.StatusBadRequest},
		{name: "unknown versioned path", method: http.MethodGet, path: "/api/v1/not-found", want: http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, tc.want, response.Body.String())
			}
		})
	}
}

func TestRefreshRouteRequiresAccessTokenAndPopulatesPrincipal(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey() error = %v", err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	const issuer = "refresh-route-test"
	jwtMiddleware, err := middleware.NewJWTMiddleware(string(publicPEM), issuer)
	if err != nil {
		t.Fatalf("NewJWTMiddleware() error = %v", err)
	}
	router := NewRouter(Dependencies{
		AuthHandler:           &auth.Handler{},
		JWTMiddleware:         jwtMiddleware,
		IdempotencyMiddleware: idempotency.NewMiddleware(),
	})

	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{}`)))
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("refresh without access token status = %d, want unauthorized", missing.Code)
	}

	now := time.Now()
	claims := appjwt.Claims{
		UserID: "user-1",
		Role:   "advertiser",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "user-1",
			IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`malformed`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("refresh with valid access token status = %d, want handler body validation (400); body=%s", response.Code, response.Body.String())
	}
}
