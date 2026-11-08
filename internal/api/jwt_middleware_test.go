package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appjwt "github.com/adorbit/authentication/internal/jwt"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/thakur-vijay/adorbit-go/middleware"
)

func TestJWTMiddlewareValidatesBearerTokenAndSetsPrincipal(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: mustPublicDER(t, &key.PublicKey)})
	const issuer = "middleware-test"
	jwtMiddleware, err := middleware.NewJWTMiddleware(string(publicPEM), issuer)
	if err != nil {
		t.Fatalf("NewJWTMiddleware() error = %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := middleware.GetPrincipal(r.Context())
		if !ok {
			t.Error("JWT middleware did not add a principal to the request context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if principal.UserID != "user-42" || principal.Role != "publisher" {
			t.Errorf("principal = %#v, want user-42 publisher", principal)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	protected := jwtMiddleware.RequireAuth(next)
	validClaims := appjwt.Claims{
		UserID: "user-42",
		Role:   "publisher",
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "user-42",
			IssuedAt:  jwtlib.NewNumericDate(time.Now().Add(-time.Minute)),
			NotBefore: jwtlib.NewNumericDate(time.Now().Add(-time.Minute)),
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	validToken, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, validClaims).SignedString(key)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "bEaReR "+validToken)
	response := httptest.NewRecorder()
	protected.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("valid token status = %d, want %d; body=%s", response.Code, http.StatusNoContent, response.Body.String())
	}

	for _, header := range []string{"", "Bearer", "Basic " + validToken, "Bearer "} {
		t.Run("header "+header, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", header)
			response := httptest.NewRecorder()
			protected.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want unauthorized", response.Code)
			}
		})
	}

	wrongIssuer := validClaims
	wrongIssuer.Issuer = "wrong"
	wrongIssuerToken, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, wrongIssuer).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	expired := validClaims
	expired.ExpiresAt = jwtlib.NewNumericDate(time.Now().Add(-time.Minute))
	expiredToken, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, expired).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	missingRole := validClaims
	missingRole.Role = ""
	missingRoleToken, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, missingRole).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}

	for _, token := range []string{"malformed", wrongIssuerToken, expiredToken, missingRoleToken} {
		t.Run("invalid token "+token[:min(len(token), 12)], func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			protected.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want unauthorized", response.Code)
			}
		})
	}
}

func mustPublicDER(t *testing.T, key *rsa.PublicKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey() error = %v", err)
	}
	return der
}
