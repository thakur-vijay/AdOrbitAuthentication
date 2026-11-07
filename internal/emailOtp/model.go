package emailotp

import "time"

type EmailOTP struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	OTPHash     string     `json:"-"`
	ExpiresAt   time.Time  `json:"expires_at"`
	Attempts    int16      `json:"attempts"`
	MaxAttempts int16      `json:"max_attempts"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	LastSentAt  time.Time  `json:"last_sent_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
