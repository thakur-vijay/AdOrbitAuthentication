package auth

import (
	"bytes"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"

	"github.com/thakur-vijay/adorbit-go/apperrors"
	"github.com/thakur-vijay/adorbit-go/middleware"
	"github.com/thakur-vijay/adorbit-go/response"
)

const maxRequestBodyBytes int64 = 1 << 20

var errRequestBodyTooLarge = errors.New("request body exceeds 1 MiB limit")
var errUnsupportedContentType = errors.New("Content-Type must be application/json")

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Signup godoc
// @Summary Register a new user
// @Description Creates a new advertiser or publisher account and initiates email verification.
// @Description Requires an Idempotency-Key to safely handle retries.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param Idempotency-Key header string true "Unique UUID for request idempotency"
// @Param request body SignupRequestDTO true "Signup details"
// @Success 201 {object} APIResponse{data=LoginResponseDTO} "User created successfully"
// @Failure 400 {object} APIErrorResponse "Invalid request or missing idempotency key"
// @Failure 409 {object} APIErrorResponse "Email already registered or idempotency conflict"
// @Failure 422 {object} APIErrorResponse "Validation failed"
// @Failure 429 {object} APIErrorResponse "Rate limit exceeded"
// @Failure 413 {object} APIErrorResponse "Request body exceeds the configured limit"
// @Failure 415 {object} APIErrorResponse "Content-Type must be application/json"
// @Failure 500 {object} APIErrorResponse "Internal server error"
// @Router /auth/signup [post]
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var signupRequest SignupRequestDTO
	ctx := r.Context()
	if err := decodeJSONRequest(w, r, &signupRequest); err != nil {
		writeDecodeError(w, err)
		return
	}

	createdUser, err := h.service.Signup(ctx, signupRequest)
	if err != nil {
		log.Println(err.Error())
		response.WriteError(w, err)
		return
	}

	response.WriteItem(w, http.StatusCreated, "User created successfully.", createdUser)
}

// Login godoc
// @Summary Authenticate a user
// @Description Authenticates a user using email and password.
// @Description Returns authentication tokens when email verification is complete.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequestDTO true "Login credentials"
// @Success 202 {object} APIResponse{data=LoginResponseDTO} "Authentication accepted"
// @Failure 400 {object} APIErrorResponse "Invalid request or validation failure"
// @Failure 401 {object} APIErrorResponse "Invalid credentials"
// @Failure 403 {object} APIErrorResponse "Email verification required or account disabled"
// @Failure 409 {object} APIErrorResponse "Account conflict"
// @Failure 429 {object} APIErrorResponse "Too many login attempts"
// @Failure 413 {object} APIErrorResponse "Request body exceeds the configured limit"
// @Failure 415 {object} APIErrorResponse "Content-Type must be application/json"
// @Failure 500 {object} APIErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var loginRequest LoginRequestDTO
	ctx := r.Context()
	if err := decodeJSONRequest(w, r, &loginRequest); err != nil {
		writeDecodeError(w, err)
		return
	}

	user, err := h.service.Login(ctx, loginRequest)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteItem(w, http.StatusAccepted, "User logged in successfully.", user)
}

// Logout godoc
// @Summary Log out the authenticated user
// @Description Invalidates the authenticated user's active session.
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse "Logged out successfully"
// @Failure 401 {object} APIErrorResponse "Missing, invalid, or expired access token"
// @Failure 500 {object} APIErrorResponse "Internal server error"
// @Router /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	principal, ok := middleware.GetPrincipal(r.Context())
	if !ok {
		response.Write(w, http.StatusUnauthorized, "authentication failed")
		return
	}

	userID := principal.UserID

	if err := h.service.Logout(ctx, userID); err != nil {
		response.WriteError(w, err)
		return
	}

	response.Write(w, http.StatusOK, "Logged out successfully.")
}

// ResendOTP godoc
// @Summary Resend email verification OTP
// @Description Generates and sends a new email verification OTP.
// @Description Requests are subject to resend cooldown and rate limiting.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body ResendOTPDTO true "Email address"
// @Success 201 {object} APIResponse "OTP sent successfully"
// @Failure 400 {object} APIErrorResponse "Invalid request or email format"
// @Failure 404 {object} APIErrorResponse "Account not found"
// @Failure 409 {object} APIErrorResponse "Email already verified or resend cooldown active"
// @Failure 429 {object} APIErrorResponse "Too many OTP requests"
// @Failure 413 {object} APIErrorResponse "Request body exceeds the configured limit"
// @Failure 415 {object} APIErrorResponse "Content-Type must be application/json"
// @Failure 500 {object} APIErrorResponse "Internal server error"
// @Router /auth/resendOTP [post]
func (h *Handler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var resendOTPDTO ResendOTPDTO
	ctx := r.Context()
	if err := decodeJSONRequest(w, r, &resendOTPDTO); err != nil {
		writeDecodeError(w, err)
		return
	}

	err := h.service.ResendOTP(ctx, resendOTPDTO)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.Write(w, http.StatusCreated, "OTP send successfully")
}

