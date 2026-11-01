package auth

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/adorbit/authentication/internal/jwt"
	"github.com/adorbit/authentication/internal/messaging"
	"github.com/adorbit/authentication/internal/security"
	"github.com/google/uuid"
	"github.com/thakur-vijay/adorbit-go/apperrors"
)

type Service struct {
	jwtService *jwt.Service
	repository *Repository
	rabbitMQ   *messaging.RabbitMQ
}

func NewService(
	jwtService *jwt.Service,
	repository *Repository,
	rabbitMQ *messaging.RabbitMQ,
) *Service {
	return &Service{
		jwtService: jwtService,
		repository: repository,
		rabbitMQ:   rabbitMQ,
	}
}

const (
	Pending = "pending"
	Active  = "active"
	Failed  = "failed"
)

func (s *Service) Signup(
	ctx context.Context,
	request SignupRequestDTO,
) (*LoginResponseDTO, error) {
	if s.repository == nil {
		return nil, errors.New("credential repository is not initialized")
	}

	if s.jwtService == nil {
		return nil, errors.New("JWT service is not initialized")
	}

	if s.rabbitMQ == nil {
		return nil, errors.New("RabbitMQ is not initialized")
	}

	// 1. Check existing credentials.
	dbCtx, dbCancel := context.WithTimeout(ctx, 3*time.Second)
	existingCredential, err := s.repository.FetchByEmail(
		dbCtx,
		request.Email,
	)
	dbCancel()

	if err != nil {
		return nil, err
	}

	var userID string

	// 2. Handle existing credentials.
	if existingCredential != nil {
		switch existingCredential.Status {
		case Active:
			return nil, apperrors.NewConflict(
				"This email is already in use. Please login.",
				nil,
			)

		case Pending:
			// Verify that the retry belongs to the same signup.
			verified, err := security.VerifyPassword(
				request.Password,
				existingCredential.PasswordHash,
			)
			if err != nil {
				return nil, err
			}

			if !verified {
				return nil, apperrors.NewConflict(
					"A signup with this email is already pending.",
					nil,
				)
			}

			userID = existingCredential.UserID
		case Failed:
			return nil, apperrors.NewConflict(
				"This signup previously failed. Please contact support.",
				nil,
			)

		default:
			return nil, errors.New(
				"credential has an invalid status",
			)
		}
	} else {
		// 3. Create new pending credentials.
		userID = uuid.NewString()

		passwordHash, err := security.HashPassword(request.Password)
		if err != nil {
			return nil, err
		}

		now := time.Now().UTC()

		credential := Credential{
			UserID:       userID,
			Email:        request.Email,
			PasswordHash: passwordHash,
			Status:       Pending,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		dbCtx, dbCancel := context.WithTimeout(ctx, 3*time.Second)
		err = s.repository.SaveCredentials(dbCtx, credential)
		dbCancel()

		if err != nil {
			return nil, err
		}
	}

	// 4. Request user creation using the same UserID.
	rabbitCtx, rabbitCancel := context.WithTimeout(ctx, 5*time.Second)

	createdUserID, err := s.rabbitMQ.CreateUser(
		rabbitCtx,
		messaging.CreateUserRequest{
			UserID:   userID,
			FullName: request.FullName,
			Email:    request.Email,
			Role:     request.Role,
		},
	)

	rabbitCancel()

	if err != nil {
		log.Printf(
			"[SIGNUP] CreateUser failed: userID=%s error=%v",
			userID,
			err,
		)

		// Keep credentials pending.
		// Timeout does not prove that user creation failed.
		return nil, err
	}

	if createdUserID != userID {
		log.Printf(
			"[SIGNUP] UserID mismatch: expected=%s received=%s",
			userID,
			createdUserID,
		)

		return nil, errors.New("user creation returned an unexpected user ID")
	}

	// 5. Activate credentials only after user creation confirmation.
	updateCtx, updateCancel := context.WithTimeout(
		ctx,
		3*time.Second,
	)

	err = s.repository.UpdateCredentialStatus(
		updateCtx,
		userID,
		Active,
	)

	updateCancel()

	if err != nil {
		log.Printf(
			"[SIGNUP] Credential activation failed: userID=%s error=%v",
			userID,
			err,
		)

		return nil, err
	}

	// 6. Create session only after credentials become active.
	sessionCtx, sessionCancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)

	tokens, err := s.jwtService.CreateSession(sessionCtx, userID)
	sessionCancel()

	if err != nil {
		log.Printf(
			"[SIGNUP] Session creation failed: userID=%s error=%v",
			userID,
			err,
		)

		return nil, err
	}

	// 7. Return signup response.
	return &LoginResponseDTO{
		User: LoginUserDTO{
			ID:       userID,
			FullName: request.FullName,
			Email:    request.Email,
		},
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}
