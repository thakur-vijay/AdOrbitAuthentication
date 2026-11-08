//go:build integration

package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/adorbit/authentication/internal/app"
	"github.com/adorbit/authentication/internal/auth"
	"github.com/adorbit/authentication/internal/config"
	appjwt "github.com/adorbit/authentication/internal/jwt"
	"github.com/adorbit/authentication/internal/messaging"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestBuildRouterWiresHealthIdempotencyAndJWTProtectedRoutes(t *testing.T) {
	db := postgresFixture(t)
	startUsersServiceHarness(t)
	rabbit, err := messaging.NewRabbitMQ(requireEnv(t, "TEST_RABBITMQ_URL"))
	if err != nil {
		t.Fatalf("NewRabbitMQ() error = %v", err)
	}
	t.Cleanup(func() {
		if err := rabbit.Close(); err != nil {
			t.Errorf("close RabbitMQ connection: %v", err)
		}
	})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate JWT key: %v", err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	const testMailAPIKey = "test-only-api-key"
	type capturedEmail struct {
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		HTML    string   `json:"html"`
		Text    string   `json:"text"`
	}
	emailMessages := make(chan capturedEmail, 8)
	mailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer "+testMailAPIKey {
			http.Error(w, "unauthorized test mail request", http.StatusUnauthorized)
			return
		}
		var message capturedEmail
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		emailMessages <- message
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"test-email-id"}`)
	}))
	t.Cleanup(mailServer.Close)
	t.Setenv("RESEND_API_KEY", "test-only-api-key")
	t.Setenv("MAIL_FROM_EMAIL", "auth@example.com")
	t.Setenv("MAIL_FROM_NAME", "AdOrbit Tests")
	t.Setenv("MAIL_API_ENDPOINT", mailServer.URL)
	router, err := app.BuildRouter(db, &config.Config{
		JWTPrivateKey:            string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})),
		JWTPublicKey:             string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})),
		OTP_HMAC_SECRET:          []byte("0123456789abcdef0123456789abcdef"),
		IdempotencyScopeKey:      []byte("abcdefghijklmnopqrstuvwxyz123456"),
		IdempotencyEncryptionKey: []byte("12345678901234567890123456789012"),
	}, rabbit)
	if err != nil {
		t.Fatalf("BuildRouter() error = %v", err)
	}

	request := func(method, path, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, jsonRequest(method, path, body))
		return recorder
	}
	if response := request(http.MethodGet, "/health", ""); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"ok"`) {
		t.Fatalf("health route = %d %s", response.Code, response.Body.String())
	}
	if response := request(http.MethodPost, "/api/v1/auth/login", `{}`); response.Code != http.StatusBadRequest {
		t.Fatalf("login validation route status = %d, want 400", response.Code)
	}
	if response := request(http.MethodPost, "/api/v1/auth/signup", `{}`); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "Idempotency-Key") {
		t.Fatalf("signup idempotency route = %d %s", response.Code, response.Body.String())
	}
	if response := request(http.MethodPost, "/api/v1/auth/refresh", `{}`); response.Code != http.StatusUnauthorized {
		t.Fatalf("refresh route without JWT status = %d, want 401", response.Code)
	}
	if response := request(http.MethodPost, "/api/v1/auth/logout", `{}`); response.Code != http.StatusUnauthorized {
		t.Fatalf("logout route without JWT status = %d, want 401", response.Code)
	}

	now := time.Now().UTC()
	claims := appjwt.Claims{
		UserID: "route-test-user",
		Role:   "advertiser",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "adorbit-authentication",
			Subject:   "route-test-user",
			IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign test access token: %v", err)
	}
	validAccessRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{}`))
	validAccessRequest.Header.Set("Content-Type", "application/json")
	validAccessRequest.Header.Set("Authorization", "Bearer "+accessToken)
	validAccessResponse := httptest.NewRecorder()
	router.ServeHTTP(validAccessResponse, validAccessRequest)
	if validAccessResponse.Code != http.StatusBadRequest {
		t.Fatalf("refresh with valid access token did not reach body validation: %d %s", validAccessResponse.Code, validAccessResponse.Body.String())
	}

	applicationServer := httptest.NewServer(router)
	t.Cleanup(applicationServer.Close)
	client := &http.Client{Timeout: 10 * time.Second}
	doRequest := func(method, path, body, key, token string) (int, []byte) {
		t.Helper()
		request, err := http.NewRequest(method, applicationServer.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatalf("construct HTTP request: %v", err)
		}
		request.Header.Set("Content-Type", "application/json")
		if key != "" {
			request.Header.Set("Idempotency-Key", key)
		}
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("send %s %s: %v", method, path, err)
		}
		defer response.Body.Close()
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("read %s %s response: %v", method, path, err)
		}
		return response.StatusCode, responseBody
	}

	const email = "router-e2e@example.com"
	const password = "RouterPass9!"
	signupBody := `{"full_name":"Router E2E","email":"router-e2e@example.com","role":"advertiser","password":"RouterPass9!"}`
	signupKey := uuid.NewString()
	signupStatus, signupBodyBytes := doRequest(http.MethodPost, "/api/v1/auth/signup", signupBody, signupKey, "")
	if signupStatus != http.StatusCreated {
		t.Fatalf("HTTP signup status = %d, want 201; body=%s", signupStatus, signupBodyBytes)
	}
	var signupResult struct {
		Data struct {
			User auth.LoginUserDTO `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(signupBodyBytes, &signupResult); err != nil || signupResult.Data.User.Email != email {
		t.Fatalf("decode HTTP signup response: user=%#v error=%v", signupResult.Data.User, err)
	}
	if strings.Contains(string(signupBodyBytes), password) || strings.Contains(string(signupBodyBytes), "password_hash") {
		t.Fatalf("HTTP signup exposed password material: %s", signupBodyBytes)
	}
	var delivered capturedEmail
	select {
	case delivered = <-emailMessages:
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP signup did not deliver verification email to the controlled test endpoint")
	}
	if len(delivered.To) != 1 || delivered.To[0] != email || !strings.Contains(delivered.Subject, "Verify your email") {
		t.Fatalf("controlled email delivery = %#v", delivered)
	}
	otp := regexp.MustCompile(`[0-9]{6}`).FindString(delivered.Text)
	if otp == "" || !strings.Contains(delivered.HTML, otp) {
		t.Fatal("controlled email did not include a consistent six digit OTP")
	}

	replayStatus, replayBody := doRequest(http.MethodPost, "/api/v1/auth/signup", signupBody, signupKey, "")
	if replayStatus != http.StatusCreated || string(replayBody) != string(signupBodyBytes) {
		t.Fatalf("idempotent signup replay = %d %s, want original response", replayStatus, replayBody)
	}
	select {
	case duplicateEmail := <-emailMessages:
		t.Fatalf("idempotent signup replay sent another email to %v", duplicateEmail.To)
	default:
	}
	if duplicateStatus, _ := doRequest(http.MethodPost, "/api/v1/auth/signup", signupBody, uuid.NewString(), ""); duplicateStatus != http.StatusConflict {
		t.Fatalf("second signup with a new idempotency key status = %d, want conflict", duplicateStatus)
	}

	verifyBody := fmt.Sprintf(`{"email":%q,"role":"advertiser","otp":%q}`, email, otp)
	verifyStatus, verifyBodyBytes := doRequest(http.MethodPost, "/api/v1/auth/verifyOTP", verifyBody, uuid.NewString(), "")
	if verifyStatus != http.StatusOK {
		t.Fatalf("HTTP VerifyOTP status = %d, want 200; body=%s", verifyStatus, verifyBodyBytes)
	}
	var verifyResult struct {
		Data struct {
			User         auth.LoginUserDTO `json:"user"`
			AccessToken  string            `json:"access_token"`
			RefreshToken string            `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(verifyBodyBytes, &verifyResult); err != nil || verifyResult.Data.User.ID != signupResult.Data.User.ID || verifyResult.Data.AccessToken == "" || verifyResult.Data.RefreshToken == "" {
		t.Fatalf("decode HTTP verification result: data=%#v error=%v", verifyResult.Data, err)
	}
	if strings.Contains(string(verifyBodyBytes), password) || strings.Contains(string(verifyBodyBytes), otp) {
		t.Fatalf("HTTP verification exposed secret material: %s", verifyBodyBytes)
	}

	loginBody := `{"email":"router-e2e@example.com","password":"RouterPass9!","role":"advertiser"}`
	loginStatus, loginBodyBytes := doRequest(http.MethodPost, "/api/v1/auth/login", loginBody, "", "")
	if loginStatus != http.StatusAccepted {
		t.Fatalf("HTTP login status = %d, want 202; body=%s", loginStatus, loginBodyBytes)
	}
	var loginResult struct {
		Data struct {
			User         auth.LoginUserDTO `json:"user"`
			AccessToken  string            `json:"access_token"`
			RefreshToken string            `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginBodyBytes, &loginResult); err != nil || loginResult.Data.User.ID != signupResult.Data.User.ID || loginResult.Data.AccessToken == "" || loginResult.Data.RefreshToken == "" {
		t.Fatalf("decode HTTP login result: data=%#v error=%v", loginResult.Data, err)
	}

	refreshBody := fmt.Sprintf(`{"refresh_token":%q}`, loginResult.Data.RefreshToken)
	refreshStatus, refreshBodyBytes := doRequest(http.MethodPost, "/api/v1/auth/refresh", refreshBody, "", loginResult.Data.AccessToken)
	if refreshStatus != http.StatusOK {
		t.Fatalf("HTTP refresh status = %d, want 200; body=%s", refreshStatus, refreshBodyBytes)
	}
	var refreshResult struct {
		Data auth.RefreshTokenResponseDTO `json:"data"`
	}
	if err := json.Unmarshal(refreshBodyBytes, &refreshResult); err != nil || refreshResult.Data.AccessToken == "" || refreshResult.Data.RefreshToken == "" {
		t.Fatalf("decode HTTP refresh result: data=%#v error=%v", refreshResult.Data, err)
	}
	logoutStatus, logoutBody := doRequest(http.MethodPost, "/api/v1/auth/logout", "", "", loginResult.Data.AccessToken)
	if logoutStatus != http.StatusOK {
		t.Fatalf("HTTP logout status = %d, want 200; body=%s", logoutStatus, logoutBody)
	}
	postLogoutStatus, postLogoutBody := doRequest(http.MethodPost, "/api/v1/auth/refresh", fmt.Sprintf(`{"refresh_token":%q}`, refreshResult.Data.RefreshToken), "", loginResult.Data.AccessToken)
	if postLogoutStatus == http.StatusOK {
		t.Fatalf("HTTP refresh after logout unexpectedly succeeded: %s", postLogoutBody)
	}
}