// Refresh godoc
// @Summary Refresh authentication tokens
// @Description Exchanges a valid refresh token for a new access token and refresh token.
// @Description Requires a valid access token to identify the caller role; the submitted refresh token is validated separately.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequestDTO true "Refresh token"
// @Success 200 {object} APIResponse{data=RefreshTokenResponseDTO} "Tokens refreshed successfully"
// @Failure 400 {object} APIErrorResponse "Invalid request or missing refresh token"
// @Failure 401 {object} APIErrorResponse "Missing or invalid access token, or invalid refresh token"
// @Failure 409 {object} APIErrorResponse "Refresh token rotation conflict"
// @Failure 429 {object} APIErrorResponse "Too many refresh attempts"
// @Failure 413 {object} APIErrorResponse "Request body exceeds the configured limit"
// @Failure 415 {object} APIErrorResponse "Content-Type must be application/json"
// @Failure 500 {object} APIErrorResponse "Internal server error"
// @Router /auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var refreshRequest RefreshTokenRequestDTO

	if err := decodeJSONRequest(w, r, &refreshRequest); err != nil {
		writeDecodeError(w, err)
		return
	}

	tokens, err := h.service.Refresh(r.Context(), refreshRequest)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteItem(
		w,
		http.StatusOK,
		"Tokens refreshed successfully.",
		tokens,
	)
}

// VerifyOTP godoc
// @Summary Verify email address using OTP
// @Description Verifies the email address using the OTP sent during registration.
// @Description Successfully verifying the OTP completes email verification and returns authentication tokens.
// @Description Requires an Idempotency-Key to safely handle retries.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param Idempotency-Key header string true "Unique UUID for request idempotency"
// @Param request body VerifyOTPDTO true "Email address and verification OTP"
// @Success 200 {object} APIResponse{data=LoginResponseDTO} "Email verified successfully"
// @Failure 400 {object} APIErrorResponse "Invalid request, OTP format, or missing idempotency key"
// @Failure 401 {object} APIErrorResponse "Invalid or expired OTP"
// @Failure 404 {object} APIErrorResponse "Account or OTP not found"
// @Failure 409 {object} APIErrorResponse "OTP already consumed or idempotency conflict"
// @Failure 422 {object} APIErrorResponse "Validation failed"
// @Failure 429 {object} APIErrorResponse "Too many verification attempts"
// @Failure 413 {object} APIErrorResponse "Request body exceeds the configured limit"
// @Failure 415 {object} APIErrorResponse "Content-Type must be application/json"
// @Failure 500 {object} APIErrorResponse "Internal server error"
// @Router /auth/verifyOTP [post]
func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var verifyOTPDTO VerifyOTPDTO
	ctx := r.Context()

	if err := decodeJSONRequest(w, r, &verifyOTPDTO); err != nil {
		writeDecodeError(w, err)
		return
	}

	user, err := h.service.VerifyOTP(ctx, verifyOTPDTO)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteItem(w, http.StatusOK, "Email verified successfully.", user)
}

func decodeJSONRequest[T any](w http.ResponseWriter, r *http.Request, destination *T) error {
	if r.ContentLength > maxRequestBodyBytes {
		return errRequestBodyTooLarge
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errUnsupportedContentType
	}
	limitedBody := http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	body, err := io.ReadAll(limitedBody)
	closeErr := r.Body.Close()
	if err == nil && closeErr != nil {
		err = closeErr
	}
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		return errRequestBodyTooLarge
	}
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return response.DecodeJSONAndValidate(r, destination)
}

func writeDecodeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errRequestBodyTooLarge):
		response.Write(w, http.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, errUnsupportedContentType):
		response.Write(w, http.StatusUnsupportedMediaType, err.Error())
	default:
		appError := apperrors.NewBadRequest(err.Error(), nil)
		response.Write(w, appError.StatusCode, appError.Message)
	}
}
