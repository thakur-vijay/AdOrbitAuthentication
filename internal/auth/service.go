package auth

import "github.com/adorbit/authentication/internal/sessions"

type Service struct {
	sessionService *sessions.Service
}

func NewService(sessionService *sessions.Service) *Service {
	return &Service{
		sessionService: sessionService,
	}
}
