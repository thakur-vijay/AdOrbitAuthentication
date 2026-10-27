package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type Principal struct {
	UserID string
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}
