package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlersRejectMalformedAndInvalidRequests(t *testing.T) {
	handler := &Handler{}
	tests := []struct {
		name        string
		run         func(http.ResponseWriter, *http.Request)
		body        string
		contentType string
		want        int
		message     string
	}{
		{name: "signup malformed JSON", run: handler.Signup, body: `{`, want: http.StatusBadRequest},
		{name: "signup missing fields", run: handler.Signup, body: `{}`, want: http.StatusBadRequest},
		{name: "signup invalid email", run: handler.Signup, body: `{"full_name":"Ada","email":"bad","role":"advertiser","password":"GoodPass1!"}`, want: http.StatusBadRequest},
		{name: "signup weak password", run: handler.Signup, body: `{"full_name":"Ada","email":"ada@example.com","role":"advertiser","password":"weak"}`, want: http.StatusBadRequest},
		{name: "signup unsupported content type", run: handler.Signup, body: `{"full_name":"Ada","email":"ada@example.com","role":"advertiser","password":"GoodPass1!"}`, contentType: "text/plain", want: http.StatusUnsupportedMediaType},
		{name: "signup invalid role", run: handler.Signup, body: `{"full_name":"Ada","email":"ada@example.com","role":"admin","password":"GoodPass1!"}`, want: http.StatusBadRequest},
		{name: "login missing password", run: handler.Login, body: `{"email":"ada@example.com","role":"advertiser"}`, want: http.StatusBadRequest},
		{name: "login invalid role", run: handler.Login, body: `{"email":"ada@example.com","password":"secret","role":"admin"}`, want: http.StatusBadRequest},
		{name: "resend invalid email", run: handler.ResendOTP, body: `{"email":"not-an-email"}`, want: http.StatusBadRequest},
		{name: "verify malformed JSON", run: handler.VerifyOTP, body: `[]`, want: http.StatusBadRequest},
		{name: "verify nonnumeric OTP", run: handler.VerifyOTP, body: `{"email":"ada@example.com","role":"advertiser","otp":"12x456"}`, want: http.StatusBadRequest},
		{name: "verify short OTP", run: handler.VerifyOTP, body: `{"email":"ada@example.com","role":"advertiser","otp":"12345"}`, want: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			contentType := tc.contentType
			if contentType == "" {
				contentType = "application/json"
			}
			request.Header.Set("Content-Type", contentType)
			response := httptest.NewRecorder()
			tc.run(response, request)
			if response.Code != tc.want {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, tc.want, response.Body.String())
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			var body struct {
				StatusCode int    `json:"status_code"`
				Message    string `json:"message"`
				Data       any    `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("response is not JSON: %v; body=%s", err, response.Body.String())
			}
			if body.StatusCode != tc.want || body.Message == "" {
				t.Fatalf("error response = %#v, want status %d and a message", body, tc.want)
			}
			if strings.Contains(response.Body.String(), "GoodPass1!") || strings.Contains(response.Body.String(), "12x456") {
				t.Fatalf("response exposed a sensitive request field: %s", response.Body.String())
			}
		})
	}
}

func TestProtectedHandlersRequirePrincipal(t *testing.T) {
	handler := &Handler{}
	for _, tc := range []struct {
		name string
		run  func(http.ResponseWriter, *http.Request)
	}{
		{name: "logout", run: handler.Logout},
		{name: "refresh", run: handler.Refresh},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"refresh_token":"secret"}`))
			tc.run(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
			}
			if strings.Contains(response.Body.String(), "secret") {
				t.Fatalf("response exposed request token: %s", response.Body.String())
			}
		})
	}
}

func TestSignupRejectsOversizedRequestBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(`{}`))
	request.ContentLength = maxRequestBodyBytes + 1
	response := httptest.NewRecorder()
	(&Handler{}).Signup(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(response.Body.String(), "1 MiB") {
		t.Fatalf("oversized request response did not explain the limit: %s", response.Body.String())
	}

	unknownLength := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(strings.Repeat(" ", int(maxRequestBodyBytes+1))))
	unknownLength.ContentLength = -1
	unknownLength.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	(&Handler{}).Signup(response, unknownLength)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("chunked oversized request status = %d, want %d; body=%s", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
	}

	trailingData := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(`{"email":"ada@example.com"}`+strings.Repeat(" ", int(maxRequestBodyBytes))))
	trailingData.ContentLength = -1
	trailingData.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	(&Handler{}).Signup(response, trailingData)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("valid JSON with oversized trailing data status = %d, want %d; body=%s", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
	}
}
