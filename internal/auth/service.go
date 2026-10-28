package auth

import (
	"context"
	"errors"
	"log"

	"github.com/adorbit/authentication/internal/jwt"
	"github.com/adorbit/authentication/internal/usersclient"
	"github.com/google/uuid"
)

type Service struct {
	jwtService  *jwt.Service
	usersClient *usersclient.Client
}

func NewService(
	jwtService *jwt.Service,
	usersClient *usersclient.Client,
) *Service {
	return &Service{
		jwtService:  jwtService,
		usersClient: usersClient,
	}
}

func (s *Service) Signup(request SignupRequestDTO) (LoginResponseDTO, error) {
	log.Printf("[SIGNUP] Started for email=%s", request.Email)

	ctx := context.Background()
	userID := uuid.NewString()

	if s.usersClient == nil {
		log.Printf("[SIGNUP] ERROR: usersClient is nil")
		return LoginResponseDTO{}, errors.New("users client is nil")
	}

	log.Printf("[SIGNUP] Calling Users gRPC CreateUser: userID=%s email=%s role=advertiser",
		userID, request.Email)

	createdUserID, err := s.usersClient.CreateUser(
		ctx,
		userID,
		request.FullName,
		request.Email,
		"advertiser",
	)
	if err != nil {
		log.Printf("[SIGNUP] Users gRPC CreateUser FAILED: %v", err)
		return LoginResponseDTO{}, err
	}

	log.Printf("[SIGNUP] Users gRPC CreateUser SUCCESS: userID=%s", createdUserID)

	tokens, err := s.jwtService.CreateSession(ctx, createdUserID)
	if err != nil {
		log.Printf("[SIGNUP] CreateSession FAILED: %v", err)
		return LoginResponseDTO{}, err
	}

	log.Printf("[SIGNUP] CreateSession SUCCESS: userID=%s", createdUserID)

	return LoginResponseDTO{
		User: LoginUserDTO{
			ID:       createdUserID,
			FullName: request.FullName,
			Email:    request.Email,
		},
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}
