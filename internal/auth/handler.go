package auth

import (
	"net/http"

	"github.com/thakur-vijay/adorbit-go/apperrors"
	"github.com/thakur-vijay/adorbit-go/middleware"
	"github.com/thakur-vijay/adorbit-go/response"
)

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
// @Failure 500 {object} APIErrorResponse "Internal server error"
// @Router /auth/signup [post]
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var signupRequest SignupRequestDTO
	ctx := r.Context()
	if err := response.DecodeJSONAndValidate(r, &signupRequest); err != nil {
		appError := apperrors.NewBadRequest(err.Error(), nil)
		response.Write(w, appError.StatusCode, appError.Message)
		return
	}

	createdUser, err := h.service.Signup(ctx, signupRequest)
	if err != nil {
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
// @Failure 500 {object} APIErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var loginRequest LoginRequestDTO
	ctx := r.Context()
	if err := response.DecodeJSONAndValidate(r, &loginRequest); err != nil {
		appError := apperrors.NewBadRequest(err.Error(), nil)
		response.Write(w, appError.StatusCode, appError.Message)
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
	userID, ok := middleware.UserID(ctx)
	if !ok {
		response.WriteError(w, apperrors.NewUnauthorized(
			"Authentication required.",
			nil,
		))
		return
	}

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
// @Failure 500 {object} APIErrorResponse "Internal server error"
// @Router /auth/resendOTP [post]
func (h *Handler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var resendOTPDTO ResendOTPDTO
	ctx := r.Context()
	if err := response.DecodeJSONAndValidate(r, &resendOTPDTO); err != nil {
		appError := apperrors.NewBadRequest(err.Error(), nil)
		response.Write(w, appError.StatusCode, appError.Message)
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
// @Description The refresh token is validated independently of the access token.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequestDTO true "Refresh token"
// @Success 200 {object} APIResponse{data=RefreshTokenResponseDTO} "Tokens refreshed successfully"
// @Failure 400 {object} APIErrorResponse "Invalid request or missing refresh token"
// @Failure 401 {object} APIErrorResponse "Invalid, expired, revoked, or reused refresh token"
// @Failure 409 {object} APIErrorResponse "Refresh token rotation conflict"
// @Failure 429 {object} APIErrorResponse "Too many refresh attempts"
// @Failure 500 {object} APIErrorResponse "Internal server error"
// @Router /auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var refreshRequest RefreshTokenRequestDTO
	ctx := r.Context()

	if err := response.DecodeJSONAndValidate(r, &refreshRequest); err != nil {
		appError := apperrors.NewBadRequest(err.Error(), nil)
		response.Write(w, appError.StatusCode, appError.Message)
		return
	}

	tokens, err := h.service.Refresh(ctx, refreshRequest)
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
// @Failure 500 {object} APIErrorResponse "Internal server error"
// @Router /auth/verifyOTP [post]
func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var verifyOTPDTO VerifyOTPDTO
	ctx := r.Context()

	if err := response.DecodeJSONAndValidate(r, &verifyOTPDTO); err != nil {
		appError := apperrors.NewBadRequest(err.Error(), nil)
		response.Write(w, appError.StatusCode, appError.Message)
		return
	}

	user, err := h.service.VerifyOTP(ctx, verifyOTPDTO)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteItem(w, http.StatusOK, "Email verified successfully.", user)
}
