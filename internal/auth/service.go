package auth

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/adorbit/authentication/internal/jwt"
	"github.com/adorbit/authentication/internal/messaging"
	"github.com/adorbit/authentication/internal/security"
	"github.com/adorbit/authentication/internal/usersclient"
	"github.com/google/uuid"
	"github.com/thakur-vijay/adorbit-go/apperrors"
)

type Service struct {
	jwtService  *jwt.Service
	usersClient *usersclient.Client
	repository  *Repository
	rabbitMQ    *messaging.RabbitMQ
}

func NewService(
	jwtService *jwt.Service,
	usersClient *usersclient.Client,
	repository *Repository,
	rabbitMQ *messaging.RabbitMQ,
) *Service {
	return &Service{
		jwtService:  jwtService,
		usersClient: usersClient,
		repository:  repository,
		rabbitMQ:    rabbitMQ,
	}
}

func (s *Service) Signup(request SignupRequestDTO) (*LoginResponseDTO, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//1. Fetch user
	existingCredential, err := s.repository.FetchByEmail(ctx, request.Email)

	if err != nil {
		return nil, err
	}

	if existingCredential != nil {
		return nil, apperrors.NewConflict(
			"This email is already in use. Please login.",
			nil,
		)
	}

	userID := uuid.NewString()

	passwordHash, err := security.HashPassword(request.Password)

	if err != nil {
		return nil, err
	}

	credential := Credential{
		UserID:       userID,
		Email:        request.Email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = s.repository.SaveCredentials(ctx, credential)

	if err != nil {
		return nil, err
	}

	if s.rabbitMQ == nil {
		log.Printf("[SIGNUP] ERROR: RabbitMQ is nil")
		return nil, errors.New("RabbitMQ is not initialized")
	}

	log.Printf("[SIGNUP] Sending CreateUser request through RabbitMQ: userID=%s email=%s role=%s",
		userID,
		request.Email,
		request.Role,
	)

	createdUserID, err := s.rabbitMQ.CreateUser(ctx, messaging.CreateUserRequest{
		UserID:   userID,
		FullName: request.FullName,
		Email:    request.Email,
		Role:     request.Role,
	})
	if err != nil {
		log.Printf("[SIGNUP] RabbitMQ CreateUser FAILED: %v", err)
		return nil, err
	}

	log.Printf("[SIGNUP] RabbitMQ CreateUser SUCCESS: userID=%s", createdUserID)

	tokens, err := s.jwtService.CreateSession(ctx, createdUserID)
	if err != nil {
		log.Printf("[SIGNUP] CreateSession FAILED: %v", err)
		return &LoginResponseDTO{}, err
	}

	log.Printf("[SIGNUP] CreateSession SUCCESS: userID=%s", createdUserID)

	return &LoginResponseDTO{
		User: LoginUserDTO{
			ID:       createdUserID,
			FullName: request.FullName,
			Email:    request.Email,
		},
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}
