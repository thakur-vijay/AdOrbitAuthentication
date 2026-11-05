package emailotp

import (
	"context"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Upsert(ctx context.Context, otp EmailOTP) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO email_otps (
			user_id,
			otp_hash,
			expires_at,
			attempts,
			verified_at,
			last_sent_at
		)
		VALUES ($1, $2, $3, 0, NULL, NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET
			otp_hash = EXCLUDED.otp_hash,
			expires_at = EXCLUDED.expires_at,
			attempts = 0,
			verified_at = NULL,
			last_sent_at = NOW(),
			updated_at = NOW()`,
		otp.UserID,
		otp.OTPHash,
		otp.ExpiresAt,
	)

	return err
}

func (r *Repository) CheckResendCooldown(
	ctx context.Context,
	userID string,
) (bool, error) {
	var allowed bool

	err := r.db.QueryRowContext(ctx,
		`UPDATE email_otps
		 SET last_sent_at = NOW()
		 WHERE user_id = $1
		   AND last_sent_at <= NOW() - INTERVAL '60 seconds'
		   AND verified_at IS NULL
		 RETURNING TRUE`,
		userID,
	).Scan(&allowed)

	if err == sql.ErrNoRows {
		return false, nil
	}

	return allowed, err
}

func (r *Repository) Create(ctx context.Context, otp EmailOTP) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO email_otps (user_id, otp_hash) VALUES ($1, $2)`, otp.UserID, otp.OTPHash)
	return err
}

func (r *Repository) FetchByUserID(ctx context.Context, userID string) (*EmailOTP, error) {
	var emailOTP EmailOTP

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, otp_hash, expires_at, attempts, max_attempts, verified_at, created_at, updated_at
		 FROM email_otps
		 WHERE user_id = $1`,
		userID,
	).Scan(
		&emailOTP.ID,
		&emailOTP.UserID,
		&emailOTP.OTPHash,
		&emailOTP.ExpiresAt,
		&emailOTP.Attempts,
		&emailOTP.MaxAttempts,
		&emailOTP.VerifiedAt,
		&emailOTP.CreatedAt,
		&emailOTP.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &emailOTP, nil
}

func (r *Repository) UpdateAttempts(ctx context.Context, userID string, attempts int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE email_otps
		 SET attempts = $1,
		     updated_at = NOW()
		 WHERE user_id = $2`,
		attempts,
		userID,
	)

	return err
}

func (r *Repository) MarkAsVerified(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE email_otps
		 SET verified_at = NOW(),
		     updated_at = NOW()
		 WHERE user_id = $1 AND verified_at IS NULL`,
		userID,
	)

	return err
}

func (r *Repository) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM email_otps WHERE user_id = $1`,
		userID,
	)

	return err
}

func (r *Repository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM email_otps
		 WHERE expires_at <= NOW()
		 AND verified_at IS NULL`,
	)

	return err
}

func (r *Repository) UpdateOTP(ctx context.Context, otp EmailOTP) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE email_otps
		 SET otp_hash = $1,
		     expires_at = $2,
		     attempts = 0,
		     verified_at = NULL,
		     updated_at = NOW()
		 WHERE user_id = $3`,
		otp.OTPHash,
		otp.ExpiresAt,
		otp.UserID,
	)

	return err
}
