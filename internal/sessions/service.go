package sessions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *Service) CreateSession(ctx context.Context, userID string) (string, string, error) {
	start := time.Now()

	tokenStart := time.Now()
	accessToken, err := generateToken()
	if err != nil {
		return "", "", err
	}

	refreshToken, err := generateToken()
	if err != nil {
		return "", "", err
	}

	log.Printf("[CreateSession] Token generation: %v", time.Since(tokenStart))

	hashStart := time.Now()
	accessTokenHash := hashToken(accessToken)
	refreshTokenHash := hashToken(refreshToken)
	log.Printf("[CreateSession] Hashing: %v", time.Since(hashStart))
	now := time.Now().UTC()

	session := Session{
		ID:               uuid.NewString(),
		UserID:           userID,
		AccessTokenHash:  accessTokenHash,
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        now.Add(15 * time.Minute).Format(time.RFC3339),
		RefreshExpiresAt: now.Add(30 * 24 * time.Hour).Format(time.RFC3339),
		CreatedAt:        now.Format(time.RFC3339),
	}
	dbStart := time.Now()
	if err := s.repository.Create(ctx, session); err != nil {
		return "", "", err
	}
	log.Printf("[CreateSession] DB Create: %v", time.Since(dbStart))
	log.Printf("[CreateSession] TOTAL: %v", time.Since(start))
	return accessToken, refreshToken, nil
}

func (s *Service) ValidateAccessToken(
	ctx context.Context,
	accessToken string,
) (Session, error) {
	if accessToken == "" {
		return Session{}, errors.New("missing access token")
	}

	session, err := s.repository.GetByAccessTokenHash(
		ctx,
		hashToken(accessToken),
	)
	if err != nil {
		return Session{}, err
	}

	if session.RevokedAt != nil {
		return Session{}, errors.New("session revoked")
	}

	expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
	if err != nil {
		return Session{}, errors.New("invalid session expiry")
	}

	if time.Now().UTC().After(expiresAt) {
		return Session{}, errors.New("access token expired")
	}

	return session, nil
}

func (s *Service) ValidateRefreshToken(
	ctx context.Context,
	refreshToken string,
) (Session, error) {
	if refreshToken == "" {
		return Session{}, errors.New("missing refresh token")
	}

	session, err := s.repository.GetByRefreshTokenHash(
		ctx,
		hashToken(refreshToken),
	)
	if err != nil {
		return Session{}, err
	}

	if session.RevokedAt != nil {
		return Session{}, errors.New("session revoked")
	}

	expiresAt, err := time.Parse(time.RFC3339, session.RefreshExpiresAt)
	if err != nil {
		return Session{}, errors.New("invalid refresh token expiry")
	}

	if time.Now().UTC().After(expiresAt) {
		return Session{}, errors.New("refresh token expired")
	}

	return session, nil
}

func (s *Service) RefreshSession(
	ctx context.Context,
	refreshToken string,
) (string, string, error) {

	session, err := s.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", err
	}

	newAccessToken, err := generateToken()
	if err != nil {
		return "", "", err
	}

	newRefreshToken, err := generateToken()
	if err != nil {
		return "", "", err
	}

	now := time.Now().UTC()

	newExpiresAt := now.Add(15 * time.Minute).Format(time.RFC3339)

	err = s.repository.UpdateTokens(
		ctx,
		session.ID,
		hashToken(newAccessToken),
		hashToken(newRefreshToken),
		newExpiresAt,
	)
	if err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.repository.Revoke(ctx, sessionID)
}
