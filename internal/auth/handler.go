package auth

import "net/http"

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {

}
