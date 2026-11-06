package auth

import (
	"net/http"

	"github.com/thakur-vijay/adorbit-go/apperrors"
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
// @Tags Authentication
// @Accept json
// @Produce json
// @Param Idempotency-Key header string true "UUID used to safely retry signup"
// @Param request body SignupRequestDTO true "Signup details"
// @Success 201 {object} APIResponse{data=LoginResponseDTO}
// @Failure 400 {object} APIErrorResponse
// @Failure 409 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
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
// @Summary Login user
// @Description Authenticates a user and returns session tokens when email verification is complete.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequestDTO true "Login credentials"
// @Success 202 {object} APIResponse{data=LoginResponseDTO}
// @Failure 400 {object} APIErrorResponse
// @Failure 401 {object} APIErrorResponse
// @Failure 409 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
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
// @Summary Logout user
// @Description Invalidates the authenticated user's session.
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse
// @Failure 401 {object} APIErrorResponse
// @Router /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {

}

// ResendOTP godoc
// @Summary Resend email verification OTP
// @Description Sends a new email verification OTP.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body ResendOTPDTO true "Email address"
// @Success 201 {object} APIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
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
// @Description Generates new access and refresh tokens.
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RefreshRequestDTO true "Refresh token"
// @Success 200 {object} APIResponse{data=RefreshResponseDTO}
// @Failure 401 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {

}
