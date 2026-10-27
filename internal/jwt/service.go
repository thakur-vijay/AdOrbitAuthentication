package jwt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
)

type Service struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	repository *Repository
	issuer     string
}

func NewService(
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	repository *Repository,
	issuer string,
) *Service {
	return &Service{
		privateKey: privateKey,
		publicKey:  publicKey,
		repository: repository,
		issuer:     issuer,
	}
}

func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *Service) generateAccessToken(
	userID string,
) (string, error) {
	now := time.Now().UTC()

	claims := Claims{
		UserID:    userID,
		Issuer:    s.issuer,
		Subject:   userID,
		ID:        uuid.NewString(),
		IssuedAt:  jwtlib.NewNumericDate(now),
		NotBefore: jwtlib.NewNumericDate(now),
		ExpiresAt: jwtlib.NewNumericDate(now.Add(AccessTokenTTL)),
	}

	token := jwtlib.NewWithClaims(
		jwtlib.SigningMethodRS256,
		claims,
	)

	return token.SignedString(s.privateKey)
}

func (s *Service) CreateSession(
	ctx context.Context,
	userID string,
) (TokenPair, error) {
	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	now := time.Now().UTC()

	record := RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: hashRefreshToken(refreshToken),
		ExpiresAt: now.Add(RefreshTokenTTL),
		CreatedAt: now,
	}

	if err := s.repository.Create(ctx, record); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    now.Add(AccessTokenTTL),
	}, nil
}

func (s *Service) ValidateAccessToken(
	tokenString string,
) (*Principal, error) {
	if tokenString == "" {
		return nil, errors.New("missing access token")
	}

	claims := &Claims{}

	token, err := jwtlib.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwtlib.Token) (any, error) {
			if token.Method != jwtlib.SigningMethodRS256 {
				return nil, errors.New("unexpected signing method")
			}

			return s.publicKey, nil
		},
		jwtlib.WithIssuer(s.issuer),
		jwtlib.WithExpirationRequired(),
		jwtlib.WithIssuedAt(),
		jwtlib.WithValidMethods([]string{"RS256"}),
		jwtlib.WithLeeway(30*time.Second),
	)
	if err != nil {
		return nil, errors.New("invalid access token")
	}

	if !token.Valid {
		return nil, errors.New("invalid access token")
	}

	if claims.UserID == "" || claims.Subject != claims.UserID {
		return nil, errors.New("invalid token claims")
	}

	return &Principal{
		UserID: claims.UserID,
	}, nil
}

func (s *Service) ValidateRefreshToken(
	ctx context.Context,
	refreshToken string,
) (RefreshToken, error) {
	if refreshToken == "" {
		return RefreshToken{}, errors.New("missing refresh token")
	}

	record, err := s.repository.GetByHash(
		ctx,
		hashRefreshToken(refreshToken),
	)
	if err != nil {
		return RefreshToken{}, err
	}

	if record.RevokedAt != nil {
		return RefreshToken{}, ErrRefreshTokenRevoked
	}

	if !time.Now().UTC().Before(record.ExpiresAt) {
		return RefreshToken{}, ErrRefreshTokenExpired
	}

	return record, nil
}

func (s *Service) RefreshSession(
	ctx context.Context,
	refreshToken string,
) (TokenPair, error) {
	record, err := s.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return TokenPair{}, err
	}

	newAccessToken, err := s.generateAccessToken(record.UserID)
	if err != nil {
		return TokenPair{}, err
	}

	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	now := time.Now().UTC()

	newRecord := RefreshToken{
		ID:        uuid.NewString(),
		UserID:    record.UserID,
		TokenHash: hashRefreshToken(newRefreshToken),
		ExpiresAt: now.Add(RefreshTokenTTL),
		CreatedAt: now,
	}

	err = s.repository.Rotate(
		ctx,
		hashRefreshToken(refreshToken),
		newRecord,
	)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    now.Add(AccessTokenTTL),
	}, nil
}

func (s *Service) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	if refreshToken == "" {
		return errors.New("missing refresh token")
	}

	return s.repository.RevokeByHash(
		ctx,
		hashRefreshToken(refreshToken),
	)
}

// PublicKey returns the public key for other services to verify JWTs.
func (s *Service) PublicKey() *rsa.PublicKey {
	return s.publicKey
}
