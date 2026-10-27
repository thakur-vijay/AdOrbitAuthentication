package auth

import (
	"context"

	"github.com/adorbit/authentication/internal/jwt"
	"github.com/google/uuid"
)

type Service struct {
	jwtService *jwt.Service
}

func NewService(jwtService *jwt.Service) *Service {
	return &Service{
		jwtService: jwtService,
	}
}

func (s *Service) Signup(request SignupRequestDTO) (LoginResponseDTO, error) {
	userID := uuid.NewString()

	tokens, err := s.jwtService.CreateSession(context.Background(), userID)
	if err != nil {
		return LoginResponseDTO{}, err
	}

	return LoginResponseDTO{
		User: LoginUserDTO{
			ID:       userID,
			FullName: request.FullName,
			Email:    request.Email,
		},
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}
