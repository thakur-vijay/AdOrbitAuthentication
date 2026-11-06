package auth

import (
	"context"
	"errors"
	"log"
	"time"

	emailotp "github.com/adorbit/authentication/internal/emailOtp"
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
	otpService *emailotp.Service
}

func NewService(
	jwtService *jwt.Service,
	repository *Repository,
	rabbitMQ *messaging.RabbitMQ,
	otpService *emailotp.Service,
) *Service {
	return &Service{
		jwtService: jwtService,
		repository: repository,
		rabbitMQ:   rabbitMQ,
		otpService: otpService,
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

	rabbitCtx, rabbitCancel := context.WithTimeout(ctx, 30*time.Second)

	createdUser, err := s.rabbitMQ.CreateUser(
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
		return nil, err
	}

	if createdUser.UserID != userID {
		return nil, errors.New("user creation returned an unexpected user ID")
	}

	updateCtx, updateCancel := context.WithTimeout(
		ctx,
		3*time.Second,
	)

	isEmailVerified, err := s.repository.UpdateCredentialStatus(
		updateCtx,
		userID,
		Active,
	)

	updateCancel()

	if err != nil {
		return nil, err
	}

	if !isEmailVerified {
		err = s.sendOTP(ctx, userID, request.Email)

		if err != nil {
			return nil, err
		}
	}
	// 7. Return signup response.
	return &LoginResponseDTO{
		User: LoginUserDTO{
			ID:              createdUser.UserID,
			FullName:        createdUser.FullName,
			Email:           createdUser.Email,
			Role:            createdUser.Role,
			IsEmailVerified: isEmailVerified,
			CreatedAt:       createdUser.CreatedAt,
			UpdatedAt:       createdUser.UpdatedAt,
		},
	}, nil
}

func (s *Service) Login(
	ctx context.Context,
	request LoginRequestDTO,
) (*LoginResponseDTO, error) {
	dbCtx, dbCancel := context.WithTimeout(ctx, 3*time.Second)
	credential, err := s.repository.FetchByEmail(
		dbCtx,
		request.Email,
	)
	dbCancel()

	if err != nil {
		return nil, err
	}

	// 2. Handle existing credentials.
	if credential != nil {
		verified, err := security.VerifyPassword(
			request.Password,
			credential.PasswordHash,
		)
		if err != nil {
			return nil, err
		}
		if verified {
			switch credential.Status {
			case Active:
				user, err := s.fetchUser(ctx, credential.UserID, credential.IsEmailVerified)
				if err != nil {
					return nil, err
				}
				if credential.IsEmailVerified {
					tokens, err := s.createSessions(ctx, credential.UserID)
					if err != nil {
						return nil, err
					}

					return &LoginResponseDTO{
						User:         *user,
						AccessToken:  &tokens.AccessToken,
						RefreshToken: &tokens.RefreshToken,
					}, nil
				} else {
					err := s.sendOTP(ctx, credential.UserID, credential.Email)
					if err != nil {
						return nil, err
					}

					return &LoginResponseDTO{
						User: *user,
					}, nil
				}
			case Pending:
				return nil, apperrors.NewConflict(
					"A signup with this email is already pending.",
					nil,
				)
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
			return nil, apperrors.NewUnauthorized("Invalid email or password.", nil)
		}
	} else {
		return nil, apperrors.NewUnauthorized("Invalid email or password.", nil)
	}
}

func (s *Service) ResendOTP(ctx context.Context, dto ResendOTPDTO) error {
	// existingCredential, err := s.repository.FetchByEmail(
	// 	ctx,
	// 	dto.Email,
	// )
	// if existingCredential != nil {
	err := s.otpService.Send(ctx, "4777abd7-7840-4e73-a0d8-afbcfb232326", dto.Email)
	// }
	log.Printf("ERROR IS", err)
	return s.sendOTP(ctx, "4777abd7-7840-4e73-a0d8-afbcfb232326", dto.Email)
}

// /helping methods
func (s *Service) createSessions(ctx context.Context, userID string) (*jwt.TokenPair, error) {
	sessionCtx, sessionCancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)
	defer sessionCancel()

	tokens, err := s.jwtService.CreateSession(sessionCtx, userID)

	if err != nil {
		log.Printf(
			"[SIGNUP] Session creation failed: userID=%s error=%v",
			userID,
			err,
		)

		return nil, err
	} else {
		return &tokens, nil
	}
}

func (s *Service) sendOTP(ctx context.Context, userID, email string) error {
	otpCtx, otpCancel := context.WithTimeout(ctx, 5*time.Second)
	defer otpCancel()

	return s.otpService.Send(otpCtx, userID, email)
}

func (s *Service) fetchUser(ctx context.Context, userID string, isEmailVerified bool) (*LoginUserDTO, error) {
	rabbitCtx, rabbitCancel := context.WithTimeout(ctx, 5*time.Second)
	defer rabbitCancel()
	user, err := s.rabbitMQ.FetchUser(
		rabbitCtx,
		messaging.FetchUserRequest{
			UserID: userID,
		},
	)

	if err != nil {
		return nil, err
	}

	if user.UserID != userID {
		return nil, apperrors.NewUnauthorized("You are not authorized.", nil)
	}

	return &LoginUserDTO{
		ID:              user.UserID,
		FullName:        user.FullName,
		Email:           user.Email,
		Role:            user.Role,
		IsEmailVerified: isEmailVerified,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}, nil
}
