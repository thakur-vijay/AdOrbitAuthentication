//go:build integration

package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/adorbit/authentication/internal/auth"
	emailotp "github.com/adorbit/authentication/internal/emailOtp"
	appjwt "github.com/adorbit/authentication/internal/jwt"
	"github.com/adorbit/authentication/internal/messaging"
	"github.com/adorbit/authentication/internal/security"
	"github.com/google/uuid"
	"github.com/thakur-vijay/adorbit-go/mailservice"
)

func TestAuthHTTPWorkflowSignupVerifyLoginRotateAndLogout(t *testing.T) {
	db := postgresFixture(t)
	startUsersServiceHarness(t)
	broker := requireEnv(t, "TEST_RABBITMQ_URL")
	rabbit, err := messaging.NewRabbitMQ(broker)
	if err != nil {
		t.Fatalf("connect authentication service to RabbitMQ: %v", err)
	}
	t.Cleanup(func() {
		if err := rabbit.Close(); err != nil {
			t.Errorf("close authentication RabbitMQ connection: %v", err)
		}
	})

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate integration JWT key: %v", err)
	}
	jwtRepository := appjwt.NewRepository(db)
	jwtService := appjwt.NewService(privateKey, &privateKey.PublicKey, jwtRepository, "integration-test")
	mailer := newCaptureMailer()
	otpRepository := emailotp.NewRepository(db)
	otpService := emailotp.NewService(mailer, otpRepository, []byte("integration OTP HMAC secret"))
	authRepository := auth.NewRepository(db)
	service := auth.NewService(jwtService, authRepository, rabbit, otpService)
	handler := auth.NewHandler(service)
	ctx := context.Background()

	missingVerify := httptest.NewRecorder()
	handler.VerifyOTP(missingVerify, jsonRequest(http.MethodPost, "/verify", `{"email":"missing@example.com","role":"publisher","otp":"123456"}`))
	if missingVerify.Code != http.StatusNotFound {
		t.Fatalf("unknown-email VerifyOTP status = %d, want %d", missingVerify.Code, http.StatusNotFound)
	}
	missingResend := httptest.NewRecorder()
	handler.ResendOTP(missingResend, jsonRequest(http.MethodPost, "/resend", `{"email":"missing@example.com"}`))
	if missingResend.Code != http.StatusNotFound {
		t.Fatalf("unknown-email ResendOTP status = %d, want %d", missingResend.Code, http.StatusNotFound)
	}

	const email = "workflow@example.com"
	const password = "StrongPass9!"
	signupRecorder := httptest.NewRecorder()
	handler.Signup(signupRecorder, jsonRequest(http.MethodPost, "/signup", `{"full_name":"Workflow User","email":"workflow@example.com","role":"publisher","password":"StrongPass9!"}`))
	if signupRecorder.Code != http.StatusCreated {
		t.Fatalf("Signup status = %d, want %d; body=%s", signupRecorder.Code, http.StatusCreated, signupRecorder.Body.String())
	}
	if strings.Contains(signupRecorder.Body.String(), password) || strings.Contains(signupRecorder.Body.String(), "password_hash") {
		t.Fatalf("signup response exposed password material: %s", signupRecorder.Body.String())
	}
	var signupResponse struct {
		StatusCode int `json:"status_code"`
		Data       struct {
			User auth.LoginUserDTO `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(signupRecorder.Body.Bytes(), &signupResponse); err != nil {
		t.Fatalf("decode signup response: %v", err)
	}
	if signupResponse.StatusCode != http.StatusCreated || signupResponse.Data.User.Email != email || signupResponse.Data.User.Role != "publisher" || signupResponse.Data.User.IsEmailVerified {
		t.Fatalf("signup response user = %#v", signupResponse.Data.User)
	}
	userID := signupResponse.Data.User.ID
	if _, err := uuid.Parse(userID); err != nil {
		t.Fatalf("signup returned invalid user ID %q: %v", userID, err)
	}
	if duplicate := serviceSignup(t, handler, email, password); duplicate != http.StatusConflict {
		t.Fatalf("duplicate signup status = %d, want %d", duplicate, http.StatusConflict)
	}
	var statusValue string
	var isEmailVerified bool
	if err := db.QueryRow(`SELECT status, is_email_verified FROM credentials WHERE user_id = $1`, userID).Scan(&statusValue, &isEmailVerified); err != nil {
		t.Fatalf("read credential after signup: %v", err)
	}
	if statusValue != auth.Active || isEmailVerified {
		t.Fatalf("credential after signup = status %q verified %v, want active and unverified", statusValue, isEmailVerified)
	}

	message := receiveMail(t, mailer)
	if len(message.To) != 1 || message.To[0] != email || !strings.Contains(message.Subject, "Verify your email") {
		t.Fatalf("verification email metadata = %#v", message)
	}
	otpMatch := regexp.MustCompile(`([0-9]{6})`).FindString(message.TextBody)
	if otpMatch == "" || !strings.Contains(message.TextBody, otpMatch) || !strings.Contains(message.HTMLBody, otpMatch) {
		t.Fatalf("test mail delivery did not contain the same six digit OTP in both bodies")
	}
	var otpHash string
	if err := db.QueryRow(`SELECT otp_hash FROM email_otps WHERE user_id = $1`, userID).Scan(&otpHash); err != nil {
		t.Fatalf("read stored OTP HMAC: %v", err)
	}
	if otpHash == otpMatch || len(otpHash) != 64 {
		t.Fatalf("OTP storage contains plaintext or malformed HMAC: %q", otpHash)
	}

	wrongOTP := httptest.NewRecorder()
	handler.VerifyOTP(wrongOTP, jsonRequest(http.MethodPost, "/verify", `{"email":"workflow@example.com","role":"publisher","otp":"000000"}`))
	if wrongOTP.Code != http.StatusBadRequest {
		t.Fatalf("incorrect OTP status = %d, want %d; body=%s", wrongOTP.Code, http.StatusBadRequest, wrongOTP.Body.String())
	}

	verifyRecorder := httptest.NewRecorder()
	verifyBody := fmt.Sprintf(`{"email":%q,"role":"publisher","otp":%q}`, email, otpMatch)
	handler.VerifyOTP(verifyRecorder, jsonRequest(http.MethodPost, "/verify", verifyBody))
	if verifyRecorder.Code != http.StatusOK {
		t.Fatalf("VerifyOTP status = %d, want %d; body=%s", verifyRecorder.Code, http.StatusOK, verifyRecorder.Body.String())
	}
	if strings.Contains(verifyRecorder.Body.String(), password) || strings.Contains(verifyRecorder.Body.String(), "password_hash") || strings.Contains(verifyRecorder.Body.String(), otpMatch) {
		t.Fatalf("verification response exposed secret material: %s", verifyRecorder.Body.String())
	}
	var verifyResponse struct {
		Data struct {
			User         auth.LoginUserDTO `json:"user"`
			AccessToken  *string           `json:"access_token"`
			RefreshToken *string           `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(verifyRecorder.Body.Bytes(), &verifyResponse); err != nil {
		t.Fatalf("decode verification response: %v", err)
	}
	if verifyResponse.Data.User.ID != userID || !verifyResponse.Data.User.IsEmailVerified || verifyResponse.Data.AccessToken == nil || verifyResponse.Data.RefreshToken == nil {
		t.Fatalf("verification result = %#v, want verified user and token pair", verifyResponse.Data)
	}
	if principal, err := jwtService.ValidateAccessToken(*verifyResponse.Data.AccessToken); err != nil || principal.UserID != userID || principal.Role != "publisher" {
		t.Fatalf("ValidateAccessToken(signup verification) = (%#v, %v)", principal, err)
	}
	if _, err := service.VerifyOTP(ctx, auth.VerifyOTPDTO{Email: email, Role: "publisher", OTP: otpMatch}); err == nil || !strings.Contains(err.Error(), "already verified") {
		t.Fatalf("repeated OTP verification error = %v, want already verified", err)
	}
	if err := db.QueryRow(`SELECT is_email_verified FROM credentials WHERE user_id = $1`, userID).Scan(&isEmailVerified); err != nil || !isEmailVerified {
		t.Fatalf("credential verification state = (%v, %v), want verified", isEmailVerified, err)
	}
	var verifiedAt sql.NullTime
	if err := db.QueryRow(`SELECT verified_at FROM email_otps WHERE user_id = $1`, userID).Scan(&verifiedAt); err != nil || !verifiedAt.Valid {
		t.Fatalf("OTP verification timestamp = (%v, %v), want set", verifiedAt, err)
	}

	badLogin := serviceLogin(t, handler, email, "IncorrectPass9!", "publisher")
	if badLogin != http.StatusUnauthorized {
		t.Fatalf("invalid-password login status = %d, want %d", badLogin, http.StatusUnauthorized)
	}
	loginRecorder := httptest.NewRecorder()
	handler.Login(loginRecorder, jsonRequest(http.MethodPost, "/login", `{"email":"workflow@example.com","password":"StrongPass9!","role":"publisher"}`))
	if loginRecorder.Code != http.StatusAccepted {
		t.Fatalf("Login status = %d, want %d; body=%s", loginRecorder.Code, http.StatusAccepted, loginRecorder.Body.String())
	}
	var loginResponse struct {
		Data struct {
			User         auth.LoginUserDTO `json:"user"`
			AccessToken  *string           `json:"access_token"`
			RefreshToken *string           `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRecorder.Body.Bytes(), &loginResponse); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResponse.Data.User.ID != userID || loginResponse.Data.User.Role != "publisher" || loginResponse.Data.AccessToken == nil || loginResponse.Data.RefreshToken == nil {
		t.Fatalf("login result = %#v, want identity and token pair", loginResponse.Data)
	}
	var storedHash string
	if err := db.QueryRow(`SELECT token_hash FROM jwt_refresh_tokens WHERE user_id = $1`, userID).Scan(&storedHash); err != nil {
		t.Fatalf("read refresh token hash: %v", err)
	}
	refreshDigest := sha256.Sum256([]byte(*loginResponse.Data.RefreshToken))
	if storedHash != hex.EncodeToString(refreshDigest[:]) || storedHash == *loginResponse.Data.RefreshToken {
		t.Fatal("refresh token was not stored as its SHA-256 hash")
	}

	start := make(chan struct{})
	type refreshResult struct {
		pair *auth.RefreshTokenResponseDTO
		err  error
	}
	results := make(chan refreshResult, 2)
	var group sync.WaitGroup
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			pair, err := service.Refresh(ctx, "publisher", auth.RefreshTokenRequestDTO{RefreshToken: *loginResponse.Data.RefreshToken})
			results <- refreshResult{pair: pair, err: err}
		}()
	}
	close(start)
	group.Wait()
	close(results)
	winners := 0
	var currentRefreshToken string
	for result := range results {
		if result.err == nil {
			winners++
			currentRefreshToken = result.pair.RefreshToken
		} else if !errors.Is(result.err, appjwt.ErrRefreshTokenReused) && !errors.Is(result.err, appjwt.ErrRefreshTokenNotFound) {
			t.Fatalf("concurrent Refresh() error = %v, want one-use conflict or old-token miss", result.err)
		}
	}
	if winners != 1 {
		t.Fatalf("concurrent refresh winners = %d, want exactly one", winners)
	}
	if _, err := jwtService.ValidateRefreshToken(ctx, *loginResponse.Data.RefreshToken); !errors.Is(err, appjwt.ErrRefreshTokenNotFound) {
		t.Fatalf("old refresh token after rotation error = %v, want not found", err)
	}
	if err := service.Logout(ctx, userID); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := jwtService.ValidateRefreshToken(ctx, currentRefreshToken); !errors.Is(err, appjwt.ErrRefreshTokenRevoked) {
		t.Fatalf("refresh after logout error = %v, want revoked", err)
	}
	if _, err := service.Refresh(ctx, "publisher", auth.RefreshTokenRequestDTO{RefreshToken: currentRefreshToken}); !errors.Is(err, appjwt.ErrRefreshTokenRevoked) {
		t.Fatalf("Refresh() after logout error = %v, want revoked", err)
	}
}

func TestEmailOTPServicePersistsHMACAndEnforcesCooldownAndOneTimeUse(t *testing.T) {
	db := postgresFixture(t)
	userID := mustUUID(t)
	insertCredential(t, db, userID, "otp-service@example.com")
	mailer := newCaptureMailer()
	repository := emailotp.NewRepository(db)
	service := emailotp.NewService(mailer, repository, []byte("otp integration hmac secret"))
	ctx := context.Background()
	if err := service.Send(ctx, userID, "otp-service@example.com"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	message := receiveMail(t, mailer)
	code := regexp.MustCompile(`[0-9]{6}`).FindString(message.TextBody)
	if code == "" || !strings.Contains(message.HTMLBody, code) || message.To[0] != "otp-service@example.com" {
		t.Fatalf("delivered OTP message = %#v", message)
	}
	stored, err := repository.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}
	if stored.OTPHash == code || len(stored.OTPHash) != 64 || !stored.ExpiresAt.After(time.Now()) {
		t.Fatalf("stored OTP has plaintext or invalid expiry: %#v", stored)
	}
	if err := service.CanResend(ctx, userID); err == nil || !strings.Contains(err.Error(), "60 seconds") {
		t.Fatalf("CanResend(immediately) error = %v, want cooldown error", err)
	}
	if err := service.Verify(ctx, userID, "000000"); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("Verify(wrong code) error = %v, want invalid code", err)
	}
	stored, err = repository.GetByUserID(ctx, userID)
	if err != nil || stored.Attempts != 1 {
		t.Fatalf("attempt count after invalid code = (%#v, %v), want one", stored, err)
	}
	if err := service.Verify(ctx, userID, code); err != nil {
		t.Fatalf("Verify(correct code) error = %v", err)
	}
	if err := service.Verify(ctx, userID, code); err == nil || !strings.Contains(err.Error(), "already verified") {
		t.Fatalf("Verify(replay) error = %v, want already-verified rejection", err)
	}

	failingMailer := newCaptureMailer()
	failingMailer.err = errors.New("mail transport unavailable")
	failingService := emailotp.NewService(failingMailer, repository, []byte("otp integration hmac secret"))
	failureUserID := mustUUID(t)
	insertCredential(t, db, failureUserID, "mail-failure@example.com")
	if err := failingService.Send(ctx, failureUserID, "mail-failure@example.com"); err == nil || !strings.Contains(err.Error(), "mail transport unavailable") {
		t.Fatalf("Send(delivery failure) error = %v", err)
	}
	if stored, err := repository.GetByUserID(ctx, failureUserID); err != nil || stored.OTPHash == "" {
		t.Fatalf("OTP was not persisted before mail failure: (%#v, %v)", stored, err)
	}
}

func TestAuthSignupPendingRetriesAndPartialFailures(t *testing.T) {
	db := postgresFixture(t)
	startUsersServiceHarness(t)
	rabbit, err := messaging.NewRabbitMQ(requireEnv(t, "TEST_RABBITMQ_URL"))
	if err != nil {
		t.Fatalf("connect to RabbitMQ: %v", err)
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
	jwtService := appjwt.NewService(key, &key.PublicKey, appjwt.NewRepository(db), "signup-retry-test")
	repository := auth.NewRepository(db)
	mailer := newCaptureMailer()
	otpRepository := emailotp.NewRepository(db)
	service := auth.NewService(jwtService, repository, rabbit, emailotp.NewService(mailer, otpRepository, []byte("signup retry OTP secret")))
	ctx := context.Background()

	const retryPassword = "PendingPass9!"
	retryHash, err := security.HashPassword(retryPassword)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	pendingID := mustUUID(t)
	if err := repository.SaveCredentials(ctx, auth.Credential{UserID: pendingID, Email: "pending-retry@example.com", PasswordHash: retryHash}); err != nil {
		t.Fatalf("seed pending credential: %v", err)
	}
	response, err := service.Signup(ctx, auth.SignupRequestDTO{FullName: "Pending Retry", Email: "pending-retry@example.com", Role: "publisher", Password: retryPassword})
	if err != nil || response.User.ID != pendingID || response.User.IsEmailVerified {
		t.Fatalf("Signup(same pending password) = (%#v, %v)", response, err)
	}
	initialMail := receiveMail(t, mailer)
	if len(initialMail.To) != 1 || initialMail.To[0] != "pending-retry@example.com" {
		t.Fatalf("pending retry email recipient = %#v", initialMail.To)
	}
	if _, err := service.Signup(ctx, auth.SignupRequestDTO{FullName: "Pending Retry", Email: "pending-retry@example.com", Role: "publisher", Password: "DifferentPass9!"}); err == nil || !strings.Contains(err.Error(), "already pending") {
		t.Fatalf("Signup(different pending password) error = %v", err)
	}

	failedID := mustUUID(t)
	if err := repository.SaveCredentials(ctx, auth.Credential{UserID: failedID, Email: "failed-signup@example.com", PasswordHash: retryHash}); err != nil {
		t.Fatalf("seed failed credential: %v", err)
	}
	if _, err := db.Exec(`UPDATE credentials SET status = 'failed' WHERE user_id = $1`, failedID); err != nil {
		t.Fatalf("set failed credential state: %v", err)
	}
	if _, err := service.Signup(ctx, auth.SignupRequestDTO{FullName: "Failed", Email: "failed-signup@example.com", Role: "publisher", Password: retryPassword}); err == nil || !strings.Contains(err.Error(), "previously failed") {
		t.Fatalf("Signup(failed account) error = %v", err)
	}

	verifiedPendingID := mustUUID(t)
	if err := repository.SaveCredentials(ctx, auth.Credential{UserID: verifiedPendingID, Email: "verified-pending@example.com", PasswordHash: retryHash}); err != nil {
		t.Fatalf("seed verified pending credential: %v", err)
	}
	if err := repository.UpdateIsEmailVerified(ctx, verifiedPendingID); err != nil {
		t.Fatalf("mark seeded credential verified: %v", err)
	}
	verifiedResponse, err := service.Signup(ctx, auth.SignupRequestDTO{FullName: "Verified Pending", Email: "verified-pending@example.com", Role: "publisher", Password: retryPassword})
	if err != nil || verifiedResponse.User.ID != verifiedPendingID || !verifiedResponse.User.IsEmailVerified {
		t.Fatalf("Signup(already verified pending account) = (%#v, %v)", verifiedResponse, err)
	}
	select {
	case message := <-mailer.messages:
		t.Fatalf("verified signup sent an OTP unexpectedly: %#v", message)
	default:
	}

	unexpectedIDResult, err := service.Signup(ctx, auth.SignupRequestDTO{FullName: "UNEXPECTED_ID", Email: "unexpected-user@example.com", Role: "publisher", Password: "UnexpectedPass9!"})
	if err == nil || unexpectedIDResult != nil || !strings.Contains(err.Error(), "unexpected user ID") {
		t.Fatalf("Signup(unexpected Users Service ID) = (%#v, %v)", unexpectedIDResult, err)
	}
	var statusValue string
	if err := db.QueryRow(`SELECT status FROM credentials WHERE email = $1`, "unexpected-user@example.com").Scan(&statusValue); err != nil || statusValue != auth.Pending {
		t.Fatalf("credential state after unexpected Users Service ID = (%q, %v), want pending", statusValue, err)
	}
	if _, err := service.Signup(ctx, auth.SignupRequestDTO{FullName: "Retry", Email: "unexpected-user@example.com", Role: "publisher", Password: "UnexpectedPass9!"}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("retry after partial user creation error = %v, want duplicate-user response", err)
	}

	failingMailer := newCaptureMailer()
	failingMailer.err = errors.New("test mail delivery failure")
	failingService := auth.NewService(jwtService, repository, rabbit, emailotp.NewService(failingMailer, otpRepository, []byte("signup retry OTP secret")))
	if _, err := failingService.Signup(ctx, auth.SignupRequestDTO{FullName: "Mail Failure", Email: "mail-failure-signup@example.com", Role: "publisher", Password: "MailFailure9!"}); err == nil || !strings.Contains(err.Error(), "test mail delivery failure") {
		t.Fatalf("Signup(mail failure) error = %v", err)
	}
	if err := db.QueryRow(`SELECT status FROM credentials WHERE email = $1`, "mail-failure-signup@example.com").Scan(&statusValue); err != nil || statusValue != auth.Active {
		t.Fatalf("credential status after mail failure = (%q, %v), want active", statusValue, err)
	}
	var persistedOTP string
	if err := db.QueryRow(`SELECT otp_hash FROM email_otps WHERE user_id = (SELECT user_id FROM credentials WHERE email = $1)`, "mail-failure-signup@example.com").Scan(&persistedOTP); err != nil || persistedOTP == "" {
		t.Fatalf("OTP persistence after delivery failure = (%q, %v), want persisted hash", persistedOTP, err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := service.Signup(cancelled, auth.SignupRequestDTO{FullName: "Cancelled", Email: "cancelled@example.com", Role: "publisher", Password: "Cancelled9!"}); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("Signup(cancelled context) error = %v, want context.Canceled", err)
	}
}

func TestAuthLoginAndResendOTPStateBranches(t *testing.T) {
	db := postgresFixture(t)
	startUsersServiceHarness(t)
	rabbit, err := messaging.NewRabbitMQ(requireEnv(t, "TEST_RABBITMQ_URL"))
	if err != nil {
		t.Fatalf("connect to RabbitMQ: %v", err)
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
	password := "LoginBranch9!"
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	jwtService := appjwt.NewService(key, &key.PublicKey, appjwt.NewRepository(db), "login-branch-test")
	repository := auth.NewRepository(db)
	mailer := newCaptureMailer()
	otpRepository := emailotp.NewRepository(db)
	otpService := emailotp.NewService(mailer, otpRepository, []byte("login branch OTP secret"))
	service := auth.NewService(jwtService, repository, rabbit, otpService)
	ctx := context.Background()

	if _, err := service.Login(ctx, auth.LoginRequestDTO{Email: "absent@example.com", Password: password, Role: "advertiser"}); err == nil || !strings.Contains(err.Error(), "Invalid email or password") {
		t.Fatalf("Login(nonexistent email) error = %v", err)
	}
	pendingID := mustUUID(t)
	insertCredential(t, db, pendingID, "pending-login@example.com")
	if _, err := db.Exec(`UPDATE credentials SET password_hash = $1 WHERE user_id = $2`, passwordHash, pendingID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Login(ctx, auth.LoginRequestDTO{Email: "pending-login@example.com", Password: password, Role: "advertiser"}); err == nil || !strings.Contains(err.Error(), "already pending") {
		t.Fatalf("Login(pending account) error = %v", err)
	}
	failedID := mustUUID(t)
	insertCredential(t, db, failedID, "failed-login@example.com")
	if _, err := db.Exec(`UPDATE credentials SET password_hash = $1, status = 'failed' WHERE user_id = $2`, passwordHash, failedID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Login(ctx, auth.LoginRequestDTO{Email: "failed-login@example.com", Password: password, Role: "advertiser"}); err == nil || !strings.Contains(err.Error(), "previously failed") {
		t.Fatalf("Login(failed account) error = %v", err)
	}

	unverifiedID := mustUUID(t)
	insertCredential(t, db, unverifiedID, "unverified-login@example.com")
	if _, err := db.Exec(`UPDATE credentials SET password_hash = $1, status = 'active' WHERE user_id = $2`, passwordHash, unverifiedID); err != nil {
		t.Fatal(err)
	}
	if _, err := rabbit.CreateUser(ctx, messaging.CreateUserRequest{UserID: unverifiedID, FullName: "Unverified", Email: "unverified-login@example.com", Role: "publisher"}); err != nil {
		t.Fatalf("create unverified test user: %v", err)
	}
	unverified, err := service.Login(ctx, auth.LoginRequestDTO{Email: "unverified-login@example.com", Password: password, Role: "publisher"})
	if err != nil || unverified.User.ID != unverifiedID || unverified.User.IsEmailVerified || unverified.AccessToken != nil || unverified.RefreshToken != nil {
		t.Fatalf("Login(unverified active account) = (%#v, %v), want OTP-only response", unverified, err)
	}
	initialMail := receiveMail(t, mailer)
	if len(initialMail.To) != 1 || initialMail.To[0] != "unverified-login@example.com" {
		t.Fatalf("unverified login email recipient = %#v", initialMail.To)
	}
	if err := service.ResendOTP(ctx, auth.ResendOTPDTO{Email: "unverified-login@example.com"}); err == nil || !strings.Contains(err.Error(), "60 seconds") {
		t.Fatalf("ResendOTP(with cooldown active) error = %v", err)
	}
	if _, err := db.Exec(`UPDATE email_otps SET last_sent_at = NOW() - INTERVAL '61 seconds' WHERE user_id = $1`, unverifiedID); err != nil {
		t.Fatalf("expire resend cooldown: %v", err)
	}
	if err := service.ResendOTP(ctx, auth.ResendOTPDTO{Email: "unverified-login@example.com"}); err != nil {
		t.Fatalf("ResendOTP(after cooldown) error = %v", err)
	}
	resentMail := receiveMail(t, mailer)
	if len(resentMail.To) != 1 || resentMail.To[0] != "unverified-login@example.com" {
		t.Fatalf("resent OTP recipient = %#v", resentMail.To)
	}

	if _, err := db.Exec(`UPDATE credentials SET status = 'pending' WHERE user_id = $1`, unverifiedID); err != nil {
		t.Fatal(err)
	}
	if err := service.ResendOTP(ctx, auth.ResendOTPDTO{Email: "unverified-login@example.com"}); err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("ResendOTP(inactive account) error = %v", err)
	}
	if _, err := db.Exec(`UPDATE credentials SET status = 'active', is_email_verified = TRUE WHERE user_id = $1`, unverifiedID); err != nil {
		t.Fatal(err)
	}
	if err := service.ResendOTP(ctx, auth.ResendOTPDTO{Email: "unverified-login@example.com"}); err == nil || !strings.Contains(err.Error(), "already verified") {
		t.Fatalf("ResendOTP(already verified) error = %v", err)
	}
	if _, err := service.Login(ctx, auth.LoginRequestDTO{Email: "unverified-login@example.com", Password: password, Role: "advertiser"}); err == nil || !strings.Contains(err.Error(), "unexpected user role") {
		t.Fatalf("Login(user role mismatch) error = %v", err)
	}

	expiredID := mustUUID(t)
	insertCredential(t, db, expiredID, "expired-otp@example.com")
	if _, err := db.Exec(`UPDATE credentials SET status = 'active' WHERE user_id = $1`, expiredID); err != nil {
		t.Fatal(err)
	}
	if err := otpRepository.Upsert(ctx, emailotp.EmailOTP{UserID: expiredID, OTPHash: "expired-hmac", ExpiresAt: time.Now().Add(-time.Minute)}); err != nil {
		t.Fatalf("seed expired OTP: %v", err)
	}
	if _, err := service.VerifyOTP(ctx, auth.VerifyOTPDTO{Email: "expired-otp@example.com", Role: "publisher", OTP: "123456"}); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("VerifyOTP(expired OTP) error = %v", err)
	}
}

func TestConcurrentOTPVerificationAndResendAreSerializedByPostgres(t *testing.T) {
	db := postgresFixture(t)
	ctx := context.Background()
	userID := mustUUID(t)
	insertCredential(t, db, userID, "otp-race@example.com")
	mailer := newCaptureMailer()
	repository := emailotp.NewRepository(db)
	service := emailotp.NewService(mailer, repository, []byte("otp race hmac secret"))
	if err := service.Send(ctx, userID, "otp-race@example.com"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	code := regexp.MustCompile(`[0-9]{6}`).FindString(receiveMail(t, mailer).TextBody)
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			results <- service.Verify(ctx, userID, code)
		}()
	}
	close(start)
	group.Wait()
	close(results)
	verified := 0
	for err := range results {
		if err == nil {
			verified++
		} else if !strings.Contains(err.Error(), "already verified") && !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("concurrent OTP verification error = %v", err)
		}
	}
	if verified != 1 {
		t.Fatalf("concurrent OTP verification successes = %d, want exactly one", verified)
	}

	if _, err := db.Exec(`UPDATE email_otps SET verified_at = NULL, last_sent_at = NOW() - INTERVAL '61 seconds' WHERE user_id = $1`, userID); err != nil {
		t.Fatalf("prepare resend cooldown race: %v", err)
	}
	start = make(chan struct{})
	allowed := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			ok, err := repository.CheckResendCooldown(ctx, userID)
			if err != nil {
				t.Errorf("CheckResendCooldown() error = %v", err)
			}
			allowed <- ok
		}()
	}
	close(start)
	group.Wait()
	close(allowed)
	approved := 0
	for ok := range allowed {
		if ok {
			approved++
		}
	}
	if approved != 1 {
		t.Fatalf("concurrent OTP resend approvals = %d, want exactly one", approved)
	}
}

func jsonRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func receiveMail(t *testing.T, mailer *captureMailer) mailservice.Message {
	t.Helper()
	select {
	case message := <-mailer.messages:
		return message
	case <-time.After(3 * time.Second):
		t.Fatal("test mail delivery did not occur before the deadline")
		return mailservice.Message{}
	}
}

func serviceSignup(t *testing.T, handler *auth.Handler, email, password string) int {
	t.Helper()
	body := fmt.Sprintf(`{"full_name":"Duplicate User","email":%q,"role":"publisher","password":%q}`, email, password)
	recorder := httptest.NewRecorder()
	handler.Signup(recorder, jsonRequest(http.MethodPost, "/signup", body))
	return recorder.Code
}

func serviceLogin(t *testing.T, handler *auth.Handler, email, password, role string) int {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":%q,"role":%q}`, email, password, role)
	recorder := httptest.NewRecorder()
	handler.Login(recorder, jsonRequest(http.MethodPost, "/login", body))
	return recorder.Code
}
