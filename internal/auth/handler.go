package auth

import (
	"net/http"

	"github.com/adorbit/authentication/internal/apperrors"
	"github.com/adorbit/authentication/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var signupRequest SignupRequestDTO
	if err := response.DecodeJSONAndValidate(r, &signupRequest); err != nil {
		appError := apperrors.NewBadRequest(err.Error(), nil)
		response.Write(w, appError.StatusCode, appError.Message)
		return
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {

}
