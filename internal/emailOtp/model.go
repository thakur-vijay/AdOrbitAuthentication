package emailotp

import "time"

type EmailOTP struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	OTPHash     string     `json:"-"`
	ExpiresAt   time.Time  `json:"expires_at"`
	Attempts    int16      `json:"attempts"`
	MaxAttempts int16      `json:"max_attempts"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
