package sessions

type Session struct {
	ID               string
	UserID           string
	AccessTokenHash  string
	RefreshTokenHash string
	ExpiresAt        string
	RefreshExpiresAt string
	CreatedAt        string
	RevokedAt        *string
}
