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

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {

}
